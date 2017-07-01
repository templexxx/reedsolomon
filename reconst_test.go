package reedsolomon

import (
	"bytes"
	"math/rand"
	"testing"
)

// verify reconst in base
func TestVerifyReconstBase(t *testing.T) {
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

	have := []int{9, 8, 7, 1, 3}
	lost := []int{5, 6, 4, 2, 0}
	newShards := NewMatrix(10, 2)
	for _, h := range have {
		copy(newShards[h], shards[h])
	}
	for _, l := range lost {
		newShards[l] = make([]byte, 2)
	}
	r.Reconst(newShards, have, lost)
	if newShards[5][0] != 97 || newShards[5][1] != 64 {
		t.Fatal("shard 5 mismatch")
	}
	if newShards[6][0] != 173 || newShards[6][1] != 3 {
		t.Fatal("shard 6 mismatch")
	}
	if newShards[4][0] != 8 || newShards[4][1] != 9 {
		t.Fatal("shard 7 mismatch")
	}
	if newShards[2][0] != 2 || newShards[2][1] != 3 {
		t.Fatal("shard 8 mismatch")
	}
	if newShards[0][0] != 0 || newShards[0][1] != 1 {
		t.Fatal("shard 9 mismatch")
	}
}

// verify reconst in avx2
func TestVerifyReconstAVX2(t *testing.T) {
	d := 10
	p := 4
	size := LoopSizeAVX2
	dp := NewMatrix(d+p, size)
	for i := 0; i < d; i++ {
		rand.Seed(int64(i))
		fillRandom(dp[i])
	}
	r, err := New(d, p)
	if err != nil {
		t.Fatal(err)
	}
	r.Encode(dp[:d], dp[d:])
	dp2 := NewMatrix(d+p, size)
	have := []int{0, 13, 2, 5, 6, 7, 8, 9, 11, 1}
	lost := []int{10, 12, 3, 4}
	for _, h := range have {
		copy(dp2[h], dp[h])
	}
	for _, l := range lost {
		dp2[l] = make([]byte, size)
	}
	r.Reconst(dp2, have, lost)
	for i := range dp {
		if !bytes.Equal(dp[i], dp2[i]) {
			t.Errorf("reconst data mismatch %d", i)
		}
	}
}

func BenchmarkReconst10x4x128KRepair1(b *testing.B) {
	benchmarkReconst(b, testNumIn, testNumOut, 128*kb, 1)
}

func BenchmarkReconst10x4x128KRepair2(b *testing.B) {
	benchmarkReconst(b, testNumIn, testNumOut, 128*kb, 2)
}

//
func BenchmarkReconst10x4x128KRepair3(b *testing.B) {
	benchmarkReconst(b, testNumIn, testNumOut, 128*kb, 3)
}

//
func BenchmarkReconst10x4x128KRepair4(b *testing.B) {
	benchmarkReconst(b, testNumIn, testNumOut, 128*kb, 4)
}

// lost only happened in data
func BenchmarkReconst10x4x128KRepair1Data(b *testing.B) {
	benchmarkReconstData(b, testNumIn, testNumOut, 128*kb, 1)
}

func BenchmarkReconst10x4x128KRepair2Data(b *testing.B) {
	benchmarkReconstData(b, testNumIn, testNumOut, 128*kb, 2)
}

func BenchmarkReconst10x4x128KRepair3Data(b *testing.B) {
	benchmarkReconstData(b, testNumIn, testNumOut, 128*kb, 3)
}

func BenchmarkReconst10x4x128KRepair4Data(b *testing.B) {
	benchmarkReconstData(b, testNumIn, testNumOut, 128*kb, 4)
}

func BenchmarkReconst10x4x16MRepair1Data(b *testing.B) {
	benchmarkReconstData(b, testNumIn, testNumOut, 16*mb, 1)
}

// lost only happened in data
func benchmarkReconstData(b *testing.B, d, p, size, repair int) {

	r, err := New(d, p)
	if err != nil {
		b.Fatal(err)
	}
	dp := NewMatrix(d+p, size)
	for s := 0; s < d; s++ {
		rand.Seed(int64(s))
		fillRandom(dp[s])
	}
	r.Encode(dp[:d], dp[d:])
	lost := randLost(d, repair)
	have := getHave(d, p, lost)
	for _, l := range lost {
		dp[l] = make([]byte, size)
	}
	b.SetBytes(int64(size * d))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.Reconst(dp, have, lost)
	}
}

func getHave(d, p int, lost []int) []int {
	have := make([]int, d)
	var cnt int
	for i := 0; i < d+p; i++ {
		if cnt < d {
			if !has(lost, i) {
				have[cnt] = i
				cnt++
			}
		}
	}
	return have
}

func benchmarkReconst(b *testing.B, d, p, size, repair int) {
	r, err := New(d, p)
	if err != nil {
		b.Fatal(err)
	}
	dp := NewMatrix(d+p, size)
	for s := 0; s < d; s++ {
		rand.Seed(int64(s))
		fillRandom(dp[s])
	}

	r.Encode(dp[:d], dp[d:])
	lost := randLost(d+p, repair)
	have := getHave(d, p, lost)
	for _, l := range lost {
		dp[l] = make([]byte, size)
	}
	b.SetBytes(int64(size * d))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.Reconst(dp, have, lost)
	}
}

func randLost(max, num int) []int {
	var lost []int
	seed := 0
	for {
		rand.Seed(int64(seed))
		r := rand.Intn(max)
		if len(lost) == num {
			return lost
		}
		if !has(lost, r) {
			lost = append(lost, r)
		}
		seed++
	}
}

func has(s []int, i int) bool {
	for _, v := range s {
		if i == v {
			return true
		}
		continue
	}
	return false
}
