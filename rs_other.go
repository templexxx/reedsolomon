// +build !amd64

package reedsolomon

func newRS(data, parity int, encodeMatrix matrix) (enc EncodeReconster) {
	gen := encodeMatrix[data*data:]
	c := make(map[uint64]matrix)
	return &encBase{data: data, parity: parity, encode: encodeMatrix, gen: gen}
}
