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
	TileSize     = 16
)

// --- Colors ---
var (
	ColGopherBlue  = color.RGBA{0x7f, 0xd5, 0xea, 0xff} // Cyan
	ColGopherDark  = color.RGBA{0x00, 0x00, 0x00, 0xff} // Black
	ColGopherSnout = color.RGBA{0xfd, 0xe6, 0x8a, 0xff} // Tan
	ColGopherTooth = color.RGBA{0xff, 0xff, 0xff, 0xff}
	
	ColFishShadow  = color.RGBA{0x00, 0x00, 0x00, 0x50}
	ColMazeWall    = color.RGBA{0x55, 0x55, 0xff, 0xff}
	ColDot         = color.RGBA{0xff, 0xb8, 0xae, 0xff}
	ColHeart       = color.RGBA{0xff, 0x6b, 0x6b, 0xff} // Red

	// Keyboard Colors
	ColDesk        = color.RGBA{0x8b, 0x5a, 0x2b, 0xff} // Wood
	ColKeyBase     = color.RGBA{0x20, 0x20, 0x20, 0xff} // Chassis
	ColKeyRow1     = color.RGBA{0x40, 0x40, 0x40, 0xff} // Dark Keys
	ColKeyRow2     = color.RGBA{0x80, 0x80, 0x80, 0xff} // Light Keys
	ColKeySpace    = color.RGBA{0xAA, 0xAA, 0xAA, 0xff} // Spacebar
)

// --- Enums ---
type GameMode int

const (
	ModeDirectory GameMode = iota
	ModeRelax
	ModeFocus
	ModeFishing
	ModePacman
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
	PacmanWinsToday  int    `json:"pacman_wins_today"`
}

// --- Sub-System States ---

type FishingGame struct {
	State        int
	ActiveSpot   int
	TargetSpot   int
	BobberX      float64
	BobberY      float64
	ReelProgress float64
	FishStrength float64
	Score        int
	WaitTimer    int
}

type PacmanGame struct {
	Map             [15][20]int
	PlayerX, PlayerY int
	GhostX, GhostY   int
	GhostMoveTimer   int
	GhostSpeedDelay  int
	Score            int
	GameOver, Win    bool
}

// --- Main Game State ---
type Game struct {
	Mode     GameMode
	Tick     int
	LastSave time.Time

	Stats    GameStats
	Settings AppSettings
	BgColor, AccentColor color.RGBA

	// Systems
	Timer   struct { 
		Active        bool
		TargetMinutes int
		TimeLeft      time.Duration
		LastTick      time.Time
		GopherState   int
		KissProgress  float64
	}
	Fishing FishingGame
	Pacman  PacmanGame
}

func NewGame() *Game {
	g := &Game{
		Mode: ModeDirectory,
		Timer: struct{Active bool; TargetMinutes int; TimeLeft time.Duration; LastTick time.Time; GopherState int; KissProgress float64}{
			TargetMinutes: 25, 
			TimeLeft: 25 * time.Minute,
		},
		LastSave: time.Now(),
	}
	g.LoadData()
	g.InitPacman()
	return g
}

func (g *Game) InitPacman() {
	layout := [15][20]int{
		{1,1,1,1,1,1,1,1,1,1,1,1,1,1,1,1,1,1,1,1},
		{1,2,2,2,2,2,1,2,2,2,2,2,2,1,2,2,2,2,2,1},
		{1,2,1,1,1,2,1,2,1,1,1,1,2,1,2,1,1,1,2,1},
		{1,2,1,2,2,2,2,2,2,2,2,2,2,2,2,2,2,1,2,1},
		{1,2,1,2,1,1,1,2,1,1,1,1,2,1,1,1,2,1,2,1},
		{1,2,2,2,2,2,2,2,2,0,0,2,2,2,2,2,2,2,2,1},
		{1,2,1,2,1,1,1,2,1,1,1,1,2,1,1,1,2,1,2,1},
		{1,2,1,2,2,2,2,2,2,2,2,2,2,2,2,2,2,1,2,1},
		{1,2,1,1,1,2,1,2,1,1,1,1,2,1,2,1,1,1,2,1},
		{1,2,2,2,2,2,1,2,2,2,2,2,2,1,2,2,2,2,2,1},
		{1,1,1,1,1,1,1,1,1,1,1,1,1,1,1,1,1,1,1,1},
	}
	for y := 0; y < 11; y++ {
		for x := 0; x < 20; x++ { g.Pacman.Map[y][x] = layout[y][x] }
	}
	g.Pacman.PlayerX = 1; g.Pacman.PlayerY = 1
	g.Pacman.GhostX = 10; g.Pacman.GhostY = 5
	g.Pacman.Score = 0; g.Pacman.GameOver = false; g.Pacman.Win = false

	// Difficulty Scaling
	delay := 30 - (g.Stats.PacmanWinsToday * 2)
	if delay < 5 { delay = 5 }
	g.Pacman.GhostSpeedDelay = delay
}

