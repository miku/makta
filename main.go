// sqlikv takes two columns and turns it into an indexed sqlite3 database.
package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	Version   string
	Buildtime string

	showVersion = flag.Bool("version", false, "show version and exit")
	outputFile  = flag.String("o", "data.db", "output filename")
	bufferSize  = flag.Int("B", 64*1<<20, "buffer size")
	indexMode   = flag.Int("I", 3, "index mode: 0=none, 1=k, 2=v, 3=kv")

	initSQL = `
CREATE TABLE IF NOT EXISTS map
(
	k TEXT,
	v TEXT
);
`
	keyIndexSQL = `
PRAGMA journal_mode = OFF;
PRAGMA synchronous = 0;
PRAGMA cache_size = 1000000;
PRAGMA locking_mode = EXCLUSIVE;
CREATE INDEX IF NOT EXISTS idx_k ON map(k);
`
	valueIndexSQL = `
PRAGMA journal_mode = OFF;
PRAGMA synchronous = 0;
PRAGMA cache_size = 1000000;
PRAGMA locking_mode = EXCLUSIVE;

CREATE INDEX IF NOT EXISTS idx_v ON map(v);
`
	importSQL = `
PRAGMA journal_mode = OFF;
PRAGMA synchronous = 0;
PRAGMA cache_size = 1000000;
PRAGMA locking_mode = EXCLUSIVE;
PRAGMA temp_store = MEMORY;

.mode tabs
.import /dev/stdin map
`
)

// TempFileReader returns path to temporary file with contents from reader.
func TempFileReader(r io.Reader) (string, error) {
	f, err := ioutil.TempFile("", "")
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(f, r); err != nil {
		return "", err
	}
	if err := f.Close(); err != nil {
		return "", err
	}
	return f.Name(), nil
}

// ByteSize returns a human-readable byte string of the form 10M, 12.5K, and so
// forth.  The following units are available: E: Exabyte, P: Petabyte, T:
// Terabyte, G: Gigabyte, M: Megabyte, K: Kilobyte, B: Byte, The unit that
// results in the smallest number greater than or equal to 1 is always chosen.
func ByteSize(bytes int) string {
	const (
		BYTE = 1 << (10 * iota)
		KB
		MB
		GB
		TB
		PB
		EB
	)
	var (
		u      = ""
		v      = float64(bytes)
		result string
	)
	switch {
	case bytes >= EB:
		u = "E"
		v = v / EB
	case bytes >= PB:
		u = "P"
		v = v / PB
	case bytes >= TB:
		u = "T"
		v = v / TB
	case bytes >= GB:
		u = "G"
		v = v / GB
	case bytes >= MB:
		u = "M"
		v = v / MB
	case bytes >= KB:
		u = "K"
		v = v / KB
	case bytes >= BYTE:
		u = "B"
	case bytes == 0:
		return "0B"
	}
	result = strconv.FormatFloat(v, 'f', 1, 64)
	result = strings.TrimSuffix(result, ".0")
	return result + u
}

func runScript(path, script, message string) error {
	cmd := exec.Command("sqlite3", path)
	cmd.Stdin = strings.NewReader(script)
	err := cmd.Run()
	if err == nil {
		log.Printf("[ok] %s -- %s", message, path)
	}
	return err
}

func runImport(r io.Reader, initFile, outputFile string) (int64, error) {
	cmd := exec.Command("sqlite3", "--init", initFile, outputFile)
	cmdStdin, err := cmd.StdinPipe()
	if err != nil {
		return 0, err
	}
	var (
		wg      sync.WaitGroup
		copyErr error
		written int64
	)
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer cmdStdin.Close()
		n, err := io.Copy(cmdStdin, r)
		if err != nil {
			copyErr = fmt.Errorf("copy failed: %w", err)
		}
		written += n
	}()
	if _, err := cmd.CombinedOutput(); err != nil {
		return written, fmt.Errorf("exec failed: %w", err)
	}
	wg.Wait()
	return written, copyErr
}

// Flushf for messages that should stay on a single line.
func Flushf(s string, vs ...interface{}) {
	// 2021/09/29 15:38:05
	t := time.Now().Format("2006/01/02 15:04:05")
	msg := fmt.Sprintf("\r"+t+" [io] "+s, vs...)
	fmt.Printf("\r" + strings.Repeat(" ", len(msg)+1))
	fmt.Printf(msg)
}

// HumanSpeed returns a human readable throughput number, e.g. 10MB/s,
// 12.3kB/s, etc.
func HumanSpeed(bytesWritten int64, elapsedSeconds float64) string {
	speed := float64(bytesWritten) / elapsedSeconds
	return fmt.Sprintf("%s/s", ByteSize(int(speed)))
}

func main() {
	flag.Parse()
	if *showVersion {
		fmt.Printf("mkocidb %s %s\n", Version, Buildtime)
		os.Exit(0)
	}
	if _, err := os.Stat(*outputFile); os.IsNotExist(err) {
		if err := runScript(*outputFile, initSQL, "initialized database"); err != nil {
			log.Fatal(err)
		}
	}
	runFile, err := TempFileReader(strings.NewReader(importSQL))
	if err != nil {
		log.Fatal(err)
	}
	var (
		br          = bufio.NewReader(os.Stdin)
		buf         bytes.Buffer
		written     int64
		started     = time.Now()
		elapsed     float64
		importBatch = func() error {
			n, err := runImport(&buf, runFile, *outputFile)
			if err != nil {
				return err
			}
			written += n
			elapsed = time.Since(started).Seconds()
			Flushf("written %s · %s", ByteSize(int(written)), HumanSpeed(written, elapsed))
			return nil
		}
	)
	for {
		b, err := br.ReadBytes('\n')
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}
		if _, err := buf.Write(b); err != nil {
			log.Fatal(err)
		}
		if buf.Len() >= *bufferSize {
			if err := importBatch(); err != nil {
				log.Fatal(err)
			}
		}
	}
	if err := importBatch(); err != nil {
		log.Fatal(err)
	}
	fmt.Println()
	var indexScripts []string
	switch *indexMode {
	case 1:
		indexScripts = append(indexScripts, keyIndexSQL)
	case 2:
		indexScripts = append(indexScripts, valueIndexSQL)
	case 3:
		indexScripts = append(indexScripts, keyIndexSQL)
		indexScripts = append(indexScripts, valueIndexSQL)
	default:
		log.Printf("no index requested")
	}
	for i, script := range indexScripts {
		msg := fmt.Sprintf("%d/%d created index", i+1, len(indexScripts))
		if err := runScript(*outputFile, script, msg); err != nil {
			log.Fatal(err)
		}
	}
}
