package game

import (
	"bytes"
	"fmt"
	"image"
	"log"
	"math/rand/v2"
	"path/filepath"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/lafriks/go-tiled"
	"github.com/solarlune/resolv"
)

type PlacedItem struct {
	X, Y float64
	Img  *ebiten.Image
}
type Portal struct {
	X, Y   float64
	Img    *ebiten.Image
	Active bool
}

type MapData struct {
	Map         *tiled.Map
	Image       *ebiten.Image
	Tiles       map[uint32]*ebiten.Image
	TileW       int
	TileH       int
	Width       int
	Height      int
	SolidTiles  []resolv.IShape
	Items       []PlacedItem
	BadItems    []PlacedItem
	Collected   int
	Portal      *Portal // 👈 add this
	EmptyTiles  [][2]int
	PortalTextX float64
	PortalTextY float64
	Enemies     []*Enemy
}

var GameOver bool

// scaleImage takes an ebiten.Image and a scale factor, and returns a new scaled image.
func scaleImage(img image.Image, scale float64) *ebiten.Image {
	original := ebiten.NewImageFromImage(img)

	sw := int(float64(original.Bounds().Dx()) * scale)
	sh := int(float64(original.Bounds().Dy()) * scale)

	if sw < 1 {
		sw = 1
	}
	if sh < 1 {
		sh = 1
	}

	scaled := ebiten.NewImage(sw, sh)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(scale, scale)
	scaled.DrawImage(original, op)

	return scaled
}

func LoadMap() *MapData {
	m, err := tiled.LoadFile("Assets/Maps/floor1.tmx", tiled.WithFileSystem(EmbeddedFS))
	if err != nil {
		log.Fatalf("Failed to load map: %v", err)
	}

	loadExternalTilesets(m)
	tileImages := loadTilesFromEmbed("Assets/Maps/floor1.tmx", m)

	w := m.Width * m.TileWidth
	h := m.Height * m.TileHeight
	img := ebiten.NewImage(w, h)
	drawMap(img, m, tileImages)

	md := &MapData{
		Map:    m,
		Image:  img,
		Tiles:  tileImages,
		TileW:  m.TileWidth,
		TileH:  m.TileHeight,
		Width:  w,
		Height: h,
	}
	md.loadCollision()
	md.spawnItems()
	return md
}

func loadExternalTilesets(m *tiled.Map) {
	for _, ts := range m.Tilesets {
		if ts.Source != "" && ts.Tiles == nil {
			tsxPath := filepath.ToSlash(filepath.Join("Assets/Maps", ts.Source))
			fmt.Println("📦 Loading external tileset:", tsxPath)
			tsx, err := tiled.LoadTilesetFile(tsxPath, tiled.WithFileSystem(EmbeddedFS))
			if err != nil {
				log.Fatalf(" Failed to load tileset %s: %v", tsxPath, err)
			}
			ts.Tiles = tsx.Tiles
			ts.Image = tsx.Image
			ts.TileCount = tsx.TileCount
			ts.Columns = tsx.Columns
			ts.TileWidth = tsx.TileWidth
			ts.TileHeight = tsx.TileHeight
			ts.Properties = tsx.Properties
		}
	}
}

func loadTilesFromEmbed(mapPath string, m *tiled.Map) map[uint32]*ebiten.Image {
	result := make(map[uint32]*ebiten.Image)
	mapDir := filepath.Dir(mapPath)

	for _, ts := range m.Tilesets {
		imgPath := filepath.ToSlash(filepath.Join(mapDir, ts.Image.Source))
		data, err := EmbeddedFS.ReadFile(imgPath)
		if err != nil {
			log.Fatalf("read tileset %s: %v", imgPath, err)
		}
		img, _, _ := image.Decode(bytes.NewReader(data))
		sheet := ebiten.NewImageFromImage(img)

		for i := 0; i < ts.TileCount; i++ {
			sx := (i % ts.Columns) * ts.TileWidth
			sy := (i / ts.Columns) * ts.TileHeight
			sub := sheet.SubImage(image.Rect(sx, sy, sx+ts.TileWidth, sy+ts.TileHeight)).(*ebiten.Image)
			gid := ts.FirstGID + uint32(i)
			result[gid] = sub
		}
	}
	return result
}

