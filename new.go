package main

import (
	"encoding/json"
	"fmt"
	"image/color"
	"log"
	"math"
	"math/rand"
	"os"
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
	SaveFileName = "panda_stats.json"
)

// --- Colors ---
var (
	ColBg         = color.RGBA{0x2d, 0x2d, 0x2d, 0xff}
	ColPandaWhite = color.RGBA{0xff, 0xff, 0xff, 0xff}
	ColPandaBlack = color.RGBA{0x10, 0x10, 0x10, 0xff}
	ColAccent     = color.RGBA{0xff, 0x6b, 0x6b, 0xff}
	ColDesk       = color.RGBA{0x8b, 0x5a, 0x2b, 0xff} // Wood color
	ColLaptop     = color.RGBA{0x50, 0x50, 0x60, 0xff} // Grey
)

// --- Enums ---
type GameMode int

const (
	ModeRelax GameMode = iota
	ModeFocus
	ModeFishing
	ModeMusic
	ModeCooking
)

// --- Save Data Structure ---
type GameStats struct {
	TotalPlayTimeSec  int64 `json:"total_play_time"`
	TotalFocusTimeSec int64 `json:"total_focus_time"`
	FishCaught        int   `json:"fish_caught"`
}

// --- Audio System ---
type ChiptuneStream struct {
	tick float64
	freq float64
	vol  float64
	beat int
}

func (s *ChiptuneStream) Read(buf []byte) (int, error) {
	for i := 0; i < len(buf); i += 4 {
		s.tick++
		if int(s.tick)%8000 == 0 {
			notes := []float64{220, 261, 329, 392, 440, 523}
			s.freq = notes[rand.Intn(len(notes))]
			s.beat = 10
		}
		if s.beat > 0 { s.beat-- }
		s.vol = math.Max(0, s.vol-0.0001)

		val := 0.0
		phase := int(s.tick * s.freq * 2 * math.Pi / SampleRate)
		if phase%2 == 0 { val = 0.1 } else { val = -0.1 }

		v := int16(val * 32767)
		buf[i] = byte(v)
		buf[i+1] = byte(v >> 8)
		buf[i+2] = byte(v)
		buf[i+3] = byte(v >> 8)
	}
	return len(buf), nil
}

// --- Sub-Systems ---
type FocusTimer struct {
	Active        bool
	TargetMinutes int
	TimeLeft      time.Duration
	LastTick      time.Time
}

type FishingGame struct {
	BobberY    float64
	IsCasted   bool
	FishHooked bool
	Score      int
}

type CookingGame struct {
	Progress   float64
	KnifeY     float64
}

// --- Main Game State ---
type Game struct {
	Mode      GameMode
	Tick      int
	Stats     GameStats
	LastSave  time.Time

	// Audio
	AudioCtx *audio.Context
	Player   *audio.Player
	Stream   *ChiptuneStream

	// Systems
	Timer   FocusTimer
	Fishing FishingGame
	Cooking CookingGame
}

func NewGame() *Game {
	// Audio Init
	ctx := audio.NewContext(SampleRate)
	stream := &ChiptuneStream{freq: 440}
	player, _ := ctx.NewPlayer(stream)
	player.SetVolume(0.5)

	g := &Game{
		Mode:     ModeRelax,
		AudioCtx: ctx,
		Player:   player,
		Stream:   stream,
		Timer:    FocusTimer{TargetMinutes: 25, TimeLeft: 25 * time.Minute},
		LastSave: time.Now(),
	}
	
	g.LoadStats()
	return g
}

// --- Persistence ---
func (g *Game) LoadStats() {
	file, err := os.ReadFile(SaveFileName)
	if err == nil {
		json.Unmarshal(file, &g.Stats)
	}
}

func (g *Game) SaveStats() {
	data, _ := json.MarshalIndent(g.Stats, "", "  ")
	os.WriteFile(SaveFileName, data, 0644)
}

// --- UPDATE ---
func (g *Game) Update() error {
	g.Tick++

	// Auto-Save every 10 seconds
	if time.Since(g.LastSave) > 10*time.Second {
		g.SaveStats()
		g.LastSave = time.Now()
	}

	// Update Total Play Time (Approximate via ticks)
	if g.Tick%60 == 0 {
		g.Stats.TotalPlayTimeSec++
	}

	// Mode Switching
	if inpututil.IsKeyJustPressed(ebiten.Key1) { g.Mode = ModeRelax; g.Player.Pause() }
	if inpututil.IsKeyJustPressed(ebiten.Key2) { g.Mode = ModeFocus; g.Player.Pause() }
	if inpututil.IsKeyJustPressed(ebiten.Key3) { g.Mode = ModeFishing; g.Player.Pause() }
	if inpututil.IsKeyJustPressed(ebiten.Key4) { g.Mode = ModeMusic; g.Player.Play() }
	if inpututil.IsKeyJustPressed(ebiten.Key5) { g.Mode = ModeCooking; g.Player.Pause() }

	switch g.Mode {
	case ModeFocus:
		g.updateFocus()
	case ModeFishing:
		g.updateFishing()
	case ModeCooking:
		g.updateCooking()
	case ModeMusic:
		// Visualizer logic handled in Draw/AudioStream
	}
	return nil
}

