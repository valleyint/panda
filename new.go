package main

import (
	"fmt"
	"image/color"
	"log"
	"math"
	"math/rand"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// --- Constants ---
const (
	ScreenWidth  = 320
	ScreenHeight = 240
	SampleRate   = 44100
)

// --- Colors (Retro Palette) ---
var (
	ColBg        = color.RGBA{0x2d, 0x2d, 0x2d, 0xff} // Dark Grey
	ColPandaWhite= color.RGBA{0xff, 0xff, 0xff, 0xff}
	ColPandaBlack= color.RGBA{0x10, 0x10, 0x10, 0xff}
	ColAccent    = color.RGBA{0xff, 0x6b, 0x6b, 0xff} // Red
	ColWater     = color.RGBA{0x4e, 0xcd, 0xc4, 0xff} // Cyan
	ColBamboo    = color.RGBA{0x6b, 0x8c, 0x42, 0xff} // Green
	ColGold      = color.RGBA{0xff, 0xd9, 0x3d, 0xff} // Gold
)

// --- Enums ---
type GameMode int

const (
	ModeRelax GameMode = iota
	ModeFocus
	ModeFishing  // Replaces Eating
	ModeMusic
	ModeCooking  // Replaces Minigame
)

// --- Audio Stream (Chiptune Generator) ---
type ChiptuneStream struct {
	tick    float64
	freq    float64
	vol     float64 // Used for Visual Sync
	beat    int
}

func (s *ChiptuneStream) Read(buf []byte) (int, error) {
	for i := 0; i < len(buf); i += 4 {
		// Procedural Melody: Change frequency every 0.2 seconds
		s.tick++
		if int(s.tick)%8000 == 0 {
			notes := []float64{220, 261, 329, 392, 440, 523} // A minor pentatonic
			s.freq = notes[rand.Intn(len(notes))]
			s.beat = 10 // Trigger visual beat
		}

		// Decay beat for visuals
		if s.beat > 0 { s.beat-- }
		s.vol = math.Max(0, s.vol-0.0001)

		// Square Wave Synthesis
		val := 0.0
		phase := int(s.tick * s.freq * 2 * math.Pi / SampleRate)
		if phase%2 == 0 { val = 0.1 } else { val = -0.1 }

		// Write to buffer (Little Endian Float32)
		v := int16(val * 32767)
		buf[i] = byte(v)
		buf[i+1] = byte(v >> 8)
		buf[i+2] = byte(v)
		buf[i+3] = byte(v >> 8)
	}
	return len(buf), nil
}

// --- Sub-Systems ---
type FishingGame struct {
	BobberY    float64
	IsCasted   bool
	FishHooked bool
	Score      int
	Tension    float64
}

type CookingGame struct {
	Progress  float64 // 0 to 100
	Chopped   int
	IsChopping bool
	KnifeY    float64
}

type FocusTimer struct {
	Active   bool
	Duration time.Duration
	TimeLeft time.Duration
	LastTick time.Time
}

// --- Main Game State ---
type Game struct {
	Mode      GameMode
	Tick      int
	PandaX    float64
	PandaY    float64
	
	// Audio
	AudioCtx *audio.Context
	Player   *audio.Player
	Stream   *ChiptuneStream

	// Systems
	Timer    FocusTimer
	Fishing  FishingGame
	Cooking  CookingGame
}

func NewGame() *Game {
	// Init Audio
	ctx := audio.NewContext(SampleRate)
	stream := &ChiptuneStream{freq: 440}
	player, _ := ctx.NewPlayer(stream)
	player.SetVolume(0.5)
	
	return &Game{
		Mode:     ModeRelax,
		PandaX:   160,
		PandaY:   140,
		AudioCtx: ctx,
		Player:   player,
		Stream:   stream,
		Timer: FocusTimer{Duration: 25 * time.Minute, TimeLeft: 25 * time.Minute},
	}
}

// --- UPDATE ---
func (g *Game) Update() error {
	g.Tick++

	// Mode Switching
	if inpututil.IsKeyJustPressed(ebiten.Key1) { g.Mode = ModeRelax; g.Player.Pause() }
	if inpututil.IsKeyJustPressed(ebiten.Key2) { g.Mode = ModeFocus; g.Player.Pause() }
	if inpututil.IsKeyJustPressed(ebiten.Key3) { g.Mode = ModeFishing; g.Player.Pause() }
	if inpututil.IsKeyJustPressed(ebiten.Key4) { g.Mode = ModeMusic; g.Player.Play() }
	if inpututil.IsKeyJustPressed(ebiten.Key5) { g.Mode = ModeCooking; g.Player.Pause() }

	switch g.Mode {
	case ModeRelax:
		// Gentle breathing animation
		g.PandaY = 140 + math.Sin(float64(g.Tick)*0.05)*2

	case ModeFocus:
		if inpututil.IsKeyJustPressed(ebiten.KeySpace) && !g.Timer.Active {
			g.Timer.Active = true
			g.Timer.LastTick = time.Now()
		}
		if g.Timer.Active {
			dt := time.Since(g.Timer.LastTick)
			g.Timer.LastTick = time.Now()
			g.Timer.TimeLeft -= dt
		}

	case ModeMusic:
		// Panda bounces to the ACTUAL audio envelope we are generating
		if g.Stream.beat > 0 {
			g.PandaY = 150 // Jump down
		} else {
			g.PandaY = 130 + math.Sin(float64(g.Tick)*0.1)*5 // Return up
		}

	case ModeFishing:
		if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
			if !g.Fishing.IsCasted {
				g.Fishing.IsCasted = true
				g.Fishing.BobberY = 180
			} else if g.Fishing.FishHooked {
				g.Fishing.Score++
				g.Fishing.IsCasted = false
				g.Fishing.FishHooked = false
			} else {
				g.Fishing.IsCasted = false // Pulled too early
			}
		}
		// Random bite logic
		if g.Fishing.IsCasted && !g.Fishing.FishHooked {
			if rand.Intn(100) < 2 { g.Fishing.FishHooked = true }
		}

	case ModeCooking:
		g.Cooking.KnifeY = 110
		if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
			g.Cooking.Chopped++
			g.Cooking.Progress += 5
			g.Cooking.KnifeY = 130 // Chop down
			if g.Cooking.Progress >= 100 { g.Cooking.Progress = 0; g.Cooking.Chopped = 0 }
		}
	}

	return nil
}

