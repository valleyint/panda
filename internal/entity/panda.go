package entity

import (
    "github.com/hajimehoshi/ebiten/v2"
    "panda/internal/assets"
)

type Panda struct {
    X, Y         float64
    
    // New GIF Animation Data
    frames       []*ebiten.Image
    delays       []int
    currentFrame int
    tickCounter  int // Accumulates time
}

func NewPanda() *Panda {
    // Load GIF instead of PNG
    // Make sure you have 'panda_idle.gif' in assets/images/
    anim := assets.LoadGIF("panda_idle.gif")

    return &Panda{
        X:      120,
        Y:      100,
        frames: anim.Frames,
        delays: anim.Delays,
    }
}

func (p *Panda) Update() {
    if len(p.frames) == 0 {
        return
    }

    p.tickCounter++

    // Check if we passed the delay for the CURRENT frame
    targetDelay := p.delays[p.currentFrame]
    
    if p.tickCounter >= targetDelay {
        p.tickCounter = 0
        p.currentFrame++
        
        // Loop back to start
        if p.currentFrame >= len(p.frames) {
            p.currentFrame = 0
        }
    }
}

func (p *Panda) Draw(screen *ebiten.Image) {
    if len(p.frames) == 0 {
        return
    }

    // Get the specific image for this moment
    img := p.frames[p.currentFrame]

    op := &ebiten.DrawImageOptions{}
    op.GeoM.Translate(p.X, p.Y)
    op.GeoM.Scale(4, 4) // Keep the retro pixel look!

    screen.DrawImage(img, op)
}