package reedsolomon

import "errors"

// set shard nil if lost
func (r *rsAVX2) Reconstruct(shards matrix) (err error) {
	return r.reconst(shards, false)
}

func (r *rsAVX2) ReconstructData(shards matrix) (err error) {
	return r.reconst(shards, true)
}

func (r *rsSSSE3) Reconstruct(shards matrix) (err error) {
	return r.reconst(shards, false)
}

func (r *rsSSSE3) ReconstructData(shards matrix) (err error) {
	return r.reconst(shards, true)
}

func (r *rsBase) Reconstruct(shards matrix) (err error) {
	return r.reconst(shards, false)
}

func (r *rsBase) ReconstructData(shards matrix) (err error) {
	return r.reconst(shards, true)
}

////////////// Internal Functions //////////////
func (r *rsAVX2) reconst(shards matrix, dataOnly bool) (err error) {
	stat, err := getReconstStat(r.data, r.parity, shards, dataOnly)
	if err != nil {
		if err == ErrNoNeedRepair {
			return nil
		}
		return
	}
	if len(stat.dataLost) > 0 {
		err := r.reconstData(shards, stat.size, stat.have, stat.dataLost)
		if err != nil {
			return err
		}
	}
	if len(stat.parityLost) > 0 && !dataOnly {
		r.reconstParity(shards, stat.size, stat.parityLost)
	}
	return nil
}

func (r *rsAVX2) reconstData(shards matrix, size int, have, dataLost []int) error {
	dpTmp, gen, err := genReconstMatrix(shards, r.data, r.parity, size, have, dataLost)
	if err != nil {
		return err
	}
	e := &rsAVX2{data: r.data, parity: len(dataLost), gen: gen}
	e.Encode(dpTmp)
	return nil
}

func (r *rsAVX2) reconstParity(shards matrix, size int, parityLost []int) {
	genTmp := genCauchyMatrix(r.data, r.parity)
	numPL := len(parityLost)
	gen := NewMatrix(numPL, r.data)
	for i, l := range parityLost {
		gen[i] = genTmp[l-r.data]
	}
	dpTmp := NewMatrix(r.data+numPL, size)
	for i := 0; i < r.data; i++ {
		dpTmp[i] = shards[i]
	}
	for i, l := range parityLost {
		shards[l] = make([]byte, size)
		dpTmp[i+r.data] = shards[l]
	}
	e := &rsAVX2{data: r.data, parity: numPL, gen: gen}
	e.Encode(dpTmp)
}

func (r *rsSSSE3) reconst(shards matrix, dataOnly bool) (err error) {
	stat, err := getReconstStat(r.data, r.parity, shards, dataOnly)
	if err != nil {
		if err == ErrNoNeedRepair {
			return nil
		}
		return
	}
	if len(stat.dataLost) > 0 {
		err := r.reconstData(shards, stat.size, stat.have, stat.dataLost)
		if err != nil {
			return err
		}
	}
	if len(stat.parityLost) > 0 && !dataOnly {
		r.reconstParity(shards, stat.size, stat.parityLost)
	}
	return nil
}

func (r *rsBase) reconst(shards matrix, dataOnly bool) (err error) {
	stat, err := getReconstStat(r.data, r.parity, shards, dataOnly)
	if err != nil {
		if err == ErrNoNeedRepair {
			return nil
		}
		return
	}
	if len(stat.dataLost) > 0 {
		err := r.reconstData(shards, stat.size, stat.have, stat.dataLost)
		if err != nil {
			return err
		}
	}
	if len(stat.parityLost) > 0 && !dataOnly {
		r.reconstParity(shards, stat.size, stat.parityLost)
	}
	return nil
}

func (r *rsSSSE3) reconstData(shards matrix, size int, have, dataLost []int) error {
	dpTmp, gen, err := genReconstMatrix(shards, r.data, r.parity, size, have, dataLost)
	if err != nil {
		return err
	}
	e := &rsSSSE3{data: r.data, parity: len(dataLost), gen: gen}
	e.Encode(dpTmp)
	return nil
}

