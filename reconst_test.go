package reedsolomon

import (
	"testing"
)

//func TestVerifyReconstBase(t *testing.T) {
//
//}

// verify reconst in base
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
	e := &encBase{data: d, parity: p, total: d + p, gen: g, encode: em}
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
