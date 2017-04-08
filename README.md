# Reed-Solomon

Reed-Solomon Erasure Code engine in pure Go.

More info in [my blogs](http://www.templex.xyz/blog/101/reedsolomon.html) (in Chinese)

It's not the fastest version here, if you want to get the fastest one, please send me email:

temple3x@gmail.com

more than 5GB/s per physics core

 * Coding over in GF(2^8).
 * Primitive Polynomial: x^8 + x^4 + x^3 + x^2 + 1 (0x1d)

It released by  [Klauspost ReedSolomon](https://github.com/klauspost/reedsolomon), with some optimizations/changes:

1. Use Cauchy matrix as generator matrix. Vandermonde matrix need more operations for preserving the
property that any square subset of rows is invertible
2. There are a math tool(mathtools/gentables.go) for generator Primitive Polynomial and it's log table, exp table, multiply table,
inverse table etc. We can get more info about how galois field work
3. Use a single core to encode. If you want use more cores, please see the history of "encode.go"(it use a "pipeline mode" for encoding concurrency,
and physics cores number will be the pipeline number)
4. 16*1024 bytes(it's a half of L1 data cache size) will be the default calculation unit,
   it improve performance greatly(especially when the data shard's size is large).
5. Go1.7 have added some new instruction, and some are what we need here. The byte sequence in asm files are changed to
instructions now (unfortunately, I added some new bytes)
6. Delete inverse matrix cache part, it’s a statistical fact that only 2-3% shards need to be repaired. And calculating invert matrix is very fast,
so I don't think it will improve performance very much
7. Instead of copying data, I use maps to save position of data. Reconstruction is as fast as encoding now
8. AVX intrinsic instructions are not mixed with any of SSE instructions, so we don't need "VZEROUPPER" to avoid AVX-SSE Transition Penalties,
it seems improve performance.
9. Some of Golang's asm OP codes make me uncomfortable, especially the "MOVQ", so I use byte codes to operate the register lower part sometimes.
I still need time to learn the golang asm more. (Thanks to [asm2plan9s](https://github.com/fwessels/asm2plan9s))
10. I drop the "TEST in_data is empty or not" part in asm file
11. No R8-R15 register in asm codes, because it need one more byte
12. Only import Golang standard library
13. I use loop unrolling to improve performance, it will help a little
14. I put the data memory address into two register, and make two mask. It's good for the CPU pipeline, but still can't improve much.
It depends on CPU's ports and ALU, pipeline is not a silver bullet
15. ...

# Installation
To get the package use the standard:
```bash
go get github.com/templexxx/reedsolomon
```

# Usage

This section assumes you know the basics of Reed-Solomon encoding. A good start is this [Backblaze blog post](https://www.backblaze.com/blog/reed-solomon/).

There are only two public function in the package: Encode, Reconst and NewMatrix

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
| E              |      10+4       |5558.60  |
| R              |      10+4(1)       | 19050.82 |
| R              |      10+4(2)       | 9725.64  |
| R              |      10+4(3)       | 6974.09  |
| R              |      10+4(4)      | 5528.68 |
# Links
* [Klauspost ReedSolomon](https://github.com/klauspost/reedsolomon)
* [intel ISA-L](https://github.com/01org/isa-l)
* [GF SIMD] (http://www.ssrc.ucsc.edu/papers/plank-fast13.pdf)
