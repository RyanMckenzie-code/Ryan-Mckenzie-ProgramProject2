package game

import (
	"github.com/hajimehoshi/ebiten/v2"
)

type Item struct {
	X, Y float64
	Img  *ebiten.Image
}
