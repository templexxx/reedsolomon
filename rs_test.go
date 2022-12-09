// Copyright (c) 2017 Temple3x (temple3x@gmail.com)
//
// Use of this source code is governed by the MIT License
// that can be found in the LICENSE file.

package reedsolomon

import (
	"bytes"
	"fmt"
	"math"
	"math/rand"
	"sync"
	"testing"
	"time"
)

const (
	testDataNum   = 10
	testParityNum = 4
	testSize      = kib // enough for covering branches when using SIMD
)

// Check basic matrix multiply.
// Powered by MATLAB.
func TestRS_mul(t *testing.T) {
	d, p := 5, 5
	r, err := New(d, p)
	if err != nil {
		t.Fatal(err)
	}
	vects := [][]byte{{0}, {4}, {2}, {6}, {8}, {0}, {0}, {0}, {0}, {0}}
	_ = r.mul(vects)
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
		t.Fatal("vect 9 mismatch")
	}
}

func TestRS_Encode(t *testing.T) {
	d, p := testDataNum, testParityNum
	max := testSize

	testEncode(t, d, p, max, featNoSIMD, featUnknown)

	switch getCPUFeature() {
	case featAVX2:
		testEncode(t, d, p, max, featAVX2, featNoSIMD)
	}
}

func testEncode(t *testing.T, d, p, maxSize, feat, cmpFeat int) {

	fs := featToStr(feat)
	cmpfs := featToStr(cmpFeat)

	for size := 1; size <= maxSize; size++ {
		exp := make([][]byte, d+p)
		act := make([][]byte, d+p)
		for j := 0; j < d+p; j++ {
			exp[j], act[j] = make([]byte, size), make([]byte, size)
		}
		for j := 0; j < d; j++ {
			fillRandom(exp[j])
			copy(act[j], exp[j])
		}
		r, err := newWithFeature(d, p, feat)
		if err != nil {
			t.Fatal(err)
		}
		err = r.Encode(act)
		if err != nil {
			t.Fatal(err)
		}

		var f func(vects [][]byte) error
		if cmpFeat == featUnknown {
			f = r.mul
		} else {
			r2, err := newWithFeature(d, p, cmpFeat)
			if err != nil {
				t.Fatal(err)
			}
			f = r2.Encode
		}
		err = f(exp)
		if err != nil {
			t.Fatal(err)
		}

		for j := range exp {
			if !bytes.Equal(exp[j], act[j]) {
				t.Fatalf("%s mismatched with %s: %d+%d, vect: %d, size: %d",
					fs, cmpfs, d, p, j, size)
			}
		}
	}

	t.Logf("%s matched %s: %d+%d, size: [1, %d)",
		fs, cmpfs, d, p, maxSize+1)
}

func TestMakeInverseCacheKey(t *testing.T) {

	type tc struct {
		survived []int
		exp      uint64
	}
	cases := []tc{
		{[]int{0}, 1},
		{[]int{1}, 2},
		{[]int{0, 1}, 3},
		{[]int{0, 1, 2}, 7},
		{[]int{0, 2}, 5},
	}
	survived := make([]int, 64)
	for i := range survived {
		survived[i] = i
	}
	cases = append(cases, tc{survived, math.MaxUint64})
	for i, c := range cases {
		got := makeInverseCacheKey(c.survived)
		if got != c.exp {
			t.Fatalf("case: %d, exp: %d, got: %d, survived: %#v", i, c.exp, got, c.survived)
		}
	}
}

func TestRS_Reconst(t *testing.T) {
	testReconst(t, testDataNum, testParityNum, testSize, 128)
}

func testReconst(t *testing.T, d, p, size, loop int) {

	rand.Seed(time.Now().UnixNano())

	r, err := New(d, p)
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < loop; i++ {

		exp := make([][]byte, d+p)
		act := make([][]byte, d+p)
		for j := 0; j < d+p; j++ {
			exp[j], act[j] = make([]byte, size), make([]byte, size)
		}
		for j := 0; j < d; j++ {
			fillRandom(exp[j])
		}

		err = r.Encode(exp)
		if err != nil {
			t.Fatal(err)
		}

		survived, needReconst := genIdxRand(d, p, rand.Intn(d+p), rand.Intn(p+1))
		for _, i := range survived {
			copy(act[i], exp[i])
		}

		// Pollute vectors need to be reconstructed.
		for _, nr := range needReconst {
			if rand.Intn(4) == 1 { // 1/4 chance.
				fillRandom(act[nr])
			}
		}

		err = r.Reconst(act, survived, needReconst)
		if err != nil {
			t.Fatal(err)
		}

		for _, n := range needReconst {
			if !bytes.Equal(exp[n], act[n]) {
				t.Fatalf("mismatched vect: %d, size: %d", n, size)
			}
		}
	}
}

func TestRS_Update(t *testing.T) {
	testUpdate(t, testDataNum, testParityNum, testSize)
}

