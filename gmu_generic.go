//go:build !amd64
// +build !amd64

package reedsolomon

func (g *gmu) initFunc(feat int) {
	g.mulVect = mulVectNoSIMD
	g.mulVectXOR = mulVectXORNoSIMD
}
