// Copyright (c) 2017 Temple3x (temple3x@gmail.com)
//
// Use of this source code is governed by the MIT License
// that can be found in the LICENSE file.

package reedsolomon

// Coefficient multiply by vector(d).
// Then write result(p).
func mulVect(c byte, d, p []byte, cpuFeature int) {
	switch cpuFeature {
	case avx512:
		tbl := lowHighTbl[int(c)*32 : int(c)*32+32]
		mulVectAVX512(tbl, d, p)
	case avx2:
		tbl := lowHighTbl[int(c)*32 : int(c)*32+32]
		mulVectAVX2(tbl, d, p)
	default:
		mulVectBase(c, d, p)
	}
}

// Coefficient multiply by vector(d).
// Then update result(p) by XOR old result(p).
func mulVectXOR(c byte, d, p []byte, cpuFeature int) {
	switch cpuFeature {
	case avx512:
		tbl := lowHighTbl[int(c)*32 : int(c)*32+32]
		mulVectXORAVX512(tbl, d, p)
	case avx2:
		tbl := lowHighTbl[int(c)*32 : int(c)*32+32]
		mulVectXORAVX2(tbl, d, p)
	default:
		mulVectXORBase(c, d, p)
	}
}

//go:noescape
func mulVectAVX2(tbl, d, p []byte)

//go:noescape
func mulVectXORAVX2(tbl, d, p []byte)

//go:noescape
func mulVectAVX512(tbl, d, p []byte)

//go:noescape
func mulVectXORAVX512(tbl, d, p []byte)
