// Copyright (c) 2017 Temple3x (temple3x@gmail.com)
//
// Use of this source code is governed by the MIT License
// that can be found in the LICENSE file.

package reedsolomon

//go:noescape
func mulVectAVX2(tbl, d, p []byte)

//go:noescape
func mulVectXORAVX2(tbl, d, p []byte)
