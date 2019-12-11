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
>- [High Performance](https://github.com/templexxx/reedsolomon#performance): More than 15GB/s per physics core. 
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
>   >-  Any submatrix of encoding matrix is invertible (See the proof [here](invertible.jpg)). 
>
>- [Galois Field Tool](mathtool/gentbls.go): Generate primitive polynomial 
and it's log, exponent, multiply and inverse tables etc. 
>
>- [Inverse Matrices Tool](mathtool/combi.go): Calculate the number of inverse matrices 
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

*AWS c5d.xlarge (Intel(R) Xeon(R) Platinum 8124M CPU @ 3.00GHz)*

**All test run on a single Core.**

### Encode:

`I/O = (data + parity) * vector_size / cost`

*Base means no SIMD.*

| Data  | Parity  | Vector size | AVX512 I/O (MB/S) |  AVX2 I/O (MB/S) |Base I/O (MB/S) |
|-------|---------|-------------|-------------|---------------|---------------|
|10|2|4KB|       29683.69   |    21371.43      |   910.45       |
|10|2|1MB|     17664.67    |    	15505.58      |   917.26       |
|10|2|8MB|      10363.05    |      9323.60     |    914.62      |
|10|4|4KB|      17708.62    |      12705.35    |    531.82      |
|10|4|1MB|     11970.42    |     9804.57     |  536.31        |
|10|4|8MB|      7957.9    |      6941.69     |    534.82      |
|12|4|4KB|      16902.12    |       12065.14   |  511.95        |
|12|4|1MB|      11478.86   |   9392.33       |   514.24       |
|12|4|8MB|       7949.81   |   6760.49        |    513.06      |

### Reconstruct:

`I/O = (data + reconstruct_data_num) * vector_size / cost`

| Data  | Parity  | Vector size | Reconstruct Data Num |  AVX512 I/O (MB/S) |
|-------|---------|-------------|-------------|---------------|
|10|4|4KB| 1         |      29830.36    |
|10|4|4KB| 2        |     21649.61     |  
|10|4|4KB| 3         |     17088.41      | 
|10|4|4KB| 4         |    14567.26       | 

### Update:

`I/O = (2 + parity_num + parity_num) * vector_size / cost`

| Data  | Parity  | Vector size | AVX512 I/O (MB/S) |
|-------|---------|-------------|-------------|
|10|4|4KB|      36444.13    |

### Replace:

`I/O = (parity_num + parity_num + replace_data_num) * vector_size / cost`

| Data  | Parity  | Vector size | Replace Data Num |  AVX512 I/O (MB/S) |
|-------|---------|-------------|-------------|---------------|
|10|4|4KB| 1         |  78464.33        |  
|10|4|4KB| 2        |     50068.71     |   
|10|4|4KB| 3         |   38808.11        |  
|10|4|4KB| 4         |    32457.60       |     
|10|4|4KB| 5         |  28679.46         |  
|10|4|4KB| 6         |    26151.85       |   

**PS:**

*And we must know the benchmark test is quite different with encoding/decoding in practice.
Because in benchmark test loops, the CPU Cache may help a lot.*

## Links & Thanks
>- [Klauspost ReedSolomon](https://github.com/klauspost/reedsolomon): It's the
most commonly used Erasure Codes library in Go. Impressive performance, friendly API, 
and it can support multi platforms(with fast Galois Field Arithmetic). Inspired me a lot.
>
>- [Intel ISA-L](https://github.com/01org/isa-l): The ideas of Cauchy matrix and saving memory
I/O are from it.
