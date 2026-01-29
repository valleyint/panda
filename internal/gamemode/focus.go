package gamemode

import (
    "fmt"
    "image/color"
    "time"

    "github.com/hajimehoshi/ebiten/v2"
    "github.com/hajimehoshi/ebiten/v2/text/v2" 
    // Note: If text/v2 is not found, use "github.com/hajimehoshi/ebiten/v2/text"
    // and standard Go font packages. For simplicity, we'll use DebugPrint first.
    "github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

type FocusState int

const (
    FocusIdle    FocusState = iota // Waiting to start
    FocusRunning                   // Timer ticking
    FocusBreak                     // Short break
)

type FocusMode struct {
    State       FocusState
    Duration    time.Duration // Target time (e.g., 25 mins)
    TimeLeft    time.Duration
    LastUpdate  time.Time
}

func NewFocusMode() *FocusMode {
    return &FocusMode{
        State:    FocusIdle,
        Duration: 25 * time.Minute,
        TimeLeft: 25 * time.Minute,
    }
}

func (f *FocusMode) Update() {
    now := time.Now()

    // Handle State Logic
    switch f.State {
    case FocusIdle:
        // Press SPACE to start timer
        if ebiten.IsKeyPressed(ebiten.KeySpace) {
            f.State = FocusRunning
            f.LastUpdate = now
        }

    case FocusRunning:
        // Calculate time passed since last frame
        dt := now.Sub(f.LastUpdate)
        f.LastUpdate = now
        
        f.TimeLeft -= dt
        
        // Timer Finished?
        if f.TimeLeft <= 0 {
            f.State = FocusBreak
            f.TimeLeft = 5 * time.Minute // Set break time
        }
    }
}

func (f *FocusMode) Draw(screen *ebiten.Image) {
    // Simple UI for now
    var status string
    var timeStr string

    // Format Duration: "25:00"
    minutes := int(f.TimeLeft.Minutes())
    seconds := int(f.TimeLeft.Seconds()) % 60
    timeStr = fmt.Sprintf("%02d:%02d", minutes, seconds)

    switch f.State {
    case FocusIdle:
        status = "PRESS SPACE TO FOCUS"
    case FocusRunning:
        status = "FOCUSED..."
    case FocusBreak:
        status = "TAKE A BREAK!"
    }

    // Render (Debug Print for now, we will add fancy fonts later)
    msg := fmt.Sprintf("%s\n\n%s", status, timeStr)
    ebitenutil.DebugPrintAt(screen, msg, 120, 150)
}