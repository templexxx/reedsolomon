# Reed-Solomon

[![GoDoc][1]][2] [![MIT licensed][3]][4] [![Build Status][5]][6] [![Go Report Card][7]][8] 

[1]: https://godoc.org/github.com/templexxx/reedsolomon?status.svg
[2]: https://godoc.org/github.com/templexxx/reedsolomon
[3]: https://img.shields.io/badge/license-MIT-blue.svg
[4]: LICENSE
[5]: https://travis-ci.org/templexxx/reedsolomon.svg?branch=master
[6]: https://travis-ci.org/templexxx/reedsolomon
[7]: https://goreportcard.com/badge/github.com/templexxx/reedsolomon
[8]: https://goreportcard.com/report/github.com/templexxx/reedsolomon

## Introduction:
1.  Reed-Solomon Erasure Code engine in pure Go.(Based on [intel ISA-L](https://github.com/01org/isa-l) & [Klauspost ReedSolomon](https://github.com/klauspost/reedsolomon))
2.  Fast: more than 10GB/s per physics core

## Installation
To get the package use the standard:
```bash
go get github.com/templexxx/reedsolomon
```

## Documentation
See the associated [GoDoc](http://godoc.org/github.com/templexxx/reedsolomon)

## Specification
### GOARCH
1. All arch are supported
2. Go1.11(for AVX512)

### Math
1. Coding over in GF(2^8)
2. Primitive Polynomial: x^8 + x^4 + x^3 + x^2 + 1 (0x1d)
3. mathtool/gentbls.go : generator Primitive Polynomial and it's log table, exp table, multiply table, inverse table etc. We can get more info about how galois field work
4. mathtool/cntinverse.go : calculate how many inverse matrix will have in different RS codes config
5. Cauchy Matrix is generator matrix

### Why so fast?
These three parts will cost too much time:

1. lookup galois-field tables
2. read/write memory
3. calculate inverse matrix in the reconstruct process

SIMD will solve no.1

Cache-friendly codes will help to solve no.2 & no.3, and more, use a sync.Map for cache inverse matrix, it will help to save about 1000ns when we need same matrix. 

## Performance

Performance depends mainly on:

1. CPU instruction extension( AVX512 or AVX2)
2. number of data/parity vects
3. unit size of calculation ( see it in rs.go )
4. size of shards
5. speed of memory (waste so much time on read/write mem, :D )
6. performance of CPU
7. the way of using (reuse memory)

And we must know the benchmark test is quite different with encoding/decoding in practice.

Because in benchmark test loops, the CPU Cache will help a lot. In practice, we must reuse the memory to make the performance become as good as the benchmark test.

Example of performance on my AWS c5d.large (Intel(R) Xeon(R) Platinum 8124M CPU @ 3.00GHz.)
DataCnt = 10; ParityCnt = 4

### Encoding:

| Vector size | AVX512 (MB/S) | AVX2 (MB/S) |
|-------------|---------------|-------------|
| 4KB         |       12775   |    9174     |
| 64KB        |       11618   |    8964     |
| 1MB         |      7918     |    6820     |

## Links & Thanks
* [Klauspost ReedSolomon](https://github.com/klauspost/reedsolomon)
* [intel ISA-L](https://github.com/01org/isa-l)
* [GF SIMD] (http://www.ssrc.ucsc.edu/papers/plank-fast13.pdf)
* [asm2plan9s] (https://github.com/fwessels/asm2plan9s)
