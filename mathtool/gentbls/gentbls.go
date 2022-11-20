// Copyright (c) 2017 Temple3x (temple3x@gmail.com)
//
// Use of this source code is governed by the MIT License
// that can be found in the LICENSE file.

// This tool generates primitive polynomial,
// and it's log, exponent, multiply and inverse tables etc.
package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
)

const deg = 8

type polynomial [deg + 1]byte

func main() {
	f, err := os.OpenFile("gf_tables", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		log.Fatalln(err)
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	ps := genPrimitivePolynomial()
	title := strconv.FormatInt(int64(deg), 10) + " degree primitive polynomial：\n"
	var pss string
	for i, p := range ps {
		pf := formatPolynomial(p)
		pf = strconv.FormatInt(int64(i+1), 10) + ". " + pf + ";\n"
		pss = pss + pf
	}
	body := fmt.Sprintf(title+"%v", pss)
	w.WriteString(body)

	// Set primitive polynomial here to generator tables.
	// Default: x^8+x^4+x^3+x^2+1
	var primitivePolynomial polynomial
	primitivePolynomial[0] = 1
	primitivePolynomial[2] = 1
	primitivePolynomial[3] = 1
	primitivePolynomial[4] = 1
	primitivePolynomial[8] = 1

	lenExpTable := (1 << deg) - 1
	expTable := genExpTable(primitivePolynomial, lenExpTable)
	body = fmt.Sprintf("expTbl: %#v\n", expTable)
	w.WriteString(body)

	logTable := genLogTable(expTable)
	body = fmt.Sprintf("logTbl: %#v\n", logTable)
	w.WriteString(body)

	mulTable := genMulTable(expTable, logTable)
	body = fmt.Sprintf("mulTbl: %#v\n", mulTable)
	w.WriteString(body)

	lowTable, highTable := genMulTableHalf(mulTable)
	body = fmt.Sprintf("lowTbl: %#v\n", lowTable)
	w.WriteString(body)
	body = fmt.Sprintf("highTbl: %#v\n", highTable)
	w.WriteString(body)

	var lowHighTbl [256 * 32]byte
	for i := 0; i < 256; i++ {
		copy(lowHighTbl[i*32:i*32+16], lowTable[i])
		copy(lowHighTbl[i*32+16:i*32+32], highTable[i])
	}

	body = fmt.Sprintf("lowHighTbl: %#v\n", lowHighTbl)
	w.WriteString(body)

	inverseTable := genInverseTable(mulTable)
	body = fmt.Sprintf("inverseTbl: %#v\n", inverseTable)
	w.WriteString(body)
	w.Flush()
}

// Generate primitive Polynomial.
func genPrimitivePolynomial() []polynomial {
	// Drop Polynomial x，so the constant term must be 1,
	// so there are 2^(deg-1) Polynomials.
	cnt := 1 << (deg - 1)
	var polynomials []polynomial
	var p polynomial
	p[0] = 1
	p[deg] = 1
	// Generate all Polynomials.
	for i := 0; i < cnt; i++ {
		p = genPolynomial(p, 1)
		polynomials = append(polynomials, p)
	}
	// Drop Polynomial x+1, so the cnt of Polynomials is odd.
	var psRaw []polynomial
	for _, p := range polynomials {
		var n int
		for _, v := range p {
			if v == 1 {
				n++
			}
		}
		if n&1 != 0 {
			psRaw = append(psRaw, p)
		}
	}
	// Order of primitive element == 2^deg -1
	var ps []polynomial
	for _, p := range psRaw {
		lenTable := (1 << deg) - 1
		table := genExpTable(p, lenTable)
		var numOf1 int
		for _, v := range table {
			// Cnt 1 in ExpTable.
			if int(v) == 1 {
				numOf1++
			}
		}
		if numOf1 == 1 {
			ps = append(ps, p)
		}
	}
	return ps
}

func genPolynomial(p polynomial, i int) polynomial {
	if p[i] == 0 {
		p[i] = 1
	} else {
		p[i] = 0
		i++
		if i == deg {
			return p
		}
		p = genPolynomial(p, i)
	}
	return p
}

func genExpTable(primitivePolynomial polynomial, exp int) []byte {
	table := make([]byte, exp)
	var rawPolynomial polynomial
	rawPolynomial[1] = 1
	table[0] = byte(1)
	table[1] = byte(2)
	for i := 2; i < exp; i++ {
		rawPolynomial = expGrowPolynomial(rawPolynomial, primitivePolynomial)
		table[i] = getValueOfPolynomial(rawPolynomial)
	}
	return table
}

func expGrowPolynomial(raw, primitivePolynomial polynomial) polynomial {
	var newP polynomial
	for i, v := range raw[:deg] {
		if v == 1 {
			newP[i+1] = 1
		}
	}
	if newP[deg] == 1 {
		for i, v := range primitivePolynomial[:deg] {
			if v == 1 {
				if newP[i] == 1 {
					newP[i] = 0
				} else {
					newP[i] = 1
				}
			}
		}
	}
	newP[deg] = 0
	return newP
}

func getValueOfPolynomial(p polynomial) uint8 {
	var v uint8
	for i, coefficient := range p[:deg] {
		if coefficient != 0 {
			add := 1 << uint8(i)
			v += uint8(add)
		}
	}
	return v
}

func genLogTable(expTable []byte) []byte {
	table := make([]byte, 1<<deg)
	table[0] = 0
	for i, v := range expTable {
		table[v] = byte(i)
	}
	return table
}

func genMulTable(expTable, logTable []byte) (result [256][256]byte) {

	for a := range result {
		for b := range result[a] {
			if a == 0 || b == 0 {
				result[a][b] = 0
				continue
			}
			logA := int(logTable[a])
			logB := int(logTable[b])
			logSum := logA + logB
			for logSum >= 255 {
				logSum -= 255
			}
			result[a][b] = expTable[logSum]
		}
	}
	return result
}

func genMulTableHalf(mulTable [256][256]byte) (low [][]byte, high [][]byte) {
	low = make([][]byte, 256)
	high = make([][]byte, 256)
	for i := range low {
		low[i] = make([]byte, 16)
		high[i] = make([]byte, 16)
	}
	for i := range low {
		for j := range low {
			//result := 0
			var result byte
			if !(i == 0 || j == 0) {
				//result = int(mulTable[i][j])
				result = mulTable[i][j]

			}
			// j & 00001111, [0,15]
			if (j & 0xf) == j {
				low[i][j] = result
			}
			// j & 11110000, [240,255]
			if (j & 0xf0) == j {
				high[i][j>>4] = result
			}
		}
	}
	return
}

func genInverseTable(mulTable [256][256]byte) [256]byte {

	var inverseTable [256]byte
	for i, t := range mulTable {
		for j, v := range t {
			if int(v) == 1 {
				inverseTable[i] = byte(j)
			}
		}
	}
	return inverseTable
}

func formatPolynomial(p polynomial) string {
	var ps string
	for i := deg; i > 1; i-- {
		if p[i] == 1 {
			ps = ps + "x^" + strconv.FormatInt(int64(i), 10) + "+"
		}
	}
	if p[1] == 1 {
		ps = ps + "x+"
	}
	if p[0] == 1 {
		ps = ps + "1"
	} else {
		strings.TrimSuffix(ps, "+")
	}
	return ps
}
