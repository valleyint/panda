package main

import (
	"fmt"
	"image/color"
	"log"
	"math"
	"math/rand"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// --- Constants ---
const (
	ScreenWidth  = 320
	ScreenHeight = 240
	ScaleFactor  = 3
)

// --- Colors ---
var (
	ColBackground = color.RGBA{0x2b, 0x2b, 0x2b, 0xff} // Dark Grey
	ColPandaWhite = color.RGBA{0xf0, 0xf0, 0xf0, 0xff}
	ColPandaBlack = color.RGBA{0x1a, 0x1a, 0x1a, 0xff}
	ColBamboo     = color.RGBA{0x6b, 0x8c, 0x42, 0xff} // Green
	ColFood       = color.RGBA{0xff, 0x6b, 0x6b, 0xff} // Red/Pink (Apple)
	ColBar        = color.RGBA{0x4e, 0xcd, 0xc4, 0xff} // Cyan for music
)

// --- Enums ---
type GameMode int

const (
	ModeRelax GameMode = iota
	ModeFocus
	ModeEating
	ModeMusic
	ModeMinigame
)

// --- Sub-System Structs ---

type FocusTimer struct {
	Active   bool
	Duration time.Duration
	TimeLeft time.Duration
	LastTick time.Time
}

type FoodItem struct {
	X, Y   float64
	Active bool
}

type MinigameState struct {
	Score      int
	BallX      float64
	BallY      float64
	BallSpeedY float64
	IsGameOver bool
}

// --- Main Game State ---
type Game struct {
	Mode      GameMode
	Tick      int
	PandaX    float64
	PandaY    float64
	AnimFrame int

	// Sub-systems
	Timer    FocusTimer
	Food     FoodItem      // One item at a time for simplicity
	MiniGame MinigameState // State for the catching game
	MusicBars []float32    // For visualizer
}

// --- Procedural Art Grids ---
// 0=Empty, 1=White, 2=Black
var pandaSprite = [][]uint8{
	{0, 2, 0, 0, 0, 2, 0},
	{2, 1, 2, 2, 2, 1, 2},
	{2, 1, 1, 2, 1, 1, 2},
	{0, 2, 2, 2, 2, 2, 0},
	{0, 2, 1, 1, 1, 2, 0},
	{2, 0, 2, 0, 2, 0, 2},
	{2, 0, 2, 0, 2, 0, 2},
}

// --- Initialization ---
func NewGame() *Game {
	g := &Game{
		Mode:   ModeRelax,
		PandaX: 140,
		PandaY: 100,
		Timer: FocusTimer{
			Duration: 25 * time.Minute,
			TimeLeft: 25 * time.Minute,
		},
		MiniGame: MinigameState{
			BallY: -10, // Start off screen
		},
		MusicBars: make([]float32, 10),
	}
	return g
}

// --- Update Loop ---
func (g *Game) Update() error {
	g.Tick++

	// 1. Global Mode Switching
	if inpututil.IsKeyJustPressed(ebiten.Key1) { g.Mode = ModeRelax }
	if inpututil.IsKeyJustPressed(ebiten.Key2) { g.Mode = ModeFocus }
	if inpututil.IsKeyJustPressed(ebiten.Key3) { g.Mode = ModeEating }
	if inpututil.IsKeyJustPressed(ebiten.Key4) { g.Mode = ModeMusic }
	if inpututil.IsKeyJustPressed(ebiten.Key5) { 
		g.Mode = ModeMinigame
		g.resetMinigame()
	}

	// 2. Mode Specific Logic
	switch g.Mode {
	case ModeRelax:
		// Idle bobbing
		if g.Tick%30 == 0 { g.AnimFrame++ }

	case ModeFocus:
		g.updateFocus()

	case ModeEating:
		g.updateEating()

	case ModeMusic:
		g.updateMusic()

	case ModeMinigame:
		g.updateMinigame()
	}

	return nil
}

// --- Logic Implementations ---

func (g *Game) updateFocus() {
	if inpututil.IsKeyJustPressed(ebiten.KeySpace) && !g.Timer.Active {
		g.Timer.Active = true
		g.Timer.LastTick = time.Now()
	}

	if g.Timer.Active {
		now := time.Now()
		dt := now.Sub(g.Timer.LastTick)
		g.Timer.LastTick = now
		g.Timer.TimeLeft -= dt
		if g.Timer.TimeLeft <= 0 {
			g.Timer.Active = false
			g.Timer.TimeLeft = 0
		}
	}
}

func (g *Game) updateEating() {
	// Click to spawn food
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		mx, my := ebiten.CursorPosition()
		g.Food.X = float64(mx)
		g.Food.Y = float64(my)
		g.Food.Active = true
	}

	// Walk towards food
	if g.Food.Active {
		dx := g.Food.X - (g.PandaX + 14) // Center offset
		dy := g.Food.Y - (g.PandaY + 14)
		dist := math.Sqrt(dx*dx + dy*dy)

		if dist > 5 {
			g.PandaX += (dx / dist) * 2
			g.PandaY += (dy / dist) * 2
			if g.Tick%10 == 0 { g.AnimFrame++ } // Walk animation
		} else {
			// Eat it
			g.Food.Active = false
			g.AnimFrame = 0 // Reset
		}
	}
}

func (g *Game) updateMusic() {
	// Simulate Audio Analysis (Randomize bars)
	if g.Tick%5 == 0 {
		for i := range g.MusicBars {
			// Smooth random movement
			target := rand.Float32() * 50
			g.MusicBars[i] += (target - g.MusicBars[i]) * 0.2
		}
	}
	// Dance to the "beat"
	if g.Tick%20 == 0 {
		g.AnimFrame++
	}
}

