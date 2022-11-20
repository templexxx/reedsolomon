// Copyright (c) 2017 Temple3x (temple3x@gmail.com)
//
// Use of this source code is governed by the MIT License
// that can be found in the LICENSE file.

package reedsolomon

import (
	"bytes"
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"
)

const (
	kb            = 1024
	mb            = 1024 * 1024
	testDataNum   = 10
	testParityNum = 4
	testSize      = kb
)

func TestRS_Encode(t *testing.T) {
	d, p := testDataNum, testParityNum
	max := testSize

	testEncode(t, d, p, max, base, -1)

	switch getCPUFeature() {
	case avx512:
		testEncode(t, d, p, max, avx2, base)
		testEncode(t, d, p, max, avx512, avx2)
	case avx2:
		testEncode(t, d, p, max, avx2, base)
	}
}

func testEncode(t *testing.T, d, p, maxSize, feat, cmpFeat int) {

	rand.Seed(time.Now().UnixNano())

	fs := featToStr(feat)
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
		r, err := New(d, p)
		if err != nil {
			t.Fatal(err)
		}
		r.cpuFeat = feat
		err = r.Encode(act)
		if err != nil {
			t.Fatal(err)
		}

		var f func(vects [][]byte) error
		if cmpFeat < 0 {
			f = r.mul
		} else {
			r.cpuFeat = cmpFeat
			f = r.Encode
		}
		err = f(exp)
		if err != nil {
			t.Fatal(err)
		}
		for j := range exp {
			if !bytes.Equal(exp[j], act[j]) {
				t.Fatalf("%s mismatched with %s, vect: %d, size: %d",
					fs, featToStr(cmpFeat), j, size)
			}
		}
	}

	t.Logf("%s pass %d+%d, max_size: %d",
		fs, d, p, maxSize)
}

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

// Wrap matrix.mul.
func (r *RS) mul(vects [][]byte) error {
	r.GenMatrix.mul(vects, r.DataNum, r.ParityNum, len(vects[0]))
	return nil
}

// m(generator matrix) * vectors,
// it's the basic matrix multiply.
func (m matrix) mul(vects [][]byte, d, p, n int) {
	src := vects[:d]
	out := vects[d:]
	for i := 0; i < p; i++ {
		for j := 0; j < n; j++ {
			var s uint8
			for k := 0; k < d; k++ {
				s ^= gfMul(src[k][j], m[i*d+k])
			}
			out[i][j] = s
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

		lost := makeLostRandom(d+p, rand.Intn(p+1))
		needReconst := lost[:rand.Intn(len(lost)+1)]
		dpHas := makeHasFromLost(d+p, lost)
		for _, h := range dpHas {
			copy(act[h], exp[h])
		}

		// Try to reconstruct some health vectors.
		// Although we want to reconstruct these vectors,
		// but it maybe a mistake.
		for _, nr := range needReconst {
			if rand.Intn(4) == 0 { // 1/4 chance.
				copy(act[nr], exp[nr])
			}
		}

		err = r.Reconst(act, dpHas, needReconst)
		if err != nil {
			t.Fatal(err)
		}

		for _, n := range needReconst {
			if !bytes.Equal(exp[n], act[n]) {
				t.Fatalf("reconst failed: vect: %d, size: %d", n, size)
			}
		}
	}
}

func makeHasFromLost(n int, lost []int) []int {
	s := make([]int, n-len(lost))
	c := 0
	for i := 0; i < n; i++ {
		if !isIn(i, lost) {
			s[c] = i
			c++
		}
	}
	return s
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
	d, p := 64, 64 // Big enough for cache effects.
	r, err := New(d, p)
	if err != nil {
		t.Fatal(err)
	}
	// Enable Cache.
	r.cacheEnabled = true
	r.inverseMatrix = new(sync.Map)

	rand.Seed(time.Now().UnixNano())
	dpHas := makeHasRandom(d+p, p)
	var dLost []int
	for _, h := range dpHas {
		if h < d {
			dLost = append(dLost, h)
		}
	}
	start1 := time.Now()
	exp, err := r.getReconstMatrix(dpHas, dLost)
	if err != nil {
		t.Fatal(err)
	}
	cost1 := time.Now().Sub(start1)

	start2 := time.Now()
	act, err := r.getReconstMatrix(dpHas, dLost)
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
		4 * kb,
		mb,
		8 * mb,
	}

	var feats []int
	switch getCPUFeature() {
	case avx512:
		feats = append(feats, avx512)
		feats = append(feats, avx2)
	case avx2:
		feats = append(feats, avx2)
	}
	feats = append(feats, base)

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
	r, err := New(d, p)
	if err != nil {
		b.Fatal(err)
	}
	r.cpuFeat = feat

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
	size := 4 * kb

	b.Run("", benchmarkReconst(benchReconst, d, p, size))
}

func benchmarkReconst(f func(*testing.B, int, int, int, []int, []int), d, p, size int) func(*testing.B) {

	datas := make([]int, d)
	for i := range datas {
		datas[i] = i
	}
	return func(b *testing.B) {
		for i := 1; i <= p; i++ {
			lost := datas[:i]
			dpHas := makeHasFromLost(d+p, lost)
			b.Run(fmt.Sprintf("(%d+%d)-%s-reconst_%d_data_vects-%s",
				d, p, byteToStr(size), i, featToStr(getCPUFeature())),
				func(b *testing.B) { f(b, d, p, size, dpHas, lost) })
		}
	}
}

func benchReconst(b *testing.B, d, p, size int, dpHas, needReconst []int) {
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
		err = r.Reconst(vects, dpHas, needReconst)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRS_Update(b *testing.B) {
	d, p := 10, 4
	size := 4 * kb

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
	size := 4 * kb

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

func featToStr(f int) string {
	switch f {
	case avx512:
		return "AVX512"
	case avx2:
		return "AVX2"
	case base:
		return "Base"
	default:
		return "Tested"
	}
}

func fillRandom(p []byte) {
	rand.Read(p)
}

func byteToStr(n int) string {
	if n >= mb {
		return fmt.Sprintf("%dMB", n/mb)
	}

	return fmt.Sprintf("%dKB", n/kb)
}
