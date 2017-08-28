// +build !amd64

package reedsolomon

func newRS(d, p int, em matrix) (enc EncodeReconster) {
	g := em[d*d:]
	return &encBase{data: d, parity: p, total: d + p, encode: em, gen: g}
}
