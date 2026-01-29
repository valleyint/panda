package entity

import (
    "image"
    "github.com/hajimehoshi/ebiten/v2"
    "panda/internal/assets"
)

type Panda struct {
    X, Y         float64
    
    // Sprite Sheet Data
    spriteSheet  *ebiten.Image
    frameWidth   int // Width of ONE frame
    frameCount   int // Total frames in sheet
    currentFrame int
    
    // Timing
    tickCounter  int
    speed        int // Ticks per frame (Lower = Faster)
}

func NewPanda() *Panda {
    // Load the Sprite Sheet
    sheet := assets.LoadImage("panda_idle.png")
    
    // Auto-detect frame count based on aspect ratio
    // Assumption: The sheet is a horizontal strip of square-ish frames
    totalW, totalH := sheet.Bounds().Dx(), sheet.Bounds().Dy()
    
    // Simple logic: If height is 32, and width is 128, we have 4 frames.
    // If it's a single image, width == height (usually).
    count := totalW / totalH 
    if count == 0 { count = 1 }

    return &Panda{
        X:           120,
        Y:           100,
        spriteSheet: sheet,
        frameWidth:  totalH, // Assuming frames are square (32x32)
        frameCount:  count,
        speed:       15,     // Update every 15 ticks (approx 4 times/sec)
    }
}

func (p *Panda) Update() {
    p.tickCounter++

    if p.tickCounter >= p.speed {
        p.tickCounter = 0
        p.currentFrame++
        
        // Loop Animation
        if p.currentFrame >= p.frameCount {
            p.currentFrame = 0
        }
    }
}

func (p *Panda) Draw(screen *ebiten.Image) {
    if p.spriteSheet == nil { return }

    // Math: Calculate where the current frame lives on the sheet
    sx := p.currentFrame * p.frameWidth
    
    // Cut out the frame
    rect := image.Rect(sx, 0, sx+p.frameWidth, p.spriteSheet.Bounds().Dy())
    subImg := p.spriteSheet.SubImage(rect).(*ebiten.Image)

    op := &ebiten.DrawImageOptions{}
    op.GeoM.Translate(p.X, p.Y)
    op.GeoM.Scale(4, 4) // Retro Zoom

    screen.DrawImage(subImg, op)
}