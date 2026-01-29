package assets

import (
    "embed"
    "image/gif"
    "log"
    "time"

    "github.com/hajimehoshi/ebiten/v2"
)

//go:embed images/*.png images/*.gif
var projectAssets embed.FS // Added *.gif support

// GIFData holds the ready-to-play frames
type GIFData struct {
    Frames []*ebiten.Image
    Delays []int // Delay in "Ticks" (1/60th of a second) for each frame
}

func LoadGIF(name string) *GIFData {
    f, err := projectAssets.Open("images/" + name)
    if err != nil {
        log.Fatalf("Failed to open GIF '%s': %v", name, err)
    }
    defer f.Close()

    // Decode the GIF structure
    g, err := gif.DecodeAll(f)
    if err != nil {
        log.Fatalf("Failed to decode GIF '%s': %v", name, err)
    }

    // Convert frames to Ebiten Images
    frames := make([]*ebiten.Image, len(g.Image))
    delays := make([]int, len(g.Image))

    for i, srcImg := range g.Image {
        frames[i] = ebiten.NewImageFromImage(srcImg)
        
        // Convert GIF delay (1/100th sec) to Ebiten Ticks (1/60th sec)
        // Formula: (gif_delay * 10ms) / 16.6ms per tick
        sec := float64(g.Delay[i]) / 100.0
        delays[i] = int(sec * 60)
        
        // Safety: Ensure at least 1 tick delay
        if delays[i] < 1 {
            delays[i] = 1
        }
    }

    return &GIFData{Frames: frames, Delays: delays}
}