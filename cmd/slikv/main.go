// slikv takes two columns and turns it into an indexed sqlite3 database.
package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/andrew-d/go-termutil"
	"github.com/miku/sqlikv"
)

var (
	Version   string
	Buildtime string

	showVersion = flag.Bool("version", false, "show version and exit")
	outputFile  = flag.String("o", "data.db", "output filename")
	bufferSize  = flag.Int("B", 64*1<<20, "buffer size")
	indexMode   = flag.Int("I", 3, "index mode: 0=none, 1=k, 2=v, 3=kv")
	cacheSize   = flag.Int("C", 100000, "sqlite3 cache size, needs memory = C x page size")
)

func main() {
	var (
		err     error
		runFile string
	)
	flag.Parse()
	var (
		pragma = fmt.Sprintf(`
PRAGMA journal_mode = OFF;
PRAGMA synchronous = 0;
PRAGMA cache_size = %d;
PRAGMA locking_mode = EXCLUSIVE;`, *cacheSize)
		initSQL = `
CREATE TABLE IF NOT EXISTS map (k TEXT, v TEXT);`
		keyIndexSQL = fmt.Sprintf(`
%s
CREATE INDEX IF NOT EXISTS idx_k ON map(k);`, pragma)
		valueIndexSQL = fmt.Sprintf(`
%s
CREATE INDEX IF NOT EXISTS idx_v ON map(v);`, pragma)
		importSQL = fmt.Sprintf(`
%s
PRAGMA temp_store = MEMORY;

.mode tabs
.import /dev/stdin map`, pragma)
	)

	if *showVersion {
		fmt.Printf("sqlikv %s %s\n", Version, Buildtime)
		os.Exit(0)
	}
	if termutil.Isatty(os.Stdin.Fd()) {
		log.Println("stdin: no data")
		os.Exit(0)
	}
	if _, err := os.Stat(*outputFile); os.IsNotExist(err) {
		if err := sqlikv.RunScript(*outputFile, initSQL, "initialized database"); err != nil {
			log.Fatal(err)
		}
	}
	if runFile, err = sqlikv.TempFileReader(strings.NewReader(importSQL)); err != nil {
		log.Fatal(err)
	}
	var (
		br          = bufio.NewReader(os.Stdin)
		buf         bytes.Buffer
		written     int64
		started     = time.Now()
		elapsed     float64
		importBatch = func() error {
			n, err := sqlikv.RunImport(&buf, runFile, *outputFile)
			if err != nil {
				return err
			}
			written += n
			elapsed = time.Since(started).Seconds()
			sqlikv.Flushf("written %s · %s",
				sqlikv.ByteSize(int(written)),
				sqlikv.HumanSpeed(written, elapsed))
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
		if err := sqlikv.RunScript(*outputFile, script, msg); err != nil {
			log.Fatal(err)
		}
	}
}
