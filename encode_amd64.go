package reedsolomon

func init() {
	getEXT()
}

func getEXT() {
	if hasAVX2() {
		extension = avx2
	} else if hasSSSE3() {
		extension = ssse3
	} else {
		extension = none
	}
}
//go:noescape
func hasAVX2() bool

//go:noescape
func hasSSSE3() bool

//go:noescape
func

// compress high&low tables from a gen matrix for speeding up encoding
// it will cost about 1000ns, so it's not a good idea to use it for reconstruct data
// because the tables will be used only once time
// especially for reconstruct small data
func compressTables(gen matrix) []byte {
	rows := len(gen)
	cols := len(gen[0])
	tables := make([]byte, 32*rows*cols)
	for i := 0; i < cols; i++ {
		for j := 0; j < rows; j++ {
			c := gen[j][i]
			offset := (i*rows + j) * 32
			l := mulTableLow[c][:]
			// TODO
			copy(tables[offset:offset+16], l)
			h := mulTableHigh[c][:]
			copy(tables[offset+16:offset+32], h)
		}
	}
	return tables
}

func (r *encAVX2) Encode(shards matrix) (err error) {
	err = CheckEncodeShards(r.data, r.parity, shards)
	if err != nil {
		return
	}
	in := shards[:r.data]
	out := shards[r.data:]
	size := len(in[0])
	start, end := 0, 0
	do := UnitSize
	for start < size {
		end = start + do
		if end <= size {
			r.matrixMul(start, end, in, out)
			start = end
		} else {
			r.matrixMulRemain(start, size, in, out)
			start = size
		}
	}
	return
}

func (r *encSSSE3) Encode(shards matrix) (err error) {
	err = CheckEncodeShards(r.data, r.parity, shards)
	if err != nil {
		return
	}
	in := shards[:r.data]
	out := shards[r.data:]
	size := len(in[0])
	start, end := 0, 0
	do := UnitSize
	for start < size {
		end = start + do
		if end <= size {
			r.matrixMul(start, end, in, out)
			start = end
		} else {
			r.matrixMulRemain(start, size, in, out)
			start = size
		}
	}
	return
}

// avx2
func (r *encAVX2) matrixMul(start, end int, in, out matrix) {
	for i := 0; i < r.data; i++ {
		for oi := 0; oi < r.parity; oi++ {
			c := r.gen[oi][i]
			low := mulTableLow[c][:]
			high := mulTableHigh[c][:]
			if i == 0 {
				mulAVX2(low, high, in[i][start:end], out[oi][start:end])
			} else {
				mulXORAVX2(low, high, in[i][start:end], out[oi][start:end])
			}
		}
	}
}

func (r *encAVX2) matrixMulRemain(start, end int, in, out matrix) {
	r.matrixMul(start, end, in, out)
	done := (end >> 5) << 5
	remain := end - done
	if remain > 0 {
		g := r.gen
		start = start + done
		for i := 0; i < r.data; i++ {
			for oi := 0; oi < r.parity; oi++ {
				if i == 0 {
					mulBase(g[oi][i], in[i][start:end], out[oi][start:end])
				} else {
					mulXORBase(g[oi][i], in[i][start:end], out[oi][start:end])
				}
			}
		}
	}
}

//go:noescape
func mulAVX2(low, high, in, out []byte)

//go:noescape
func mulXORAVX2(low, high, in, out []byte)

// ssse3
func (r *encSSSE3) matrixMul(start, end int, in, out matrix) {
	for i := 0; i < r.data; i++ {
		for oi := 0; oi < r.parity; oi++ {
			c := r.gen[oi][i]
			low := mulTableLow[c][:]
			high := mulTableHigh[c][:]
			if i == 0 {
				mulSSSE3(low, high, in[i][start:end], out[oi][start:end])
			} else {
				mulXORSSSE3(low, high, in[i][start:end], out[oi][start:end])
			}
		}
	}
}

func (r *encSSSE3) matrixMulRemain(start, end int, in, out matrix) {
	r.matrixMul(start, end, in, out)
	done := (end >> 4) << 4
	remain := end - done
	if remain > 0 {
		gen := r.gen
		start = start + done
		for i := 0; i < r.data; i++ {
			for oi := 0; oi < r.parity; oi++ {
				if i == 0 {
					mulBase(gen[oi][i], in[i][start:end], out[oi][start:end])
				} else {
					mulXORBase(gen[oi][i], in[i][start:end], out[oi][start:end])
				}
			}
		}
	}
}

//go:noescape
func mulSSSE3(low, high, in, out []byte)

//go:noescape
func mulXORSSSE3(low, high, in, out []byte)

func getINS() int {
	if hasAVX2() {
		return avx2
	} else if hasSSSE3() {
		return ssse3
	} else {
		return none
	}
}