func testUpdate(t *testing.T, d, p, size int) {

	rand.Seed(time.Now().UnixNano())

	for i := 0; i < d; i++ {
		act := make([][]byte, d+p)
		exp := make([][]byte, d+p)
		for j := 0; j < d+p; j++ {
			act[j], exp[j] = make([]byte, size), make([]byte, size)
		}
		for j := 0; j < d; j++ {
			fillRandom(exp[j])
			copy(act[j], exp[j])
		}

		r, err := New(d, p)
		if err != nil {
			t.Fatal(err)
		}
		err = r.Encode(act)
		if err != nil {
			t.Fatal(err)
		}

		newData := make([]byte, size)
		fillRandom(newData)
		updateRow := i
		err = r.Update(act[updateRow], newData, updateRow, act[d:d+p])
		if err != nil {
			t.Fatal(err)
		}

		copy(exp[updateRow], newData)
		err = r.Encode(exp)
		if err != nil {
			t.Fatal(err)
		}
		for j := d; j < d+p; j++ {
			if !bytes.Equal(act[j], exp[j]) {
				t.Fatalf("update failed: vect: %d, size: %d", j, size)
			}
		}
	}
}

func TestRS_Replace(t *testing.T) {
	testReplace(t, testDataNum, testParityNum, testSize, 128, true)
	testReplace(t, testDataNum, testParityNum, testSize, 128, false)
}

func testReplace(t *testing.T, d, p, size, loop int, toZero bool) {

	rand.Seed(time.Now().UnixNano())

	for i := 0; i < loop; i++ {
		replaceRows := makeReplaceRowRandom(d)
		act := make([][]byte, d+p)
		exp := make([][]byte, d+p)
		for j := 0; j < d+p; j++ {
			act[j], exp[j] = make([]byte, size), make([]byte, size)
		}
		for j := 0; j < d; j++ {
			fillRandom(exp[j])
			copy(act[j], exp[j])
		}

		data := make([][]byte, len(replaceRows))
		for i, rr := range replaceRows {
			data[i] = make([]byte, size)
			copy(data[i], exp[rr])
		}

		if toZero {
			for _, rr := range replaceRows {
				exp[rr] = make([]byte, size)
			}
		}

		r, err := New(d, p)
		if err != nil {
			t.Fatal(err)
		}
		err = r.Encode(exp)
		if err != nil {
			t.Fatal(err)
		}

		if !toZero {
			for _, rr := range replaceRows {
				act[rr] = make([]byte, size)
			}
		}
		err = r.Encode(act)
		if err != nil {
			t.Fatal(err)
		}

		err = r.Replace(data, replaceRows, act[d:])
		if err != nil {
			t.Fatal(err)
		}

		for j := d; j < d+p; j++ {
			if !bytes.Equal(act[j], exp[j]) {
				t.Fatalf("replace failed: vect: %d, size: %d", j, size)
			}
		}

	}
}

func makeReplaceRowRandom(d int) []int {
	rand.Seed(time.Now().UnixNano())

	n := rand.Intn(d + 1)
	s := make([]int, 0)
	c := 0
	for i := 0; i < 64; i++ {
		if c == n {
			break
		}
		v := rand.Intn(d)
		if !isIn(v, s) {
			s = append(s, v)
			c++
		}
	}
	if c == 0 {
		s = []int{0}
	}
	return s
}

func TestRS_getReconstMatrixFromCache(t *testing.T) {
	d, p := 64, 64 // Big enough for showing cache effects.
	r, err := New(d, p)
	if err != nil {
		t.Fatal(err)
	}
	// Enable Cache.
	r.inverseCacheEnabled = true
	r.inverseCache = new(sync.Map)
	r.inverseCacheMax = 1

	rand.Seed(time.Now().UnixNano())

	var survived, needReconst []int // genReconstMatrix needs survived vectors & data vectors need to be reconstructed.
	for {
		var needReconstData int
		survived, needReconst = genIdxRand(d, p, d, p)
		survived, needReconst, needReconstData, err = r.checkReconst(survived, needReconst)
		if err != nil {
			t.Fatal(err)
		}
		if needReconstData != 0 { // At least has one.
			needReconst = needReconst[:needReconstData]
			break
		}
	}

	start1 := time.Now()
	exp, err := r.getReconstMatrix(survived, needReconst)
	if err != nil {
		t.Fatal(err)
	}
	cost1 := time.Now().Sub(start1)

	start2 := time.Now()
	act, err := r.getReconstMatrix(survived, needReconst)
	if err != nil {
		t.Fatal(err)
	}
	cost2 := time.Now().Sub(start2)

	if cost2 >= cost1 {
		t.Fatal("cache is much slower than expect")
	}

	if !bytes.Equal(act, exp) {
		t.Fatal("cache matrix mismatched")
	}
}

func BenchmarkRS_Encode(b *testing.B) {
	dps := [][]int{
		{10, 2},
		{10, 4},
		{12, 4},
	}

	sizes := []int{
		4 * kib,
		mib,
		8 * mib,
	}

	var feats []int
	switch getCPUFeature() {
	case featAVX2:
		feats = append(feats, featAVX2)
	}
	feats = append(feats, featNoSIMD)

	b.Run("", benchmarkEncode(benchEnc, feats, dps, sizes))
}

