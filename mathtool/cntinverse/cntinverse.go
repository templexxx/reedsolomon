// Copyright (c) 2017 Temple3x (temple3x@gmail.com)
//
// Use of this source code is governed by the MIT License
// that can be found in the LICENSE file.
//
// Copyright ©2016 The Gonum Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This tools will calculate the number of inverse matrices
// with specific data & parity number.
package main

import (
	"flag"
	"fmt"
	"math"
	"os"
)

var vects = flag.Uint64("vects", 20, "number of vectors (data+parity)")
var data = flag.Uint64("data", 0, "number of data vectors; keep it empty if you want to "+
	"get the max num of inverse matrix")

func init() {
	flag.Usage = func() {
		fmt.Printf("Usage of %s:\n", os.Args[0])
		fmt.Println("  cntinverse [-flags]")
		fmt.Println("  Valid flags:")
		flag.PrintDefaults()
	}
}

func main() {
	flag.Parse()

	n := float64(*vects)
	k := float64(*data)

	if k == 0 {
		k = n / 2
	}
	fmt.Printf("num of inverse matrices for vectors ≈ %d, data: %d: %.f \n",
		uint64(n),
		uint64(k),
		generalizedBinomial(n, k))
}

const (
	errNegInput = "combination: negative input"
	badSetSize  = "combination: n < k"
)

// generalizedBinomial returns the generalized binomial coefficient of (n, k),
// defined as
//
//	Γ(n+1) / (Γ(k+1) Γ(n-k+1))
//
// where Γ is the Gamma function. generalizedBinomial is useful for continuous
// relaxations of the binomial coefficient, or when the binomial coefficient value
// may overflow int. In the latter case, one may use math/big for an exact
// computation.
//
// n and k must be non-negative with n >= k, otherwise generalizedBinomial will panic.
func generalizedBinomial(n, k float64) float64 {
	return math.Exp(logGeneralizedBinomial(n, k))
}

// logGeneralizedBinomial returns the log of the generalized binomial coefficient.
// See generalizedBinomial for more information.
func logGeneralizedBinomial(n, k float64) float64 {
	if n < 0 || k < 0 {
		panic(errNegInput)
	}
	if n < k {
		panic(badSetSize)
	}
	a, _ := math.Lgamma(n + 1)
	b, _ := math.Lgamma(k + 1)
	c, _ := math.Lgamma(n - k + 1)
	return a - b - c
}
