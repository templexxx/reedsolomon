package reedsolomon

import (
	"math/rand"
	"sort"
	"testing"
	"time"
)

func TestDedup(t *testing.T) {

	rand.Seed(time.Now().UnixNano())

	round := 1024
	minN := 4
	maxN := 4096
	s := make([]int, maxN)

	for i := 0; i < round; i++ {
		n := rand.Intn(maxN + 1)
		if n < minN {
			n = minN
		}
		for j := 0; j < n/minN; j++ {
			copy(s[j*4:j*4+4], []int{0, 1, 2, 3})
		}
		s2 := s[:n]
		s2 = dedup(s2)
		if len(s2) != minN {
			t.Fatal("failed to dedup: wrong length")
		}
		for j := range s2 {
			if s2[j] != j {
				t.Fatal("failed to dedup: wrong result")
			}
		}
	}
}

// generates survived & needReconst indexes.
func genIdxForReconst(d, p, survivedN, needReconstN int) ([]int, []int) {
	if survivedN < d {
		survivedN = d
	}
	if needReconstN > p {
		needReconstN = p
	}
	if survivedN+needReconstN > d+p {
		survivedN = d
	}

	idxR := genIdxNeedReconst(d, p, needReconstN)

	idxS := make([]int, 0, survivedN)

	fullIdx := make([]int, d+p)
	for i := range fullIdx {
		fullIdx[i] = i
	}
	rand.Shuffle(d+p, func(i, j int) { // More chance to get balanced survived indexes
		fullIdx[i], fullIdx[j] = fullIdx[j], fullIdx[i]
	})

	for i := 0; i < d+p; i++ {
		if len(idxS) == survivedN {
			break
		}
		if !isIn(fullIdx[i], idxR) {
			idxS = append(idxS, fullIdx[i])
		}
	}

	sort.Ints(idxS)
	sort.Ints(idxR)

	return idxS, idxR
}

func TestGenIdxForReconst(t *testing.T) {

	d, p := 10, 4

	ret := make([]int, 0, d+p)

	for i := 0; i < d+p; i++ {
		for j := 0; j < d+p; j++ {
			is, ir := genIdxForReconst(d, p, 10, 4)
			checkGenIdx(t, d, p, is, ir, ret)
			ret = ret[:0]
		}
	}
}

func checkGenIdx(t *testing.T, d, p int, is, ir, all []int) {

	for _, v := range is {
		if v < 0 || v >= d+p {
			t.Fatal(ErrIllegalVectIndex)
		}
		all = append(all, v)
	}
	for _, v := range ir {
		if v < 0 || v >= d+p {
			t.Fatal(ErrIllegalVectIndex)
		}
		all = append(all, v)
	}
	if len(is) < d {
		t.Fatal("too few survived")
	}
	da := dedup(all)
	if len(da) != len(all) {
		t.Fatal("survived & needReconst conflicting")
	}
	if !sort.IsSorted(sort.IntSlice(is)) || !sort.IsSorted(sort.IntSlice(ir)) {
		t.Fatal("idx unsorted")
	}
}

func genIdxNeedReconst(d, p, needReconstN int) []int {
	rand.Seed(time.Now().UnixNano())

	s := make([]int, needReconstN)
	n := 0
	for {
		if n == needReconstN {
			break
		}
		v := rand.Intn(d + p)
		if !isIn(v, s) {
			s[n] = v
			n++
		}
	}
	return s
}

func isIn(e int, s []int) bool {
	for _, v := range s {
		if e == v {
			return true
		}
	}
	return false
}

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
