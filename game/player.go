package game

import (
	"bytes"
	"image"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/solarlune/resolv"
)

type Player struct {
	Anim          [][]*ebiten.Image
	X, Y          float64
	Dir, Frame    int
	Box           resolv.IShape
	HitboxOffsetX float64
	HitboxOffsetY float64
}

func NewPlayer(x, y float64) *Player {
	p := &Player{
		X:             x,
		Y:             y,
		HitboxOffsetX: 8,
		HitboxOffsetY: 35,
	}
	p.Anim = loadPlayerAnim()
	p.Box = resolv.NewRectangle(
		x+p.HitboxOffsetX, y+p.HitboxOffsetY,
		16, 27,
	)
	return p
}

func loadPlayerAnim() [][]*ebiten.Image {
	data, _ := EmbeddedFS.ReadFile("Assets/Sprites/player.png")
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		log.Fatal(err)
	}
	sheet := ebiten.NewImageFromImage(img)

	cols := 12
	rows := 4
	frameW := sheet.Bounds().Dx() / cols
	frameH := sheet.Bounds().Dy() / rows

	out := make([][]*ebiten.Image, rows)
	for row := 0; row < rows; row++ {
		for col := 0; col < cols; col++ {
			sx := col * frameW
			sy := row * frameH
			sub := sheet.SubImage(image.Rect(sx, sy, sx+frameW, sy+frameH)).(*ebiten.Image)
			out[row] = append(out[row], sub)
		}
	}
	return out
}

func (p *Player) Update(solids []resolv.IShape, mapW, mapH int) error {
	speed := 3.0
	moving := false
	var dx, dy float64

	if ebiten.IsKeyPressed(ebiten.KeyLeft) {
		dx -= speed
		p.Dir = 1
		moving = true
	}
	if ebiten.IsKeyPressed(ebiten.KeyRight) {
		dx += speed
		p.Dir = 2
		moving = true
	}
	if ebiten.IsKeyPressed(ebiten.KeyUp) {
		dy -= speed
		p.Dir = 3
		moving = true
	}
	if ebiten.IsKeyPressed(ebiten.KeyDown) {
		dy += speed
		p.Dir = 0
		moving = true
	}

	if moving {
		p.move(dx, dy, solids, mapW, mapH)
		p.Frame++
	} else {
		p.Frame = 0
	}
	return nil
}

func (p *Player) move(dx, dy float64, solids []resolv.IShape, mapW, mapH int) {
	// --- Horizontal movement ---
	if dx != 0 {
		newX := p.X + dx
		p.Box.SetPosition(newX+p.HitboxOffsetX, p.Y+p.HitboxOffsetY)
		if p.collides(solids) {
			// if we hit something horizontally, stop horizontal motion
			p.Box.SetPosition(p.X+p.HitboxOffsetX, p.Y+p.HitboxOffsetY)
		} else {
			p.X = newX
		}
	}

	// --- Vertical movement ---
	if dy != 0 {
		newY := p.Y + dy
		p.Box.SetPosition(p.X+p.HitboxOffsetX, newY+p.HitboxOffsetY)
		if p.collides(solids) {
			// if we hit something vertically, stop vertical motion
			p.Box.SetPosition(p.X+p.HitboxOffsetX, p.Y+p.HitboxOffsetY)
		} else {
			p.Y = newY
		}
	}

	// --- Clamp player to map boundaries ---
	if p.X < 0 {
		p.X = 0
	}
	if p.Y < 0 {
		p.Y = 0
	}
	maxX := float64(mapW - 32)
	maxY := float64(mapH - 32)
	if p.X > maxX {
		p.X = maxX
	}
	if p.Y > maxY {
		p.Y = maxY
	}

	// Update box position at end
	p.Box.SetPosition(p.X+p.HitboxOffsetX, p.Y+p.HitboxOffsetY)
}

func (p *Player) collides(solids []resolv.IShape) bool {
	for _, s := range solids {
		if p.Box.IsIntersecting(s) {
			return true
		}
	}
	return false
}
func NewLanternPlayer(x, y float64) *Player {
	p := &Player{
		X:             x,
		Y:             y,
		HitboxOffsetX: 8,
		HitboxOffsetY: 35,
	}
	p.Anim = loadLanternAnim()
	p.Box = resolv.NewRectangle(
		x+p.HitboxOffsetX, y+p.HitboxOffsetY,
		16, 27,
	)
	return p
}
func loadLanternAnim() [][]*ebiten.Image {
	data, _ := EmbeddedFS.ReadFile("Assets/Sprites/player_lantern.png")
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		log.Fatal(err)
	}
	sheet := ebiten.NewImageFromImage(img)

	cols := 12
	rows := 4
	frameW := sheet.Bounds().Dx() / cols
	frameH := sheet.Bounds().Dy() / rows

	out := make([][]*ebiten.Image, rows)
	for row := 0; row < rows; row++ {
		for col := 0; col < cols; col++ {
			sx := col * frameW
			sy := row * frameH
			sub := sheet.SubImage(image.Rect(sx, sy, sx+frameW, sy+frameH)).(*ebiten.Image)
			out[row] = append(out[row], sub)
		}
	}
	return out
}
