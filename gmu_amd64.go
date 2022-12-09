// Copyright (c) 2017 Temple3x (temple3x@gmail.com)
//
// Use of this source code is governed by the MIT License
// that can be found in the LICENSE file.

package reedsolomon

func (g *gmu) initFunc(feat int) {
	switch feat {
	case featAVX2:
		g.mulVect = mulVectAVX2C
		g.mulVectXOR = mulVectXORAVX2C
	default:
		g.mulVect = mulVect
		g.mulVectXOR = mulVectXOR
	}
}

func mulVectAVX2C(c byte, d, p []byte) {
	tbl := lowHighTbl[int(c)*32 : int(c)*32+32]
	mulVectAVX2(tbl, d, p)
}

func mulVectXORAVX2C(c byte, d, p []byte) {
	tbl := lowHighTbl[int(c)*32 : int(c)*32+32]
	mulVectXORAVX2(tbl, d, p)
}

//go:noescape
func mulVectAVX2(tbl, d, p []byte)

//go:noescape
func mulVectXORAVX2(tbl, d, p []byte)
