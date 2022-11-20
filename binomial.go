// Copyright ©2016 The Gonum Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package reedsolomon

import "math"

const (
	errNegInput = "combination: negative input"
	badSetSize  = "combination: n < k"
)

// GeneralizedBinomial returns the generalized binomial coefficient of (n, k),
// defined as
//
//	Γ(n+1) / (Γ(k+1) Γ(n-k+1))
//
// where Γ is the Gamma function. GeneralizedBinomial is useful for continuous
// relaxations of the binomial coefficient, or when the binomial coefficient value
// may overflow int. In the latter case, one may use math/big for an exact
// computation.
//
// n and k must be non-negative with n >= k, otherwise GeneralizedBinomial will panic.
func GeneralizedBinomial(n, k float64) float64 {
	return math.Exp(logGeneralizedBinomial(n, k))
}

// logGeneralizedBinomial returns the log of the generalized binomial coefficient.
// See GeneralizedBinomial for more information.
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
