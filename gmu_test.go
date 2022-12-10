// Copyright (c) 2017 Temple3x (temple3x@gmail.com)
//
// Use of this source code is governed by the MIT License
// that can be found in the LICENSE file.

package reedsolomon

import (
	"bytes"
	"math/rand"
	"testing"
	"time"
)

func TestGMU(t *testing.T) {
	max := testSize

	switch getCPUFeature() {
	case featAVX2:
		testGMU(t, max, featAVX2, featNoSIMD)
	}
}

func testGMU(t *testing.T, maxSize, feat, cmpFeat int) {

	rand.Seed(time.Now().UnixNano())

	fs := featToStr(feat)

	start, n := 1, 1
	if feat != featNoSIMD {
		start, n = 16, 16 // The min size for SIMD instructions.
	}

	g := new(gmu)
	g.initFunc(feat)

	cg := new(gmu)
	cg.initFunc(cmpFeat)

	for size := start; size <= maxSize; size += n {
		for c := 0; c <= 255; c++ {
			input := make([]byte, size)
			act := make([]byte, size)
			fillRandom(input)

			g.mulVect(byte(c), input, act)
			exp := make([]byte, size)
			cg.mulVect(byte(c), input, exp)
			if !bytes.Equal(act, exp) {
				t.Fatalf("%s mismatched with %s, size: %d",
					fs, featToStr(cmpFeat), size)
			}

			g.mulVectXOR(byte(c), input, act)
			cg.mulVectXOR(byte(c), input, exp)
			if !bytes.Equal(act, exp) {
				t.Fatalf("%s mismatched with %s, size: %d",
					fs, featToStr(cmpFeat), size)
			}
		}
	}

	t.Logf("%s passed, size: [%d, %d), size = i * %d",
		fs, start, maxSize+1, n)
}
