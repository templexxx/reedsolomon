package reedsolomon

import (
	"bytes"
	"math/rand"
	"testing"

	"fmt"

	krs "github.com/klauspost/reedsolomon"
)

const (
	kb         = 1024
	mb         = 1024 * 1024
	testNumIn  = 10
	testNumOut = 4
)

//func TestSwap(t *testing.T) {
//	vects := make([][]byte, testNumIn+testNumOut)
//	for j := 0; j < testNumIn+testNumOut; j++ {
//		vects[j] = make([]byte, 1)
//		rand.Seed(int64(j))
//		fillRandom(vects[j])
//	}
//	pos := []int{9, 8, 0, 1, 11, 13, 5, 4, 6, 2}
//	dLost := []int{3, 7}
//	fmt.Println(vects)
//	swap2(vects, pos, dLost, 10, 4)
//}

//func swap2(vects [][]byte, pos, dLost []int, d, p int) {
//	order := make([]int, len(vects))
//	for i := 0; i < len(vects); i++ {
//		order[i] = i
//	}
//	ppos := make([]int, len(dLost))
//	pc := 0
//	for _, p := range pos {
//		if pc == len(dLost) {
//			break
//		}
//		if p >= d {
//			ppos[pc] = p
//			pc++
//		}
//	}
//	sort.Ints(ppos)
//	sort.Ints(dLost)
//	m := make(map[int]int)
//
//	for i, l := range dLost {
//
//		order[l], order[ppos[i]] = order[ppos[i]], order[l]
//		vects[l], vects[ppos[i]] = vects[ppos[i]], vects[l]
//		if ppos[i] != i+d {
//			m[ppos[i]] = i + d
//			order[i+d], order[ppos[i]] = order[ppos[i]], order[i+d]
//			vects[i+d], vects[ppos[i]] = vects[ppos[i]], vects[i+d]
//		}
//	}
//	fmt.Println("after fill data", order)
//	fmt.Println("after fill data", vects)
//	if len(dLost) == p {
//		for i, l := range dLost {
//			order[l], order[ppos[i]] = order[ppos[i]], order[l]
//			vects[l], vects[ppos[i]] = vects[ppos[i]], vects[l]
//		}
//		//swapback(order, d, len(dLost))
//		fmt.Println("back", order)
//		fmt.Println("back", vects)
//		return
//	}
//	//done := 0
//	//for i := d; i < len(order); i++ {
//	//	if done == len(dLost) {
//	//		break
//	//	}
//	//	if order[i] >= d {
//	//		for j := i + 1; j < len(order); j++ {
//	//			if order[j] < d {
//	//				m[j] = i
//	//				order[i], order[j] = order[j], order[i]
//	//				vects[i], vects[j] = vects[j], vects[i]
//	//				done++
//	//				break
//	//			}
//	//		}
//	//	}
//	//}
//	//fmt.Println("now can calc lost", order)
//	//fmt.Println("now can calc lost", vects)
//	fmt.Println(m)
//	for i := len(vects) - 1; i >= d; i-- {
//		if v, ok := m[i]; ok {
//			order[i], order[v] = order[v], order[i]
//			vects[i], vects[v] = vects[v], vects[i]
//		}
//	}
//
//	fmt.Println("after map", order)
//	fmt.Println("after map", vects)
//	for i, l := range dLost {
//		order[l], order[ppos[i]] = order[ppos[i]], order[l]
//		vects[l], vects[ppos[i]] = vects[ppos[i]], vects[l]
//	}
//	//swapback(order, d, len(dLost))
//	fmt.Println("back", order)
//	fmt.Println("back", vects)
//}

const verifySize = 256 + 32 + 16 + 15

