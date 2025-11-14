package game

import (
	"github.com/hajimehoshi/ebiten/v2"
)

// Camera controls what region of the world is rendered.
type Camera struct {
	W, H int
}

func NewCamera(w, h int) *Camera {
	return &Camera{W: w, H: h}
}

// Draw draws the map, items, player, enemies, and optionally the heart in camera/world space.
func (c *Camera) Draw(screen *ebiten.Image, md *MapData, player *Player, heart *Heart) {
	// Center camera on the player
	camX := player.X - float64(c.W)/2
	camY := player.Y - float64(c.H)/2

	// Clamp left & top
	if camX < 0 {
		camX = 0
	}
	if camY < 0 {
		camY = 0
	}

	// Clamp right & bottom
	if md != nil {
		maxCamX := float64(md.Width) - float64(c.W)
		if camX > maxCamX {
			camX = maxCamX
		}

		maxCamY := float64(md.Height) - float64(c.H)
		if camY > maxCamY {
			camY = maxCamY
		}
	}

	// Camera view buffer
	cameraView := ebiten.NewImage(c.W, c.H)

	// Draw tilemap + world objects (items, portal, enemies)
	if md != nil {
		// Draw tilemap (world)
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(-camX, -camY)
		cameraView.DrawImage(md.Image, op)

		// Draw good items
		for _, it := range md.Items {
			op2 := &ebiten.DrawImageOptions{}
			op2.GeoM.Translate(it.X-camX, it.Y-camY)
			cameraView.DrawImage(it.Img, op2)
		}

		// Draw bad items
		for _, it := range md.BadItems {
			op3 := &ebiten.DrawImageOptions{}
			op3.GeoM.Translate(it.X-camX, it.Y-camY)
			cameraView.DrawImage(it.Img, op3)
		}

		// Draw portal
		if md.Portal != nil && md.Portal.Active {
			op4 := &ebiten.DrawImageOptions{}
			op4.GeoM.Translate(md.Portal.X-camX, md.Portal.Y-camY)
			cameraView.DrawImage(md.Portal.Img, op4)
		}

		// Draw enemies (if any)
		for _, e := range md.Enemies {
			if len(e.Images) == 0 {
				continue
			}
			frame := (e.Frame / 6) % len(e.Images)
			eImg := e.Images[frame]
			opE := &ebiten.DrawImageOptions{}
			opE.GeoM.Translate(e.X-camX, e.Y-camY)
			cameraView.DrawImage(eImg, opE)
		}
	}

	// Draw player
	if player != nil && len(player.Anim) > 0 {
		frames := player.Anim[player.Dir]
		if len(frames) > 0 {
			frame := (player.Frame / 6) % len(frames)
			img := frames[frame]

			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(player.X-camX, player.Y-camY)
			cameraView.DrawImage(img, op)
		}
	}

	// Draw heart (world space) – used on Game Over screen
	if heart != nil {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(heart.X-camX, heart.Y-camY)
		cameraView.DrawImage(heart.Img, op)
	}

	// Scale camera view to final window
	scaleX := float64(screen.Bounds().Dx()) / float64(c.W)
	scaleY := float64(screen.Bounds().Dy()) / float64(c.H)

	finalOp := &ebiten.DrawImageOptions{}
	finalOp.GeoM.Scale(scaleX, scaleY)
	screen.DrawImage(cameraView, finalOp)
}
