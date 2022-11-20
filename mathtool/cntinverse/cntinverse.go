// Copyright (c) 2017 Temple3x (temple3x@gmail.com)
//
// Use of this source code is governed by the MIT License
// that can be found in the LICENSE file.

// This tools will calculate the number of inverse matrices
// with specific data & parity number.
package main

import (
	"flag"
	"fmt"
	"github.com/templexxx/reedsolomon"
	"os"
)

var vects = flag.Uint64("vects", 20, "number of vectors (data+parity)")
var data = flag.Uint64("data", 0, "number of data vectors; keep it empty if you want to "+
	"get the max num of inverse matrix")

func init() {
	flag.Usage = func() {
		fmt.Printf("Usage of %s:\n", os.Args[0])
		fmt.Println("  cntinverse [-flags]")
		fmt.Println("  Valid flags:")
		flag.PrintDefaults()
	}
}

func main() {
	flag.Parse()

	n := float64(*vects)
	k := float64(*data)

	if k == 0 {
		k = n / 2
	}
	fmt.Printf("num of inverse matrices for vectors: %d, data: %d: %.f \n",
		uint64(n),
		uint64(k),
		reedsolomon.GeneralizedBinomial(n, k))
}
