# SLIKV

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

![](static/439256.gif)

## How it works

Data is chopped up into smaller chunks (defaults to about 64MB) and imported with
the `.import` [command](https://www.sqlite.org/cli.html). Indexes are created
only after all data has been imported.

## Motivation

> SQLite is likely used more than all other database engines combined. Billions
> and billions of copies of SQLite exist in the wild. -- [https://www.sqlite.org/mostdeployed.html](https://www.sqlite.org/mostdeployed.html)

Sometimes, programs need lookup tables to map values between two domains. A
[dictionary](https://xlinux.nist.gov/dads/HTML/dictionary.html) is a perfect
data structure as long as the data fits in memory. For larger sets (hundreds of
millions of entries), a dictionary will not work.

The *slikv* tool takes a two-column tabular file and turns it into an sqlite3
database, which you can query in your program. Depending on the size of the
data, you can expect 1K-50K queries per second.

## Usage

```shsh
$ slikv -h
Usage of slikv:
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

$ time slikv < fixtures/sample-10m.tsv
2021/09/30 16:58:07 [ok] initialized database -- data.db
2021/09/30 16:58:17 [io] written 523M · 56.6M/s
2021/09/30 16:58:21 [ok] 1/2 created index -- data.db
2021/09/30 16:58:34 [ok] 2/2 created index -- data.db

real    0m26.267s
user    0m24.122s
sys     0m3.224s
```

* 10M rows stored, with indexed keys and values in 27s, 370370 rows/s
