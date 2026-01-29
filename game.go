package main

import (
    "github.com/hajimehoshi/ebiten/v2"
    "github.com/hajimehoshi/ebiten/v2/ebitenutil"
    "image/color"
)

// Define Modes
type GameMode int

const (
    ModeRelax GameMode = iota
    ModeFocus
    ModeEating
    ModeMusic
    ModeMinigame
)

// Game holds global state
type Game struct {
    CurrentMode GameMode
    Tick        int
    // Placeholder for sub-systems
    // panda *entity.Panda
    // music *gamemode.MusicPlayer
}

func NewGame() *Game {
    return &Game{
        CurrentMode: ModeRelax,
    }
}

// Update: Logic (60 TPS)
func (g *Game) Update() error {
    g.Tick++

    // Input Handling (Global Debug Keys for testing)
    if ebiten.IsKeyPressed(ebiten.Key1) { g.CurrentMode = ModeRelax }
    if ebiten.IsKeyPressed(ebiten.Key2) { g.CurrentMode = ModeFocus }
    if ebiten.IsKeyPressed(ebiten.Key3) { g.CurrentMode = ModeEating }
    if ebiten.IsKeyPressed(ebiten.Key4) { g.CurrentMode = ModeMusic }
    if ebiten.IsKeyPressed(ebiten.Key5) { g.CurrentMode = ModeMinigame }

    // Mode-Specific Logic Router
    switch g.CurrentMode {
    case ModeRelax:
        // g.panda.UpdateRelax()
    case ModeFocus:
        // g.timer.Update()
    case ModeEating:
        // g.inventory.Update()
    case ModeMusic:
        // g.visualizer.Update()
    case ModeMinigame:
        // g.minigameManager.Update()
    }

    return nil
}

// Draw: Rendering (VSync)
func (g *Game) Draw(screen *ebiten.Image) {
    // 1. Clear Screen (Background Color)
    screen.Fill(color.RGBA{0x2b, 0x2b, 0x2b, 0xff}) // Dark Grey

    // 2. Mode-Specific Render Router
    switch g.CurrentMode {
    case ModeRelax:
        ebitenutil.DebugPrint(screen, "MODE: RELAX\n(Panda Wandering...)")
        // g.panda.Draw(screen)
    case ModeFocus:
        ebitenutil.DebugPrint(screen, "MODE: FOCUS\n(Timer: 25:00)")
    case ModeEating:
        ebitenutil.DebugPrint(screen, "MODE: EATING\n(Drag food here)")
    case ModeMusic:
        ebitenutil.DebugPrint(screen, "MODE: MUSIC\n(Visualizer Active)")
    case ModeMinigame:
        ebitenutil.DebugPrint(screen, "MODE: MINIGAME\n(Select Game)")
    }
}

// Layout: Scaling Strategy
func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
    // Always render at 320x240, let Ebiten scale it up
    return ScreenWidth, ScreenHeight
}