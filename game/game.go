package game

import (
	"fmt"
	"image/color"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
)

// -------------------------------
// FloatText structure
// -------------------------------
type FloatText struct {
	Text  string
	X, Y  float64
	Life  int
	Alpha float64
}

// -------------------------------
// Game struct
// -------------------------------
type Game struct {
	MapData    *MapData
	Player     *Player
	Camera     *Camera
	screenW    int
	screenH    int
	level      int
	floatTexts []*FloatText
	smallFont  font.Face
}

func NewGame() *Game {
	g := &Game{
		screenW: 800,
		screenH: 800,
	}

	InitFont()

	// --- Create smaller font for +1 text ---
	data, err := EmbeddedFS.ReadFile("Assets/Fonts/Square-Black.ttf")
	if err != nil {
		log.Fatalf(" Could not load font for +1 text: %v", err)
	}
	tt, _ := opentype.Parse(data)
	g.smallFont, _ = opentype.NewFace(tt, &opentype.FaceOptions{
		Size:    18,
		DPI:     72,
		Hinting: font.HintingFull,
	})

	g.MapData = LoadMap()
	g.Player = NewPlayer(float64(g.MapData.Width/2-16), float64(g.MapData.Height/2-16))
	g.Camera = NewCamera(400, 400)
	g.level = 1

	fmt.Println("✅ Game initialized successfully")
	return g
}

// -------------------------------
// FloatText handling
// -------------------------------
func (g *Game) AddFloatText(x, y float64) {
	ft := &FloatText{
		Text:  "+1",
		X:     x,
		Y:     y,
		Life:  60,
		Alpha: 1.0,
	}
	g.floatTexts = append(g.floatTexts, ft)
}

func (g *Game) updateFloatTexts() {
	active := []*FloatText{}
	for _, ft := range g.floatTexts {
		ft.Y -= 0.5
		ft.Life--
		ft.Alpha -= 0.02
		if ft.Life > 0 {
			active = append(active, ft)
		}
	}
	g.floatTexts = active
}

func (g *Game) drawFloatTexts(screen *ebiten.Image, camX, camY float64) {
	for _, ft := range g.floatTexts {
		clr := color.RGBA{255, 255, 255, uint8(ft.Alpha * 255)}
		drawFace := text.NewGoXFace(g.smallFont)
		opts := &text.DrawOptions{}
		opts.GeoM.Translate(ft.X-camX, ft.Y-camY)
		opts.ColorScale.ScaleWithColor(clr)
		text.Draw(screen, ft.Text, drawFace, opts)
	}
}

// -------------------------------
// Level loading + Update + Draw
// -------------------------------
func (g *Game) LoadLevel(level int) {
	switch level {
	case 1:
		g.MapData = LoadMapFile("Assets/Maps/floor1.tmx")
	case 2:
		g.MapData = LoadMapFile("Assets/Maps/floor2.tmx")
		g.Player.X = 160
		g.Player.Y = 280
		g.MapData.Items = nil
		g.MapData.BadItems = nil
		g.MapData.Portal = nil
		g.MapData.SpawnEnemies(2)
	default:
		log.Fatalf(" Unknown level: %d", level)
	}

	g.Player.Box.SetPosition(g.Player.X+g.Player.HitboxOffsetX, g.Player.Y+g.Player.HitboxOffsetY)
	g.Camera = NewCamera(400, 400)
}

func (g *Game) Update() error {
	g.Player.Update(g.MapData.SolidTiles, g.MapData.Width, g.MapData.Height)
	g.MapData.CheckItemCollection(g.Player, g)

	if g.MapData.Portal != nil && g.MapData.Portal.Active {
		playerBox := g.Player.Box
		portalRect := makePortalRect(g.MapData.Portal.X, g.MapData.Portal.Y, g.MapData.Portal.Img)

		if playerBox.IsIntersecting(portalRect) {
			fmt.Println("🌀 Portal entered! Loading Level 2...")

			g.MapData.Items = nil
			g.MapData.Collected = 0
			g.MapData.Portal.Active = false
			g.level = 2
			g.LoadLevel(2)
			return nil
		}
	}

	for _, e := range g.MapData.Enemies {
		e.Update()
	}

	g.updateFloatTexts()
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	if GameOver {
		screen.Fill(color.Black)
		drawFace := text.NewGoXFace(ScoreFont)
		opts := &text.DrawOptions{}
		opts.GeoM.Translate(200, 400)
		opts.ColorScale.ScaleWithColor(color.RGBA{255, 0, 0, 255})
		text.Draw(screen, "GAME OVER", drawFace, opts)
		return
	}

	// Draw map, items, player, etc.
	camX := g.Player.X - float64(g.Camera.W)/2
	camY := g.Player.Y - float64(g.Camera.H)/2
	if camX < 0 {
		camX = 0
	}
	if camY < 0 {
		camY = 0
	}

	g.Camera.Draw(screen, g.MapData, g.Player)
	g.drawFloatTexts(screen, camX, camY)
	ebitenutil.DebugPrint(screen, "Arrow keys to move 🦊")

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
