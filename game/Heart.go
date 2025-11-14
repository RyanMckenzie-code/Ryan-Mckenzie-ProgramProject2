package game

import "github.com/hajimehoshi/ebiten/v2"

type Heart struct {
	X, Y                  float64
	Img                   *ebiten.Image
	OrigWidth, OrigHeight int
}
