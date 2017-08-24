package reedsolomon

import (
	"bytes"
	"errors"
	"testing"
)

func TestEncMatrixVand(t *testing.T) {
	a, err := genEncMatrixVand(10, 4)
	if err != nil {
		t.Fatal("gen EncMatrixVand fault")
	}
	e := []byte{}
	if !bytes.Equal(a, e) {
		t.Fatal("gen EncMatrixVand fault")
	}
}

func TestEncMatrixCauchy(t *testing.T) {
	a := genEncMatrixCauchy(10, 4)
	e := []byte{}
	if !bytes.Equal(a, e) {
		t.Fatal("gen EncMatrixCauchy fault")
	}
}

func TestMatrixInverse(t *testing.T) {
	testCases := []struct {
		matrixData  []byte
		cols        int
		expect      []byte
		ok          bool
		expectedErr error
	}{
		{
			[]byte{56, 23, 98, 3, 100, 200, 45, 201, 123},
			3,
			[]byte{175, 133, 33, 130, 13, 245, 112, 35, 126},
			true,
			nil,
		},
		{
			[]byte{0, 23, 98, 3, 100, 200, 45, 201, 123},
			3,
			[]byte{245, 128, 152, 188, 64, 135, 231, 81, 239},
			true,
			nil,
		},
		{
			[]byte{1, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 1, 7, 7, 6, 6, 1},
			5,
			[]byte{1, 0, 0, 0, 0, 0, 1, 0, 0, 0, 123, 123, 1, 122, 122, 0, 0, 1, 0, 0, 0, 0, 0, 1, 0},
			true,
			nil,
		},
		{
			[]byte{4, 2, 12, 6},
			2,
			nil,
			false,
			errors.New("rs.invert: matrix is singular"),
		},
	}

	for i, c := range testCases {
		m := matrix(c.matrixData)
		actual, actualErr := m.invert(c.cols)
		if actualErr != nil && c.ok {
			t.Errorf("Test %e: Expected to pass, but failed with: <ERROR> %s", i+1, actualErr.Error())
		}
		if actualErr == nil && !c.ok {
			t.Errorf("Test %e: Expected to fail with <ERROR> \"%s\", but passed instead.", i+1, c.expectedErr)
		}
		if actualErr != nil && !c.ok {
			if c.expectedErr != actualErr {
				t.Errorf("Test %e: Expected to fail with error \"%s\", but instead failed with error \"%s\" instead.", i+1, c.expectedErr, actualErr)
			}
		}
		if actualErr == nil && c.ok {
			if !bytes.Equal(c.expect, actual) {
				t.Errorf("Test %e: The mc matrix doesnt't match the expected result", i+1)
			}
		}
	}
}

func BenchmarkInvert5x5(b *testing.B) {
	benchmarkInvert(b, 5)
}

func BenchmarkInvert10x10(b *testing.B) {
	benchmarkInvert(b, 10)
}

func BenchmarkInvert20x20(b *testing.B) {
	benchmarkInvert(b, 20)
}

func benchmarkInvert(b *testing.B, size int) {
	m := genEncMatrixCauchy(size, 2)
	m.swap(0, size, size)
	m.swap(1, size+1, size)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := matrix(m[:size*size]).invert(size)
		if err != nil {
			b.Fatal(b)
		}
	}
}
