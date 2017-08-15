package reedsolomon

import "errors"

// size of sub-vector
const UnitSize int = 16 * 1024


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

func mulBase(c byte, in, out []byte) {
	mt := mulTbl[c]
	for i := 0; i < len(in); i++ {
		out[i] = mt[in[i]]
	}
}

func mulXORBase(c byte, in, out []byte) {
	mt := mulTbl[c]
	for i := 0; i < len(in); i++ {
		out[i] ^= mt[in[i]]
	}
}
