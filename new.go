package main

import (
	"encoding/json"
	"fmt"
	"image/color"
	"log"
	"math"
	"math/rand"
	"os"
	"strconv"
	"strings"
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
	SettingsFile = "settings.json"
	StatsFile    = "panda_stats.json"
)

// --- Colors ---
var (
	ColSharkBlue  = color.RGBA{0x6e, 0xa8, 0xfe, 0xff} // Blahaj Blue
	ColSharkBelly = color.RGBA{0xff, 0xff, 0xff, 0xff}
	ColHeart      = color.RGBA{0xff, 0x6b, 0x6b, 0xff} // Red
)

// --- Enums ---
type GameMode int

const (
	ModeDirectory GameMode = iota
	ModeRelax
	ModeFocus
	ModeFishing
	ModeCooking
	ModeSettings
)

// --- Structs ---
type ColorProfile struct {
	Name      string `json:"name"`
	BgHex     string `json:"bg_hex"`
	AccentHex string `json:"accent_hex"`
}

type AppSettings struct {
	ActiveIndex int            `json:"active_profile_index"`
	Profiles    []ColorProfile `json:"profiles"`
}

type GameStats struct {
	TotalPlayTimeSec int64  `json:"total_play_time"`
	TodayPlayTimeSec int64  `json:"today_play_time"`
	LastLoginDate    string `json:"last_login_date"`
	FishCaught       int    `json:"fish_caught"`
}

// --- Main Game State ---
type Game struct {
	Mode     GameMode
	Tick     int
	LastSave time.Time

	Stats    GameStats
	Settings AppSettings
	
	// Active Colors
	BgColor     color.RGBA
	AccentColor color.RGBA

	// Game Systems
	Timer   struct { 
		Active bool
		TargetMinutes int
		TimeLeft time.Duration
		LastTick time.Time
		SharkState int // 0=Hidden, 1=Swimming (Last 10%), 2=Kissing (Done)
		KissTimer  int // How long to show the kiss
	}
	Fishing struct { BobberY float64; IsCasted, FishHooked bool; Score int }
	Cooking struct { Progress, KnifeY float64 }
}

func NewGame() *Game {
	g := &Game{
		Mode: ModeDirectory,
		Timer: struct{Active bool; TargetMinutes int; TimeLeft time.Duration; LastTick time.Time; SharkState int; KissTimer int}{
			TargetMinutes: 25, 
			TimeLeft: 25 * time.Minute,
		},
		LastSave: time.Now(),
	}
	g.LoadData()
	return g
}

// --- IO Logic ---
func (g *Game) LoadData() {
	if d, err := os.ReadFile(StatsFile); err == nil { json.Unmarshal(d, &g.Stats) }
	
	today := time.Now().Format("2006-01-02")
	if g.Stats.LastLoginDate != today {
		g.Stats.TodayPlayTimeSec = 0
		g.Stats.LastLoginDate = today
	}

	if d, err := os.ReadFile(SettingsFile); err == nil {
		json.Unmarshal(d, &g.Settings)
	} else {
		g.Settings = AppSettings{0, []ColorProfile{
			{"Retro Dark", "#2d2d2d", "#ff6b6b"},
			{"Cozy Light", "#fdf6e3", "#2aa198"},
			{"Matrix", "#000000", "#00ff00"},
		}}
		g.SaveSettings()
	}
	g.ApplyProfile()
}

func (g *Game) SaveSettings() { d, _ := json.MarshalIndent(g.Settings, "", " "); os.WriteFile(SettingsFile, d, 0644) }
func (g *Game) SaveStats()    { d, _ := json.MarshalIndent(g.Stats, "", " "); os.WriteFile(StatsFile, d, 0644) }

func (g *Game) ApplyProfile() {
	idx := g.Settings.ActiveIndex
	if idx < 0 || idx >= len(g.Settings.Profiles) { idx = 0 }
	p := g.Settings.Profiles[idx]
	g.BgColor = ParseHex(p.BgHex)
	g.AccentColor = ParseHex(p.AccentHex)
}

func ParseHex(s string) color.RGBA {
	s = strings.TrimPrefix(s, "#")
	if len(s) != 6 { return color.RGBA{0,0,0,255} }
	v, _ := strconv.ParseUint(s, 16, 32)
	return color.RGBA{uint8(v >> 16), uint8((v >> 8) & 0xFF), uint8(v & 0xFF), 255}
}

