package main

import (
	"log"

	"programProject2/game"

	"github.com/hajimehoshi/ebiten/v2"
)

func main() {
	ebiten.SetWindowSize(800, 800)
	ebiten.SetWindowTitle("Tile + Camera + Resolv")

	g := game.NewGame()
	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}