func (r *rsSSSE3) reconstParity(shards matrix, size int, parityLost []int) {
	genTmp := genCauchyMatrix(r.data, r.parity)
	numPL := len(parityLost)
	gen := NewMatrix(numPL, r.data)
	for i, l := range parityLost {
		gen[i] = genTmp[l-r.data]
	}
	dpTmp := NewMatrix(r.data+numPL, size)
	for i := 0; i < r.data; i++ {
		dpTmp[i] = shards[i]
	}
	for i, l := range parityLost {
		shards[l] = make([]byte, size)
		dpTmp[i+r.data] = shards[l]
	}
	e := &rsSSSE3{data: r.data, parity: numPL, gen: gen}
	e.Encode(dpTmp)
}

func (r *rsBase) reconstData(shards matrix, size int, have, dataLost []int) error {
	dpTmp, gen, err := genReconstMatrix(shards, r.data, r.parity, size, have, dataLost)
	if err != nil {
		return err
	}
	e := &rsBase{data: r.data, parity: len(dataLost), gen: gen}
	e.Encode(dpTmp)
	return nil
}

func (r *rsBase) reconstParity(shards matrix, size int, parityLost []int) {
	genTmp := genCauchyMatrix(r.data, r.parity)
	numPL := len(parityLost)
	gen := NewMatrix(numPL, r.data)
	for i, l := range parityLost {
		gen[i] = genTmp[l-r.data]
	}
	dpTmp := NewMatrix(r.data+numPL, size)
	for i := 0; i < r.data; i++ {
		dpTmp[i] = shards[i]
	}
	for i, l := range parityLost {
		shards[l] = make([]byte, size)
		dpTmp[i+r.data] = shards[l]
	}
	e := &rsBase{data: r.data, parity: numPL, gen: gen}
	e.Encode(dpTmp)
}

func genReconstMatrix(shards matrix, data, parity, size int, have, dataLost []int) (dpTmp, gen matrix, err error) {
	e := GenEncodeMatrix(data, parity)
	decodeM := NewMatrix(data, data)
	numDL := len(dataLost)
	dpTmp = NewMatrix(data+numDL, size)
	for i := 0; i < data; i++ {
		h := have[i]
		dpTmp[i] = shards[h]
		decodeM[i] = e[h]
	}
	for i, l := range dataLost {
		shards[l] = make([]byte, size)
		dpTmp[i+data] = shards[l]
	}
	decodeM, err = decodeM.invert()
	if err != nil {
		return
	}
	gen = NewMatrix(numDL, data)
	for i, l := range dataLost {
		gen[i] = decodeM[l]
	}
	return
}

type reconstStat struct {
	have       []int
	dataLost   []int
	parityLost []int
	size       int
}

var ErrTooFewShards = errors.New("reedsolomon: too few shards for repair")
var ErrNoNeedRepair = errors.New("reedsolomon: no shard need repair")

func getReconstStat(in, out int, shards matrix, dataOnly bool) (stat reconstStat, err error) {
	err = CheckMatrixRows(in, out, shards)
	if err != nil {
		return
	}
	size := 0
	var have, dataLost, parityLost []int
	for i, s := range shards {
		if s != nil {
			sSize := len(s)
			if sSize == 0 {
				err = ErrShardEmpty
				return
			}
			if size == 0 {
				size = sSize
				have = append(have, i)
			} else {
				if size != sSize {
					err = ErrShardSizeNoMatch
					return
				} else {
					have = append(have, i)
				}
			}
		} else {
			if i < in {
				dataLost = append(dataLost, i)
			} else {
				parityLost = append(parityLost, i)
			}
		}
	}
	if len(have) < in {
		err = ErrTooFewShards
		return
	}
	if len(dataLost)+len(parityLost) == 0 {
		err = ErrNoNeedRepair
		return
	}
	if len(dataLost)+len(parityLost) > out {
		err = ErrTooFewShards
		return
	}
	if len(have)+len(parityLost) == in+out && dataOnly {
		err = ErrNoNeedRepair
		return
	}
	stat.have = have
	stat.dataLost = dataLost
	stat.parityLost = parityLost
	stat.size = size
	return
}