func verifyKEnc(t *testing.T, d, p, ins int) {
	for i := 1; i <= verifySize; i++ {
		vects1 := make([][]byte, testNumIn+testNumOut)
		vects2 := make([][]byte, testNumIn+testNumOut)
		for j := 0; j < testNumIn+testNumOut; j++ {
			vects1[j] = make([]byte, i)
			vects2[j] = make([]byte, i)
		}
		for j := 0; j < testNumIn; j++ {
			rand.Seed(int64(j))
			fillRandom(vects1[j])
			copy(vects2[j], vects1[j])
		}
		em, err := genEncMatrixVand(testNumIn, testNumOut)
		if err != nil {
			t.Fatal("rs.verifySIMDEnc: ", err)
		}
		g := em[testNumIn*testNumIn:]
		tbl := initTbl(g, testNumOut, testNumIn)
		var e EncodeReconster
		switch ins {
		case avx2:
			e = &encAVX2{data: d, parity: p, gen: g, tbl: tbl}
		case ssse3:
			e = &encSSSE3{data: d, parity: p, gen: g, tbl: tbl}
		default:
			e = &encBase{data: d, parity: p, gen: g}
		}
		err = e.Encode(vects1)
		if err != nil {
			t.Fatal("rs.verifySIMDEnc: ", err)
		}
		ek, err := krs.New(testNumIn, testNumOut)
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

func TestVerifyKAVX2(t *testing.T) {
	if !hasAVX2() {
		t.Fatal("rs.TestVerifyAVX2: no AVX2")
	}
	verifyKEnc(t, testNumIn, testNumOut, avx2)
}

func TestVerifyKSSSE3(t *testing.T) {
	if !hasAVX2() {
		t.Fatal("rs.TestVerifyAVX2: no SSSE3")
	}
	verifyKEnc(t, testNumIn, testNumOut, ssse3)
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

func TestVerifyInitTbl(t *testing.T) {
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

func benchEnc(b *testing.B, d, p, size int) {
	vects := make([][]byte, d+p)
	for j := 0; j < d+p; j++ {
		vects[j] = make([]byte, size)
	}
	for j := 0; j < d; j++ {
		rand.Seed(int64(j))
		fillRandom(vects[j])
	}
	//e, _ := New(d, p)
	em := genEncMatrixCauchy(d, p)
	g := em[d*d:]
	e := &encAVX2{data: d, parity: p, gen: g}
	//err := e.Encode(vects)
	err := e.encodeWithGen(vects)
	if err != nil {
		b.Fatal(err)
	}
	b.SetBytes(int64(d * size))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err = e.encodeWithGen(vects)
		//	err = e.Encode(vects)
		if err != nil {
			b.Fatal(err)
		}
	}
}

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
	pos := make([]int, d)
	dLost := make([]int, 0)
	pLost := make([]int, 0)
	for _, l := range lost {
		if l < d {
			dLost = append(dLost, l)
		} else {
			pLost = append(pLost, l)
		}
	}
	cnt := 0
	for i := 0; i < d+p; i++ {
		ok := true
		for _, l := range lost {
			if i == l {
				ok = false
			}
		}
		if ok {
			pos[cnt] = i
			cnt++
		}
	}
	b.SetBytes(int64(d * size))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err = e.ReconstWithPos(vects, pos, dLost, pLost)
		if err != nil {
			b.Fatal(err)
		}
	}
}

//
func verifySIMDReconst(t *testing.T, d, p, ins int) {
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
			e = &encAVX2{data: d, parity: p, encode: em, gen: g, tbl: tbl, enableCache: true}
		case ssse3:
			e = &encSSSE3{data: d, parity: p, encode: em, gen: g, tbl: tbl, enableCache: true}
		}
		//e = &encBase{data: d, parity: p, gen: g, encode: em}
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
		//fmt.Println("expect", vects1)
		for k, v1 := range vects1 {
			if !bytes.Equal(v1, vects2[k]) {
				var ext string
				switch ins {
				case avx2:
					ext = "avx2"
				case ssse3:
					ext = "ssse3"
				}
				t.Fatalf("%s no match reconst; vect: %d; size: %d", ext, k, i)
			}
		}
	}
}

//
func TestVerifyAVX2Reconst(t *testing.T) {
	verifySIMDReconst(t, testNumIn, testNumOut, avx2)
}

func TestVerifySSSE3Reconst(t *testing.T) {
	verifySIMDReconst(t, testNumIn, testNumOut, ssse3)
}

//func TestCorrupt(t *testing.T) {
//	vects := make([][]byte, 14)
//	corruptRandom(vects, 10, 4)
//}

func corruptRandom(shards [][]byte, dataShards, parityShards int) {
	shardsToCorrupt := rand.Intn(parityShards)
	fmt.Println(shardsToCorrupt)
	for i := 1; i <= shardsToCorrupt; i++ {
		k := rand.Intn(dataShards + parityShards)
		fmt.Println(k)
		shards[k] = nil
	}
}
