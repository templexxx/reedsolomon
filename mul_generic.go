// Copyright (c) 2017 Temple3x (temple3x@gmail.com)
//
// Use of this source code is governed by the MIT License
// that can be found in the LICENSE file.

// +build !amd64

package reedsolomon

// Coefficient multiply by vector(d).
// Then write result(p).
func mulVect(c byte, data, parity []byte, cpuFeature int) {
	mulVectBase(c, data, parity)
}

// Coefficient multiply by vector(d).
// Then update result(p) by XOR old result(p).
func mulVectXOR(c byte, data, parity []byte, cpuFeature int) {
	mulVectXORBase(c, data, parity)
}
