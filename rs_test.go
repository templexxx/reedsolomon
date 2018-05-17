package reedsolomon

import (
	"bytes"
	"fmt"
	"go/xec/cpu"
	"math/rand"
	"testing"
)

const (
	kb            = 1024
	mb            = 1024 * 1024
	testDataCnt   = 14
	testParityCnt = 4
	// 256: avx_loop/sse_loop, 32: ymm_register/xmm_register, 16: ymm_register/xmm_register, 8: byte by byte
	verifySize    = 256 + 32 + 16 + 8
	testUpdateRow = 3
)

var (
	testDPHas       = []int{10, 8, 5, 6, 7, 4, 2, 12, 13, 1, 14, 15, 16, 17}
	testNeedReconst = []int{3, 9, 0, 11}
)

// when vect_size < 16, encode won't use SIMD
// Powered by MATLAB
func TestVerifyEncodeBase(t *testing.T) {
	d, p := 5, 5
	r, err := New(d, p)
	if err != nil {
		t.Fatal(err)
	}
	vects := [][]byte{{0}, {4}, {2}, {6}, {8}, {0}, {0}, {0}, {0}, {0}}
	err = r.Encode(vects)
	if err != nil {
		t.Fatal(err)
	}
	if vects[5][0] != 97 {
		t.Fatal("vect 5 mismatch")
	}
	if vects[6][0] != 173 {
		t.Fatal("vect 6 mismatch")
	}
	if vects[7][0] != 218 {
		t.Fatal("vect 7 mismatch")
	}
	if vects[8][0] != 107 {
		t.Fatal("vect 8 mismatch")
	}
	if vects[9][0] != 110 {
		t.Fatal("shard 9 mismatch")
	}
}

func TestVerifyEncodeSIMD(t *testing.T) {
	d, p := testDataCnt, testParityCnt
	if cpu.X86.HasAVX2 {
		verifyEncodeSIMD(t, d, p, avx2)
		verifyEncodeSIMD(t, d, p, ssse3)
	} else if cpu.X86.HasSSSE3 {
		verifyEncodeSIMD(t, d, p, ssse3)
	} else {
		t.Log("not support SIMD")
	}
}

// compare encodeBase & encodeSIMD(avx2 or ssse3)
// step1: copy data from expect to result
// step2: encodeSIMD & ecodeBase
// step3: compare
func verifyEncodeSIMD(t *testing.T, d, p, cpuFeature int) {
	for size := 1; size <= verifySize; size++ {
		expect := make([][]byte, d+p)
		result := make([][]byte, d+p)
		for j := 0; j < d+p; j++ {
			expect[j], result[j] = make([]byte, size), make([]byte, size)
		}
		for j := 0; j < d; j++ {
			rand.Seed(int64(j))
			fillRandom(expect[j])
			copy(result[j], expect[j])
		}
		r, err := New(d, p)
		if err != nil {
			t.Fatal(err)
		}
		r.cpuFeature = cpuFeature
		err = r.Encode(result)
		if err != nil {
			t.Fatal(err)
		}
		r.cpuFeature = base
		err = r.Encode(expect)
		if err != nil {
			t.Fatal(err)
		}
		for j := range expect {
			if !bytes.Equal(expect[j], result[j]) {
				var cpuFeatureStr string
				if cpuFeature == avx2 {
					cpuFeatureStr = "avx2"
				} else {
					cpuFeatureStr = "ssse3"
				}
				t.Fatalf("no match encodeSIMD with encodeBase; vect: %d; size: %d; feature: %s", j, size, cpuFeatureStr)
			}
		}
	}
}

func TestVerifyReconst(t *testing.T) {
	verifyReconst(t, testDataCnt, testParityCnt, testDPHas, testNeedReconst)
}

