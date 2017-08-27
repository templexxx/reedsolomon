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

func TestVerifyBaseEncodeCauchy(t *testing.T) {
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
	e := &encBase{data: d, parity: p, genMatrix: g}
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

func TestVerifyBaseEncodeVand(t *testing.T) {
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
	e := &encBase{data: d, parity: p, genMatrix: g}
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

func TestVerifyMakeTbl(t *testing.T) {
	g := []byte{17, 34, 23, 211}
	expect := make([]byte, 4*32)
	copy(expect[:32], lowhighTbl[17][:])
	copy(expect[32:64], lowhighTbl[23][:])
	copy(expect[64:96], lowhighTbl[34][:])
	copy(expect[96:128], lowhighTbl[211][:])
	tbl := initTbl(g, 2, 2)
	if !bytes.Equal(expect, tbl) {
		t.Fatal("mismatch")
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

//const verifySize = unitSize + 128 + 32 + 7
const verifySize = 256 + 32 + 16 + 15

func verifySIMDEnc(t *testing.T, d, p, ins int) {
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
		tbl := initTbl(g, p, d)
		var e EncodeReconster
		switch ins {
		case avx2:
			e = &encAVX2{data: d, parity: p, genMatrix: g, tbl: tbl}
		case ssse3:
			e = &encSSSE3{data: d, parity: p, genMatrix: g, tbl: tbl}
		}
		err = e.Encode(vects1)
		if err != nil {
			t.Fatal(err)
		}
		eb := &encBase{data: d, parity: p, genMatrix: g}
		err = eb.Encode(vects2)
		if err != nil {
			t.Fatal(err)
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
				t.Fatalf("%s no match base enc; vect: %d; size: %d", ext, k, i)
			}
		}
	}
}

func TestVerifyAVX2(t *testing.T) {
	if !hasAVX2() {
		t.Fatal("no AVX2")
	}
	verifySIMDEnc(t, testNumIn, testNumOut, avx2)
}

func TestVerifySSSE3(t *testing.T) {
	if !hasAVX2() {
		t.Fatal("rs.TestVerifyAVX2: no SSSE3")
	}
	verifySIMDEnc(t, testNumIn, testNumOut, ssse3)
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

func BenchmarkEnc10x4_1KB(b *testing.B) {
	benchEnc(b, testNumIn, testNumOut, kb)
}

func BenchmarkEnc10x4_1350B(b *testing.B) {
	benchEnc(b, testNumIn, testNumOut, 1350)
}
func BenchmarkEnc10x4_1400B(b *testing.B) {
	benchEnc(b, testNumIn, testNumOut, 1400)
}

func BenchmarkEnc10x4_4KB(b *testing.B) {
	benchEnc(b, testNumIn, testNumOut, 4*kb)
}

func BenchmarkEnc10x4_16KB(b *testing.B) {
	benchEnc(b, testNumIn, testNumOut, 16*kb)
}

func BenchmarkEnc10x4_64KB(b *testing.B) {
	benchEnc(b, testNumIn, testNumOut, 64*kb)
}

func BenchmarkEnc10x4_256KB(b *testing.B) {
	benchEnc(b, testNumIn, testNumOut, 256*kb)
}

func BenchmarkEnc10x4_1M(b *testing.B) {
	benchEnc(b, testNumIn, testNumOut, mb)
}

func BenchmarkEnc10x4_4M(b *testing.B) {
	benchEnc(b, testNumIn, testNumOut, 4*mb)
}

func BenchmarkEnc10x4_16M(b *testing.B) {
	benchEnc(b, testNumIn, testNumOut, 16*mb)
}

func benchEncBase(b *testing.B, d, p, size int) {
	vects := make([][]byte, d+p)
	for j := 0; j < d+p; j++ {
		vects[j] = make([]byte, size)
	}
	for j := 0; j < d; j++ {
		rand.Seed(int64(j))
		fillRandom(vects[j])
	}
	em, err := genEncMatrixVand(d, p)
	if err != nil {
		b.Fatal(err)
	}
	g := em[d*d:]
	e := &encBase{data: d, parity: p, genMatrix: g}
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

func BenchmarkEncBase10x4_1KB(b *testing.B) {
	benchEncBase(b, testNumIn, testNumOut, kb)
}

func BenchmarkEncBase10x4_1400B(b *testing.B) {
	benchEncBase(b, testNumIn, testNumOut, 1400)
}

func BenchmarkEncBase10x4_256KB(b *testing.B) {
	benchEncBase(b, testNumIn, testNumOut, 256*kb)
}

func BenchmarkEncBase10x4_16M(b *testing.B) {
	benchEncBase(b, testNumIn, testNumOut, 16*mb)
}
