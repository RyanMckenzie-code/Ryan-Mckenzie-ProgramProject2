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

// -------------------------------
// Data structures
// -------------------------------
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
	Portal      *Portal
	EmptyTiles  [][2]int
	PortalTextX float64
	PortalTextY float64
	Enemies     []*Enemy
}

var GameOver bool

// -------------------------------
// Helpers for item hitboxes
// -------------------------------
const (
	itemBoxW         = 21.0
	itemBoxH         = 19.0
	badItemBoxW      = 23.0 // slightly wider for bad item PNG
	badItemBoxH      = 18.0
	badItemYOffset   = 8.0 // adjust downward if image sits higher visually
	badItemXOffset   = -8.0
	portalBoxW       = 16.0 // width of portal hitbox
	portalBoxH       = 16.0 // height of portal hitbox
	portalBoxYOffset = 0.0  // nudge downward if needed
)

// Good item box (centered)
func makeItemRect(x, y float64, img *ebiten.Image) resolv.IShape {
	iw := float64(img.Bounds().Dx())
	ih := float64(img.Bounds().Dy())
	offX := (iw - itemBoxW) / 2.0
	offY := (ih - itemBoxH) / 2.0
	return resolv.NewRectangle(x+offX, y+offY, itemBoxW, itemBoxH)
}

// Bad item box (slightly wider and vertically offset)
func makeBadItemRect(x, y float64, img *ebiten.Image) resolv.IShape {
	iw := float64(img.Bounds().Dx())
	ih := float64(img.Bounds().Dy())
	offX := (iw-badItemBoxW)/2.0 + badItemXOffset
	offY := (ih-badItemBoxH)/2.0 + badItemYOffset
	return resolv.NewRectangle(x+offX, y+offY, badItemBoxW, badItemBoxH)
}

func makePortalRect(x, y float64, img *ebiten.Image) resolv.IShape {
	iw := float64(img.Bounds().Dx())
	ih := float64(img.Bounds().Dy())
	offX := (iw - portalBoxW) / 2.0
	offY := (ih-portalBoxH)/2.0 + portalBoxYOffset
	return resolv.NewRectangle(x+offX, y+offY, portalBoxW, portalBoxH)
}

// -------------------------------
// Image scaling helper
// -------------------------------
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

// -------------------------------
// Map loading functions
// -------------------------------
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

// -------------------------------
// Spawn items (good + bad)
// -------------------------------
func (md *MapData) spawnItems() {
	data, err := EmbeddedFS.ReadFile("Assets/Sprites/tuna_closed.png")
	if err != nil {
		log.Printf("⚠️ Could not load fish item: %v", err)
		return
	}
	img, _, _ := image.Decode(bytes.NewReader(data))
	fishImg := scaleImage(img, 0.2)

	var emptyTiles [][2]int
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

	dataBad, err := EmbeddedFS.ReadFile("Assets/Sprites/tuna_open.png")
	if err != nil {
		log.Printf("⚠️ Could not load bad item image: %v", err)
		return
	}
	imgBad, _, _ := image.Decode(bytes.NewReader(dataBad))
	badImg := scaleImage(imgBad, 0.2)

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

// -------------------------------
// Collision + collection
// -------------------------------
func (md *MapData) CheckItemCollection(player *Player, g *Game) {
	var remaining []PlacedItem
	var lastCollectedX, lastCollectedY float64
	collectedThisFrame := false
	playerBox := player.Box

	for _, item := range md.Items {
		itemRect := makeItemRect(item.X, item.Y, item.Img)
		if playerBox.IsIntersecting(itemRect) {
			if md.Collected < 9 {
				md.Collected++
				collectedThisFrame = true
				lastCollectedX = item.X
				lastCollectedY = item.Y
				g.AddFloatText(player.X+8, player.Y-10)
			}
			continue
		}
		remaining = append(remaining, item)
	}
	md.Items = remaining

	if collectedThisFrame {
		md.PortalTextX = lastCollectedX
		md.PortalTextY = lastCollectedY
	}

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

	// --- Unified bad item collision ---
	var remainingBad []PlacedItem
	for _, bad := range md.BadItems {
		badRect := makeBadItemRect(bad.X, bad.Y, bad.Img)
		if player.Box.IsIntersecting(badRect) {
			md.BadItems = nil
			GameOver = true
			log.Println("💀 Hit a bad can — GAME OVER")
			return
		}
		remainingBad = append(remainingBad, bad)
	}
	md.BadItems = remainingBad
}

// -------------------------------
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
