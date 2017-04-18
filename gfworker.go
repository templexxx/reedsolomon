package reedsolomon

//go:noescape
func gfMulAVX2(low, high, in, out []byte)

//go:noescape
func gfMulXorAVX2(low, high, in, out []byte)

//go:noescape
func gfMulSSSE3(low, high, in, out []byte)

//go:noescape
func gfMulXorSSSE3(low, high, in, out []byte)

// use AVX2 to calc remain
func gfMulRemain(coeff byte, input, output []byte, size int) {
	var done int
	if size < 32 {
		mt := mulTable[coeff]
		for i := done; i < size; i++ {
			output[i] = mt[input[i]]
		}
	} else {
		gfMulAVX2(mulTableLow[coeff][:], mulTableHigh[coeff][:], input, output)
		done = (size >> 5) << 5
		remain := size - done
		if remain > 0 {
			mt := mulTable[coeff]
			for i := done; i < size; i++ {
				output[i] = mt[input[i]]
			}
		}
	}
}

// use AVX2 to calc remain
func gfMulRemainXor(coeff byte, input, output []byte, size int) {
	var done int
	if size < 32 {
		mt := mulTable[coeff]
		for i := done; i < size; i++ {
			output[i] ^= mt[input[i]]
		}
	} else {
		gfMulXorAVX2(mulTableLow[coeff][:], mulTableHigh[coeff][:], input, output)
		done = (size >> 5) << 5
		remain := size - done
		if remain > 0 {
			mt := mulTable[coeff]
			for i := done; i < size; i++ {
				output[i] ^= mt[input[i]]
			}
		}
	}
}

// use SSSE3 to calc remain
func gfMulRemainS(coeff byte, input, output []byte, size int) {
	var done int
	if size < 16 {
		mt := mulTable[coeff]
		for i := done; i < size; i++ {
			output[i] = mt[input[i]]
		}
	} else {
		gfMulSSSE3(mulTableLow[coeff][:], mulTableHigh[coeff][:], input, output)
		done = (size >> 4) << 4
		remain := size - done
		if remain > 0 {
			mt := mulTable[coeff]
			for i := done; i < size; i++ {
				output[i] = mt[input[i]]
			}
		}
	}
}

// use SSSE3 to calc remain
func gfMulRemainXorS(coeff byte, input, output []byte, size int) {
	var done int
	if size < 16 {
		mt := mulTable[coeff]
		for i := done; i < size; i++ {
			output[i] ^= mt[input[i]]
		}
	} else {
		gfMulXorSSSE3(mulTableLow[coeff][:], mulTableHigh[coeff][:], input, output)
		done = (size >> 4) << 4
		remain := size - done
		if remain > 0 {
			mt := mulTable[coeff]
			for i := done; i < size; i++ {
				output[i] ^= mt[input[i]]
			}
		}
	}
}