// step1: encode expect
// step2: copy dhHas from expect to result
// step3: reconst result
// step4: compare
func verifyReconst(t *testing.T, d, p int, dpHas, needReconst []int) {
	for size := 1; size <= verifySize; size++ {
		expect := make([][]byte, d+p)
		result := make([][]byte, d+p)
		for j := 0; j < d+p; j++ {
			expect[j], result[j] = make([]byte, size), make([]byte, size)
		}
		for j := 0; j < d; j++ {
			rand.Seed(int64(j))
			fillRandom(expect[j])
		}
		r, err := New(d, p)
		err = r.Encode(expect)
		if err != nil {
			t.Fatal(err)
		}
		for _, h := range dpHas {
			copy(result[h], expect[h])
		}
		err = r.Reconst(result, dpHas, needReconst)
		if err != nil {
			t.Fatal(err)
		}
		for _, n := range needReconst {
			if !bytes.Equal(expect[n], result[n]) {
				t.Fatalf("no match reconst; vect: %d; size: %d", n, size)
			}
		}
	}
}

func TestVerifyUpdateParity(t *testing.T) {
	verifyUpdateParity(t, testDataCnt, testParityCnt, testUpdateRow)
}

// compare encode&update results
func verifyUpdateParity(t *testing.T, d, p, updateRow int) {
	for size := 1; size <= verifySize; size++ {
		updateRet := make([][]byte, d+p)
		encodeRet := make([][]byte, d+p)
		for j := 0; j < d+p; j++ {
			updateRet[j], encodeRet[j] = make([]byte, size), make([]byte, size)
		}
		for j := 0; j < d; j++ {
			rand.Seed(int64(j))
			fillRandom(encodeRet[j])
			copy(updateRet[j], encodeRet[j])
		}
		r, err := New(d, p)
		if err != nil {
			t.Fatal(err)
		}
		err = r.Encode(updateRet)
		if err != nil {
			t.Fatal(err)
		}
		newData := make([]byte, size)
		fillRandom(newData)
		err = r.UpdateParity(updateRet[updateRow], newData, updateRow, updateRet[d:d+p])
		if err != nil {
			t.Fatal(err)
		}

		copy(encodeRet[updateRow], newData)
		err = r.Encode(encodeRet)
		if err != nil {
			t.Fatal(err)
		}
		for j := d; j < d+p; j++ {
			if !bytes.Equal(updateRet[j], encodeRet[j]) {
				t.Fatalf("update mismatch; vect: %d; size: %d", j, size)
			}
		}
	}
}

func TestEncodeMatrixCache(t *testing.T) {
	d, p := testDataCnt, testParityCnt
	r, err := New(d, p)
	if err != nil {
		t.Fatal(err)
	}
	if r.cacheEnabled == false {
		t.Fatal("cache enable failed")
	}
	dLost, _ := SplitNeedReconst(d, testNeedReconst)
	// store a matrix in cache
	gm, err := r.getGenMatrixFromCache(testDPHas, dLost)
	if err != nil {
		t.Fatal(err)
	}
	// read cache
	var bitmap uint64 // indicate dpHas
	for _, i := range testDPHas {
		bitmap += 1 << uint8(i)
	}
	d, lostCnt := r.DataCnt, len(dLost)
	v, ok := r.Load(bitmap)
	gmFromCache := make([]byte, lostCnt*d)
	if ok {
		im := v.([]byte)
		for i, l := range dLost {
			copy(gmFromCache[i*d:i*d+d], im[l*d:l*d+d])
		}
	}
	if !bytes.Equal(gm, gmFromCache) {
		t.Fatal("matrix misamtch")
	}
}

func BenchmarkEncode(b *testing.B) {
	sizes := []int{4 * kb, 4 * mb}
	b.Run("", benchEncRun(benchEnc, testDataCnt, testParityCnt, sizes))
}

func benchEncRun(f func(*testing.B, int, int, int), d, p int, sizes []int) func(*testing.B) {
	return func(b *testing.B) {
		for _, s := range sizes {
			b.Run(fmt.Sprintf("(%d+%d)*%dKB", d, p, s/kb), func(b *testing.B) {
				f(b, d, p, s)
			})
		}
	}
}

