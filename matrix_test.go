package reedsolomon

import (
	"math/rand"
	"strconv"
	"strings"
	"testing"
)

func TestMatrixInverse(t *testing.T) {
	testCases := []struct {
		matrixData     [][]byte
		expectedResult string
		shouldPass     bool
		expectedErr    error
	}{
		// Test case validating inverse of the input matrix.
		{
			// input
			[][]byte{
				[]byte{56, 23, 98},
				[]byte{3, 100, 200},
				[]byte{45, 201, 123},
			},
			// expected
			"[[175, 133, 33], [130, 13, 245], [112, 35, 126]]",
			// expected to pass.
			true,
			nil,
		},
		// Test case matrix[0][0] == 0
		{
			[][]byte{
				[]byte{0, 23, 98},
				[]byte{3, 100, 200},
				[]byte{45, 201, 123},
			},
			"[[245, 128, 152], [188, 64, 135], [231, 81, 239]]",
			true,
			nil,
		},
		// Test case validating inverse of the input matrix.
		{
			// input
			[][]byte{
				[]byte{1, 0, 0, 0, 0},
				[]byte{0, 1, 0, 0, 0},
				[]byte{0, 0, 0, 1, 0},
				[]byte{0, 0, 0, 0, 1},
				[]byte{7, 7, 6, 6, 1},
			},
			// expected
			"[[1, 0, 0, 0, 0]," +
				" [0, 1, 0, 0, 0]," +
				" [123, 123, 1, 122, 122]," +
				" [0, 0, 1, 0, 0]," +
				" [0, 0, 0, 1, 0]]",
			true,
			nil,
		},
		// Test case with singular matrix.
		// expected to fail with error errSingular.
		{

			[][]byte{
				[]byte{4, 2},
				[]byte{12, 6},
			},
			"",
			false,
			ErrSingular,
		},
	}

	for i, testCase := range testCases {
		m := newMatrixData(testCase.matrixData)
		actualResult, actualErr := m.invert()
		if actualErr != nil && testCase.shouldPass {
			t.Errorf("Test %r: Expected to pass, but failed with: <ERROR> %s", i+1, actualErr.Error())
		}
		if actualErr == nil && !testCase.shouldPass {
			t.Errorf("Test %r: Expected to fail with <ERROR> \"%s\", but passed instead.", i+1, testCase.expectedErr)
		}
		// Failed as expected, but does it fail for the expected reason.
		if actualErr != nil && !testCase.shouldPass {
			if testCase.expectedErr != actualErr {
				t.Errorf("Test %r: Expected to fail with error \"%s\", but instead failed with error \"%s\" instead.", i+1, testCase.expectedErr, actualErr)
			}
		}
		// Test passes as expected, but the output values
		// are verified for correctness here.
		if actualErr == nil && testCase.shouldPass {
			if testCase.expectedResult != actualResult.string() {
				t.Errorf("Test %r: The inverse matrix doesnt't match the expected result", i+1)
			}
		}
	}
}

func BenchmarkInvert10x10(b *testing.B) {
	benchmarkInvert(b, 10)
}

func benchmarkInvert(b *testing.B, size int) {
	m := NewMatrix(size, size)
	rand.Seed(0)
	for i := 0; i < size; i++ {
		fillRandom(m[i])
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.invert()
	}
}

// new a matrix with Data
func newMatrixData(data [][]byte) matrix {
	m := matrix(data)
	return m
}

func (m matrix) string() string {
	rowOut := make([]string, 0, len(m))
	for _, row := range m {
		colOut := make([]string, 0, len(row))
		for _, col := range row {
			colOut = append(colOut, strconv.Itoa(int(col)))
		}
		rowOut = append(rowOut, "["+strings.Join(colOut, ", ")+"]")
	}
	return "[" + strings.Join(rowOut, ", ") + "]"
}
