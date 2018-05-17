package reedsolomon

import (
	"bytes"
	"testing"
)

func TestVerifyEncMatrixCauchy(t *testing.T) {
	a := genEncMatrix(4, 4)
	e := []byte{1, 0, 0, 0,
		0, 1, 0, 0,
		0, 0, 1, 0,
		0, 0, 0, 1,
		71, 167, 122, 186,
		167, 71, 186, 122,
		122, 186, 71, 167,
		186, 122, 167, 71}
	if !bytes.Equal(a, e) {
		t.Fatal("mismatch")
	}
}

func TestMatrixInvert(t *testing.T) {
	testCases := []struct {
		matrixData  []byte
		n           int
		expect      []byte
		ok          bool
		expectedErr error
	}{
		{
			[]byte{56, 23, 98,
				3, 100, 200,
				45, 201, 123},
			3,
			[]byte{175, 133, 33,
				130, 13, 245,
				112, 35, 126},
			true,
			nil,
		},
		{
			[]byte{0, 23, 98,
				3, 100, 200,
				45, 201, 123},
			3,
			[]byte{245, 128, 152,
				188, 64, 135,
				231, 81, 239},
			true,
			nil,
		},
		{
			[]byte{1, 0, 0, 0, 0,
				0, 1, 0, 0, 0,
				0, 0, 0, 1, 0,
				0, 0, 0, 0, 1,
				7, 7, 6, 6, 1},
			5,
			[]byte{1, 0, 0, 0, 0,
				0, 1, 0, 0, 0,
				123, 123, 1, 122, 122,
				0, 0, 1, 0, 0,
				0, 0, 0, 1, 0},
			true,
			nil,
		},
		{
			[]byte{4, 2,
				12, 6},
			2,
			nil,
			false,
			ErrSingularMatrix,
		},
	}

	for i, c := range testCases {
		m := matrix(c.matrixData)
		actual, actualErr := m.invert(c.n)
		if actualErr != nil && c.ok {
			t.Errorf("case.%d, expected to pass, but failed with: <ERROR> %s", i+1, actualErr.Error())
		}
		if actualErr == nil && !c.ok {
			t.Errorf("case.%d, expected to fail with <ERROR> \"%s\", but passed", i+1, c.expectedErr)
		}
		if actualErr != nil && !c.ok {
			if c.expectedErr != actualErr {
				t.Errorf("case.%d, expected to fail with error \"%s\", but instead failed with error \"%s\"", i+1, c.expectedErr, actualErr)
			}
		}
		if actualErr == nil && c.ok {
			if !bytes.Equal(c.expect, actual) {
				t.Errorf("case.%d, mismatch", i+1)
			}
		}
	}
}

func benchmarkInvert(b *testing.B, n int) {
	m := genEncMatrix(n, 2)
	m.swap(0, n, n)
	m.swap(1, n+1, n)
	raw := matrix(m[:n*n])
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := raw.invert(n)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkInvert5x5(b *testing.B) {
	benchmarkInvert(b, 5)
}
