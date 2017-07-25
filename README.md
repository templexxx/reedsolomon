# Reed-Solomon

## Branch

**Master** : accept any size of data

**0.1**    : only accept 256*x or 16*x byte per data shard

## Reed-Solomon Erasure Code engine in pure Go.

more than 5GB/s per physics core, almost as fast as Intel ISA-L

faster than ISA-L when the data_size is small

but slower if the data size is big ( in the my EC2, 10+4 encode, the "big" will be 20MB)

**My EC2**:

AWS t2.micro Intel(R) Xeon(R) CPU E5-2676 v3 @ 2.40GHz, Memory 1GB, ubuntu-trusty-16.04-amd64

## About Decode

We can have many ways to write codes about decoding(it call reconst here). It will be good for our system, but maybe it's not the best way for your system.
You can make a new decoding through encoding, and it's not hard, if you need help, here is my email:

temple3x@gmail.com

## More Info

More info in [my blogs](http://www.templex.xyz/blog/101/reedsolomon.html) (in Chinese)

 * Coding over in GF(2^8).
 * Primitive Polynomial: x^8 + x^4 + x^3 + x^2 + 1 (0x1d)

It released by  [Klauspost ReedSolomon](https://github.com/klauspost/reedsolomon), with some optimizations/changes:

1. Use Cauchy matrix as generator matrix. Vandermonde matrix need more operations for preserving the property that any square subset of rows is invertible
2. There are a math tool(mathtool/gentables.go) for generator Primitive Polynomial and it's log table, exp table, multiply table, inverse table etc. We can get more info about how galois field work
3. Use a single core to encode
4. New Go version have added some new instruction, and some are what we need here. The byte sequence in asm files are changed to instructions now (unfortunately, I added some new bytes codes)
5. Delete inverse matrix cache part, it’s a statistical fact that only 2-3% shards need to be repaired. And calculating invert matrix is very fast, so I don't think it will improve performance very much
6. Instead of copying data, I use maps to save position of data. Reconstruction is as fast as encoding now
7. AVX intrinsic instructions are not mixed with any of SSE instructions, so we don't need "VZEROUPPER" to avoid AVX-SSE Transition Penalties, it seems improve performance.
8. Some of Golang's asm OP codes make me uncomfortable, especially the "MOVQ", so I use byte codes to operate the register lower part sometimes. (Thanks to [asm2plan9s](https://github.com/fwessels/asm2plan9s))
9. I drop the "TEST in_data is empty or not" part in asm file
10. No R8-R15 register in asm codes, because it need one more byte
11. Only import Golang standard library
12. More Cache-friendly
13. ...

Actually, mine is almost entirely different with Klauspost's. But his' do inspire me

## Installation
To get the package use the standard:
```bash
go get github.com/templexxx/reedsolomon
```

## Usage

The most important part of these codes are :

1. Encode
2. Reconst
3. Matrix
4. Check args

You need check args before sending data to the engine

all data & parity are store in Matrix([][]byte)

**warning:**

the shard size must be integral multiple of 256B (avx2) or 16B (ssse3), in practice, we always have a fixed shard size,
and I'm too lazy to make it flexible

sorry about that :D


## Performance

Performance depends mainly on:

1. CPU instruction extension(AVX2 or SSSE3)
2. number of data/parity shards
3. unit size of calculation (see it in encode.go)
4. size of shards
5. speed of memory(waste so much time on read/write mem, :D )
6. performance of CPU
7. the way of using

And we must know the benchmark test is quite different with encoding in practice.

Because in benchmark test loops, the CPU Cache will help a lot. We must reuse the
memory space well to make the performance as good as the benchmark test.

Example of performance on my MacBook 2014-mid(i5-4278U 2.6GHz 2 physical cores). 10+4.
Single core work here(avx2):

| Shard size | Speed (MB/S) |
|----------------|--------------|
| 1KB              |5670.41  |
| 2KB             |   6341.92 |
| 4KB              |    6660.46  |
| 8KB              |       6259.54  |
| 16KB              |     6301.48 |
| 32KB              |     5922.17 |
| 64KB              |       5875.41 |
| 128KB              |       5688.15 |
| 256KB              |      4973.67 |
| 512KB              |       4406.47 |
| 1MB              |      4570.09 |
| 2MB              |      4669.23 |
| 4MB              |      4548.71 |
| 8MB              |      4552.08 |
| 16MB              |      4479.27 |
| 32MB              |      4613.49 |

## Links & Thanks
* [Klauspost ReedSolomon](https://github.com/klauspost/reedsolomon)
* [intel ISA-L](https://github.com/01org/isa-l)
* [GF SIMD] (http://www.ssrc.ucsc.edu/papers/plank-fast13.pdf)
