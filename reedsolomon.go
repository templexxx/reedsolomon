package reedsolomon

import "errors"

type EncodeReconster interface {
	Encode(in, out Matrix) error
	Reconst(dp Matrix, have, lost []int) error
}

type rsAVX2 rsSIMD
type rsSSSE3 rsSIMD

type rsBase struct {
	gen Matrix
	in  int
	out int
}

type rsSIMD struct {
	tables []byte
	in     int
	out    int
}

const (
	Base = 1 << iota
	AVX2
	SSSE3
)

var ErrNonSupportINS = errors.New("reedsolomon: nonsupport SIMD Extensions, need avx2 or ssse3")

func New(data, parity int) (rs EncodeReconster, err error) {
	err = CheckShardsNum(data, parity)
	if err != nil {
		return
	}
	ins := GetINS()
	g := genCauchyMatrix(data, parity)
	if ins == Base {
		return rsBase{gen: g, in: data, out: parity}, nil
	}
	t := genTables(g)
	switch ins {
	case AVX2:
		return rsAVX2{tables: t, in: data, out: parity}, nil
	case SSSE3:
		return rsSSSE3{tables: t, in: data, out: parity}, nil
	default:
		err = ErrNonSupportINS
		return
	}
}

// check should be done before transport data to the rs engine server
var ErrTooFewShards = errors.New("reedsolomon: too few shards given for encoding")
var ErrTooManyShards = errors.New("reedsolomon: too many shards given for encoding")

func CheckShardsNum(d, p int) error {
	if (d <= 0) || (p <= 0) {
		return ErrTooFewShards
	}
	if d+p >= 255 {
		return ErrTooManyShards
	}
	return nil
}

var ErrShardEmpty = errors.New("reedsolomon: shards size equal 0")
var ErrShardNoMatch = errors.New("reedsolomon: shards size not match")
var ErrShardSize = errors.New("reedsolomon: shard size must be integral multiple of 256B(AVX2)/128B(SSSE3)")

const (
	LoopSizeAVX2  int = 256 // size of per avx2 encode loop
	LoopSizeSSSE3 int = 16  // size of per ssse3 encode loop
)

var ErrNoNeedRepair = errors.New("reedsolomon: no shard need repair")

func CheckShardsReconst(d, p, size, ins int, dp Matrix, have, lost []int) error {
	err := CheckShards(d, p, size, ins, dp[:d], dp[d:])
	if err != nil {
		return err
	}
	if len(lost) == 0 {
		return ErrNoNeedRepair
	}
	if len(lost) > p {
		return ErrTooFewShards
	}
	if len(have) != d {
		return ErrTooFewShards
	}
	return nil
}

func CheckShards(d, p, size, ins int, in, out Matrix) error {
	err := CheckMatrixRows(d, p, in, out)
	if err != nil {
		return err
	}
	err = CheckShardSize(size, ins, in, out)
	if err != nil {
		return err
	}
	return nil
}

func CheckShardSize(size, ins int, in, out Matrix) error {
	if size == 0 {
		return ErrShardEmpty
	}
	for _, v := range in {
		if len(v) != size {
			return ErrShardNoMatch
		}
	}
	for _, v := range out {
		if len(v) != size {
			return ErrShardNoMatch
		}
	}
	loopSize := 1
	switch ins {
	case AVX2:
		loopSize = LoopSizeAVX2
	case SSSE3:
		loopSize = LoopSizeSSSE3
	}
	if (size/loopSize)*(loopSize) != size {
		return ErrShardSize
	}
	return nil
}

var ErrDataShards = errors.New("reedsolomon: num of data shards not match")
var ErrParityShards = errors.New("reedsolomon: num of parity shards not match")

func CheckMatrixRows(d, p int, in, out Matrix) error {
	if d != len(in) {
		return ErrDataShards
	}
	if p != len(out) {
		return ErrParityShards
	}
	return nil
}

func GetINS() int {
	if hasAVX2() {
		return AVX2
	} else if hasSSSE3() {
		return SSSE3
	} else {
		return Base
	}
}

func genTables(gen Matrix) []byte {
	rows := len(gen)
	cols := len(gen[0])
	tables := make([]byte, 32*rows*cols)
	for i := 0; i < cols; i++ {
		for j := 0; j < rows; j++ {
			c := gen[j][i]
			offset := (i*rows + j) * 32
			l := mulTableLow[c][:]
			copy(tables[offset:offset+16], l)
			h := mulTableHigh[c][:]
			copy(tables[offset+16:offset+32], h)
		}
	}
	return tables
}

//go:noescape
func hasAVX2() bool

//go:noescape
func hasSSSE3() bool