// --- UPDATE ---
func (g *Game) Update() error {
	g.Tick++
	if time.Since(g.LastSave) > 10*time.Second { g.SaveStats(); g.LastSave = time.Now() }
	if g.Tick%60 == 0 { g.Stats.TotalPlayTimeSec++; g.Stats.TodayPlayTimeSec++ }

	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) { g.Mode = ModeDirectory }

	switch g.Mode {
	case ModeDirectory:
		if inpututil.IsKeyJustPressed(ebiten.Key1) { g.Mode = ModeRelax }
		if inpututil.IsKeyJustPressed(ebiten.Key2) { g.Mode = ModeFocus }
		if inpututil.IsKeyJustPressed(ebiten.Key3) { g.Mode = ModeFishing }
		if inpututil.IsKeyJustPressed(ebiten.Key4) { g.Mode = ModeCooking }
		if inpututil.IsKeyJustPressed(ebiten.KeyS) { g.Mode = ModeSettings }

	case ModeSettings:
		change := false
		if inpututil.IsKeyJustPressed(ebiten.KeyRight) {
			g.Settings.ActiveIndex = (g.Settings.ActiveIndex + 1) % len(g.Settings.Profiles)
			change = true
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyLeft) {
			g.Settings.ActiveIndex--
			if g.Settings.ActiveIndex < 0 { g.Settings.ActiveIndex = len(g.Settings.Profiles) - 1 }
			change = true
		}
		if change { g.ApplyProfile(); g.SaveSettings() }

	case ModeFocus:
		// Kissing Phase (Post-Timer)
		if g.Timer.SharkState == 2 {
			g.Timer.KissTimer--
			if g.Timer.KissTimer <= 0 {
				g.Timer.SharkState = 0 // Reset
				g.Timer.Active = false
			}
			return
		}

		if !g.Timer.Active {
			// Configuration
			if inpututil.IsKeyJustPressed(ebiten.KeyUp) { g.Timer.TargetMinutes += 5 }
			if inpututil.IsKeyJustPressed(ebiten.KeyDown) { 
				g.Timer.TargetMinutes -= 5
				if g.Timer.TargetMinutes < 5 { g.Timer.TargetMinutes = 5 }
			}
			g.Timer.TimeLeft = time.Duration(g.Timer.TargetMinutes) * time.Minute
			if inpututil.IsKeyJustPressed(ebiten.KeySpace) { 
				g.Timer.Active = true
				g.Timer.LastTick = time.Now()
				g.Timer.SharkState = 0
			}
		} else {
			// Timer Running
			g.Timer.TimeLeft -= time.Since(g.Timer.LastTick)
			g.Timer.LastTick = time.Now()
			
			// SHARK CHECK: Last 10%
			totalDur := time.Duration(g.Timer.TargetMinutes) * time.Minute
			ratio := float64(g.Timer.TimeLeft) / float64(totalDur)
			
			if ratio <= 0.10 {
				g.Timer.SharkState = 1 // Swimming
			}

			// Finished?
			if g.Timer.TimeLeft <= 0 {
				g.Timer.TimeLeft = 0
				g.Timer.SharkState = 2 // Kissing
				g.Timer.KissTimer = 180 // 3 Seconds (60fps * 3)
			}
		}

	case ModeFishing:
		if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
			if !g.Fishing.IsCasted { g.Fishing.IsCasted = true; g.Fishing.BobberY = 180
			} else if g.Fishing.FishHooked {
				g.Fishing.Score++; g.Stats.FishCaught++; g.Fishing.IsCasted = false; g.Fishing.FishHooked = false
			} else { g.Fishing.IsCasted = false }
		}
		if g.Fishing.IsCasted && !g.Fishing.FishHooked && rand.Intn(100) < 2 { g.Fishing.FishHooked = true }

	case ModeCooking:
		g.Cooking.KnifeY = 110
		if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
			g.Cooking.Progress += 5; g.Cooking.KnifeY = 130
			if g.Cooking.Progress >= 100 { g.Cooking.Progress = 0 }
		}
	}
	return nil
}