func (g *Game) updateFocus() {
	// 1. Variable Timer Configuration
	if !g.Timer.Active {
		if inpututil.IsKeyJustPressed(ebiten.KeyUp) {
			g.Timer.TargetMinutes += 5
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyDown) {
			g.Timer.TargetMinutes -= 5
			if g.Timer.TargetMinutes < 5 { g.Timer.TargetMinutes = 5 }
		}
		// Reset TimeLeft to display current selection
		g.Timer.TimeLeft = time.Duration(g.Timer.TargetMinutes) * time.Minute
		
		// Start
		if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
			g.Timer.Active = true
			g.Timer.LastTick = time.Now()
		}
	} else {
		// 2. Countdown Logic
		dt := time.Since(g.Timer.LastTick)
		g.Timer.LastTick = time.Now()
		g.Timer.TimeLeft -= dt
		
		// Track Focus Stats
		if g.Tick%60 == 0 {
			g.Stats.TotalFocusTimeSec++
		}

		if g.Timer.TimeLeft <= 0 {
			g.Timer.Active = false
			g.Timer.TimeLeft = 0
		}
	}
}

func (g *Game) updateFishing() {
	if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		if !g.Fishing.IsCasted {
			g.Fishing.IsCasted = true
			g.Fishing.BobberY = 180
		} else if g.Fishing.FishHooked {
			g.Fishing.Score++
			g.Stats.FishCaught++ // Save stat
			g.Fishing.IsCasted = false
			g.Fishing.FishHooked = false
		} else {
			g.Fishing.IsCasted = false
		}
	}
	if g.Fishing.IsCasted && !g.Fishing.FishHooked && rand.Intn(100) < 2 {
		g.Fishing.FishHooked = true
	}
}

func (g *Game) updateCooking() {
	g.Cooking.KnifeY = 110
	if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		g.Cooking.Progress += 5
		g.Cooking.KnifeY = 130
		if g.Cooking.Progress >= 100 { g.Cooking.Progress = 0 }
	}
}

// --- DRAW ---
func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(ColBg)

	switch g.Mode {
	case ModeRelax:
		// Show Stats
		msg := fmt.Sprintf("RELAX MODE\nTotal Play: %dm\nFocus Time: %dm\nFish Caught: %d", 
			g.Stats.TotalPlayTimeSec/60, g.Stats.TotalFocusTimeSec/60, g.Stats.FishCaught)
		ebitenutil.DebugPrint(screen, msg)
		
		y := 140.0 + math.Sin(float64(g.Tick)*0.05)*2
		g.DrawPanda(screen, 160, y, "none")

	case ModeFocus:
		// Center Layout
		cx, cy := float64(ScreenWidth/2), float64(ScreenHeight/2)
		
		// UI
		mins := int(g.Timer.TimeLeft.Minutes())
		secs := int(g.Timer.TimeLeft.Seconds()) % 60
		
		status := "Use UP/DOWN to Set Time"
		if g.Timer.Active { status = "WORKING..." }
		
		msg := fmt.Sprintf("%s\n     %02d:%02d\n(Space to Start)", status, mins, secs)
		ebitenutil.DebugPrintAt(screen, msg, int(cx)-60, 40)

		// Draw Typing Panda
		g.DrawPanda(screen, cx, cy, "typing")

	case ModeFishing:
		ebitenutil.DebugPrint(screen, fmt.Sprintf("FISH: %d", g.Fishing.Score))
		vector.DrawFilledRect(screen, 0, 180, ScreenWidth, 60, color.RGBA{0x4e, 0xcd, 0xc4, 0xff}, false)
		if g.Fishing.IsCasted {
			by := float32(g.Fishing.BobberY)
			if g.Fishing.FishHooked {
				by += float32(math.Sin(float64(g.Tick)*0.5) * 5)
				ebitenutil.DebugPrintAt(screen, "!!!", 180, 120)
			}
			vector.DrawFilledCircle(screen, 200, by, 3, ColAccent, false)
			vector.StrokeLine(screen, 175, 140, 200, by, 1, ColPandaWhite, false)
		}
		g.DrawPanda(screen, 160, 140, "rod")

	case ModeMusic:
		ebitenutil.DebugPrint(screen, "MUSIC")
		h := float32(g.Stream.freq) / 10.0
		vector.DrawFilledRect(screen, 50, 200-h, 20, h, ColAccent, false)
		vector.DrawFilledRect(screen, 250, 200-h, 20, h, ColAccent, false)
		
		y := 140.0
		if g.Stream.beat > 0 { y = 150 }
		g.DrawPanda(screen, 160, y, "headphones")

	case ModeCooking:
		ebitenutil.DebugPrint(screen, "COOKING")
		vector.DrawFilledRect(screen, 50, 20, float32(g.Cooking.Progress)*2, 10, color.RGBA{0x6b, 0x8c, 0x42, 0xff}, false)
		vector.DrawFilledRect(screen, 100, 150, 120, 10, color.RGBA{100,100,100,255}, false)
		vector.DrawFilledRect(screen, 180, float32(g.Cooking.KnifeY), 5, 20, ColPandaWhite, false)
		g.DrawPanda(screen, 160, 140, "chef")
	}
}

