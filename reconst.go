package reedsolomon

import "sort"

// dp : data+parity shards, all shards size must be equal
// lost : row number in dp
func (r *rs) Reconst(dp matrix, lost []int, repairParity bool) error {
	if len(dp) != r.shards {
		return ErrTooFewShards
	}
	size, err := checkShardSize(dp)
	if err != nil {
		return err
	}
	if len(lost) == 0 {
		return nil
	}
	if len(lost) > r.parity {
		return ErrTooFewShards
	}
	dataLost, parityLost := splitLost(lost, r.data)
	sort.Ints(dataLost)
	sort.Ints(parityLost)
	if len(dataLost) > 0 {
		err = reconstData(r.m, dp, dataLost, parityLost, r.data, size, r.ins)
		if err != nil {
			return err
		}
	}
	if len(parityLost) > 0 && repairParity {
		reconstParity(r.m, dp, parityLost, r.data, size, r.ins)
	}
	return nil
}

func reconstData(encodeMatrix, dp matrix, dataLost, parityLost []int, numData, size, ins int) error {
	decodeMatrix := NewMatrix(numData, numData)
	survivedMap := make(map[int]int)
	numShards := len(encodeMatrix)
	// fill with survived data
	for i := 0; i < numData; i++ {
		if survived(i, dataLost) {
			decodeMatrix[i] = encodeMatrix[i]
			survivedMap[i] = i
		}
	}
	// "borrow" from survived parity
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
	// fill generator matrix with lost rows of decode matrix
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

func reconstParity(encodeMatrix, dp matrix, parityLost []int, numData, size, ins int) {
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
