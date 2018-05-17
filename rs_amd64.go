package reedsolomon

// parity = c *data
func coeffMulVect(c byte, data, parity []byte, cpuFeature int) {
	switch cpuFeature {
	case avx2:
		tbl := lowHighTbl[c][:]
		coeffMulVectAVX2(tbl, data, parity)
	case ssse3:
		tbl := lowHighTbl[c][:]
		coeffMulVectSSSE3(tbl, data, parity)
	default:
		coeffMulVectBase(c, data, parity)
	}
}

//go:noescape
func coeffMulVectAVX2(tbl, d, p []byte)

//go:noescape
func coeffMulVectSSSE3(tbl, d, p []byte)

// parity = parity xor (c * data)
func coeffMulVectUpdate(c byte, data, parity []byte, cpuFeature int) {
	switch cpuFeature {
	case avx2:
		tbl := lowHighTbl[c][:]
		coeffMulVectUpdateAVX2(tbl, data, parity)
	case ssse3:
		tbl := lowHighTbl[c][:]
		coeffMulVectUpdateSSSE3(tbl, data, parity)
	default:
		coeffMulVectUpdateBase(c, data, parity)
	}
}

//go:noescape
func coeffMulVectUpdateAVX2(tbl, d, p []byte)

//go:noescape
func coeffMulVectUpdateSSSE3(tbl, d, p []byte)