// --- IO Logic ---
func (g *Game) LoadData() {
	if d, err := os.ReadFile(StatsFile); err == nil { json.Unmarshal(d, &g.Stats) }
	today := time.Now().Format("2006-01-02")
	if g.Stats.LastLoginDate != today { 
		g.Stats.TodayPlayTimeSec = 0
		g.Stats.PacmanWinsToday = 0
		g.Stats.LastLoginDate = today 
	}
	if d, err := os.ReadFile(SettingsFile); err == nil { json.Unmarshal(d, &g.Settings) } else {
		g.Settings = AppSettings{0, []ColorProfile{{"Retro", "#2d2d2d", "#ff6b6b"}, {"Light", "#fdf6e3", "#2aa198"}, {"Matrix", "#000000", "#00ff00"}}}
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
	g.BgColor = ParseHex(p.BgHex); g.AccentColor = ParseHex(p.AccentHex)
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
		if inpututil.IsKeyJustPressed(ebiten.Key4) { g.Mode = ModePacman; g.InitPacman() }
		if inpututil.IsKeyJustPressed(ebiten.KeyS) { g.Mode = ModeSettings }

	case ModeSettings:
		change := false
		if inpututil.IsKeyJustPressed(ebiten.KeyRight) { g.Settings.ActiveIndex = (g.Settings.ActiveIndex + 1) % len(g.Settings.Profiles); change = true }
		if inpututil.IsKeyJustPressed(ebiten.KeyLeft) { g.Settings.ActiveIndex--; if g.Settings.ActiveIndex < 0 { g.Settings.ActiveIndex = len(g.Settings.Profiles) - 1 }; change = true }
		if change { g.ApplyProfile(); g.SaveSettings() }

	case ModeFocus:
		g.updateFocus()

	case ModeFishing:
		g.updateFishing()

	case ModePacman:
		g.updatePacman()
	}
	return nil
}

func (g *Game) updateFocus() {
	if g.Timer.GopherState == 2 {
		if g.Timer.KissProgress < 1.0 { g.Timer.KissProgress += 0.01 }
		// Menu
		if inpututil.IsKeyJustPressed(ebiten.Key3) { g.Timer.GopherState = 0; g.Timer.Active = false; g.Mode = ModeFishing }
		if inpututil.IsKeyJustPressed(ebiten.Key4) { g.Timer.GopherState = 0; g.Timer.Active = false; g.Mode = ModePacman; g.InitPacman() }
		if inpututil.IsKeyJustPressed(ebiten.KeySpace) { g.Timer.GopherState = 0; g.Timer.Active = false }
		return
	}

	if !g.Timer.Active {
		if inpututil.IsKeyJustPressed(ebiten.KeyUp) { g.Timer.TargetMinutes += 5 }
		if inpututil.IsKeyJustPressed(ebiten.KeyDown) { g.Timer.TargetMinutes -= 5; if g.Timer.TargetMinutes<5{g.Timer.TargetMinutes=5} }
		g.Timer.TimeLeft = time.Duration(g.Timer.TargetMinutes)*time.Minute
		if inpututil.IsKeyJustPressed(ebiten.KeySpace) { 
			g.Timer.Active = true; g.Timer.LastTick = time.Now()
			g.Timer.GopherState = 0; g.Timer.KissProgress = 0
		}
	} else {
		g.Timer.TimeLeft -= time.Since(g.Timer.LastTick); g.Timer.LastTick = time.Now()
		totalDur := time.Duration(g.Timer.TargetMinutes)*time.Minute
		if float64(g.Timer.TimeLeft)/float64(totalDur) <= 0.10 { g.Timer.GopherState = 1 }
		if g.Timer.TimeLeft <= 0 { g.Timer.TimeLeft=0; g.Timer.GopherState=2 }
	}
}

