package game

import (
	"bytes"
	"image"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
)

// Enemy represents one animated enemy on the map
type Enemy struct {
	X, Y   float64
	Frame  int
	Images []*ebiten.Image
}

func (e *Enemy) Update() {
	e.Frame++
}

// LoadEnemySprites loads and splits enemies.png (1 row, 6 columns)
// LoadEnemySprites loads and splits enemies.png (1 row, 6 columns)
func LoadEnemySprites() []*ebiten.Image {
	data, err := EmbeddedFS.ReadFile("Assets/Sprites/enemies.png")
	if err != nil {
		log.Fatalf("Could not load enemy sprite sheet: %v", err)
	}

	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		log.Fatalf("Could not decode enemy sprite sheet: %v", err)
	}
	sheet := ebiten.NewImageFromImage(img)

	const cols = 6
	frameW := sheet.Bounds().Dx() / cols
	frameH := sheet.Bounds().Dy()

	// Your original fallback for uneven division
	if sheet.Bounds().Dx()%cols != 0 {
		frameW = 19
	}

	frames := make([]*ebiten.Image, cols)

	for col := 0; col < cols; col++ {
		sx := col * frameW
		sy := 0
		ex := sx + frameW
		if ex > sheet.Bounds().Dx() {
			ex = sheet.Bounds().Dx()
		}
		ey := sy + frameH

		sub := sheet.SubImage(image.Rect(sx, sy, ex, ey)).(*ebiten.Image)

		// --- SCALE ENEMY SPRITE (add this) ---
		scale := 2.0 // change to 1.5 or 3.0 if needed
		scaledW := int(float64(ex-sx) * scale)
		scaledH := int(float64(ey-sy) * scale)

		scaled := ebiten.NewImage(scaledW, scaledH)
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(scale, scale)
		scaled.DrawImage(sub, op)

		frames[col] = scaled
	}

	return frames
}