func (g *Game) updateMinigame() {
	if g.MiniGame.IsGameOver {
		if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
			g.resetMinigame()
		}
		return
	}

	// Player Movement
	if ebiten.IsKeyPressed(ebiten.KeyLeft) { g.PandaX -= 3 }
	if ebiten.IsKeyPressed(ebiten.KeyRight) { g.PandaX += 3 }

	// Clamp Screen
	if g.PandaX < 0 { g.PandaX = 0 }
	if g.PandaX > ScreenWidth-30 { g.PandaX = ScreenWidth - 30 }

	// Ball Logic
	g.MiniGame.BallY += g.MiniGame.BallSpeedY
	
	// Check Collision (Simple AABB)
	pandaRect := struct{x,y,w,h float64}{g.PandaX, 200, 28, 28}
	ballRect := struct{x,y,w,h float64}{g.MiniGame.BallX, g.MiniGame.BallY, 8, 8}

	if g.checkCollision(pandaRect, ballRect) {
		g.MiniGame.Score++
		g.MiniGame.BallY = -10
		g.MiniGame.BallX = float64(rand.Intn(ScreenWidth - 10))
		g.MiniGame.BallSpeedY += 0.2 // Get harder
	}

	// Miss?
	if g.MiniGame.BallY > ScreenHeight {
		g.MiniGame.IsGameOver = true
	}
}

func (g *Game) resetMinigame() {
	g.MiniGame.Score = 0
	g.MiniGame.IsGameOver = false
	g.MiniGame.BallY = -10
	g.MiniGame.BallX = float64(rand.Intn(ScreenWidth - 10))
	g.MiniGame.BallSpeedY = 2.0
	g.PandaX = ScreenWidth / 2
	g.PandaY = 200 // Lock Y for minigame
}

func (g *Game) checkCollision(r1, r2 struct{x,y,w,h float64}) bool {
	return r1.x < r2.x+r2.w &&
		r1.x+r1.w > r2.x &&
		r1.y < r2.y+r2.h &&
		r1.y+r1.h > r2.y
}

// --- Draw Loop ---
func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(ColBackground)

	switch g.Mode {
	case ModeRelax:
		ebitenutil.DebugPrint(screen, "MODE: RELAX\n[1-5] to Switch Modes")
		g.drawPanda(screen, g.PandaX, g.PandaY, false)

	case ModeFocus:
		mins := int(g.Timer.TimeLeft.Minutes())
		secs := int(g.Timer.TimeLeft.Seconds()) % 60
		msg := fmt.Sprintf("FOCUS TIMER\n\n%02d:%02d\n\n(Space to Start)", mins, secs)
		ebitenutil.DebugPrint(screen, msg)
		g.drawPanda(screen, 20, 200, false) // Small panda in corner

	case ModeEating:
		ebitenutil.DebugPrint(screen, "MODE: EATING\n(Click to feed)")
		g.drawPanda(screen, g.PandaX, g.PandaY, false)
		if g.Food.Active {
			// Draw Bamboo (Green Box)
			vector.DrawFilledRect(screen, float32(g.Food.X), float32(g.Food.Y), 8, 20, ColBamboo, false)
		}

	case ModeMusic:
		ebitenutil.DebugPrint(screen, "MODE: MUSIC")
		// Draw Visualizer
		barW := float32(ScreenWidth) / float32(len(g.MusicBars))
		for i, h := range g.MusicBars {
			x := float32(i) * barW
			vector.DrawFilledRect(screen, x, ScreenHeight-h, barW-2, h, ColBar, false)
		}
		// Draw Dancing Panda
		g.drawPanda(screen, ScreenWidth/2-14, ScreenHeight/2, true)

	case ModeMinigame:
		ebitenutil.DebugPrint(screen, fmt.Sprintf("SCORE: %d", g.MiniGame.Score))
		if g.MiniGame.IsGameOver {
			ebitenutil.DebugPrintAt(screen, "GAME OVER\n(Enter to Restart)", 100, 100)
		} else {
			// Draw Ball (Bamboo)
			vector.DrawFilledRect(screen, float32(g.MiniGame.BallX), float32(g.MiniGame.BallY), 8, 8, ColBamboo, false)
			// Draw Panda
			g.drawPanda(screen, g.PandaX, 200, false)
		}
	}
}

// --- Helper: Procedural Panda Renderer ---
func (g *Game) drawPanda(screen *ebiten.Image, x, y float64, dance bool) {
	pixelSize := float32(4)
	
	// Animation Logic
	bounce := float32(0)
	if dance {
		// Squash and stretch effect
		if g.AnimFrame%2 == 0 { pixelSize = 3; y += 4 } // Squash
	} else {
		if g.AnimFrame%2 == 0 { bounce = -2 } // Bob
	}

	for r, row := range pandaSprite {
		for c, val := range row {
			if val == 0 { continue }
			px := float32(x) + (float32(c) * pixelSize)
			py := float32(y) + (float32(r) * pixelSize) + bounce
			
			var col color.Color
			if val == 1 { col = ColPandaWhite }
			if val == 2 { col = ColPandaBlack }
			vector.DrawFilledRect(screen, px, py, pixelSize, pixelSize, col, false)
		}
	}
}

func (g *Game) Layout(w, h int) (int, int) {
	return ScreenWidth, ScreenHeight
}

func main() {
	ebiten.SetWindowSize(ScreenWidth*ScaleFactor, ScreenHeight*ScaleFactor)
	ebiten.SetWindowTitle("Panda Go: All Modes")
	if err := ebiten.RunGame(NewGame()); err != nil {
		log.Fatal(err)
	}
}