package assets

import (
    "bytes"
    "embed"
    "image"
    _ "image/png" // Register PNG format
    "log"

    "github.com/hajimehoshi/ebiten/v2"
)

//go:embed images/*.png
var projectAssets embed.FS

// LoadImage loads a PNG sprite sheet into VRAM
func LoadImage(name string) *ebiten.Image {
    fileData, err := projectAssets.ReadFile("images/" + name)
    if err != nil {
        log.Fatalf("Failed to read image '%s': %v", name, err)
    }

    img, _, err := image.Decode(bytes.NewReader(fileData))
    if err != nil {
        log.Fatalf("Failed to decode image '%s': %v", name, err)
    }

    return ebiten.NewImageFromImage(img)
}