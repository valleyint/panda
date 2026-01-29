package main

import (
    "log"
    "github.com/hajimehoshi/ebiten/v2"
)

// Screen Constants (Retro 4:3)
const (
    ScreenWidth  = 320
    ScreenHeight = 240
    WindowTitle  = "Panda Focus"
)

func main() {
    // 1. Window Setup
    ebiten.SetWindowSize(ScreenWidth*3, ScreenHeight*3) // 3x Scale for desktop
    ebiten.SetWindowTitle(WindowTitle)
    ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

    // 2. Initialize Game
    game := NewGame()

    // 3. Run Loop
    if err := ebiten.RunGame(game); err != nil {
        log.Fatal(err)
    }
}