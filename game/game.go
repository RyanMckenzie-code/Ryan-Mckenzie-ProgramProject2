package game

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	text "github.com/hajimehoshi/ebiten/v2/text/v2"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
)

const (
	ScreenCenterX           = 400.0
	ScreenCenterY           = 400.0
	StatePlaying  GameState = iota
	StateGameOver
)

type GameState int

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
	MapData        *MapData
	Player         *Player
	Camera         *Camera
	screenW        int
	screenH        int
	level          int
	floatTexts     []*FloatText
	smallFont      font.Face
	State          GameState
	GameOverPlayer *Player
	Heart          *Heart

	// --- Portal popup animation ---
	portalTextTimer int
	portalAlpha     float64
	portalY         float64
}

// -------------------------------
// Init Game
// -------------------------------
func NewGame() *Game {
	g := &Game{
		screenW: 800,
		screenH: 800,
	}

	InitFont()

	// small +1 font
	data, err := EmbeddedFS.ReadFile("Assets/Fonts/Square-Black.ttf")
	if err != nil {
		log.Fatalf("Could not load small font: %v", err)
	}
	tt, _ := opentype.Parse(data)
	g.smallFont, _ = opentype.NewFace(tt, &opentype.FaceOptions{
		Size:    18,
		DPI:     72,
		Hinting: font.HintingFull,
	})

	// Initial map + player
	g.MapData = LoadMap()
	g.Player = NewPlayer(float64(g.MapData.Width/2-16), float64(g.MapData.Height/2-16))
	g.Camera = NewCamera(400, 400)
	g.level = 1

	return g
}

// -------------------------------
// LoadLevel (RESTORED)
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
		log.Fatalf("Unknown level: %d", level)
	}

	g.Player.Box.SetPosition(g.Player.X+g.Player.HitboxOffsetX, g.Player.Y+g.Player.HitboxOffsetY)
	g.Camera = NewCamera(400, 400)
}

// -------------------------------
// Floating +1 Text
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
// Portal Text Animation
// -------------------------------
func (g *Game) updatePortalTextAnimation() {
	if g.portalTextTimer <= 0 {
		return
	}

	g.portalTextTimer--

	if g.portalAlpha < 1 {
		g.portalAlpha += 0.04
		if g.portalAlpha > 1 {
			g.portalAlpha = 1
		}
	}

	// float upward
	g.portalY -= 0.3
}

// -------------------------------
// UPDATE
// -------------------------------
func (g *Game) Update() error {

	// -------- GAME OVER MODE --------
	if g.State == StateGameOver {

		g.GameOverPlayer.Update(nil, g.screenW, g.screenH)

		heartRect := makeHeartRect(g.Heart.X, g.Heart.Y, g.Heart.Img)
		if g.GameOverPlayer.Box.IsIntersecting(heartRect) {
			g.RestartGame()
		}

		return nil
	}

	// -------- NORMAL UPDATE --------
	prevCollected := g.MapData.Collected

	// Move player & check items
	g.Player.Update(g.MapData.SolidTiles, g.MapData.Width, g.MapData.Height)
	g.MapData.CheckItemCollection(g.Player, g)

	// Detect 9th fish → start popup animation
	if g.MapData.Collected == 9 && prevCollected != 9 {
		g.portalTextTimer = 90
		g.portalAlpha = 0
		g.portalY = g.MapData.PortalTextY
	}

	// update effects
	g.updateFloatTexts()
	g.updatePortalTextAnimation()
	for _, e := range g.MapData.Enemies {
		e.Update()
	}

	// portal collision
	if g.MapData.Portal != nil && g.MapData.Portal.Active {
		portalRect := makePortalRect(g.MapData.Portal.X, g.MapData.Portal.Y, g.MapData.Portal.Img)
		if g.Player.Box.IsIntersecting(portalRect) {
			g.LoadLevel(2)
		}
	}

	return nil
}

// -------------------------------
// DRAW
// -------------------------------
func (g *Game) Draw(screen *ebiten.Image) {

	// --------- GAME OVER SCREEN ---------
	if g.State == StateGameOver {
		screen.Fill(color.Black)

		g.Camera.Draw(screen, nil, g.GameOverPlayer, g.Heart)

		drawCenteredText(screen, "GAME OVER", ScoreFont, ScreenCenterY-150, color.White)
		drawCenteredText(screen, "Touch the Heart to Restart", ScoreFont, ScreenCenterY-90, color.White)
		return
	}

	// --------- NORMAL RENDER ---------
	g.Camera.Draw(screen, g.MapData, g.Player, nil)

	camX := g.Player.X - float64(g.Camera.W)/2
	camY := g.Player.Y - float64(g.Camera.H)/2

	// Floating +1 text
	g.drawFloatTexts(screen, camX, camY)

	// -------- Animated Portal Popup Text --------
	if g.portalTextTimer > 0 {
		drawFace := text.NewGoXFace(ScoreFont)
		opts := &text.DrawOptions{}

		col := color.RGBA{255, 255, 255, uint8(255 * g.portalAlpha)}
		opts.ColorScale.ScaleWithColor(col)

		opts.GeoM.Translate(
			g.MapData.PortalTextX-camX,
			g.portalY-camY-20,
		)

		text.Draw(screen, "A portal has appeared!", drawFace, opts)
	}

	// -------- HUD (Fish count) --------
	count := g.MapData.Collected
	if count > 9 {
		count = 9
	}

	msg := fmt.Sprintf("Fish: %d / 9", count)

	drawFace := text.NewGoXFace(ScoreFont)
	opts := &text.DrawOptions{}
	opts.GeoM.Translate(30, 50)
	opts.ColorScale.ScaleWithColor(color.White)

	text.Draw(screen, msg, drawFace, opts)
}

// -------------------------------
// Restart Game
// -------------------------------
func (g *Game) RestartGame() {

	g.State = StatePlaying
	g.level = 1
	g.MapData = LoadMap()
	g.Player = NewPlayer(float64(g.MapData.Width/2-16), float64(g.MapData.Height/2-16))
	g.Camera = NewCamera(400, 400)
	g.floatTexts = nil

	g.GameOverPlayer = nil
	g.Heart = nil

	g.portalTextTimer = 0
	g.portalAlpha = 0
}

// -------------------------------
func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return g.screenW, g.screenH
}

// -------------------------------
func drawCenteredText(screen *ebiten.Image, msg string, face font.Face, y float64, col color.Color) {
	drawFace := text.NewGoXFace(face)
	width, _ := text.Measure(msg, drawFace, 0)

	opts := &text.DrawOptions{}
	opts.GeoM.Translate(ScreenCenterX-(width/2), y)
	opts.ColorScale.ScaleWithColor(col)

	text.Draw(screen, msg, drawFace, opts)
}

// -------------------------------
// Game Over Objects
// -------------------------------
func (g *Game) initGameOverHeart() {
	data, err := EmbeddedFS.ReadFile("Assets/Sprites/heart.png")
	if err != nil {
		log.Fatal("Could not load heart.png:", err)
	}

	img, _, _ := image.Decode(bytes.NewReader(data))
	heartImg := ebiten.NewImageFromImage(img)

	g.Heart = &Heart{
		X:   ScreenCenterX - float64(heartImg.Bounds().Dx())/2,
		Y:   ScreenCenterY + 325,
		Img: heartImg,
	}
}

func (g *Game) initGameOverPlayer() {
	g.initGameOverHeart()
	g.GameOverPlayer = NewLanternPlayer(ScreenCenterX, g.Heart.Y+140)
	g.Camera = NewCamera(g.screenW, g.screenH)
}
