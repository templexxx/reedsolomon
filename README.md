# Reed-Solomon

[![GoDoc][1]][2] [![MIT licensed][3]][4] [![Build Status][5]][6] [![Go Report Card][7]][8] [![Sourcegraph][9]][10]

[1]: https://godoc.org/github.com/templexxx/reedsolomon?status.svg
[2]: https://godoc.org/github.com/templexxx/reedsolomon
[3]: https://img.shields.io/badge/license-MIT-blue.svg
[4]: LICENSE
[5]: https://github.com/templexxx/reedsolomon/workflows/unit-test/badge.svg
[6]: https://github.com/templexxx/reedsolomon
[7]: https://goreportcard.com/badge/github.com/templexxx/reedsolomon
[8]: https://goreportcard.com/report/github.com/templexxx/reedsolomon
[9]: https://sourcegraph.com/github.com/templexxx/reedsolomon/-/badge.svg
[10]: https://sourcegraph.com/github.com/templexxx/reedsolomon?badge


## Introduction:

>- Erasure Codes(based on Reed-Solomon Codes) engine in pure Go.
>
>- It's a kind of [Systematic Codes](https://en.wikipedia.org/wiki/Systematic_code), which means 
the input data is embedded in the encoded output .
>
>- [High Performance](https://github.com/templexxx/reedsolomon#performance): dozens GikB/s per physics core. 
>
>- High Reliability: 
>  1. At least two companies are using this library in their storage system.
    (More than dozens PB data)
>  2. Full test of galois field calculation and invertible matrices
>   (You can also find the [mathematical proof](invertible.jpg) in this repo).
>
>- Based on [Klauspost ReedSolomon](https://github.com/klauspost/reedsolomon) 
& [Intel ISA-L](https://github.com/01org/isa-l) with some additional changes/optimizations.
>
>- It's the backend of [XRS](https://github.com/templexxx/xrs) (Erasure Codes
which can save about 30% I/O in reconstruction process).

## Specification
### Math

>- Coding over in GF(2^8).
>
>- Primitive Polynomial: x^8 + x^4 + x^3 + x^2 + 1 (0x1d).
>
>- [Cauchy Matrix](matrix.go) is the generator matrix.
>   >-  Any sub-matrix of encoding matrix is invertible (See the proof [here](invertible.jpg)). 
>
>- [Galois Field Tool](mathtool/gentbls/gentbls.go): Generate primitive polynomial,
and it's log, exponent, multiply and inverse tables etc. 
>
>- [Inverse Matrices Tool](mathtool/cntinverse/cntinverse.go): Calculate the number of inverse matrices 
with specific data & parity number.
>

[XP](https://github.com/drmingdrmer) has written an excellent article ([Here, in Chinese](http://drmingdrmer.github.io/tech/distributed/2017/02/01/ec.html)) about how
Erasure Codes works and the math behind it. It's a good start to read it.

### Accelerate

>- SIMD: [Screaming Fast Galois Field Arithmetic Using Intel SIMD Instructions](http://web.eecs.utk.edu/~jplank/plank/papers/FAST-2013-GF.html)
>
>- Reduce memory I/O: Write cache-friendly code. In the process of two matrices multiply, we will have to
read data times, and keep the temporary results, then write to memory. If we could put more data into
CPU's Cache but not read/write memory again and again, the performance should
improve a lot. 
>
>- Cache inverse matrices: It'll save thousands ns, not much, but it's still meaningful
for small data.
>
>- ...

[Here](http://www.templex.xyz/blog/101/reedsolomon.html) (in Chinese) is an article about
how to write a fast Erasure Codes engine. 
(Written by me years ago, need update, but the main ideas still work)

## Performance

Performance depends mainly on:

>- CPU instruction extension.
>
>- Number of data/parity row vectors.

**Platform:** 

```
goos: linux
goarch: amd64
pkg: github.com/templexxx/reedsolomon
cpu: 12th Gen Intel(R) Core(TM) i7-12700K
```

**All test run on a single Core.**

### Encode:

`I/O = (data + parity) * vector_size / cost`

| Data | Parity | Vector size | AVX2 I/O (MiB/S) | no SIMD I/O (MiB/S) |
|------|--------|-------------|------------------|---------------------|
| 10   | 2      | 8KiB        | 35640.29         | 2226.84             |
| 10   | 2      | 1MiB        | 	30136.69        | 2214.45             |
| 10   | 4      | 8KiB        | 19936.79         | 1294.25             |
| 10   | 4      | 1MiB        | 17845.68         | 1284.02             |
| 12   | 4      | 8KiB        | 19072.93         | 1229.14             |
| 12   | 4      | 1MiB        | 16851.19         | 1219.29             |

### Reconstruct:

`I/O = (data + reconstruct_data_num) * vector_size / cost`

| Data | Parity | Vector size | Reconstruct Data Num | AVX2 I/O (MiB/s) |
|------|--------|-------------|----------------------|------------------|
| 10   | 4      | 8KiB        | 1                    | 55775.91         |
| 10   | 4      | 8KiB        | 2                    | 33037.90         |  
| 10   | 4      | 8KiB        | 3                    | 23917.16         | 
| 10   | 4      | 8KiB        | 4                    | 19363.26         | 

### Update:

`I/O = (2 + parity_num + parity_num) * vector_size / cost`

| Data | Parity | Vector size | AVX2 I/O (MiB/S) |
|------|--------|-------------|------------------|
| 10   | 4      | 8KiB        | 55710.83         |

### Replace:

`I/O = (parity_num + parity_num + replace_data_num) * vector_size / cost`

| Data | Parity | Vector size | Replace Data Num | AVX2 I/O (MiB/S) |
|------|--------|-------------|------------------|------------------|
| 10   | 4      | 8KiB        | 1                | 116193.04        |  
| 10   | 4      | 8KiB        | 2                | 65375.73         |   
| 10   | 4      | 8KiB        | 3                | 48775.47         |  
| 10   | 4      | 8KiB        | 4                | 40398.79         |     
| 10   | 4      | 8KiB        | 5                | 35262.89         |  
| 10   | 4      | 8KiB        | 6                | 31881.60         |   

**PS:**

*We must know the benchmark test is quite different with encoding/decoding in practice.
Because in benchmark test loops, the CPU Cache may help a lot.*

## Acknowledgements
>- [Klauspost ReedSolomon](https://github.com/klauspost/reedsolomon): It's the
most commonly used Erasure Codes library in Go. Impressive performance, friendly API, 
and it can support multi platforms(with fast Galois Field Arithmetic). Inspired me a lot.
>
>- [Intel ISA-L](https://github.com/01org/isa-l): The ideas of Cauchy matrix and saving memory
I/O are from it.
