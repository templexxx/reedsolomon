package reedsolomon

// size of sub-matrix
const UnitSize int = 16 * 1024

// Size of Shard must be integral multiple of 256B
func (r rsAVX2) Encode(in, out Matrix) (err error) {
	size := len(in[0])
	start, end := 0, 0
	do := UnitSize
	if size <= UnitSize {
		r.matrixMul(start, size, in, out)
	} else {
		for start < size {
			end = start + do
			if end <= size {
				r.matrixMul(start, end, in, out)
				start = end
			} else {
				r.matrixMul(start, size, in, out)
				start = size
			}
		}
	}
	return
}

// Size of Shard must be integral multiple of 16B
func (r rsSSSE3) Encode(in, out Matrix) (err error) {
	size := len(in[0])
	start, end := 0, 0
	do := UnitSize
	for start < size {
		end = start + do
		if end <= size {
			r.matrixMul(start, end, in, out)
			start = end
		} else {
			r.matrixMul(start, size, in, out)
			start = size
		}
	}
	return
}

func (r rsBase) Encode(in, out Matrix) (err error) {
	gen := r.gen
	for i := 0; i < r.in; i++ {
		data := in[i]
		for oi := 0; oi < r.out; oi++ {
			if i == 0 {
				mulBase(gen[oi][i], data, out[oi])
			} else {
				mulXORBase(gen[oi][i], data, out[oi])
			}
		}
	}
	return
}

////////////// internal functions //////////////

func (r rsAVX2) matrixMul(start, end int, in, out Matrix) {
	for i := 0; i < r.in; i++ {
		tmp := i * r.out
		for oi := 0; oi < r.out; oi++ {
			offset := (tmp + oi) * 32
			table := r.tables[offset : offset+32]
			if i == 0 {
				mulAVX2(table, in[i][start:end], out[oi][start:end])
			} else {
				mulXORAVX2(table, in[i][start:end], out[oi][start:end])
			}
		}
	}
}

//go:noescape
func mulAVX2(table, in, out []byte)

//go:noescape
func mulXORAVX2(table, in, out []byte)

func (r rsSSSE3) matrixMul(start, end int, in, out Matrix) {
	for i := 0; i < r.in; i++ {
		for oi := 0; oi < r.out; oi++ {
			offset := (i*len(out) + oi) * 32
			table := r.tables[offset : offset+32]
			if i == 0 {
				mulSSSE3(table, in[i][start:end], out[oi][start:end])
			} else {
				mulXORSSSE3(table, in[i][start:end], out[oi][start:end])
			}
		}
	}
}

//go:noescape
func mulSSSE3(table, in, out []byte)

//go:noescape
func mulXORSSSE3(table, in, out []byte)

func mulBase(c byte, in, out []byte) {
	mt := mulTable[c]
	for i := 0; i < len(in); i++ {
		out[i] = mt[in[i]]
	}
}

func mulXORBase(c byte, in, out []byte) {
	mt := mulTable[c]
	for i := 0; i < len(in); i++ {
		out[i] ^= mt[in[i]]
	}
}