// --- DRAW ---
func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(ColBg)

	switch g.Mode {
	case ModeRelax:
		ebitenutil.DebugPrint(screen, "RELAX MODE")
		g.DrawPanda(screen, g.PandaX, g.PandaY, "none")

	case ModeFocus:
		mins := int(g.Timer.TimeLeft.Minutes())
		secs := int(g.Timer.TimeLeft.Seconds()) % 60
		ebitenutil.DebugPrint(screen, fmt.Sprintf("FOCUS: %02d:%02d (Space to Start)", mins, secs))
		g.DrawPanda(screen, 40, 200, "glasses") // Study Glasses

	case ModeFishing:
		ebitenutil.DebugPrint(screen, fmt.Sprintf("FISHING SCORE: %d (Space to Cast/Reel)", g.Fishing.Score))
		// Draw Water
		vector.DrawFilledRect(screen, 0, 180, ScreenWidth, 60, ColWater, false)
		// Draw Bobber
		if g.Fishing.IsCasted {
			by := float32(g.Fishing.BobberY)
			if g.Fishing.FishHooked {
				by += float32(math.Sin(float64(g.Tick)*0.5) * 5) // Shake if bitten
				ebitenutil.DebugPrintAt(screen, "!!!", int(g.PandaX)+20, int(g.PandaY)-40)
			}
			vector.DrawFilledCircle(screen, float32(g.PandaX)+40, by, 3, ColAccent, false)
			vector.StrokeLine(screen, float32(g.PandaX)+15, float32(g.PandaY), float32(g.PandaX)+40, by, 1, ColPandaWhite, false)
		}
		g.DrawPanda(screen, g.PandaX, g.PandaY, "rod")

	case ModeMusic:
		ebitenutil.DebugPrint(screen, "MUSIC SYNC (Chiptune Generated Realtime)")
		// Visualizer Bars (Driven by Synth Frequency)
		h := float32(g.Stream.freq) / 10.0
		vector.DrawFilledRect(screen, 50, 200-h, 20, h, ColAccent, false)
		vector.DrawFilledRect(screen, 250, 200-h, 20, h, ColAccent, false)
		g.DrawPanda(screen, g.PandaX, g.PandaY, "headphones")

	case ModeCooking:
		ebitenutil.DebugPrint(screen, "COOKING: Press Space to Chop!")
		// Progress Bar
		vector.DrawFilledRect(screen, 50, 20, float32(g.Cooking.Progress)*2, 10, ColBamboo, false)
		// Table
		vector.DrawFilledRect(screen, 100, 150, 120, 10, color.RGBA{100,100,100,255}, false)
		// Knife
		vector.DrawFilledRect(screen, float32(g.PandaX)+20, float32(g.Cooking.KnifeY), 5, 20, ColPandaWhite, false)
		g.DrawPanda(screen, g.PandaX, g.PandaY, "chef")
	}
}

