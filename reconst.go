package reedsolomon

// Reconst steps:
// 1. read survived data ( In practice, recommend read more than num of data shards for avoiding read err)
// 2. reconst data (if lost)
// 3. reconst parity (if lost)
func (r rsAVX2) Reconst(dp Matrix, have, lost []int) (err error) {
	size := len(dp[0])
	dataLost, parityLost := splitLost(lost, r.in)
	if len(dataLost) > 0 {
		err := r.reconstData(dp, size, have, dataLost)
		if err != nil {
			return err
		}
	}
	if len(parityLost) > 0 {
		r.reconstParity(dp, size, parityLost)
	}
	return nil
}

func (r rsSSSE3) Reconst(dp Matrix, have, lost []int) (err error) {
	size := len(dp[0])
	dataLost, parityLost := splitLost(lost, r.in)
	if len(dataLost) > 0 {
		err := r.reconstData(dp, size, have, dataLost)
		if err != nil {
			return err
		}
	}
	if len(parityLost) > 0 {
		r.reconstParity(dp, size, parityLost)
	}
	return nil
}

func (r rsBase) Reconst(dp Matrix, have, lost []int) (err error) {
	size := len(dp[0])
	dataLost, parityLost := splitLost(lost, r.in)
	if len(dataLost) > 0 {
		err := r.reconstData(dp, size, have, dataLost)
		if err != nil {
			return err
		}
	}
	if len(parityLost) > 0 {
		r.reconstParity(dp, size, parityLost)
	}
	return nil
}

func (r rsAVX2) reconstData(dp Matrix, size int, have, dataLost []int) error {
	dpTmp, gen, err := genReconstMatrix(dp, r.in, r.out, size, have, dataLost)
	if err != nil {
		return err
	}
	t := genTables(gen)
	e := rsAVX2{tables: t, in: r.in, out: len(dataLost)}
	e.Encode(dpTmp[:r.in], dpTmp[r.in:])
	for i, l := range dataLost {
		dp[l] = dpTmp[r.in+i]
	}
	return nil
}

func (r rsAVX2) reconstParity(dp Matrix, size int, parityLost []int) {
	genTmp := genCauchyMatrix(r.in, r.out)
	numPL := len(parityLost)
	gen := NewMatrix(numPL, r.in)
	for i, l := range parityLost {
		gen[i] = genTmp[l-r.in]
	}
	out := NewMatrix(numPL, size)
	t := genTables(gen)
	e := rsAVX2{tables: t, in: r.in, out: numPL}
	e.Encode(dp[:r.in], out)
	for i, l := range parityLost {
		dp[l] = out[i]
	}
}

func (r rsSSSE3) reconstData(dp Matrix, size int, have, dataLost []int) error {
	dpTmp, gen, err := genReconstMatrix(dp, r.in, r.out, size, have, dataLost)
	if err != nil {
		return err
	}
	t := genTables(gen)
	e := rsSSSE3{tables: t, in: r.in, out: len(dataLost)}
	e.Encode(dpTmp[:r.in], dpTmp[r.in:])
	for i, l := range dataLost {
		dp[l] = dpTmp[r.in+i]
	}
	return nil
}

func (r rsSSSE3) reconstParity(dp Matrix, size int, parityLost []int) {
	genTmp := genCauchyMatrix(r.in, r.out)
	numPL := len(parityLost)
	gen := NewMatrix(numPL, r.in)
	for i, l := range parityLost {
		gen[i] = genTmp[l-r.in]
	}
	out := NewMatrix(numPL, size)
	t := genTables(gen)
	e := rsSSSE3{tables: t, in: r.in, out: numPL}
	e.Encode(dp[:r.in], out)
	for i, l := range parityLost {
		dp[l] = out[i]
	}
}

func (r rsBase) reconstData(dp Matrix, size int, have, dataLost []int) error {
	dpTmp, gen, err := genReconstMatrix(dp, r.in, r.out, size, have, dataLost)
	if err != nil {
		return err
	}
	e := rsBase{gen: gen, in: r.in, out: len(dataLost)}
	e.Encode(dpTmp[:r.in], dpTmp[r.in:])
	for i, l := range dataLost {
		dp[l] = dpTmp[r.in+i]
	}
	return nil
}

func (r rsBase) reconstParity(dp Matrix, size int, parityLost []int) {
	genTmp := genCauchyMatrix(r.in, r.out)
	numPL := len(parityLost)
	gen := NewMatrix(numPL, r.in)
	for i, l := range parityLost {
		gen[i] = genTmp[l-r.in]
	}
	out := NewMatrix(numPL, size)
	e := rsBase{gen: gen, in: r.in, out: numPL}
	e.Encode(dp[:r.in], out)
	for i, l := range parityLost {
		dp[l] = out[i]
	}
}

func genReconstMatrix(dp Matrix, data, parity, size int, have, dataLost []int) (dpTmp, gen Matrix, err error) {
	e := GenEncodeMatrix(data, parity)
	decodeM := NewMatrix(data, data)
	numDL := len(dataLost)
	dpTmp = NewMatrix(data+numDL, size)
	for i, h := range have {
		copy(decodeM[i], e[h])
		dpTmp[i] = dp[h]
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

func splitLost(lost []int, d int) ([]int, []int) {
	var dataLost []int
	var parityLost []int
	for _, l := range lost {
		if l < d {
			dataLost = append(dataLost, l)
		} else {
			parityLost = append(parityLost, l)
		}
	}
	return dataLost, parityLost
}
