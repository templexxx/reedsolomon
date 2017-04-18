package reedsolomon

import "errors"

const unitSize int = 1024

// Encode : cauchy_matrix * data_matrix(input) -> parity_matrix(output)
// dp : data_matrix(upper) parity_matrix(lower, empty now)
func (r *RS) Encode(dp Matrix) error {
	if len(dp) != r.Shards {
		return ErrTooFewShards
	}
	size, err := checkShardSize(dp)
	if err != nil {
		return err
	}
	inMap := make(map[int]int)
	outMap := make(map[int]int)
	for i := 0; i < r.Data; i++ {
		inMap[i] = i
	}
	for i := r.Data; i < r.Shards; i++ {
		outMap[i-r.Data] = i
	}
	encodeSSSE3(r.Gen, dp, r.Data, r.Parity, size, inMap, outMap)
	return nil
}

func encodeSSSE3(gen, dp Matrix, numIn, numOut, size int, inMap, outMap map[int]int) {
	start := 0
	do := unitSize
	for start < size {
		if start+do <= size {
			encodeWorkerS(gen, dp, start, do, numIn, numOut, inMap, outMap)
			start = start + do
		} else {
			encodeRemainS(start, size, gen, dp, numIn, numOut, inMap, outMap)
			start = size
		}
	}
}

func encodeWorkerS(gen, dp Matrix, start, do, numIn, numOut int, inMap, outMap map[int]int) {
	end := start + do
	for i := 0; i < numIn; i++ {
		j := inMap[i]
		in := dp[j]
		for oi := 0; oi < numOut; oi++ {
			k := outMap[oi]
			c := gen[oi][i]
			if i == 0 { // it means don't need to copy Parity Data for xor
				gfMulSSSE3(mulTableLow[c][:], mulTableHigh[c][:], in[start:end], dp[k][start:end])
			} else {
				gfMulXorSSSE3(mulTableLow[c][:], mulTableHigh[c][:], in[start:end], dp[k][start:end])
			}
		}
	}
}

func encodeRemainS(start, size int, gen, dp Matrix, numIn, numOut int, inMap, outMap map[int]int) {
	do := size - start
	for i := 0; i < numIn; i++ {
		j := inMap[i]
		in := dp[j]
		for oi := 0; oi < numOut; oi++ {
			k := outMap[oi]
			c := gen[oi][i]
			if i == 0 {
				gfMulRemainS(c, in[start:size], dp[k][start:size], do)
			} else {
				gfMulRemainXorS(c, in[start:size], dp[k][start:size], do)
			}
		}
	}
}

var ErrShardSize = errors.New("reedsolomon: Shards size equal 0 or not match")

func checkShardSize(m Matrix) (int, error) {
	size := len(m[0])
	if size == 0 {
		return size, ErrShardSize
	}
	for _, v := range m {
		if len(v) != size {
			return 0, ErrShardSize
		}
	}
	return size, nil
}
