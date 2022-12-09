//go:build !amd64
// +build !amd64

package reedsolomon

func (g *gmu) init(feat int) {
	g.mulVect = mulVect
	g.mulVectXOR = mulVectXOR
}