func benchmarkEncode(f func(*testing.B, int, int, int, int), feats []int, dps [][]int, sizes []int) func(*testing.B) {
	return func(b *testing.B) {
		for _, feat := range feats {
			for _, dp := range dps {
				d, p := dp[0], dp[1]
				for _, size := range sizes {
					b.Run(fmt.Sprintf("(%d+%d)-%s-%s", d, p, byteToStr(size), featToStr(feat)), func(b *testing.B) {
						f(b, d, p, size, feat)
					})
				}
			}
		}
	}
}

func benchEnc(b *testing.B, d, p, size, feat int) {

	vects := make([][]byte, d+p)
	for j := 0; j < d+p; j++ {
		vects[j] = make([]byte, size)
	}
	for j := 0; j < d; j++ {
		fillRandom(vects[j])
	}
	r, err := newWithFeature(d, p, feat)
	if err != nil {
		b.Fatal(err)
	}

	b.SetBytes(int64((d + p) * size))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err = r.Encode(vects)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRS_Reconst(b *testing.B) {
	d, p := 10, 4
	size := 4 * kib

	b.Run("", benchmarkReconst(benchReconst, d, p, size))
}

func benchmarkReconst(f func(*testing.B, int, int, int, []int, []int), d, p, size int) func(*testing.B) {

	datas := make([]int, d)
	for i := range datas {
		datas[i] = i
	}
	return func(b *testing.B) {
		for i := 1; i <= p; i++ {
			survived, needReconst := genIdxRand(d, p, d+p-i, i)
			b.Run(fmt.Sprintf("(%d+%d)-%s-reconst_%d_data_vects-%s",
				d, p, byteToStr(size), i, featToStr(getCPUFeature())),
				func(b *testing.B) { f(b, d, p, size, survived, needReconst) })
		}
	}
}

func benchReconst(b *testing.B, d, p, size int, survived, needReconst []int) {
	vects := make([][]byte, d+p)
	for j := 0; j < d+p; j++ {
		vects[j] = make([]byte, size)
	}
	for j := 0; j < d; j++ {
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

	b.SetBytes(int64((d + len(needReconst)) * size))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err = r.Reconst(vects, survived, needReconst)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRS_checkReconst(b *testing.B) {
	dps := [][2]int{
		{10, 4},
	}
	for _, dp := range dps {
		d := dp[0]
		p := dp[1]
		r, err := New(d, p)
		if err != nil {
			b.Fatal(err)
		}
		for i := 1; i <= p; i++ {
			is, ir := genIdxRand(d, p, d, i)
			b.Run(fmt.Sprintf("d:%d,p:%d,survived:%d,need_reconst:%d", d, p, len(is), len(ir)),
				func(b *testing.B) {
					b.ResetTimer()
					for j := 0; j < b.N; j++ {
						_, _, _, err = r.checkReconst(is, ir)
						if err != nil {
							b.Fatal(err)
						}
					}
				})
		}
	}
}

func BenchmarkRS_Update(b *testing.B) {
	d, p := 10, 4
	size := 4 * kib

	b.Run("", benchmarkUpdate(benchUpdate, d, p, size))
}

func benchmarkUpdate(f func(*testing.B, int, int, int, int), d, p, size int) func(*testing.B) {

	return func(b *testing.B) {
		updateRow := rand.Intn(d)
		b.Run(fmt.Sprintf("(%d+%d)-%s-%s",
			d, p, byteToStr(size), featToStr(getCPUFeature())),
			func(b *testing.B) { f(b, d, p, size, updateRow) })
	}
}

func benchUpdate(b *testing.B, d, p, size, updateRow int) {
	vects := make([][]byte, d+p)
	for j := 0; j < d+p; j++ {
		vects[j] = make([]byte, size)
	}
	for j := 0; j < d; j++ {
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

	b.SetBytes(int64((p + 2 + p) * size))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err = r.Update(vects[updateRow], newData, updateRow, vects[d:])
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRS_Replace(b *testing.B) {
	d, p := 10, 4
	size := 4 * kib

	b.Run("", benchmarkReplace(benchReplace, d, p, size))
}

func benchmarkReplace(f func(*testing.B, int, int, int, int), d, p, size int) func(*testing.B) {

	return func(b *testing.B) {
		for i := 1; i <= d-p; i++ {
			b.Run(fmt.Sprintf("(%d+%d)-%s-replace_%d_data_vects-%s",
				d, p, byteToStr(size), i, featToStr(getCPUFeature())),
				func(b *testing.B) { f(b, d, p, size, i) })
		}
	}
}

func benchReplace(b *testing.B, d, p, size, n int) {
	vects := make([][]byte, d+p)
	for j := 0; j < d+p; j++ {
		vects[j] = make([]byte, size)
	}
	for j := 0; j < d; j++ {
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

	updateRows := make([]int, n)
	for i := range updateRows {
		updateRows[i] = i
	}
	b.SetBytes(int64((n + p + p) * size))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err = r.Replace(vects[:n], updateRows, vects[d:])
		if err != nil {
			b.Fatal(err)
		}
	}
}
