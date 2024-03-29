# MAKTA

> **mak**e a database from **ta**bular data

[![DOI](https://zenodo.org/badge/412089136.svg)](https://zenodo.org/badge/latestdoi/412089136)

![](static/table.jpg)

Turn [tabular data](https://en.wikipedia.org/wiki/Tab-separated_values) into a
lookup table using [sqlite3](https://sqlite.org/). This is a working PROTOTYPE
with limitations, e.g. no customizations, the table definition is fixed, etc.

> CREATE TABLE IF NOT EXISTS map (k TEXT, v TEXT)

As a performance data point, an example dataset with 1B+ rows can be inserted
and indexed in less than two hours (on a [recent
CPU](https://ark.intel.com/content/www/us/en/ark/products/122589/intel-core-i7-8550u-processor-8m-cache-up-to-4-00-ghz.html)
and an [nvme](https://en.wikipedia.org/wiki/NVM_Express) drive; database file
size: 400G).

![](static/443238.gif)

## Installation

> [https://github.com/miku/makta/releases](https://github.com/miku/makta/releases)

```sh
$ go install github.com/miku/makta/cmd/makta@latest
```

## How it works

Data is chopped up into smaller chunks (defaults to about 64MB) and imported with
the `.import` [command](https://www.sqlite.org/cli.html). Indexes are created
only after all data has been imported.

## Example

```sh
$ cat fixtures/sample-xs.tsv | column -t
10.1001/10-v4n2-hsf10003                    10.1177/003335490912400218
10.1001/10-v4n2-hsf10003                    10.1097/01.bcr.0000155527.76205.a2
10.1001/amaguidesnewsletters.1996.novdec01  10.1056/nejm199312303292707
10.1001/amaguidesnewsletters.1996.novdec01  10.1016/s0363-5023(05)80265-5
10.1001/amaguidesnewsletters.1996.novdec01  10.1001/jama.1994.03510440069036
10.1001/amaguidesnewsletters.1997.julaug01  10.1097/00007632-199612150-00003
10.1001/amaguidesnewsletters.1997.mayjun01  10.1164/ajrccm/147.4.1056
10.1001/amaguidesnewsletters.1997.mayjun01  10.1136/thx.38.10.760
10.1001/amaguidesnewsletters.1997.mayjun01  10.1056/nejm199507133330207
10.1001/amaguidesnewsletters.1997.mayjun01  10.1378/chest.88.3.376

$ makta -o xs.db < fixtures/sample-xs.tsv
2021/10/04 16:13:06 [ok] initialized database · xs.db
2021/10/04 16:13:06 [io] written 679B · 361.3K/s
2021/10/04 16:13:06 [ok] 1/2 created index · xs.db
2021/10/04 16:13:06 [ok] 2/2 created index · xs.db

$ sqlite3 xs.db 'select * from map'
10.1001/10-v4n2-hsf10003|10.1177/003335490912400218
10.1001/10-v4n2-hsf10003|10.1097/01.bcr.0000155527.76205.a2
10.1001/amaguidesnewsletters.1996.novdec01|10.1056/nejm199312303292707
10.1001/amaguidesnewsletters.1996.novdec01|10.1016/s0363-5023(05)80265-5
10.1001/amaguidesnewsletters.1996.novdec01|10.1001/jama.1994.03510440069036
10.1001/amaguidesnewsletters.1997.julaug01|10.1097/00007632-199612150-00003
10.1001/amaguidesnewsletters.1997.mayjun01|10.1164/ajrccm/147.4.1056
10.1001/amaguidesnewsletters.1997.mayjun01|10.1136/thx.38.10.760
10.1001/amaguidesnewsletters.1997.mayjun01|10.1056/nejm199507133330207
10.1001/amaguidesnewsletters.1997.mayjun01|10.1378/chest.88.3.376

$ sqlite3 xs.db 'select * from map where k = "10.1001/amaguidesnewsletters.1997.mayjun01" '
10.1001/amaguidesnewsletters.1997.mayjun01|10.1164/ajrccm/147.4.1056
10.1001/amaguidesnewsletters.1997.mayjun01|10.1136/thx.38.10.760
10.1001/amaguidesnewsletters.1997.mayjun01|10.1056/nejm199507133330207
10.1001/amaguidesnewsletters.1997.mayjun01|10.1378/chest.88.3.376
```

## Motivation

> SQLite is likely used more than all other database engines combined. Billions
> and billions of copies of SQLite exist in the wild. -- [https://www.sqlite.org/mostdeployed.html](https://www.sqlite.org/mostdeployed.html)

Sometimes, programs need lookup tables to map values between two domains. A
[dictionary](https://xlinux.nist.gov/dads/HTML/dictionary.html) is a perfect
data structure as long as the data fits in memory. For larger sets (hundreds of
millions of entries), a dictionary may not work.

The *makta* tool currently takes a two-column TSV and turns it into an sqlite3
database, which you can query in your program. Depending on a couple of
factors, you maybe be able to query the lookup database with about 1-50K
queries per second.

Finally, sqlite3 is just an awesome database and [recommeded storage
format](https://www.sqlite.org/locrsf.html).

## Usage

```sh
$ makta -h
Usage of makta:
  -B int
        buffer size (default 67108864)
  -C int
        sqlite3 cache size, needs memory = C x page size (default 1000000)
  -I int
        index mode: 0=none, 1=k, 2=v, 3=kv (default 3)
  -o string
        output filename (default "data.db")
  -version
        show version and exit
```

## Performance

```sh
$ wc -l fixtures/sample-10m.tsv
10000000 fixtures/sample-10m.tsv

$ stat --format "%s" fixtures/sample-10m.tsv
548384897

$ time makta < fixtures/sample-10m.tsv
2021/09/30 16:58:07 [ok] initialized database -- data.db
2021/09/30 16:58:17 [io] written 523M · 56.6M/s
2021/09/30 16:58:21 [ok] 1/2 created index -- data.db
2021/09/30 16:58:34 [ok] 2/2 created index -- data.db

real    0m26.267s
user    0m24.122s
sys     0m3.224s
```

* 10M rows stored, with indexed keys and values in 27s, 370370 rows/s

## TODO

* [ ] allow tab-importing to be done programmatically, for any number of columns
* [x] a better name: mktabdb, mktabs, dbize - go with makta for now
* [ ] could write a tool for *burst* queries, e.g. split data into N shard,
      create N databases and distribute queries across files - e.g. `dbize db.json`
      with the same repl, etc. -- if we've seen 300K inserts per db, we may see 0.X x CPU x 300K, maybe millions/s.

## Design ideas

A design that works with 50M rows per database, e.g. 20 files for 1B rows;
grouped under a single directory. Every interaction only involves the
directory, not the individual files.

