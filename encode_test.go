package reedsolomon

import (
	"bytes"
	"math/rand"
	"testing"
)

const (
	kb         = 1024
	mb         = 1024 * 1024
	testNumIn  = 10
	testNumOut = 4
)

// test if lookup Tables work
func TestVerifyBaseEncode(t *testing.T) {
	d := 5
	p := 5
	shards := [][]byte{
		{0, 1},
		{4, 5},
		{2, 3},
		{6, 7},
		{8, 9},
		{0, 0},
		{0, 0},
		{0, 0},
		{0, 0},
		{0, 0},
	}
	gen := genCauchyMatrix(d, p)
	r := rsBase{gen: gen, in: d, out: p}
	r.Encode(shards[:d], shards[d:])
	if shards[5][0] != 97 || shards[5][1] != 64 {
		t.Fatal("shard 5 mismatch")
	}
	if shards[6][0] != 173 || shards[6][1] != 3 {
		t.Fatal("shard 6 mismatch")
	}
	if shards[7][0] != 218 || shards[7][1] != 14 {
		t.Fatal("shard 7 mismatch")
	}
	if shards[8][0] != 107 || shards[8][1] != 35 {
		t.Fatal("shard 8 mismatch")
	}
	if shards[9][0] != 110 || shards[9][1] != 177 {
		t.Fatal("shard 9 mismatch")
	}
}

// Check Tables
func TestGenTables(t *testing.T) {
	gen := genCauchyMatrix(testNumIn, testNumOut)
	tables := genTables(gen)
	if len(tables) != testNumIn*testNumOut*32 {
		t.Errorf("tables len error, expected %r, got %r", len(gen[0]), len(tables))
	}

	for i, row := range gen {
		for j, c := range row {
			l := mulTableLow[c][:]
			h := mulTableHigh[c][:]
			offset := (j*testNumOut + i) * 32
			table := tables[offset : offset+32]
			l1 := table[:16]
			h1 := table[16:32]
			if !bytes.Equal(l, l1) {
				t.Fatal(l, l1)
			}
			if !bytes.Equal(h, h1) {
				t.Fatal(h, h1)
			}
		}
	}
}

// Check AVX2
func TestVerifyAVX2_10x4x32K(t *testing.T) {
	if !hasAVX2() {
		t.Fatal("Verify AVX2: there is no AVX2")
	}
	verifyFastEncode(t, testNumIn, testNumOut, LoopSizeAVX2*4*32, AVX2)
}

// Check SSSE3
func TestVerifySSSE3_10x4x32K(t *testing.T) {
	if !hasSSSE3() {
		t.Fatal("Verify SSSE3: there is no SSSE3")
	}
	verifyFastEncode(t, testNumIn, testNumOut, LoopSizeSSSE3*4*32, SSSE3)
}

// 1KB
func BenchmarkAVX2Encode10x4x1KB(b *testing.B) {
	benchAVX2Encode(b, testNumIn, testNumOut, kb)
}

// 2KB
func BenchmarkAVX2Encode10x4x2KB(b *testing.B) {
	benchAVX2Encode(b, testNumIn, testNumOut, 2*kb)
}

// 4KB
func BenchmarkAVX2Encode10x4x4KB(b *testing.B) {
	benchAVX2Encode(b, testNumIn, testNumOut, 4*kb)
}

// 8KB
func BenchmarkAVX2Encode10x4x8KB(b *testing.B) {
	benchAVX2Encode(b, testNumIn, testNumOut, 8*kb)
}

// 16KB
func BenchmarkAVX2Encode10x4x16KB(b *testing.B) {
	benchAVX2Encode(b, testNumIn, testNumOut, 16*kb)
}

// 32KB
func BenchmarkAVX2Encode10x4x32KB(b *testing.B) {
	benchAVX2Encode(b, testNumIn, testNumOut, 32*kb)
}

// 64KB
func BenchmarkAVX2Encode10x4x64KB(b *testing.B) {
	benchAVX2Encode(b, testNumIn, testNumOut, 64*kb)
}

// 128KB
func BenchmarkAVX2Encode10x4x128KB(b *testing.B) {
	benchAVX2Encode(b, testNumIn, testNumOut, 128*kb)
}

// 256KB
func BenchmarkAVX2Encode10x4x256KB(b *testing.B) {
	benchAVX2Encode(b, testNumIn, testNumOut, 256*kb)
}

// 512KB
func BenchmarkAVX2Encode10x4x512KB(b *testing.B) {
	benchAVX2Encode(b, testNumIn, testNumOut, 512*kb)
}

// 1MB
func BenchmarkAVX2Encode10x4x1MB(b *testing.B) {
	benchAVX2Encode(b, testNumIn, testNumOut, mb)
}

// 2MB
func BenchmarkAVX2Encode10x4x2MB(b *testing.B) {
	benchAVX2Encode(b, testNumIn, testNumOut, 2*mb)
}

// 4MB
func BenchmarkAVX2Encode10x4x4MB(b *testing.B) {
	benchAVX2Encode(b, testNumIn, testNumOut, 4*mb)
}

// 8MB
func BenchmarkAVX2Encode10x4x8MB(b *testing.B) {
	benchAVX2Encode(b, testNumIn, testNumOut, 8*mb)
}

// 16MB
func BenchmarkAVX2Encode10x4x16MB(b *testing.B) {
	benchAVX2Encode(b, testNumIn, testNumOut, 16*mb)
}

// 32MB
func BenchmarkAVX2Encode10x4x32MB(b *testing.B) {
	benchAVX2Encode(b, testNumIn, testNumOut, 32*mb)
}

func benchAVX2Encode(b *testing.B, d, p, size int) {
	gen := genCauchyMatrix(d, p)
	table := genTables(gen)
	dp := NewMatrix(d+p, size)
	for i := 0; i < d; i++ {
		rand.Seed(int64(i))
		fillRandom(dp[i])
	}
	e := rsAVX2{tables: table, in: d, out: p}
	e.Encode(dp[:d], dp[d:])
	b.SetBytes(int64(d * size))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		e.Encode(dp[:d], dp[d:])
	}
}

func verifyFastEncode(t *testing.T, d, p, size, ins int) {
	gen := genCauchyMatrix(d, p)
	table := genTables(gen)
	dp := NewMatrix(d+p, size)
	for i := 0; i < d; i++ {
		rand.Seed(int64(i))
		fillRandom(dp[i])
	}
	var e EncodeReconster
	switch ins {
	case AVX2:
		e = rsAVX2{tables: table, in: d, out: p}
	case SSSE3:
		e = rsSSSE3{tables: table, in: d, out: p}
	}
	e.Encode(dp[:d], dp[d:])
	// mulTable
	bDP := NewMatrix(d+p, size)
	for i := 0; i < d; i++ {
		copy(bDP[i], dp[i])
	}
	e2 := rsBase{gen: gen, in: d, out: p}
	e2.Encode(bDP[:d], bDP[d:])
	for i, asm := range dp {
		if !bytes.Equal(asm, bDP[i]) {
			t.Fatal("verify failed, no match base version; shards: ", i)
		}
	}
}

func fillRandom(p []byte) {
	for i := 0; i < len(p); i += 7 {
		val := rand.Int63()
		for j := 0; i+j < len(p) && j < 7; j++ {
			p[i+j] = byte(val)
			val >>= 8
		}
	}
}
