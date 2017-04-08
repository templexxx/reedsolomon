package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
)

// set deg here
const deg = 8 // <= 8

type polynomial [deg + 1]uint8

func main() {
	f, err := os.OpenFile("tables", os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		log.Fatalln(err)
	}
	defer f.Close()
	outputWriter := bufio.NewWriter(f)
	ps := genPrimitivePolynomial()
	title := strconv.FormatInt(int64(deg), 10) + " degree primitive polynomial：\n"
	var pss string
	for i, p := range ps {
		pf := formatPolynomial(p)
		pf = strconv.FormatInt(int64(i+1), 10) + ". " + pf + ";\n"
		pss = pss + pf
	}
	body := fmt.Sprintf(title+"%v", pss)
	outputWriter.WriteString(body)

	//set primitive polynomial here to generator tables
	//x^8+x^4+x^3+x^2+1
	var primitivePolynomial polynomial
	primitivePolynomial[0] = 1
	primitivePolynomial[2] = 1
	primitivePolynomial[3] = 1
	primitivePolynomial[4] = 1
	primitivePolynomial[8] = 1

	lenExpTable := (1 << deg) - 1
	expTable := genExpTable(primitivePolynomial, lenExpTable)
	body = fmt.Sprintf("expTable: %#v\n", expTable)
	outputWriter.WriteString(body)

	logTable := genLogTable(expTable)
	body = fmt.Sprintf("logTable: %#v\n", logTable)
	outputWriter.WriteString(body)

	mulTable := genMulTable(expTable, logTable)
	body = fmt.Sprintf("mulTable: %#v\n", mulTable)
	outputWriter.WriteString(body)

	lowTable, highTable := genMulTableHalf(mulTable)
	body = fmt.Sprintf("lowTable: %#v\n", lowTable)
	outputWriter.WriteString(body)
	body = fmt.Sprintf("highTable: %#v\n", highTable)
	outputWriter.WriteString(body)

	inverseTable := genInverseTable(mulTable)
	body = fmt.Sprintf("inverseTable: %#v\n", inverseTable)
	outputWriter.WriteString(body)
	outputWriter.Flush()
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

func genInverseTable(mulTable [256][256]byte) [256]byte {
	var inVerseTable [256]byte
	for i, t := range mulTable {
		for j, v := range t {
			if int(v) == 1 {
				inVerseTable[i] = byte(j)
			}
		}
	}
	return inVerseTable
}

func genMulTable(expTable, logTable []byte) [256][256]byte {
	var result [256][256]byte
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

func genMulTableHalf(mulTable [256][256]byte) (low [256][16]byte, high [256][16]byte) {
	for a := range low {
		for b := range low {
			result := 0
			if !(a == 0 || b == 0) {
				result = int(mulTable[a][b])
			}
			if (b & 0xf) == b {
				low[a][b] = byte(result)
			}
			if (b & 0xf0) == b {
				high[a][b>>4] = byte(result)
			}
		}
	}
	return
}

// generate primitive Polynomial
func genPrimitivePolynomial() []polynomial {
	// 去除多项式x，则常数项必须为1
	// 因此总共有 2^(deg-1) 个多项式
	len := 1 << (deg - 1)
	var polynomials []polynomial
	var p polynomial
	p[0] = 1
	p[deg] = 1
	// 生成所有多项式
	for i := 0; i < len; i++ {
		p = binaryAddOfPolynomial(p, 1)
		polynomials = append(polynomials, p)
	}
	// 去除多项式 x+1, 可得多项式的项总数为奇数
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
	// 计算本原元的阶是否为2^deg-1
	var ps []polynomial
	for _, p := range psRaw {
		lenTable := (1 << deg) - 1
		table := genExpTable(p, lenTable)
		var numOf1 int
		for _, v := range table {
			// 累加指数表中为 1 的元素
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

// 将多项式系数提取出来做二进制加法
func binaryAddOfPolynomial(p polynomial, i int) polynomial {
	if p[i] == 0 {
		p[i] = 1
	} else {
		p[i] = 0
		i++
		if i == deg {
			return p
		}
		p = binaryAddOfPolynomial(p, i)
	}
	return p
}

// 生成对数表
func genLogTable(expTable []byte) []byte {
	table := make([]byte, (1 << deg))
	//table[0] 无法由本原元的幂得到
	table[0] = 0
	for i, v := range expTable {
		table[v] = byte(i)
	}
	return table
}

// 生成指数表
// 例如，2^8 expTable的length为255，即指数表中最高指数为255-1
func genExpTable(primitivePolynomial polynomial, exp int) []byte {
	table := make([]byte, exp)
	var rawPolynomial polynomial
	rawPolynomial[1] = 1
	table[0] = byte(1)
	table[1] = byte(2)
	for i := 2; i < exp; i++ {
		rawPolynomial = expGrowPolynomial(rawPolynomial, primitivePolynomial)
		table[i] = byte(getValueOfPolynomial(rawPolynomial))
	}
	return table
}

// 本原元升幂得到的多项式
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

// 获取多项式的值
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