func (g *Game) updateFishing() {
	g.Fishing.WaitTimer++
	if g.Fishing.WaitTimer > 120 { g.Fishing.WaitTimer = 0; g.Fishing.TargetSpot = rand.Intn(3) + 1 }

	if g.Fishing.State == 0 {
		target := 0
		if inpututil.IsKeyJustPressed(ebiten.KeyA) { target = 1 }
		if inpututil.IsKeyJustPressed(ebiten.KeyS) { target = 2 }
		if inpututil.IsKeyJustPressed(ebiten.KeyD) { target = 3 }
		if target > 0 {
			g.Fishing.ActiveSpot = target; g.Fishing.State = 1; g.Fishing.BobberY = 180
			switch target {
			case 1: g.Fishing.BobberX = 80
			case 2: g.Fishing.BobberX = 160
			case 3: g.Fishing.BobberX = 240
			}
		}
	} else if g.Fishing.State == 1 {
		if g.Fishing.ActiveSpot == g.Fishing.TargetSpot && rand.Intn(100) < 2 {
			g.Fishing.State = 2; g.Fishing.ReelProgress = 30; g.Fishing.FishStrength = 0.5 + rand.Float64()
		}
		if inpututil.IsKeyJustPressed(ebiten.KeySpace) { g.Fishing.State = 0 }
	} else if g.Fishing.State == 2 {
		g.Fishing.ReelProgress -= g.Fishing.FishStrength
		if inpututil.IsKeyJustPressed(ebiten.KeySpace) { g.Fishing.ReelProgress += 8.0 }
		if g.Fishing.ReelProgress >= 100 { g.Fishing.Score++; g.Stats.FishCaught++; g.Fishing.State = 0 }
		if g.Fishing.ReelProgress <= 0 { g.Fishing.State = 0 }
	}
}

func (g *Game) updatePacman() {
	if g.Pacman.GameOver || g.Pacman.Win {
		if inpututil.IsKeyJustPressed(ebiten.KeySpace) { g.InitPacman() }
		return
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowLeft) { g.movePlayer(-1, 0) }
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowRight) { g.movePlayer(1, 0) }
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) { g.movePlayer(0, -1) }
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) { g.movePlayer(0, 1) }

	g.Pacman.GhostMoveTimer++
	if g.Pacman.GhostMoveTimer > g.Pacman.GhostSpeedDelay {
		g.Pacman.GhostMoveTimer = 0
		dx := g.Pacman.PlayerX - g.Pacman.GhostX
		dy := g.Pacman.PlayerY - g.Pacman.GhostY
		mx, my := 0, 0
		if math.Abs(float64(dx)) > math.Abs(float64(dy)) {
			if dx > 0 { mx=1 } else { mx=-1 }
		} else {
			if dy > 0 { my=1 } else { my=-1 }
		}
		if g.Pacman.Map[g.Pacman.GhostY+my][g.Pacman.GhostX+mx] != 1 {
			g.Pacman.GhostX += mx; g.Pacman.GhostY += my
		}
	}
	if g.Pacman.PlayerX == g.Pacman.GhostX && g.Pacman.PlayerY == g.Pacman.GhostY { g.Pacman.GameOver = true }
}

func (g *Game) movePlayer(dx, dy int) {
	nx, ny := g.Pacman.PlayerX + dx, g.Pacman.PlayerY + dy
	if g.Pacman.Map[ny][nx] != 1 {
		g.Pacman.PlayerX = nx; g.Pacman.PlayerY = ny
		if g.Pacman.Map[ny][nx] == 2 {
			g.Pacman.Map[ny][nx] = 0; g.Pacman.Score++
			if g.Pacman.Score >= 80 { g.Pacman.Win = true; g.Stats.PacmanWinsToday++ }
		}
	}
}

