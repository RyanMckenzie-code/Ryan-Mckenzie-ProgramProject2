package game

import (
	"log"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
)

var ScoreFont font.Face

// Load and prepare the font using opentype.NewFace
func InitFont() {
	data, err := EmbeddedFS.ReadFile("Assets/Fonts/Square-Black.ttf")
	if err != nil {
		log.Fatalf(" Could not load font: %v", err)
	}

	ttFont, err := opentype.Parse(data)
	if err != nil {
		log.Fatalf(" Could not parse font: %v", err)
	}

	ScoreFont, err = opentype.NewFace(ttFont, &opentype.FaceOptions{
		Size:    36, // font size (adjust to fit screen)
		DPI:     72,
		Hinting: font.HintingFull,
	})
	if err != nil {
		log.Fatalf(" Could not create font face: %v", err)
	}
}
