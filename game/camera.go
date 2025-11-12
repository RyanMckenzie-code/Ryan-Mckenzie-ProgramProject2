package game

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

type Camera struct {
	W, H int
}

func NewCamera(w, h int) *Camera {
	return &Camera{W: w, H: h}
}

func (c *Camera) Draw(screen *ebiten.Image, md *MapData, player *Player) {
	camX := player.X - float64(c.W)/2
	camY := player.Y - float64(c.H)/2

	if camX < 0 {
		camX = 0
	}
	if camY < 0 {
		camY = 0
	}
	if camX > float64(md.Width-c.W) {
		camX = float64(md.Width - c.W)
	}
	if camY > float64(md.Height-c.H) {
		camY = float64(md.Height - c.H)
	}

	cameraView := ebiten.NewImage(c.W, c.H)

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(-camX, -camY)
	cameraView.DrawImage(md.Image, op)

	// --- Draw items (fish cans) ---
	for _, item := range md.Items {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(item.X-camX, item.Y-camY)
		cameraView.DrawImage(item.Img, op)
	}
	// --- Draw bad items ---
	for _, bad := range md.BadItems {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(bad.X-camX, bad.Y-camY)
		cameraView.DrawImage(bad.Img, op)
	}

	// --- Draw portal if active ---
	if md.Portal != nil && md.Portal.Active {
		// draw the portal sprite
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(md.Portal.X-camX, md.Portal.Y-camY)
		cameraView.DrawImage(md.Portal.Img, op)

		// draw "Portal Now Open!" text at last collected fish position
		drawFace := text.NewGoXFace(ScoreFont)
		textOpts := &text.DrawOptions{}
		textOpts.GeoM.Scale(0.5, 0.5)
		textOpts.GeoM.Translate(md.PortalTextX-camX+10, md.PortalTextY-camY-10) // 👈 position near the last fish
		textOpts.ColorScale.ScaleWithColor(color.RGBA{255, 0, 0, 255})          // red text
		text.Draw(cameraView, "Portal Now Open, Find It!", drawFace, textOpts)
	}
	// --- Draw enemies ---
	// --- Draw enemies ---
	for _, enemy := range md.Enemies {
		if len(enemy.Images) == 0 {
			continue
		}

		frame := (enemy.Frame / 8) % len(enemy.Images)
		img := enemy.Images[frame]
		op := &ebiten.DrawImageOptions{}

		scale := 3.0 // 👈 triple the size
		op.GeoM.Scale(scale, scale)
		op.GeoM.Translate(enemy.X-camX, enemy.Y-camY)
		cameraView.DrawImage(img, op)
	}
	// --- Draw player ---
	dir := player.Dir
	frames := player.Anim[dir]
	frame := (player.Frame / 6) % len(frames)
	img := frames[frame]

	pOp := &ebiten.DrawImageOptions{}
	pOp.GeoM.Translate(player.X-camX, player.Y-camY)
	cameraView.DrawImage(img, pOp)

	scaleX := float64(800) / float64(c.W)
	scaleY := float64(800) / float64(c.H)
	finalOp := &ebiten.DrawImageOptions{}
	finalOp.GeoM.Scale(scaleX, scaleY)
	screen.DrawImage(cameraView, finalOp)
}
