/**
 * Reed-Solomon Coding over in GF(2^8).
 * Primitive Polynomial: x^8 + x^4 + x^3 + x^2 + 1 (0x1d)
 */

package reedsolomon

import "errors"

type RS struct {
	Data   int    // Number of Data Shards
	Parity int    // Number of Parity Shards
	Shards int    // Total number of Shards
	M      Matrix // encoding matrix, identity Matrix(upper) + generator Matrix(lower)
	Gen    Matrix // generator matrix(cauchy Matrix)
	INS    int    // Extensions Instruction(AVX2 or SSSE3)
}

const (
	AVX2  = 0
	SSSE3 = 1
)

var ErrTooFewShards = errors.New("reedsolomon: too few Shards given for encoding")
var ErrTooManyShards = errors.New("reedsolomon: too many Shards given for encoding")
var ErrNoSupportINS = errors.New("reedsolomon: there is no AVX2 or SSSE3")

// New : create a encoding matrix for encoding, reconstruction
func New(d, p int) (*RS, error) {
	err := checkShards(d, p)
	if err != nil {
		return nil, err
	}
	r := RS{
		Data:   d,
		Parity: p,
		Shards: d + p,
	}
	if hasSSSE3() {
		r.INS = SSSE3
	} else {
		return &r, ErrNoSupportINS
	}
	e := genEncodeMatrix(r.Shards, d) // create encoding matrix
	r.M = e
	r.Gen = NewMatrix(p, d)
	for i := range r.Gen {
		r.Gen[i] = r.M[d+i]
	}
	return &r, err
}

func checkShards(d, p int) error {
	if (d <= 0) || (p <= 0) {
		return ErrTooFewShards
	}
	if d+p >= 255 {
		return ErrTooManyShards
	}
	return nil
}

//go:noescape
func hasSSSE3() bool