// --- Vector Panda Renderer ---
func (g *Game) DrawPanda(screen *ebiten.Image, x, y float64, costume string) {
	px, py := float32(x), float32(y)

	// 1. Ears
	vector.DrawFilledCircle(screen, px-12, py-15, 8, ColPandaBlack, true)
	vector.DrawFilledCircle(screen, px+12, py-15, 8, ColPandaBlack, true)

	// 2. Head
	vector.DrawFilledCircle(screen, px, py, 20, ColPandaWhite, true)

	// 3. Eyes
	vector.DrawFilledCircle(screen, px-8, py-2, 6, ColPandaBlack, true)
	vector.DrawFilledCircle(screen, px+8, py-2, 6, ColPandaBlack, true)
	vector.DrawFilledCircle(screen, px-8, py-3, 2, ColPandaWhite, true)
	vector.DrawFilledCircle(screen, px+8, py-3, 2, ColPandaWhite, true)

	// 4. Nose
	vector.DrawFilledCircle(screen, px, py+5, 3, ColPandaBlack, true)

	// 5. Body
	vector.DrawFilledRect(screen, px-15, py+15, 30, 25, ColPandaWhite, true)
	
	// 6. Arms & Legs (Default)
	if costume != "typing" {
		vector.DrawFilledCircle(screen, px-18, py+20, 7, ColPandaBlack, true) // L Arm
		vector.DrawFilledCircle(screen, px+18, py+20, 7, ColPandaBlack, true) // R Arm
	}
	vector.DrawFilledCircle(screen, px-12, py+40, 7, ColPandaBlack, true) // L Foot
	vector.DrawFilledCircle(screen, px+12, py+40, 7, ColPandaBlack, true) // R Foot

	// --- COSTUME LOGIC ---
	switch costume {
	case "typing":
		// Desk
		vector.DrawFilledRect(screen, px-40, py+25, 80, 20, ColDesk, true)
		// Laptop
		vector.DrawFilledRect(screen, px-15, py+15, 30, 15, ColLaptop, true) // Screen back
		vector.DrawFilledRect(screen, px-15, py+28, 30, 5, color.RGBA{0x30,0x30,0x30,255}, true) // Keyboard base
		
		// Typing Animation: Arms move up/down
		offset := float32(0)
		if g.Timer.Active && g.Tick%10 < 5 { offset = -3 }
		
		// Arms reaching forward
		vector.DrawFilledCircle(screen, px-15, py+30+offset, 6, ColPandaBlack, true)
		vector.DrawFilledCircle(screen, px+15, py+30-offset, 6, ColPandaBlack, true)

	case "headphones":
		vector.StrokeLine(screen, px-20, py, px+20, py, 3, ColAccent, true)
		vector.DrawFilledRect(screen, px-24, py-5, 6, 15, ColPandaBlack, true)
		vector.DrawFilledRect(screen, px+18, py-5, 6, 15, ColPandaBlack, true)
	
	case "chef":
		vector.DrawFilledRect(screen, px-10, py-35, 20, 15, ColPandaWhite, true)
		vector.DrawFilledCircle(screen, px, py-35, 12, ColPandaWhite, true)

	case "rod":
		vector.StrokeLine(screen, px+15, py+20, px+40, py-10, 2, ColDesk, true)
	}
}

func (g *Game) Layout(w, h int) (int, int) {
	return ScreenWidth, ScreenHeight
}

func main() {
	ebiten.SetWindowSize(ScreenWidth*3, ScreenHeight*3)
	ebiten.SetWindowTitle("Panda: Stats & Focus Update")
	if err := ebiten.RunGame(NewGame()); err != nil {
		log.Fatal(err)
	}
}