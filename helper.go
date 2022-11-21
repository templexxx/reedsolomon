package reedsolomon

import "sort"

// dedup removes duplicates from a given slice
func dedup(s []int) []int {

	sort.Ints(s)

	cnt := len(s)
	cntDup := 0
	for i := 1; i < cnt; i++ {
		if s[i] == s[i-1] {
			cntDup++
		} else {
			s[i-cntDup] = s[i]
		}
	}

	return s[:cnt-cntDup]
}
