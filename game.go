package main

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	
	"panda/internal/entity"
	"panda/internal/gamemode"
)

// Define Modes
type GameMode int

const (
	ModeRelax    GameMode = iota
	ModeFocus
	ModeEating
	ModeMusic
	ModeMinigame
)

// Game holds global state and sub-systems
type Game struct {
	CurrentMode GameMode
	Tick        int

	// Entities & Sub-systems
	Panda *entity.Panda
	Focus *gamemode.FocusMode
}

// NewGame initializes the state and loads initial assets
func NewGame() *Game {
	return &Game{
		CurrentMode: ModeRelax,
		Panda:       entity.NewPanda(),       // Loads panda_idle.png
		Focus:       gamemode.NewFocusMode(), // Sets up 25min timer
	}
}

// Update: Logic Loop (60 TPS)
func (g *Game) Update() error {
	g.Tick++

	// --- GLOBAL INPUT (Mode Switching) ---
	// Press 1-5 to switch screens
	if ebiten.IsKeyPressed(ebiten.Key1) { g.CurrentMode = ModeRelax }
	if ebiten.IsKeyPressed(ebiten.Key2) { g.CurrentMode = ModeFocus }
	if ebiten.IsKeyPressed(ebiten.Key3) { g.CurrentMode = ModeEating }
	if ebiten.IsKeyPressed(ebiten.Key4) { g.CurrentMode = ModeMusic }
	if ebiten.IsKeyPressed(ebiten.Key5) { g.CurrentMode = ModeMinigame }

	// --- MODE SPECIFIC LOGIC ---
	switch g.CurrentMode {
	case ModeRelax:
		// In Relax mode, the Panda wanders/animates freely
		if g.Panda != nil {
			g.Panda.Update()
		}

	case ModeFocus:
		// In Focus mode, update the timer
		if g.Focus != nil {
			g.Focus.Update()
		}
		// Optional: Still animate the panda (maybe slower?)
		if g.Panda != nil {
			g.Panda.Update()
		}

	case ModeEating:
		// Placeholder for Eating Logic
	case ModeMusic:
		// Placeholder for Music Logic
	case ModeMinigame:
		// Placeholder for Minigame Logic
	}

	return nil
}

// Draw: Render Loop (VSync)
func (g *Game) Draw(screen *ebiten.Image) {
	// 1. Clear Screen (Retro Dark Grey Background)
	screen.Fill(color.RGBA{0x2b, 0x2b, 0x2b, 0xff})

	// 2. Mode Specific Rendering
	switch g.CurrentMode {
	case ModeRelax:
		// Draw Scene Elements
		ebitenutil.DebugPrint(screen, "MODE: RELAX (1)\n[Panda is chilling]")
		
		if g.Panda != nil {
			g.Panda.Draw(screen)
		}

	case ModeFocus:
		// Draw Focus UI
		ebitenutil.DebugPrint(screen, "MODE: FOCUS (2)")
		
		// Draw Panda in background (optional)
		if g.Panda != nil {
			g.Panda.Draw(screen)
		}
		
		// Draw Timer Overlay
		if g.Focus != nil {
			g.Focus.Draw(screen)
		}

	case ModeEating:
		ebitenutil.DebugPrint(screen, "MODE: EATING (3)\n(Coming Soon)")
	case ModeMusic:
		ebitenutil.DebugPrint(screen, "MODE: MUSIC (4)\n(Coming Soon)")
	case ModeMinigame:
		ebitenutil.DebugPrint(screen, "MODE: MINIGAME (5)\n(Coming Soon)")
	}
}

// Layout: Scaling Strategy
func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	// The game logical resolution is 320x240.
	// Ebiten will automatically scale this up to fit the window.
	return 320, 240
}