func benchEnc(b *testing.B, d, p, size int) {
	vects := make([][]byte, d+p)
	for j := 0; j < d+p; j++ {
		vects[j] = make([]byte, size)
	}
	for j := 0; j < d; j++ {
		rand.Seed(int64(j))
		fillRandom(vects[j])
	}
	r, err := New(d, p)
	if err != nil {
		b.Fatal(err)
	}
	err = r.Encode(vects)
	if err != nil {
		b.Fatal(err)
	}
	b.SetBytes(int64(d * size))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err = r.Encode(vects)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkReconst(b *testing.B) {
	sizes := []int{4 * kb, 4 * mb}
	b.Run("", benchmarkReconst(benchReconst, testDataCnt, testParityCnt, sizes, testDPHas, testNeedReconst))
}

func benchmarkReconst(f func(*testing.B, int, int, int, []int, []int),
	d, p int, sizes, has, needReconst []int) func(*testing.B) {
	return func(b *testing.B) {
		for _, s := range sizes {
			b.Run(fmt.Sprintf("(%d+%d)*%dKB reconst %d vects", d, p, s/kb, len(needReconst)), func(b *testing.B) {
				f(b, d, p, s, has, needReconst)
			})
		}
	}
}

func benchReconst(b *testing.B, d, p, size int, has, needReconst []int) {
	vects := make([][]byte, d+p)
	for j := 0; j < d+p; j++ {
		vects[j] = make([]byte, size)
	}
	for j := 0; j < d; j++ {
		rand.Seed(int64(j))
		fillRandom(vects[j])
	}
	r, err := New(d, p)
	if err != nil {
		b.Fatal(err)
	}
	err = r.Encode(vects)
	if err != nil {
		b.Fatal(err)
	}
	err = r.Reconst(vects, has, needReconst)
	if err != nil {
		b.Fatal(err)
	}
	b.SetBytes(int64(d * size))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err = r.Reconst(vects, has, needReconst)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkUpdateParity(b *testing.B) {
	sizes := []int{4 * kb, 4 * mb}
	b.Run("", benchmarkUpdateParity(benchUpdateParity, testDataCnt, testParityCnt, sizes, testUpdateRow))
}

func benchmarkUpdateParity(f func(*testing.B, int, int, int, int), d, p int, sizes []int, updateRow int) func(*testing.B) {
	return func(b *testing.B) {
		for _, s := range sizes {
			b.Run(fmt.Sprintf("(%d+%d)*%dKB update", d, p, s/kb), func(b *testing.B) {
				f(b, d, p, s, updateRow)
			})
		}
	}
}

func benchUpdateParity(b *testing.B, d, p, size, updateRow int) {
	vects := make([][]byte, d+p)
	for j := 0; j < d+p; j++ {
		vects[j] = make([]byte, size)
	}
	for j := 0; j < d; j++ {
		rand.Seed(int64(j))
		fillRandom(vects[j])
	}
	r, err := New(d, p)
	if err != nil {
		b.Fatal(err)
	}
	err = r.Encode(vects)
	if err != nil {
		b.Fatal(err)
	}
	newData := make([]byte, size)
	fillRandom(newData)
	err = r.UpdateParity(vects[updateRow], newData, updateRow, vects[d:])
	if err != nil {
		b.Fatal(err)
	}
	b.SetBytes(int64(d * size))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err = r.UpdateParity(vects[updateRow], newData, updateRow, vects[d:])
		if err != nil {
			b.Fatal(err)
		}
	}
}

func fillRandom(v []byte) {
	for i := 0; i < len(v); i += 7 {
		val := rand.Int63()
		for j := 0; i+j < len(v) && j < 7; j++ {
			v[i+j] = byte(val)
			val >>= 8
		}
	}
}