// --- DRAW ---
func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(g.BgColor)

	switch g.Mode {
	case ModeDirectory:
		ebitenutil.DebugPrint(screen, "--- PANDA OS ---\n\n[1] Chill\n[2] Focus Timer\n[3] Gone Fishing\n[4] Chef Mode\n\n[S] Settings")
		g.DrawPanda(screen, 240, 150, "none")
		msg := fmt.Sprintf("STATS:\nToday: %dm\nTotal: %dm", g.Stats.TodayPlayTimeSec/60, g.Stats.TotalPlayTimeSec/60)
		ebitenutil.DebugPrintAt(screen, msg, 10, 180)

	case ModeSettings:
		p := g.Settings.Profiles[g.Settings.ActiveIndex]
		msg := fmt.Sprintf("SETTINGS\n(Left/Right to Switch)\n\nProfile: %s\nBG: %s\nAccent: %s", p.Name, p.BgHex, p.AccentHex)
		ebitenutil.DebugPrint(screen, msg)
		vector.DrawFilledRect(screen, 100, 160, 120, 30, g.AccentColor, false)
		g.DrawPanda(screen, 160, 200, "none")

	case ModeRelax:
		ebitenutil.DebugPrint(screen, "RELAX MODE")
		y := 140.0 + math.Sin(float64(g.Tick)*0.05)*2
		g.DrawPanda(screen, 160, y, "none")

	case ModeFocus:
		cx, cy := float64(ScreenWidth/2), float64(ScreenHeight/2)
		mins := int(g.Timer.TimeLeft.Minutes())
		secs := int(g.Timer.TimeLeft.Seconds()) % 60
		
		status := "SET TIME:"
		if g.Timer.Active { status = "WORKING..." }
		if g.Timer.SharkState == 2 { status = "GREAT JOB!" } // Kiss phase

		msg := fmt.Sprintf("%s\n   %02d:%02d", status, mins, secs)
		ebitenutil.DebugPrintAt(screen, msg, int(cx)-40, 40)
		g.DrawPanda(screen, cx, cy, "typing")

		// SHARK LOGIC
		if g.Timer.SharkState > 0 {
			sharkX := cx + 80 // To the right of panda
			sharkY := cy + math.Sin(float64(g.Tick)*0.08)*5 // Hovering
			
			// Draw Shark
			g.DrawShark(screen, sharkX, sharkY)

			// Draw Kiss Heart
			if g.Timer.SharkState == 2 {
				// Heart travels from Shark to Panda
				progress := 1.0 - (float64(g.Timer.KissTimer) / 180.0) // 0 to 1
				hx := sharkX - (progress * 60) // Move left 60px
				hy := sharkY - 10 - (math.Sin(progress*math.Pi) * 20) // Arc up
				g.DrawHeart(screen, hx, hy)
			}
		}

	case ModeFishing:
		ebitenutil.DebugPrint(screen, fmt.Sprintf("SCORE: %d", g.Fishing.Score))
		vector.DrawFilledRect(screen, 0, 180, ScreenWidth, 60, color.RGBA{0x4e, 0xcd, 0xc4, 0xff}, false)
		if g.Fishing.IsCasted {
			by := float32(g.Fishing.BobberY)
			if g.Fishing.FishHooked {
				by += float32(math.Sin(float64(g.Tick)*0.5) * 5)
				ebitenutil.DebugPrintAt(screen, "!!!", 180, 120)
			}
			vector.DrawFilledCircle(screen, 200, by, 3, g.AccentColor, false)
			vector.StrokeLine(screen, 175, 140, 200, by, 1, color.White, false)
		}
		g.DrawPanda(screen, 160, 140, "rod")

	case ModeCooking:
		ebitenutil.DebugPrint(screen, "CHEF MODE")
		vector.DrawFilledRect(screen, 50, 20, float32(g.Cooking.Progress)*2, 10, color.RGBA{0x6b, 0x8c, 0x42, 0xff}, false)
		vector.DrawFilledRect(screen, 100, 150, 120, 10, color.RGBA{100,100,100,255}, false)
		vector.DrawFilledRect(screen, 180, float32(g.Cooking.KnifeY), 5, 20, color.White, false)
		g.DrawPanda(screen, 160, 140, "chef")
	}
}