// --- DRAW ---
func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(g.BgColor)

	switch g.Mode {
	case ModeDirectory:
		ebitenutil.DebugPrint(screen, "--- PANDA OS ---\n\n[1] Chill\n[2] Focus Timer\n[3] Fishing Spots\n[4] Panda-Man\n\n[S] Settings")
		g.DrawPanda(screen, 240, 150, "none")
		msg := fmt.Sprintf("STATS:\nToday: %dm\nTotal: %dm", g.Stats.TodayPlayTimeSec/60, g.Stats.TotalPlayTimeSec/60)
		ebitenutil.DebugPrintAt(screen, msg, 10, 180)

	case ModeSettings:
		p := g.Settings.Profiles[g.Settings.ActiveIndex]
		ebitenutil.DebugPrint(screen, fmt.Sprintf("SETTINGS\n< %s >", p.Name))
		vector.DrawFilledRect(screen, 100, 160, 120, 30, g.AccentColor, false)
		g.DrawPanda(screen, 160, 200, "none")

	case ModeRelax:
		ebitenutil.DebugPrint(screen, "RELAX")
		g.DrawPanda(screen, 160, 140+math.Sin(float64(g.Tick)*0.05)*2, "none")

	case ModeFocus:
		status := "TIME:"
		if g.Timer.GopherState == 2 { status = "DONE!" }
		ebitenutil.DebugPrintAt(screen, fmt.Sprintf("%s\n%02d:%02d", status, int(g.Timer.TimeLeft.Minutes()), int(g.Timer.TimeLeft.Seconds())%60), 120, 40)
		g.DrawPanda(screen, 160, 120, "typing")
		
		if g.Timer.GopherState > 0 {
			gx := 240.0; gy := 120 + math.Sin(float64(g.Tick)*0.08)*5
			g.DrawGopher(screen, gx, gy)
			if g.Timer.GopherState == 2 {
				progress := g.Timer.KissProgress
				hx := gx - (progress * 60)
				hy := gy - 10 - (math.Sin(progress*math.Pi) * 20)
				g.DrawHeart(screen, hx, hy)
				ebitenutil.DebugPrintAt(screen, "GREAT JOB!", 120, 180)
				ebitenutil.DebugPrintAt(screen, "[3] Fishing  [4] Pacman", 100, 200)
			}
		}

	case ModeFishing:
		ebitenutil.DebugPrint(screen, fmt.Sprintf("FISH: %d", g.Fishing.Score))
		vector.DrawFilledRect(screen, 0, 180, ScreenWidth, 60, color.RGBA{0x4e, 0xcd, 0xc4, 0xff}, false)
		for i, label := range []string{"A", "S", "D"} {
			sx := float32(80 * (i + 1))
			ebitenutil.DebugPrintAt(screen, label, int(sx)-4, 220)
			if g.Fishing.TargetSpot == i+1 { vector.DrawFilledCircle(screen, sx, 200, 10, ColFishShadow, true) }
		}
		if g.Fishing.State > 0 {
			bx, by := float32(g.Fishing.BobberX), float32(g.Fishing.BobberY)
			if g.Fishing.State == 2 { by += float32(math.Sin(float64(g.Tick)*0.8)*5) }
			vector.StrokeLine(screen, 160, 140, bx, by, 1, color.White, false)
			vector.DrawFilledCircle(screen, bx, by, 3, g.AccentColor, false)
			if g.Fishing.State == 2 {
				vector.DrawFilledRect(screen, 110, 120, 100, 10, color.RGBA{50,50,50,255}, false)
				vector.DrawFilledRect(screen, 110, 120, float32(g.Fishing.ReelProgress), 10, g.AccentColor, false)
			}
		}
		g.DrawPanda(screen, 160, 140, "rod")

	case ModePacman:
		for y := 0; y < 15; y++ {
			for x := 0; x < 20; x++ {
				px, py := float32(x*TileSize), float32(y*TileSize)
				if g.Pacman.Map[y][x] == 1 { vector.DrawFilledRect(screen, px, py, TileSize, TileSize, ColMazeWall, false)
				} else if g.Pacman.Map[y][x] == 2 { vector.DrawFilledCircle(screen, px+8, py+8, 2, ColDot, true) }
			}
		}
		ppx, ppy := float64(g.Pacman.PlayerX*TileSize)+8, float64(g.Pacman.PlayerY*TileSize)+8
		g.DrawPandaHead(screen, ppx, ppy, 8)
		gpx, gpy := float64(g.Pacman.GhostX*TileSize)+8, float64(g.Pacman.GhostY*TileSize)+8
		g.DrawGopherHead(screen, gpx, gpy)

		if g.Pacman.GameOver { ebitenutil.DebugPrintAt(screen, "GAME OVER (Space)", 100, 100) }
		if g.Pacman.Win { ebitenutil.DebugPrintAt(screen, "YOU WIN! (Space)", 100, 100) }
	}
}

