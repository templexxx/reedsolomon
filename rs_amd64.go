package reedsolomon

// parity = c *data
func coeffMulVect(c byte, data, parity []byte, cpuFeature int) {
	switch cpuFeature {
	case avx512:
		tbl := lowHighTbl[c][:]
		coeffMulVectAVX512(tbl, data, parity)
	case avx2:
		tbl := lowHighTbl[c][:]
		coeffMulVectAVX2(tbl, data, parity)
	default:
		coeffMulVectBase(c, data, parity)
	}
}

//go:noescape
func coeffMulVectAVX512(tbl, d, p []byte)

//go:noescape
func coeffMulVectAVX2(tbl, d, p []byte)

// parity = parity xor (c * data)
func coeffMulVectUpdate(c byte, data, parity []byte, cpuFeature int) {
	switch cpuFeature {
	case avx512:
		tbl := lowHighTbl[c][:]
		coeffMulVectUpdateAVX512(tbl, data, parity)
	case avx2:
		tbl := lowHighTbl[c][:]
		coeffMulVectUpdateAVX2(tbl, data, parity)
	default:
		coeffMulVectUpdateBase(c, data, parity)
	}
}

//go:noescape
func coeffMulVectUpdateAVX512(tbl, d, p []byte)

//go:noescape
func coeffMulVectUpdateAVX2(tbl, d, p []byte)
