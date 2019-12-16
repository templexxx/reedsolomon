// Copyright (c) 2017 Temple3x (temple3x@gmail.com)
//
// Use of this source code is governed by the MIT License
// that can be found in the LICENSE file.

package reedsolomon

// Coefficient multiply by vector(d) in basic way (byte by byte search multiply table).
// Then write result(p).
func mulVectBase(c byte, d, p []byte) {

	t := mulTbl[c][:256]
	for i := 0; i < len(d); i++ {
		p[i] = t[d[i]]
	}
}

// Coefficient multiply by vector(d) in basic method (byte by byte search multiply table).
// Then update result(p) by XOR old result(p).
func mulVectXORBase(c byte, d, p []byte) {

	t := mulTbl[c][:256]
	for i := 0; i < len(d); i++ {
		p[i] ^= t[d[i]]
	}
}

func gfmul(a, b uint8) uint8 {
	return mulTbl[a][b]
}