func drawMap(dst *ebiten.Image, m *tiled.Map, tiles map[uint32]*ebiten.Image) {
	for _, layer := range m.Layers {
		if !layer.Visible {
			continue
		}
		for y := 0; y < m.Height; y++ {
			for x := 0; x < m.Width; x++ {
				idx := y*m.Width + x
				if idx >= len(layer.Tiles) {
					continue
				}
				tile := layer.Tiles[idx]
				if tile == nil || tile.Tileset == nil {
					continue
				}
				gid := tile.Tileset.FirstGID + tile.ID
				img := tiles[gid]
				if img == nil {
					continue
				}
				op := &ebiten.DrawImageOptions{}
				op.GeoM.Translate(float64(x*m.TileWidth), float64(y*m.TileHeight))
				dst.DrawImage(img, op)
			}
		}
	}
}

func (md *MapData) loadCollision() {
	for _, layer := range md.Map.Layers {
		if !layer.Visible {
			continue
		}
		for y := 0; y < md.Map.Height; y++ {
			for x := 0; x < md.Map.Width; x++ {
				idx := y*md.Map.Width + x
				if idx >= len(layer.Tiles) {
					continue
				}
				tile := layer.Tiles[idx]
				if tile == nil || tile.Tileset == nil {
					continue
				}

				isSolid := false
				if int(tile.ID) < len(tile.Tileset.Tiles) {
					tsTile := tile.Tileset.Tiles[int(tile.ID)]
					if tsTile != nil && tsTile.Properties != nil && tsTile.Properties.GetBool("solid") {
						isSolid = true
					}
				}

				if isSolid {
					rect := resolv.NewRectangle(float64(x*md.TileW), float64(y*md.TileH), float64(md.TileW), float64(md.TileH))
					md.SolidTiles = append(md.SolidTiles, rect)
				}
			}
		}
	}
}

// Randomly spawn items (fish cans) on non-solid tiles
func (md *MapData) spawnItems() {
	// --- Load and scale the fish image ---
	data, err := EmbeddedFS.ReadFile("Assets/Sprites/tuna_closed.png")
	if err != nil {
		log.Printf("⚠️ Could not load fish item: %v", err)
		return
	}
	img, _, _ := image.Decode(bytes.NewReader(data))
	fishImg := scaleImage(img, 0.2) // 👈 just one line now

	var emptyTiles [][2]int

	// go through every tile in the map
	for y := 0; y < md.Map.Height; y++ {
		for x := 0; x < md.Map.Width; x++ {
			tileX := float64(x * md.TileW)
			tileY := float64(y * md.TileH)
			tileRect := resolv.NewRectangle(tileX, tileY, float64(md.TileW), float64(md.TileH))

			solid := false
			for _, s := range md.SolidTiles {
				if tileRect.IsIntersecting(s) {
					solid = true
					break
				}
			}

			if !solid {
				emptyTiles = append(emptyTiles, [2]int{x, y})
			}
		}
	}
	md.EmptyTiles = emptyTiles

	// --- Spawn 15 fish items ---
	for i := 0; i < 15 && len(emptyTiles) > 0; i++ {
		idx := rand.IntN(len(emptyTiles))
		tile := emptyTiles[idx]
		emptyTiles = append(emptyTiles[:idx], emptyTiles[idx+1:]...)

		md.Items = append(md.Items, PlacedItem{
			X:   float64(tile[0] * md.TileW),
			Y:   float64(tile[1] * md.TileH),
			Img: fishImg,
		})
	}

	// --- Load and scale the bad item image ---
	dataBad, err := EmbeddedFS.ReadFile("Assets/Sprites/tuna_open.png")
	if err != nil {
		log.Printf("⚠️ Could not load bad item image: %v", err)
		return
	}
	imgBad, _, _ := image.Decode(bytes.NewReader(dataBad))
	badImg := scaleImage(imgBad, 0.2) // 👈 same scale as fish

	// --- Spawn 5 bad items ---
	for i := 0; i < 5 && len(emptyTiles) > 0; i++ {
		idx := rand.IntN(len(emptyTiles))
		tile := emptyTiles[idx]
		emptyTiles = append(emptyTiles[:idx], emptyTiles[idx+1:]...)

		md.BadItems = append(md.BadItems, PlacedItem{
			X:   float64(tile[0] * md.TileW),
			Y:   float64(tile[1] * md.TileH),
			Img: badImg,
		})
	}

	md.EmptyTiles = emptyTiles
}

