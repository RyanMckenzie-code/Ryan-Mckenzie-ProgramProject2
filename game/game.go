package game

import (
	"fmt"
	"image/color"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/solarlune/resolv"
)

type Game struct {
	MapData *MapData
	Player  *Player
	Camera  *Camera
	screenW int
	screenH int
	level   int
}

func NewGame() *Game {
	g := &Game{
		screenW: 800,
		screenH: 800,
	}

	InitFont()

	g.MapData = LoadMap()
	g.Player = NewPlayer(float64(g.MapData.Width/2-16), float64(g.MapData.Height/2-16))
	g.Camera = NewCamera(400, 400)
	g.level = 1

	fmt.Println("✅ Game initialized successfully")
	return g
}
func (g *Game) LoadLevel(level int) {
	switch level {
	case 1:
		g.MapData = LoadMapFile("Assets/Maps/floor1.tmx")
	case 2:
		g.MapData = LoadMapFile("Assets/Maps/floor2.tmx")
		g.Player.X = 160
		g.Player.Y = 280

		// Remove items and portal for floor2
		g.MapData.Items = nil
		g.MapData.BadItems = nil
		g.MapData.Portal = nil
		g.MapData.SpawnEnemies(2) // spawn 2 enemies
	default:
		log.Fatalf(" Unknown level: %d", level)
	}

	// Update hitbox and camera
	g.Player.Box.SetPosition(g.Player.X+g.Player.HitboxOffsetX, g.Player.Y+g.Player.HitboxOffsetY)
	g.Camera = NewCamera(400, 400)

	// update hitbox and camera
	g.Player.Box.SetPosition(g.Player.X+g.Player.HitboxOffsetX, g.Player.Y+g.Player.HitboxOffsetY)
	g.Camera = NewCamera(400, 400)
}

func (g *Game) Update() error {

	g.Player.Update(g.MapData.SolidTiles, g.MapData.Width, g.MapData.Height)
	g.MapData.CheckItemCollection(g.Player)

	if g.MapData.Portal != nil && g.MapData.Portal.Active {
		playerBox := g.Player.Box
		portalRect := resolv.NewRectangle(
			g.MapData.Portal.X,
			g.MapData.Portal.Y,
			float64(g.MapData.Portal.Img.Bounds().Dx()),
			float64(g.MapData.Portal.Img.Bounds().Dy()),
		)

		if playerBox.IsIntersecting(portalRect) {
			fmt.Println("🌀 Portal entered! Loading Level 2...")

			g.MapData.Items = nil
			g.MapData.Collected = 0
			g.MapData.Portal.Active = false

			// 🚪 Load Level 2
			g.level = 2
			g.LoadLevel(2)
			return nil
		}
	}
	for _, e := range g.MapData.Enemies {
		e.Update()
	}

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	if GameOver {
		// --- Draw Game Over Screen ---
		screen.Fill(color.Black)
		drawFace := text.NewGoXFace(ScoreFont)
		opts := &text.DrawOptions{}
		opts.GeoM.Translate(200, 400)
		opts.ColorScale.ScaleWithColor(color.RGBA{255, 0, 0, 255})
		text.Draw(screen, "GAME OVER", drawFace, opts)
		return
	}

	// --- Normal drawing ---
	g.Camera.Draw(screen, g.MapData, g.Player)
	ebitenutil.DebugPrint(screen, "Arrow keys to move 🦊")

	// Only show fish text on level 1
	if g.level == 1 {
		count := g.MapData.Collected
		if count > 9 {
			count = 9
		}
		msg := fmt.Sprintf("Fish: %d / 9", count)

		drawFace := text.NewGoXFace(ScoreFont)
		opts := &text.DrawOptions{}
		opts.GeoM.Translate(30, 50)
		opts.ColorScale.ScaleWithColor(color.RGBA{255, 255, 255, 255})
		text.Draw(screen, msg, drawFace, opts)
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return g.screenW, g.screenH
}