// --- Artist ---

func (g *Game) DrawGopher(screen *ebiten.Image, x, y float64) {
	px, py := float32(x), float32(y)
	// Body
	vector.DrawFilledCircle(screen, px, py+15, 18, ColGopherBlue, true) // Bot
	vector.DrawFilledCircle(screen, px-5, py-10, 16, ColGopherBlue, true) // Top
	vector.DrawFilledRect(screen, px-20, py-10, 35, 25, ColGopherBlue, true) // Mid
	// Eyes
	vector.DrawFilledCircle(screen, px-12, py-12, 7, color.White, true)
	vector.DrawFilledCircle(screen, px-10, py-12, 2, ColGopherDark, true)
	vector.DrawFilledCircle(screen, px+2, py-12, 7, color.White, true)
	vector.DrawFilledCircle(screen, px+4, py-12, 2, ColGopherDark, true)
	// Snout
	vector.DrawFilledRect(screen, px-10, py-2, 14, 8, ColGopherSnout, true)
	vector.DrawFilledCircle(screen, px-10, py+2, 4, ColGopherSnout, true)
	vector.DrawFilledCircle(screen, px+4, py+2, 4, ColGopherSnout, true)
	vector.DrawFilledCircle(screen, px-3, py-1, 3, ColGopherDark, true)
	// Tooth
	vector.DrawFilledRect(screen, px-5, py+4, 4, 5, ColGopherTooth, true)
	// Ears
	vector.DrawFilledCircle(screen, px-18, py-18, 4, ColGopherBlue, true)
	vector.DrawFilledCircle(screen, px+8, py-20, 4, ColGopherBlue, true)
}

func (g *Game) DrawGopherHead(screen *ebiten.Image, x, y float64) {
	// Scaled down head for Pacman
	px, py := float32(x), float32(y)
	vector.DrawFilledCircle(screen, px, py, 7, ColGopherBlue, true)
	vector.DrawFilledCircle(screen, px-6, py-5, 2, ColGopherBlue, true)
	vector.DrawFilledCircle(screen, px+6, py-5, 2, ColGopherBlue, true)
	vector.DrawFilledCircle(screen, px-3, py-2, 3, color.White, true)
	vector.DrawFilledCircle(screen, px+3, py-2, 3, color.White, true)
	vector.DrawFilledCircle(screen, px-3, py-2, 1, ColGopherDark, true)
	vector.DrawFilledCircle(screen, px+3, py-2, 1, ColGopherDark, true)
	vector.DrawFilledCircle(screen, px, py+2, 3, ColGopherSnout, true)
	vector.DrawFilledCircle(screen, px, py+1, 1, ColGopherDark, true)
	vector.DrawFilledRect(screen, px-1, py+3, 2, 2, ColGopherTooth, true)
}

func (g *Game) DrawHeart(screen *ebiten.Image, x, y float64) {
	px, py := float32(x), float32(y)
	vector.DrawFilledCircle(screen, px-3, py, 3, ColHeart, true)
	vector.DrawFilledCircle(screen, px+3, py, 3, ColHeart, true)
	vector.DrawFilledCircle(screen, px, py+4, 3, ColHeart, true)
}

