# Reed-Solomon

Reed-Solomon Erasure Code engine in pure Go.

more than 5GB/s per physics core, almost as fast as Intel ISA-L

10+4 encode run on AWS t2.micro Intel(R) Xeon(R) CPU E5-2676 v3 @ 2.40GHz, Memory 1GB, ubuntu-trusty-16.04-amd64:

xxx is my code

intel is Intel ISA-L

![isal](http://templex.xyz/images/reedsolomon/isal.jpg)

More info in [my blogs](http://www.templex.xyz/blog/101/reedsolomon.html) (in Chinese)

It's not the fastest version here, if you want to get the fastest one, please send me email (I'm sorry for that, I wouldn't do this if I didn't have to):

temple3x@gmail.com

 * Coding over in GF(2^8).
 * Primitive Polynomial: x^8 + x^4 + x^3 + x^2 + 1 (0x1d)

It released by  [Klauspost ReedSolomon](https://github.com/klauspost/reedsolomon), with some optimizations/changes:

1. Use Cauchy matrix as generator matrix. Vandermonde matrix need more operations for preserving the property that any square subset of rows is invertible
2. There are a math tool(mathtools/gentables.go) for generator Primitive Polynomial and it's log table, exp table, multiply table, inverse table etc. We can get more info about how galois field work
3. Use a single core to encode
4. New Go version have added some new instruction, and some are what we need here. The byte sequence in asm files are changed to instructions now (unfortunately, I added some new bytes codes)
5. Delete inverse matrix cache part, it’s a statistical fact that only 2-3% shards need to be repaired. And calculating invert matrix is very fast, so I don't think it will improve performance very much
6. Instead of copying data, I use maps to save position of data. Reconstruction is as fast as encoding now
7. AVX intrinsic instructions are not mixed with any of SSE instructions, so we don't need "VZEROUPPER" to avoid AVX-SSE Transition Penalties, it seems improve performance.
8. Some of Golang's asm OP codes make me uncomfortable, especially the "MOVQ", so I use byte codes to operate the register lower part sometimes. (Thanks to [asm2plan9s](https://github.com/fwessels/asm2plan9s))
9. I drop the "TEST in_data is empty or not" part in asm file
10. No R8-R15 register in asm codes, because it need one more byte
11. Only import Golang standard library
12. ...

# Installation
To get the package use the standard:
```bash
go get github.com/templexxx/reedsolomon
```

# Usage

This section assumes you know the basics of Reed-Solomon encoding. A good start is this [Backblaze blog post](https://www.backblaze.com/blog/reed-solomon/).

There are only three public function in the package: Encode, Reconst and NewMatrix

NewMatrix: return a [][]byte for encode and reconst

Encode : calculate parity of data shards;

Reconst: calculate data or parity from present shards;

# Performance
Performance depends mainly on:

1. number of parity shards
2. number of cores of CPU (if you want to use parallel version)
3. CPU instruction extension(AVX2 or SSSE3)
4. unit size of calculation
5. size of shards
6. speed of memory(waste so much time on read/write mem, :D )

Example of performance on my MacBook 2014-mid(i5-4278U 2.6GHz 2 physical cores). The 128KB per shards.
Single core work here(fast version):

| Encode/Reconst | data+parity/data+parity(lost_data)   | Speed (MB/S) |
|----------------|-------------------|--------------|
| E              |      10+4       |5849.41  |
| R              |      10+4(1)       | 19050.82 |
| R              |      10+4(2)       | 9725.64  |
| R              |      10+4(3)       | 6974.09  |
| R              |      10+4(4)      | 5528.68 |
# Links
* [Klauspost ReedSolomon](https://github.com/klauspost/reedsolomon)
* [intel ISA-L](https://github.com/01org/isa-l)
* [GF SIMD] (http://www.ssrc.ucsc.edu/papers/plank-fast13.pdf)
