package reedsolomon

import "errors"

// size of sub-vector
const UnitSize int = 16 * 1024

func (r *rsAVX2) Encode(shards matrix) (err error) {
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

func (r *rsSSSE3) Encode(shards matrix) (err error) {
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

func (r *rsBase) Encode(shards matrix) (err error) {
	err = CheckEncodeShards(r.data, r.parity, shards)
	if err != nil {
		return
	}
	in := shards[:r.data]
	out := shards[r.data:]
	gen := r.gen
	for i := 0; i < r.data; i++ {
		data := in[i]
		for oi := 0; oi < r.parity; oi++ {
			if i == 0 {
				mulBase(gen[oi][i], data, out[oi])
			} else {
				mulXORBase(gen[oi][i], data, out[oi])
			}
		}
	}
	return
}

// Check Encode Args
func CheckEncodeShards(in, out int, shards matrix) error {
	err := CheckMatrixRows(in, out, shards)
	if err != nil {
		return err
	}
	err = CheckShardSize(shards)
	if err != nil {
		return err
	}
	return nil
}

var ErrNumShards = errors.New("reedsolomon: num of shards not match")

func CheckMatrixRows(in, out int, shards matrix) error {
	if in+out != len(shards) {
		return ErrNumShards
	}
	return nil
}

var ErrShardEmpty = errors.New("reedsolomon: shards size equal 0")
var ErrShardSizeNoMatch = errors.New("reedsolomon: shards size not match")

func CheckShardSize(shards matrix) error {
	size := len(shards[0])
	if size == 0 {
		return ErrShardEmpty
	}
	for i := 1; i < len(shards); i++ {
		if len(shards[i]) != size {
			return ErrShardSizeNoMatch
		}
	}
	return nil
}

////////////// Internal Functions //////////////
// matrix multiply
// avx2
func (r *rsAVX2) matrixMul(start, end int, in, out matrix) {
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

func (r *rsAVX2) matrixMulRemain(start, end int, in, out matrix) {
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
func (r *rsSSSE3) matrixMul(start, end int, in, out matrix) {
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

func (r *rsSSSE3) matrixMulRemain(start, end int, in, out matrix) {
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
