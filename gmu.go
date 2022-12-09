package reedsolomon

// galois field multiplying unit
type gmu struct {
	// output = c * input
	mulVect func(c byte, input, output []byte)
	// output ^= c * input
	mulVectXOR func(c byte, input, output []byte)
}

func mulVect(c byte, input, output []byte) {
	t := mulTbl[c][:256]
	for i := 0; i < len(input); i++ {
		output[i] = t[input[i]]
	}
}

func mulVectXOR(c byte, d, p []byte) {
	t := mulTbl[c][:256]
	for i := 0; i < len(d); i++ {
		p[i] ^= t[d[i]]
	}
}

// a * b
func gfMul(a, b uint8) uint8 {
	return mulTbl[a][b]
}
