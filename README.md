# Minimal Perfect Hashing for Go

This library provides [Minimal Perfect Hashing](http://en.wikipedia.org/wiki/Perfect_hash_function) (MPH) using the [Compress, Hash and Displace](http://cmph.sourceforge.net/papers/esa09.pdf) (CHD) algorithm.

## keegancsmith/mph FORK NOTE

Upstream is https://github.com/alecthomas/mph

2020-05-01: This is an experiment to see how this library behaves when specialised for uint64 keys and values. It seems to be quite promising if you can afford to build the maps:

``` shellsession
go test -run TestMemoryUsage -v
=== RUN   TestMemoryUsage
    TestMemoryUsage: chd_test.go:192: CHD of size 235886 uses 4016614 bytes
    TestMemoryUsage: chd_test.go:201: Map of size 235886 uses 10043448 bytes
--- PASS: TestMemoryUsage (0.38s)
PASS
ok      github.com/alecthomas/mph       1.317s
$ go test -count 10 -run '^$' -bench=. -benchmem > benchmem.txt
$ benchstat benchmem.txt
name               time/op
BuiltinMap-8       43.5ns ± 0%
BuiltinMapBuild-8  10.5ms ± 1%
CHD-8              30.2ns ± 8%
CHDBuild-8          366ms ± 6%

name               alloc/op
BuiltinMap-8        0.00B
BuiltinMapBuild-8  10.0MB ± 0%
CHD-8               0.00B
CHDBuild-8         84.4MB ± 2%

name               allocs/op
BuiltinMap-8         0.00
BuiltinMapBuild-8    14.0 ± 0%
CHD-8                0.00
CHDBuild-8          3.79M ± 6%
```


## What is this useful for?

Primarily, extremely efficient access to potentially very large static datasets, such as geographical data, NLP data sets, etc.

On my 2012 vintage MacBook Air, a benchmark against a wikipedia index with 300K keys against a 2GB TSV dump takes about ~200ns per lookup.

## How would it be used?

Typically, the table would be used as a fast index into a (much) larger data set, with values in the table being file offsets or similar.

The tables can be serialized. Numeric values are written in little endian form.

## Example code

Building and serializing an MPH hash table (error checking omitted for clarity):

```go
b := mph.Builder()
for k, v := range data {
    b.Add(k, v)
}
h, _ := b.Build()
w, _ := os.Create("data.idx")
_ := h.Write(w)
```

Deserializing the hash table and performing lookups:

```go
r, _ := os.Open("data.idx")
h, _ := mph.Read(r)

v := h.Get([]byte("some key"))
if v == nil {
    // Key not found
}
```

MMAP is also indirectly supported, by deserializing from a byte slice and slicing the keys and values.

The [API documentation](http://godoc.org/github.com/alecthomas/mph) has more details.
