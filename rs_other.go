// +build !amd64 noasm

package reedsolomon

func newRS(data, parity int, encodeMatrix matrix) (enc EncodeReconster) {
	c := make(map[uint64]matrix)
	return &encBase{data: data, parity: parity, encodeMatrix: encodeMatrix}
}
