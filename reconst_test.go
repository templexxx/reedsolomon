package reedsolomon

import (
	"bytes"
	"math/rand"
	"testing"
)

func TestReconst(t *testing.T) {
	size := 64 * 1024
	r, err := New(10, 3)
	if err != nil {
		t.Fatal(err)
	}
	dp := NewMatrix(13, size)
	rand.Seed(0)
	for s := 0; s < 10; s++ {
		fillRandom(dp[s])
	}
	err = r.Encode(dp)
	if err != nil {
		t.Fatal(err)
	}
	// restore encode result
	store := NewMatrix(3, size)
	copy(store[0], dp[0])
	copy(store[1], dp[4])
	copy(store[2], dp[12])
	dp[0] = make([]byte, size)
	dp[4] = make([]byte, size)
	dp[12] = make([]byte, size)
	// Reconstruct with all dp present
	var lost []int
	err = r.Reconst(dp, lost, true)
	if err != nil {
		t.Fatal(err)
	}
	// 3 dp "missing"
	lost = append(lost, 4)
	lost = append(lost, 0)
	lost = append(lost, 12)
	err = r.Reconst(dp, lost, true)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(store[0], dp[0]) {
		t.Fatal("reconst Data mismatch: dp[0]")
	}
	if !bytes.Equal(store[1], dp[4]) {
		t.Fatal("reconst Data mismatch: dp[4]")
	}
	if !bytes.Equal(store[2], dp[12]) {
		t.Fatal("reconst Data mismatch: dp[12]")
	}
	// Reconstruct with 9 dp present (should fail)
	lost = append(lost, 11)
	err = r.Reconst(dp, lost, true)
	if err != ErrTooFewShards {
		t.Errorf("expected %v, got %v", ErrTooFewShards, err)
	}
}

func BenchmarkReconst10x4x128KRepair1(b *testing.B) {
	benchmarkReconst(b, 10, 4, 128*1024, 1)
}

func BenchmarkReconst10x4x128KRepair2(b *testing.B) {
	benchmarkReconst(b, 10, 4, 128*1024, 2)
}

func BenchmarkReconst10x4x128KRepair3(b *testing.B) {
	benchmarkReconst(b, 10, 4, 128*1024, 3)
}

func BenchmarkReconst10x4x128KRepair4(b *testing.B) {
	benchmarkReconst(b, 10, 4, 128*1024, 4)
}

//// lost only happened in Data
func BenchmarkReconst10x4x128KRepair1Data(b *testing.B) {
	benchmarkReconstData(b, 10, 4, 128*1024, 1)
}

func BenchmarkReconst10x4x128KRepair2Data(b *testing.B) {
	benchmarkReconstData(b, 10, 4, 128*1024, 2)
}

func BenchmarkReconst10x4x128KRepair3Data(b *testing.B) {
	benchmarkReconstData(b, 10, 4, 128*1024, 3)
}

func BenchmarkReconst10x4x128KRepair4Data(b *testing.B) {
	benchmarkReconstData(b, 10, 4, 128*1024, 4)
}

// lost only happened in Data
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
	err = r.Encode(dp)
	if err != nil {
		b.Fatal(err)
	}
	lost := randLost(d, repair)
	for _, l := range lost {
		dp[l] = make([]byte, size)
	}
	r.Reconst(dp, lost, false)
	b.SetBytes(int64(size * d))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.Reconst(dp, lost, false)
	}
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
	err = r.Encode(dp)
	if err != nil {
		b.Fatal(err)
	}
	lost := randLost(d+p, repair)
	for _, l := range lost {
		dp[l] = make([]byte, size)
	}
	r.Reconst(dp, lost, true)
	b.SetBytes(int64(size * d))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.Reconst(dp, lost, true)
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