// --- Better Panda Renderer (Vector Based) ---
func (g *Game) DrawPanda(screen *ebiten.Image, x, y float64, costume string) {
	px, py := float32(x), float32(y)

	// 1. Ears (Black Circles)
	vector.DrawFilledCircle(screen, px-12, py-15, 8, ColPandaBlack, true) // Left
	vector.DrawFilledCircle(screen, px+12, py-15, 8, ColPandaBlack, true) // Right

	// 2. Head (White Circle) - Much rounder than a grid
	vector.DrawFilledCircle(screen, px, py, 20, ColPandaWhite, true)

	// 3. Eyes (Black Patches + White Pupils)
	vector.DrawFilledCircle(screen, px-8, py-2, 6, ColPandaBlack, true)
	vector.DrawFilledCircle(screen, px+8, py-2, 6, ColPandaBlack, true)
	vector.DrawFilledCircle(screen, px-8, py-3, 2, ColPandaWhite, true) // Pupil
	vector.DrawFilledCircle(screen, px+8, py-3, 2, ColPandaWhite, true) // Pupil

	// 4. Nose
	vector.DrawFilledCircle(screen, px, py+5, 3, ColPandaBlack, true)

	// 5. Body (Rounded Rect)
	vector.DrawFilledRect(screen, px-15, py+15, 30, 25, ColPandaWhite, true)
	
	// 6. Arms/Legs
	vector.DrawFilledCircle(screen, px-18, py+20, 7, ColPandaBlack, true) // L Arm
	vector.DrawFilledCircle(screen, px+18, py+20, 7, ColPandaBlack, true) // R Arm
	vector.DrawFilledCircle(screen, px-12, py+40, 7, ColPandaBlack, true) // L Foot
	vector.DrawFilledCircle(screen, px+12, py+40, 7, ColPandaBlack, true) // R Foot

	// --- COSTUMES ---
	switch costume {
	case "headphones":
		// Band
		vector.StrokeLine(screen, px-20, py, px+20, py, 3, ColAccent, true)
		// Cups
		vector.DrawFilledRect(screen, px-24, py-5, 6, 15, ColPandaBlack, true)
		vector.DrawFilledRect(screen, px+18, py-5, 6, 15, ColPandaBlack, true)
	
	case "chef":
		// Hat
		vector.DrawFilledRect(screen, px-10, py-35, 20, 15, ColPandaWhite, true)
		vector.DrawFilledCircle(screen, px, py-35, 12, ColPandaWhite, true)

	case "rod":
		// Fishing Rod Line
		vector.StrokeLine(screen, px+15, py+20, px+40, py-10, 2, color.RGBA{139, 69, 19, 255}, true)
	
	case "glasses":
		vector.StrokeLine(screen, px-12, py-2, px+12, py-2, 1, ColPandaBlack, true)
		vector.DrawFilledCircle(screen, px-8, py-2, 7, color.RGBA{0,0,0,50}, true)
		vector.DrawFilledCircle(screen, px+8, py-2, 7, color.RGBA{0,0,0,50}, true)
	}
}

func (g *Game) Layout(w, h int) (int, int) {
	return ScreenWidth, ScreenHeight
}

func main() {
	ebiten.SetWindowSize(ScreenWidth*3, ScreenHeight*3)
	ebiten.SetWindowTitle("Panda: Ultimate Edition")
	if err := ebiten.RunGame(NewGame()); err != nil {
		log.Fatal(err)
	}
}