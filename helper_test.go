package reedsolomon

import (
	"math/rand"
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