func (g *Game) DrawPanda(screen *ebiten.Image, x, y float64, costume string) {
	px, py := float32(x), float32(y)
	pDark := color.RGBA{20, 20, 20, 255}
	// Standard Body
	vector.DrawFilledCircle(screen, px-12, py-15, 8, pDark, true) 
	vector.DrawFilledCircle(screen, px+12, py-15, 8, pDark, true)
	vector.DrawFilledCircle(screen, px, py, 20, color.White, true) 
	vector.DrawFilledCircle(screen, px-8, py-2, 6, pDark, true) 
	vector.DrawFilledCircle(screen, px+8, py-2, 6, pDark, true)
	vector.DrawFilledCircle(screen, px-8, py-3, 2, color.White, true)
	vector.DrawFilledCircle(screen, px+8, py-3, 2, color.White, true)
	vector.DrawFilledCircle(screen, px, py+5, 3, pDark, true) 
	vector.DrawFilledRect(screen, px-15, py+15, 30, 25, color.White, true) 

	if costume == "typing" {
		// Desk
		vector.DrawFilledRect(screen, px-40, py+25, 80, 20, ColDesk, true)
		
		// KEYBOARD
		kx, ky := px-25, py+25
		vector.DrawFilledRect(screen, kx, ky, 50, 15, ColKeyBase, true) // Chassis
		// Row 1 (Numbers - Dark)
		for i:=0; i<10; i++ { vector.DrawFilledRect(screen, kx+1+float32(i*5), ky+1, 4, 3, ColKeyRow1, true) }
		// Row 2 (Letters)
		for i:=0; i<9; i++ { vector.DrawFilledRect(screen, kx+3+float32(i*5), ky+5, 4, 3, ColKeyRow2, true) }
		// Row 3 (Home)
		for i:=0; i<9; i++ { vector.DrawFilledRect(screen, kx+3+float32(i*5), ky+9, 4, 3, ColKeyRow2, true) }
		// Spacebar
		vector.DrawFilledRect(screen, kx+15, ky+13, 20, 2, ColKeySpace, true)

		// Hands typing
		offset := float32(0); if g.Timer.Active && g.Tick%10 < 5 { offset = -3 }
		vector.DrawFilledCircle(screen, px-15, py+30+offset, 6, pDark, true)
		vector.DrawFilledCircle(screen, px+15, py+30-offset, 6, pDark, true)

	} else if costume == "rod" {
		vector.StrokeLine(screen, px+15, py+20, px+40, py-10, 2, color.RGBA{139,69,19,255}, true)
		vector.DrawFilledCircle(screen, px-12, py+40, 7, pDark, true)
		vector.DrawFilledCircle(screen, px+12, py+40, 7, pDark, true)
	} else {
		vector.DrawFilledCircle(screen, px-18, py+20, 7, pDark, true)
		vector.DrawFilledCircle(screen, px+18, py+20, 7, pDark, true)
		vector.DrawFilledCircle(screen, px-12, py+40, 7, pDark, true)
		vector.DrawFilledCircle(screen, px+12, py+40, 7, pDark, true)
	}
}

func (g *Game) DrawPandaHead(screen *ebiten.Image, x, y, r float64) {
	px, py := float32(x), float32(y)
	pDark := color.RGBA{20, 20, 20, 255}
	vector.DrawFilledCircle(screen, px-4, py-5, float32(r/2), pDark, true)
	vector.DrawFilledCircle(screen, px+4, py-5, float32(r/2), pDark, true)
	vector.DrawFilledCircle(screen, px, py, float32(r), color.White, true)
	vector.DrawFilledCircle(screen, px-3, py-1, 2, pDark, true)
	vector.DrawFilledCircle(screen, px+3, py-1, 2, pDark, true)
}

func (g *Game) Layout(w, h int) (int, int) { return ScreenWidth, ScreenHeight }

func main() {
	ebiten.SetWindowSize(ScreenWidth*3, ScreenHeight*3)
	ebiten.SetWindowTitle("Panda OS: Final")
	if err := ebiten.RunGame(NewGame()); err != nil { log.Fatal(err) }
}