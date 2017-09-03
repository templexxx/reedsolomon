package reedsolomon

import (
	"bytes"
	"math/rand"
	"testing"

	krs "github.com/klauspost/reedsolomon"
)

const (
	kb         = 1024
	mb         = 1024 * 1024
	testNumIn  = 10
	testNumOut = 4
)

// TODO drop all AVX2 combine them
func TestVerifyKBase(t *testing.T) {
	verifyKEnc(t, testNumIn, testNumOut, none)
}

func TestVerifyKAVX2(t *testing.T) {
	verifyKEnc(t, testNumIn, testNumOut, avx2)
}

const verifySize = 256 + 32 + 16 + 15

func verifyKEnc(t *testing.T, d, p, ins int) {
	em, err := genEncMatrixVand(d, p)
	if err != nil {
		t.Fatal(err)
	}
	g := em[d*d:]
	tbl := make([]byte, p*d*32)
	initTbl(g, p, d, tbl)
	var e EncodeReconster
	switch ins {
	case avx2:
		e = &encAVX2{data: d, parity: p, gen: g, tbl: tbl}
		//case ssse3:
		//	e = &encSSSE3{data: d, parity: p, gen: g, tbl: tbl}
	default:
		e = &encBase{data: d, parity: p, gen: g}
	}
	for i := 1; i <= verifySize; i++ {
		vects1 := make([][]byte, d+p)
		vects2 := make([][]byte, d+p)
		for j := 0; j < d+p; j++ {
			vects1[j] = make([]byte, i)
			vects2[j] = make([]byte, i)
		}
		for j := 0; j < d; j++ {
			rand.Seed(int64(j))
			fillRandom(vects1[j])
			copy(vects2[j], vects1[j])
		}

		err = e.Encode(vects1)
		if err != nil {
			t.Fatal("rs.verifySIMDEnc: ", err)
		}
		ek, err := krs.New(d, p)
		if err != nil {
			t.Fatal(err)
		}
		err = ek.Encode(vects2)
		if err != nil {
			t.Fatal("rs.verifySIMDEnc: ", err)
		}
		for k, v1 := range vects1 {
			if !bytes.Equal(v1, vects2[k]) {
				var ext string
				switch ins {
				case avx2:
					ext = "avx2"
				case ssse3:
					ext = "ssse3"
				}
				t.Fatalf("rs.verifySIMDEnc: %s no match base enc; vect: %d; size: %d", ext, k, i)
			}
		}
	}
}

