package reedsolomon

import (
	"bytes"
	"math/rand"
	"testing"
)

func TestEncode(t *testing.T) {
	size := 500
	r, err := New(10, 3)
	if err != nil {
		t.Fatal(err)
	}
	dp := NewMatrix(13, size)
	for s := 0; s < 10; s++ {
		rand.Seed(int64(s))
		fillRandom(dp[s])
	}
	err = r.Encode(dp)
	if err != nil {
		t.Fatal(err)
	}
	badDP := NewMatrix(13, 100)
	badDP[0] = make([]byte, 1)
	err = r.Encode(badDP)
	if err != ErrShardSize {
		t.Errorf("expected %v, got %v", ErrShardSize, err)
	}
}

// test if lookup table work
func TestVerifyBaseEncode(t *testing.T) {
	r, err := New(5, 5)
	if err != nil {
		t.Fatal(err)
	}
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
	r.baseEncode(shards)
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

func TestVerifySSSE3_10x4x9999(t *testing.T) {
	if !hasSSSE3() {
		t.Fatal("Verify SSSE3: there is no SSSE3")
	}
	verifyFastEncode(t, 10, 4, 9999, SSSE3)
}

func BenchmarkSSSE3Encode10x4x128K(b *testing.B) {
	if !hasSSSE3() {
		b.Fatal("benchSSSE3: there is no SSSE3")
	}
	benchSIMDEncode(b, 10, 4, 128*1024, SSSE3)
}

func BenchmarkBaseEncode10x4x128K(b *testing.B) {
	benchBaseEncode(b, 10, 4, 128*1024)
}

func BenchmarkSSSE3Encode10x4x256K(b *testing.B) {
	benchSIMDEncode(b, 10, 4, 256*1024, SSSE3)
}

func BenchmarkSSSE3Encode10x4x36M(b *testing.B) {
	benchSIMDEncode(b, 10, 4, 4*1024*1024, SSSE3)
}

func (r *RS) baseEncode(dp matrix) error {
	// check args
	if len(dp) != r.Shards {
		return ErrTooFewShards
	}
	_, err := checkShardSize(dp)
	if err != nil {
		return err
	}
	// encoding
	input := dp[0:r.Data]
	output := dp[r.Data:]
	baseRunner(r.Gen, input, output, r.Data, r.Parity)
	return nil
}

func baseRunner(gen, input, output matrix, numData, numParity int) {
	for i := 0; i < numData; i++ {
		in := input[i]
		for oi := 0; oi < numParity; oi++ {
			if i == 0 {
				baseVectMul(gen[oi][i], in, output[oi])
			} else {
				baseVectMulXor(gen[oi][i], in, output[oi])
			}
		}
	}
}

func baseVectMul(c byte, in, out []byte) {
	mt := mulTable[c]
	for i := 0; i < len(in); i++ {
		out[i] = mt[in[i]]
	}
}

func baseVectMulXor(c byte, in, out []byte) {
	mt := mulTable[c]
	for i := 0; i < len(in); i++ {
		out[i] ^= mt[in[i]]
	}
}

func verifyFastEncode(t *testing.T, d, p, size, ins int) {
	r, err := New(d, p)
	if err != nil {
		t.Fatal(err)
	}
	// asm or nosimd
	dp := NewMatrix(d+p, size)
	for i := 0; i < d; i++ {
		rand.Seed(int64(i))
		fillRandom(dp[i])
	}
	r.INS = ins
	err = r.Encode(dp)
	if err != nil {
		t.Fatal(err)
	}
	// mulTable
	mDP := NewMatrix(d+p, size)
	for i := 0; i < d; i++ {
		copy(mDP[i], dp[i])
	}
	err = r.baseEncode(mDP)
	if err != nil {
		t.Fatal(err)
	}
	for i, asm := range dp {
		if !bytes.Equal(asm, mDP[i]) {
			t.Fatal("verify failed, no match base version; Shards: ", i)
		}
	}
}

func benchSIMDEncode(b *testing.B, d, p, size, ins int) {
	r, err := New(d, p)
	if err != nil {
		b.Fatal(err)
	}
	if ins == SSSE3 {
		r.INS = ins
	}
	dp := NewMatrix(d+p, size)
	for i := 0; i < d; i++ {
		rand.Seed(int64(i))
		fillRandom(dp[i])
	}
	r.Encode(dp)
	b.SetBytes(int64(size * d))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.Encode(dp)
	}
}

func benchBaseEncode(b *testing.B, d, p, size int) {
	r, err := New(d, p)
	if err != nil {
		b.Fatal(err)
	}
	dp := NewMatrix(d+p, size)
	for i := 0; i < d; i++ {
		rand.Seed(int64(i))
		fillRandom(dp[i])
	}
	r.baseEncode(dp)
	b.SetBytes(int64(size * d))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.baseEncode(dp)
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
