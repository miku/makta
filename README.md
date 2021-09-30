# sqlikv

Turn [tabular data](https://en.wikipedia.org/wiki/Tab-separated_values) into a
lookup table in [sqlite3](https://sqlite.org/). Focus on simplicity and
performance.

## Motivation

> SQLite is likely used more than all other database engines combined. Billions
> and billions of copies of SQLite exist in the wild. -- [https://www.sqlite.org/mostdeployed.html](https://www.sqlite.org/mostdeployed.html)

Sometimes, programs need lookup tables to map values between two domains. A
dictionary is a perfect data structure as long as the data fits in memory. For
larger sets (millions of items, tens or hundreds of GB), a dictionary will not
work.

The *sliqkv* tool takes a two-column tabular file and turns it into an sqlite3
database, which you can query in your program. Depending on the size of the
data, you can expect 1K-50K queries per second.