func (md *MapData) CheckItemCollection(player *Player) {
	var remaining []PlacedItem
	var lastCollectedX, lastCollectedY float64
	collectedThisFrame := false

	playerBox := player.Box

	for _, item := range md.Items {
		itemRect := resolv.NewRectangle(
			item.X, item.Y,
			float64(item.Img.Bounds().Dx()), float64(item.Img.Bounds().Dy()),
		)

		if playerBox.IsIntersecting(itemRect) {
			if md.Collected < 9 {
				md.Collected++
				collectedThisFrame = true
				lastCollectedX = item.X
				lastCollectedY = item.Y
			}
			continue
		}

		remaining = append(remaining, item)
	}

	md.Items = remaining

	// Save the position of the last collected fish
	if collectedThisFrame {
		md.PortalTextX = lastCollectedX
		md.PortalTextY = lastCollectedY
	}

	// If we just collected the 9th fish, spawn the portal somewhere random
	if collectedThisFrame && md.Collected == 9 && md.Portal == nil {
		data, err := EmbeddedFS.ReadFile("Assets/Sprites/portal.png")
		if err != nil {
			log.Printf("⚠️ Could not load portal image: %v", err)
			return
		}
		img, _, _ := image.Decode(bytes.NewReader(data))
		portalImg := ebiten.NewImageFromImage(img)

		if len(md.EmptyTiles) == 0 {
			log.Println("⚠️ No empty tiles available for portal spawn.")
			return
		}

		randomTile := md.EmptyTiles[rand.IntN(len(md.EmptyTiles))]

		md.Portal = &Portal{
			X:      float64(randomTile[0] * md.TileW),
			Y:      float64(randomTile[1] * md.TileH),
			Img:    portalImg,
			Active: true,
		}

		log.Println("🌀 Portal spawned randomly! Text remains at last collected fish.")
	}
	// --- Check if player hit any bad items ---
	// --- Check if player hit any bad items ---
	var remainingBad []PlacedItem
	for _, bad := range md.BadItems {
		// offsets to adjust hitbox position
		offsetX := 4.2  // move hitbox 5px left
		offsetY := 18.0 // move hitbox 5px down

		badRect := resolv.NewRectangle(
			bad.X+offsetX,
			bad.Y+offsetY,
			float64(bad.Img.Bounds().Dx()),
			float64(bad.Img.Bounds().Dy()),
		)

		if player.Box.IsIntersecting(badRect) {
			md.BadItems = nil
			GameOver = true
			log.Println(" Hit a bad can — GAME OVER")
			return
		}

		remainingBad = append(remainingBad, bad)
	}
	md.BadItems = remainingBad
}

func LoadMapFile(path string) *MapData {
	m, err := tiled.LoadFile(path, tiled.WithFileSystem(EmbeddedFS))
	if err != nil {
		log.Fatalf("Failed to load map: %v", err)
	}

	loadExternalTilesets(m)
	tileImages := loadTilesFromEmbed(path, m)

	w := m.Width * m.TileWidth
	h := m.Height * m.TileHeight
	img := ebiten.NewImage(w, h)
	drawMap(img, m, tileImages)

	md := &MapData{
		Map:    m,
		Image:  img,
		Tiles:  tileImages,
		TileW:  m.TileWidth,
		TileH:  m.TileHeight,
		Width:  w,
		Height: h,
	}
	md.loadCollision()
	md.spawnItems()
	return md
}
func (md *MapData) SpawnEnemies(count int) {
	enemyFrames := LoadEnemySprites()

	if len(md.EmptyTiles) == 0 {
		log.Println(" No empty tiles available for enemies")
		return
	}

	for i := 0; i < count; i++ {
		tile := md.EmptyTiles[rand.IntN(len(md.EmptyTiles))]
		enemy := &Enemy{
			X:      float64(tile[0]*md.TileW + 8),
			Y:      float64(tile[1]*md.TileH + 8),
			Frame:  0,
			Images: enemyFrames,
		}
		md.Enemies = append(md.Enemies, enemy)
	}
}
