# reedsolomon

[![pkg.go.dev](https://pkg.go.dev/badge/github.com/templexxx/reedsolomon.svg)](https://pkg.go.dev/github.com/templexxx/reedsolomon)
[![MIT License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Unit Test](https://github.com/templexxx/reedsolomon/actions/workflows/unit-test.yml/badge.svg)](https://github.com/templexxx/reedsolomon/actions/workflows/unit-test.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/templexxx/reedsolomon)](https://goreportcard.com/report/github.com/templexxx/reedsolomon)
[![Sourcegraph](https://sourcegraph.com/github.com/templexxx/reedsolomon/-/badge.svg)](https://sourcegraph.com/github.com/templexxx/reedsolomon?badge)

A high-performance, systematic Reed-Solomon erasure coding engine in pure Go.

This repository focuses on two goals:
- mathematically sound coding over `GF(2^8)`
- low-latency, high-throughput implementation for storage systems

## Why This Library

- Pure Go implementation with optional AVX2 acceleration on x86.
- Systematic code layout: original data vectors are embedded directly in the output stripe.
- Cauchy-based encoding matrix with invertibility proof included in this repo.
- Production-oriented APIs: `Encode`, `Reconst`, `Update`, and `Replace`.
- Extensive tests for finite-field arithmetic, matrix operations, and end-to-end correctness.

## Install

```bash
go get github.com/templexxx/reedsolomon
```

## Quick Start

```go
package main

import (
	"fmt"

	rs "github.com/templexxx/reedsolomon"
)

func main() {
	const (
		dataNum   = 10
		parityNum = 4
		size      = 8 * 1024
	)

	codec, err := rs.New(dataNum, parityNum)
	if err != nil {
		panic(err)
	}

	// Stripe layout: [data vectors..., parity vectors...]
	vects := make([][]byte, dataNum+parityNum)
	for i := range vects {
		vects[i] = make([]byte, size)
	}

	// Fill data vectors [0:dataNum) with your payload.
	for i := 0; i < dataNum; i++ {
		for j := 0; j < size; j++ {
			vects[i][j] = byte(i + j)
		}
	}

	// 1) Encode parity vectors.
	if err := codec.Encode(vects); err != nil {
		panic(err)
	}

	// 2) Reconstruct lost vectors (example: data #1, parity #11).
	lost := []int{1, 11}
	for _, idx := range lost {
		for i := range vects[idx] {
			vects[idx][i] = 0
		}
	}
	survived := []int{0, 2, 3, 4, 5, 6, 7, 8, 9, 10, 12, 13}
	if err := codec.Reconst(vects, survived, lost); err != nil {
		panic(err)
	}

	fmt.Println("encode + reconstruct succeeded")
}
```

## API Overview

- `Encode(vects [][]byte)`
  - Generates parity vectors from data vectors.
- `Reconst(vects [][]byte, survived []int, needReconst []int)`
  - Reconstructs missing data/parity vectors from surviving vectors.
- `Update(oldData, newData []byte, row int, parity [][]byte)`
  - Incrementally updates parity when one data vector changes.
- `Replace(data [][]byte, replaceRows []int, parity [][]byte)`
  - Efficiently updates parity for replacing multiple data rows.

## Mathematical Foundation

- Field: `GF(2^8)`
- Primitive polynomial: `x^8 + x^4 + x^3 + x^2 + 1` (`0x1d`)
- Encoding matrix:
  - upper part is identity matrix (systematic form)
  - lower part is Cauchy matrix
- Invertibility proof for reconstruction matrix:
  - [proof_invertible.md](proof_invertible.md)

Reference tools in this repo:
- Galois-field table generator: [`mathtool/gentbls/gentbls.go`](mathtool/gentbls/gentbls.go)
- Invertible-matrix counting tool: [`mathtool/cntinverse/cntinverse.go`](mathtool/cntinverse/cntinverse.go)

## Performance

Performance depends on:
- CPU instruction set support (AVX2 vs non-SIMD)
- data/parity layout (`k + m`)
- vector size and cache behavior

Benchmark platform:

```text
goos: linux
goarch: amd64
cpu: 12th Gen Intel(R) Core(TM) i7-12700K
```

All numbers below are single-core results.

### Encode Throughput

`I/O = (data + parity) * vector_size / cost`

| Data | Parity | Vector size | AVX2 (MiB/s) | No SIMD (MiB/s) |
|------|--------|-------------|--------------|-----------------|
| 10   | 2      | 8KiB        | 35640.29     | 2226.84         |
| 10   | 2      | 1MiB        | 30136.69     | 2214.45         |
| 10   | 4      | 8KiB        | 19936.79     | 1294.25         |
| 10   | 4      | 1MiB        | 17845.68     | 1284.02         |
| 12   | 4      | 8KiB        | 19072.93     | 1229.14         |
| 12   | 4      | 1MiB        | 16851.19     | 1219.29         |

### Reconstruct Throughput

`I/O = (data + reconstruct_data_num) * vector_size / cost`

| Data | Parity | Vector size | Reconstruct data num | AVX2 (MiB/s) |
|------|--------|-------------|----------------------|--------------|
| 10   | 4      | 8KiB        | 1                    | 55775.91     |
| 10   | 4      | 8KiB        | 2                    | 33037.90     |
| 10   | 4      | 8KiB        | 3                    | 23917.16     |
| 10   | 4      | 8KiB        | 4                    | 19363.26     |

### Update Throughput

`I/O = (2 + parity_num + parity_num) * vector_size / cost`

| Data | Parity | Vector size | AVX2 (MiB/s) |
|------|--------|-------------|--------------|
| 10   | 4      | 8KiB        | 55710.83     |

### Replace Throughput

`I/O = (parity_num + parity_num + replace_data_num) * vector_size / cost`

| Data | Parity | Vector size | Replace data num | AVX2 (MiB/s) |
|------|--------|-------------|------------------|--------------|
| 10   | 4      | 8KiB        | 1                | 116193.04    |
| 10   | 4      | 8KiB        | 2                | 65375.73     |
| 10   | 4      | 8KiB        | 3                | 48775.47     |
| 10   | 4      | 8KiB        | 4                | 40398.79     |
| 10   | 4      | 8KiB        | 5                | 35262.89     |
| 10   | 4      | 8KiB        | 6                | 31881.60     |

Notes:
- Micro-benchmarks can overestimate real-world throughput due to cache locality and hot loops.
- For representative results, benchmark with your target stripe sizes and I/O path.

To run benchmarks:

```bash
go test -bench BenchmarkRS_ -benchmem
```

## Correctness and Reliability

- Full unit tests for GF arithmetic and matrix inversion behavior.
- End-to-end tests for encode/reconstruct/update/replace.
- Additional mathematical proof for invertibility in [`proof_invertible.md`](proof_invertible.md).

Run tests:

```bash
go test -v ./...
```

## Compatibility Notes

- Max vectors: `dataNum + parityNum <= 256`.
- This project uses a Cauchy-style generator layout compatible with its own implementation.
- Do not assume matrix compatibility with libraries that use different RS matrix construction strategies.

## Related

- [templexxx/xrs](https://github.com/templexxx/xrs): upper-layer erasure-coding (saving about 30% I/O in a reconstruction process) using this library.

## Acknowledgements

- [klauspost/reedsolomon](https://github.com/klauspost/reedsolomon)
- [intel/isa-l](https://github.com/intel/isa-l)
- [FAST 2013 paper: SIMD GF arithmetic](http://web.eecs.utk.edu/~jplank/plank/papers/FAST-2013-GF.html)