func TestVerifyEncBaseCauchy(t *testing.T) {
	d := 5
	p := 5
	vects := [][]byte{
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
	em := genEncMatrixCauchy(d, p)
	g := em[d*d:]
	e := &encBase{data: d, parity: p, gen: g}
	err := e.Encode(vects)
	if err != nil {
		t.Fatal(err)
	}
	if vects[5][0] != 97 || vects[5][1] != 64 {
		t.Fatal("vect 5 mismatch")
	}
	if vects[6][0] != 173 || vects[6][1] != 3 {
		t.Fatal("vect 6 mismatch")
	}
	if vects[7][0] != 218 || vects[7][1] != 14 {
		t.Fatal("vect 7 mismatch")
	}
	if vects[8][0] != 107 || vects[8][1] != 35 {
		t.Fatal("vect 8 mismatch")
	}
	if vects[9][0] != 110 || vects[9][1] != 177 {
		t.Fatal("vect 9 mismatch")
	}
}

func TestVerifyEncBaseVand(t *testing.T) {
	d := 5
	p := 5
	vects := [][]byte{
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
	em, err := genEncMatrixVand(d, p)
	if err != nil {
		t.Fatal(err)
	}
	g := em[d*d:]
	e := &encBase{data: d, parity: p, gen: g}
	err = e.Encode(vects)
	if err != nil {
		t.Fatal(err)
	}
	if vects[5][0] != 12 || vects[5][1] != 13 {
		t.Fatal("vect 5 mismatch")
	}
	if vects[6][0] != 10 || vects[6][1] != 11 {
		t.Fatal("vect 6 mismatch")
	}
	if vects[7][0] != 14 || vects[7][1] != 15 {
		t.Fatal("vect 7 mismatch")
	}
	if vects[8][0] != 90 || vects[8][1] != 91 {
		t.Fatal("vect 8 mismatch")
	}
	if vects[9][0] != 94 || vects[9][1] != 95 {
		t.Fatal("shard 9 mismatch")
	}
}

// TODO add verify Enc

func fillRandom(v []byte) {
	for i := 0; i < len(v); i += 7 {
		val := rand.Int63()
		for j := 0; i+j < len(v) && j < 7; j++ {
			v[i+j] = byte(val)
			val >>= 8
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
	e, err := New(d, p)
	if err != nil {
		b.Fatal(err)
	}
	err = e.Encode(vects)
	if err != nil {
		b.Fatal(err)
	}
	b.SetBytes(int64(d * size))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err = e.Encode(vects)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// TODO learn test from map_test
func BenchmarkEnc10x4_4KB(b *testing.B) {
	benchEnc(b, testNumIn, testNumOut, 4*kb)
}

func BenchmarkEnc10x4_64KB(b *testing.B) {
	benchEnc(b, testNumIn, testNumOut, 64*kb)
}

func BenchmarkEnc10x4_1M(b *testing.B) {
	benchEnc(b, testNumIn, testNumOut, mb)
}

func BenchmarkEnc10x4_16M(b *testing.B) {
	benchEnc(b, testNumIn, testNumOut, 16*mb)
}

func BenchmarkEnc10x4_1400B(b *testing.B) {
	benchEnc(b, testNumIn, testNumOut, 1400)
}

func BenchmarkEncLittle10x4_1280B(b *testing.B) {
	benchEnc(b, testNumIn, testNumOut, 1280)
}
func BenchmarkEncLittle10x4_1281B(b *testing.B) {
	benchEnc(b, testNumIn, testNumOut, 1281)
}
func BenchmarkEncLittle10x4_1282B(b *testing.B) {
	benchEnc(b, testNumIn, testNumOut, 1282)
}
func BenchmarkEncLittle10x4_1283B(b *testing.B) {
	benchEnc(b, testNumIn, testNumOut, 1283)
}
func BenchmarkEncLittle10x4_1284B(b *testing.B) {
	benchEnc(b, testNumIn, testNumOut, 1284)
}
func BenchmarkEncLittle10x4_1285B(b *testing.B) {
	benchEnc(b, testNumIn, testNumOut, 1285)
}
func BenchmarkEncLittle10x4_1286B(b *testing.B) {
	benchEnc(b, testNumIn, testNumOut, 1286)
}
func BenchmarkEncLittle10x4_1287B(b *testing.B) {
	benchEnc(b, testNumIn, testNumOut, 1287)
}
func BenchmarkEncLittle10x4_1288B(b *testing.B) {
	benchEnc(b, testNumIn, testNumOut, 1288)
}
func BenchmarkEncLittle10x4_1289B(b *testing.B) {
	benchEnc(b, testNumIn, testNumOut, 1289)
}

func BenchmarkEnc10x3_1350B(b *testing.B) {
	benchEnc(b, testNumIn, 3, 1350)
}

func TestVerifyReconstBase(t *testing.T) {
	d := 5
	p := 5
	vects := [][]byte{
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
	em := genEncMatrixCauchy(d, p)
	g := em[d*d:]
	e := &encBase{data: d, parity: p, gen: g, encode: em}
	err := e.Encode(vects)
	if err != nil {
		t.Fatal(err)
	}
	lost := []int{5, 6, 4, 2, 0}
	for _, i := range lost {
		vects[i] = nil
	}
	err = e.Reconstruct(vects)
	if err != nil {
		t.Fatal(err)
	}
	if vects[5][0] != 97 || vects[5][1] != 64 {
		t.Fatal("shard 5 mismatch")
	}
	if vects[6][0] != 173 || vects[6][1] != 3 {
		t.Fatal("shard 6 mismatch")
	}
	if vects[4][0] != 8 || vects[4][1] != 9 {
		t.Fatal("shard 7 mismatch")
	}
	if vects[2][0] != 2 || vects[2][1] != 3 {
		t.Fatal("shard 8 mismatch")
	}
	if vects[0][0] != 0 || vects[0][1] != 1 {
		t.Fatal("shard 9 mismatch")
	}
}

func verifyReconst(t *testing.T, d, p, ins int) {
	for i := 1; i <= verifySize; i++ {
		vects1 := make([][]byte, d+p)
		vects2 := make([][]byte, d+p)
		for j := 0; j < d+p; j++ {
			vects1[j] = make([]byte, i)
			vects2[j] = make([]byte, i)
		}
		for j := 0; j < d; j++ {
			rand.Seed(int64(j))
			fillRandom(vects1[j])
			copy(vects2[j], vects1[j])
		}
		em, err := genEncMatrixVand(d, p)
		if err != nil {
			t.Fatal(err)
		}
		g := em[d*d:]
		tbl := make([]byte, p*d*32)
		initTbl(g, p, d, tbl)
		var e EncodeReconster
		switch ins {
		case avx2:
			e = &encAVX2{data: d, parity: p, gen: g, tbl: tbl, encode: em}
			//case ssse3:
			//	e = &encSSSE3{data: d, parity: p, gen: g, tbl: tbl}
		default:
			e = &encBase{data: d, parity: p, gen: g, encode: em}
		}
		err = e.Encode(vects1)
		if err != nil {
			t.Fatal(err)
		}
		for j := 0; j < d+p; j++ {
			copy(vects2[j], vects1[j])
		}
		lost := []int{11, 6, 2, 0}
		for _, i := range lost {
			vects2[i] = nil
		}
		err = e.Reconstruct(vects2)
		if err != nil {
			t.Fatal(err)
		}
		for k, v1 := range vects1 {
			if !bytes.Equal(v1, vects2[k]) {
				exts := "none"
				t.Fatalf("%s no match reconst; vect: %d; size: %d", exts, k, i)
			}
		}
	}
}

//
func TestVerifyReconstNone(t *testing.T) {
	verifyReconst(t, testNumIn, testNumOut, none)
}

func verifyReconstWithPos(t *testing.T, d, p, ins int) {
	for i := 1; i <= verifySize; i++ {
		vects1 := make([][]byte, d+p)
		vects2 := make([][]byte, d+p)
		for j := 0; j < d+p; j++ {
			vects1[j] = make([]byte, i)
			vects2[j] = make([]byte, i)
		}
		for j := 0; j < d; j++ {
			rand.Seed(int64(j))
			fillRandom(vects1[j])
			copy(vects2[j], vects1[j])
		}
		em, err := genEncMatrixVand(d, p)
		if err != nil {
			t.Fatal(err)
		}
		g := em[d*d:]
		tbl := make([]byte, p*d*32)
		initTbl(g, p, d, tbl)
		var e EncodeReconster
		switch ins {
		case avx2:
			e = &encAVX2{data: d, parity: p, gen: g, tbl: tbl, encode: em}
			//case ssse3:
			//	e = &encSSSE3{data: d, parity: p, gen: g, tbl: tbl}
		default:
			e = &encBase{data: d, parity: p, gen: g, encode: em}
		}
		err = e.Encode(vects1)
		if err != nil {
			t.Fatal(err)
		}
		dLost := []int{0, 1, 7}
		pLost := []int{12}
		has := []int{2, 3, 4, 5, 6, 8, 9, 10, 11, 13}
		for j := 0; j < d+p; j++ {
			copy(vects2[j], vects1[j])
		}
		err = e.ReconstWithPos(vects2, has, dLost, pLost)
		if err != nil {
			t.Fatal(err)
		}
		for k, v1 := range vects1 {
			if !bytes.Equal(v1, vects2[k]) {
				exts := "none"
				t.Fatalf("%s no match reconst; vect: %d; size: %d", exts, k, i)
			}
		}
	}
}

//
func TestVerifyReconstWithPosBase(t *testing.T) {
	verifyReconstWithPos(t, testNumIn, testNumOut, none)
}

func TestVerifyReconstWithPosAVX2(t *testing.T) {
	verifyReconstWithPos(t, testNumIn, testNumOut, avx2)
}

//func TestVerifySSSE3Reconst(t *testing.T) {
//	verifySIMDReconst(t, testNumIn, testNumOut, ssse3)
//}

// TODO add lost parity test
func BenchmarkReconst10x4x4KB_4DataCache(b *testing.B) {
	lost := []int{0, 1, 2, 3}
	benchReconst(b, lost, 10, 4, 4*kb, true)
}

func BenchmarkReconst10x4x4KB_4DataNoCache(b *testing.B) {
	lost := []int{2, 4, 5, 7}
	benchReconst(b, lost, 10, 4, 4*kb, false)
}

func BenchmarkReconst10x4x64KB_4DataCache(b *testing.B) {
	lost := []int{2, 4, 5, 7}
	benchReconst(b, lost, 10, 4, 64*kb, true)
}

func BenchmarkReconst10x4x64KB_4DataNoCache(b *testing.B) {
	lost := []int{2, 4, 5, 7}
	benchReconst(b, lost, 10, 4, 64*kb, false)
}

func BenchmarkReconst10x4x1M_4DataCache(b *testing.B) {
	lost := []int{2, 4, 5, 7}
	benchReconst(b, lost, 10, 4, mb, true)
}

func BenchmarkReconst10x4x1M_4DataNoCache(b *testing.B) {
	lost := []int{2, 4, 5, 7}
	benchReconst(b, lost, 10, 4, mb, false)
}

func BenchmarkReconst10x4x16m_4DataCache(b *testing.B) {
	lost := []int{2, 4, 5, 7}
	benchReconst(b, lost, 10, 4, 16*mb, true)
}

func BenchmarkReconst10x4x16m_4DataNoCache(b *testing.B) {
	lost := []int{2, 4, 5, 7}
	benchReconst(b, lost, 10, 4, 16*mb, false)
}

func BenchmarkReconst10x3x1350B_4DataCache(b *testing.B) {
	lost := []int{2, 4, 5}
	benchReconst(b, lost, 10, 3, 1350, true)
}

func BenchmarkReconst10x3x1350B_4DataNoCache(b *testing.B) {
	lost := []int{2, 4, 5}
	benchReconst(b, lost, 10, 3, 1350, false)
}

// TODO add reconst
func benchReconst(b *testing.B, lost []int, d, p, size int, cache bool) {
	vects := make([][]byte, d+p)
	for j := 0; j < d+p; j++ {
		vects[j] = make([]byte, size)
	}
	for j := 0; j < d; j++ {
		rand.Seed(int64(j))
		fillRandom(vects[j])
	}
	e, err := New(d, p)
	if err != nil {
		b.Fatal(err)
	}
	if !cache {
		e.CloseCache()
	}
	err = e.Encode(vects)
	if err != nil {
		b.Fatal(err)
	}
	for _, i := range lost {
		vects[i] = nil
	}
	err = e.Reconstruct(vects)
	if err != nil {
		b.Fatal(err)
	}
	b.SetBytes(int64(d * size))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, i := range lost {
			vects[i] = nil
		}
		err = e.Reconstruct(vects)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkReconstPos10x4x4KB_4DataCache(b *testing.B) {
	has := []int{0, 1, 3, 6, 8, 9, 10, 11, 12, 13}
	dLost := []int{2, 4, 5, 7}
	var pLost []int
	benchReconstWithPos(b, has, dLost, pLost, 10, 4, 4*kb, true)
}

func BenchmarkReconstPos10x4x4KB_4DataNoCache(b *testing.B) {
	has := []int{0, 1, 3, 6, 8, 9, 10, 11, 12, 13}
	dLost := []int{2, 4, 5, 7}
	var pLost []int
	benchReconstWithPos(b, has, dLost, pLost, 10, 4, 4*kb, false)
}

func BenchmarkReconstPos10x4x64KB_4DataCache(b *testing.B) {
	has := []int{0, 1, 3, 6, 8, 9, 10, 11, 12, 13}
	dLost := []int{2, 4, 5, 7}
	var pLost []int
	benchReconstWithPos(b, has, dLost, pLost, 10, 4, 64*kb, true)
}

func BenchmarkReconstPos10x4x64KB_4DataNoCache(b *testing.B) {
	has := []int{0, 1, 3, 6, 8, 9, 10, 11, 12, 13}
	dLost := []int{2, 4, 5, 7}
	var pLost []int
	benchReconstWithPos(b, has, dLost, pLost, 10, 4, 64*kb, false)
}

func BenchmarkReconstPos10x4x1M_4DataCache(b *testing.B) {
	has := []int{0, 1, 3, 6, 8, 9, 10, 11, 12, 13}
	dLost := []int{2, 4, 5, 7}
	var pLost []int
	benchReconstWithPos(b, has, dLost, pLost, 10, 4, mb, true)
}

func BenchmarkReconstPos10x4x1M_4DataNoCache(b *testing.B) {
	has := []int{0, 1, 3, 6, 8, 9, 10, 11, 12, 13}
	dLost := []int{2, 4, 5, 7}
	var pLost []int
	benchReconstWithPos(b, has, dLost, pLost, 10, 4, mb, false)
}

func BenchmarkReconstPos10x4x16m_4DataCache(b *testing.B) {
	has := []int{0, 1, 3, 6, 8, 9, 10, 11, 12, 13}
	dLost := []int{2, 4, 5, 7}
	var pLost []int
	benchReconstWithPos(b, has, dLost, pLost, 10, 4, 16*mb, true)
}

func BenchmarkReconstPos10x4x16m_4DataNoCache(b *testing.B) {
	has := []int{0, 1, 3, 6, 8, 9, 10, 11, 12, 13}
	dLost := []int{2, 4, 5, 7}
	var pLost []int
	benchReconstWithPos(b, has, dLost, pLost, 10, 4, 16*mb, false)
}

func BenchmarkReconstPos10x3x1350B_4DataCache(b *testing.B) {
	has := []int{0, 1, 3, 6, 7, 8, 9, 10, 11, 12}
	dLost := []int{2, 4, 5}
	var pLost []int
	benchReconstWithPos(b, has, dLost, pLost, 10, 3, 1350, true)
}

func BenchmarkReconstPos10x3x1350B_4DataNoCache(b *testing.B) {
	has := []int{0, 1, 3, 6, 7, 8, 9, 10, 11, 12}
	dLost := []int{2, 4, 5}
	var pLost []int
	benchReconstWithPos(b, has, dLost, pLost, 10, 3, 1350, false)
}

func benchReconstWithPos(b *testing.B, has, dLost, pLost []int, d, p, size int, cache bool) {
	vects := make([][]byte, d+p)
	for j := 0; j < d+p; j++ {
		vects[j] = make([]byte, size)
	}
	for j := 0; j < d; j++ {
		rand.Seed(int64(j))
		fillRandom(vects[j])
	}
	e, err := New(d, p)
	if err != nil {
		b.Fatal(err)
	}
	if !cache {
		e.CloseCache()
	}
	err = e.Encode(vects)
	if err != nil {
		b.Fatal(err)
	}
	err = e.ReconstWithPos(vects, has, dLost, pLost)
	if err != nil {
		b.Fatal(err)
	}
	b.SetBytes(int64(d * size))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err = e.ReconstWithPos(vects, has, dLost, pLost)
		if err != nil {
			b.Fatal(err)
		}
	}
}
