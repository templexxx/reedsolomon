// +build !amd64

package reedsolomon

// parity = c *data
func coeffMulVect(c byte, data, parity []byte, cpuFeature int) {
	coeffMulVectBase(c, data, parity)
}

// parity = parity xor (c * data)
func coeffMulVectUpdate(c byte, data, parity []byte, cpuFeature int) {
	coeffMulVectUpdateBase(c, data, parity)
}
