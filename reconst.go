package reedsolomon

import "sort"

// dp : Data+Parity Shards, all Shards size must be equal
// lost : row number in dp
func (r *RS) Reconst(dp Matrix, lost []int, repairParity bool) error {
	if len(dp) != r.Shards {
		return ErrTooFewShards
	}
	size, err := checkShardSize(dp)
	if err != nil {
		return err
	}
	if len(lost) == 0 {
		return nil
	}
	if len(lost) > r.Parity {
		return ErrTooFewShards
	}
	dataLost, parityLost := splitLost(lost, r.Data)
	sort.Ints(dataLost)
	sort.Ints(parityLost)
	if len(dataLost) > 0 {
		err = reconstData(r.M, dp, dataLost, parityLost, r.Data, size, r.INS)
		if err != nil {
			return err
		}
	}
	if len(parityLost) > 0 && repairParity {
		reconstParity(r.M, dp, parityLost, r.Data, size, r.INS)
	}
	return nil
}

func reconstData(encodeMatrix, dp Matrix, dataLost, parityLost []int, numData, size, ins int) error {
	decodeMatrix := NewMatrix(numData, numData)
	survivedMap := make(map[int]int)
	numShards := len(encodeMatrix)
	// fill with survived Data
	for i := 0; i < numData; i++ {
		if survived(i, dataLost) {
			decodeMatrix[i] = encodeMatrix[i]
			survivedMap[i] = i
		}
	}
	// "borrow" from survived Parity
	k := numData
	for _, dl := range dataLost {
		for j := k; j < numShards; j++ {
			k++
			if survived(j, parityLost) {
				decodeMatrix[dl] = encodeMatrix[j]
				survivedMap[dl] = j
				break
			}
		}
	}
	var err error
	decodeMatrix, err = decodeMatrix.invert()
	if err != nil {
		return err
	}
	// fill generator matrix with lost rows of decode Matrix
	numDL := len(dataLost)
	gen := NewMatrix(numDL, numData)
	outputMap := make(map[int]int)
	for i, l := range dataLost {
		gen[i] = decodeMatrix[l]
		outputMap[i] = l
	}
	encodeSSSE3(gen, dp, numData, numDL, size, survivedMap, outputMap)
	return nil
}

func reconstParity(encodeMatrix, dp Matrix, parityLost []int, numData, size, ins int) {
	gen := NewMatrix(len(parityLost), numData)
	outputMap := make(map[int]int)
	for i := range gen {
		l := parityLost[i]
		gen[i] = encodeMatrix[l]
		outputMap[i] = l
	}
	inMap := make(map[int]int)
	for i := 0; i < numData; i++ {
		inMap[i] = i
	}
	encodeSSSE3(gen, dp, numData, len(parityLost), size, inMap, outputMap)
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

func survived(i int, lost []int) bool {
	for _, l := range lost {
		if i == l {
			return false
		}
	}
	return true
}