// --- Character Renderers ---

func (g *Game) DrawShark(screen *ebiten.Image, x, y float64) {
	px, py := float32(x), float32(y)
	
	// Tail Fin
	vector.DrawFilledCircle(screen, px+25, py, 10, ColSharkBlue, true)
	// Body (Oval-ish)
	vector.DrawFilledRect(screen, px-20, py-12, 40, 24, ColSharkBlue, true)
	// Belly (White)
	vector.DrawFilledRect(screen, px-20, py+2, 40, 10, ColSharkBelly, true)
	// Top Fin
	vector.DrawFilledCircle(screen, px, py-15, 8, ColSharkBlue, true)
	
	// Eyes (Black)
	vector.DrawFilledCircle(screen, px-12, py-4, 2, color.Black, true)
	vector.DrawFilledCircle(screen, px-4, py-4, 2, color.Black, true)
	
	// Mouth (Pinkish line?) - Simple line
	vector.StrokeLine(screen, px-15, py+2, px-5, py+2, 1, color.Black, true)
}

func (g *Game) DrawHeart(screen *ebiten.Image, x, y float64) {
	px, py := float32(x), float32(y)
	// Simple V-shape heart
	vector.DrawFilledCircle(screen, px-3, py, 3, ColHeart, true)
	vector.DrawFilledCircle(screen, px+3, py, 3, ColHeart, true)
	vector.DrawFilledCircle(screen, px, py+4, 3, ColHeart, true)
}

func (g *Game) DrawPanda(screen *ebiten.Image, x, y float64, costume string) {
	px, py := float32(x), float32(y)
	pDark := color.RGBA{20, 20, 20, 255}

	// Ears
	vector.DrawFilledCircle(screen, px-12, py-15, 8, pDark, true)
	vector.DrawFilledCircle(screen, px+12, py-15, 8, pDark, true)
	// Head
	vector.DrawFilledCircle(screen, px, py, 20, color.White, true)
	// Eyes
	vector.DrawFilledCircle(screen, px-8, py-2, 6, pDark, true)
	vector.DrawFilledCircle(screen, px+8, py-2, 6, pDark, true)
	vector.DrawFilledCircle(screen, px-8, py-3, 2, color.White, true)
	vector.DrawFilledCircle(screen, px+8, py-3, 2, color.White, true)
	vector.DrawFilledCircle(screen, px, py+5, 3, pDark, true)
	// Body
	vector.DrawFilledRect(screen, px-15, py+15, 30, 25, color.White, true)
	
	if costume != "typing" {
		vector.DrawFilledCircle(screen, px-18, py+20, 7, pDark, true)
		vector.DrawFilledCircle(screen, px+18, py+20, 7, pDark, true)
	}
	vector.DrawFilledCircle(screen, px-12, py+40, 7, pDark, true)
	vector.DrawFilledCircle(screen, px+12, py+40, 7, pDark, true)

	switch costume {
	case "typing":
		vector.DrawFilledRect(screen, px-40, py+25, 80, 20, color.RGBA{139,69,19,255}, true)
		vector.DrawFilledRect(screen, px-15, py+15, 30, 15, color.RGBA{80,80,90,255}, true)
		offset := float32(0)
		if g.Timer.Active && g.Tick%10 < 5 { offset = -3 }
		vector.DrawFilledCircle(screen, px-15, py+30+offset, 6, pDark, true)
		vector.DrawFilledCircle(screen, px+15, py+30-offset, 6, pDark, true)
	case "chef":
		vector.DrawFilledRect(screen, px-10, py-35, 20, 15, color.White, true)
		vector.DrawFilledCircle(screen, px, py-35, 12, color.White, true)
	case "rod":
		vector.StrokeLine(screen, px+15, py+20, px+40, py-10, 2, color.RGBA{139,69,19,255}, true)
	}
}

func (g *Game) Layout(w, h int) (int, int) { return ScreenWidth, ScreenHeight }

func main() {
	ebiten.SetWindowSize(ScreenWidth*3, ScreenHeight*3)
	ebiten.SetWindowTitle("Panda & Shark")
	if err := ebiten.RunGame(NewGame()); err != nil { log.Fatal(err) }
}