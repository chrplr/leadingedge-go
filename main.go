package main

import (
	"flag"
	"fmt"
	"math"

	"github.com/chrplr/pgzgo"
)

type State int

const (
	StateTitle State = iota
	StatePlay
	StateGameOver
)

var (
	state           State
	game            *Game
	assets          *Assets
	audio           *Audio
	accumulatedTime float64
	demoResetTimer  float64
	demoStartTimer  float64
)

func update(dt float64) {
	controls := Controls{}

	switch state {
	case StateTitle:
		if controls.buttonPressed(0) {
			state = StatePlay
			game = NewGame(assets, audio, controls, true)
		}
		demoResetTimer -= dt
		demoStartTimer += dt
		if demoResetTimer <= 0 {
			game = NewGame(assets, audio, controls, false)
			demoResetTimer = 60 * 2
			demoStartTimer = 0
		}
	case StatePlay:
		if game.raceComplete {
			state = StateGameOver
		}
	case StateGameOver:
		if controls.buttonPressed(0) {
			if game.playerCar != nil {
				audio.StopEngineSound()
				audio.SkidSound(0)
			}
			state = StateTitle
			game = NewGame(assets, audio, controls, false)
			audio.PlayMusic("title_theme")
		}
	}

	// Fixed-timestep updates; if the frame took long we step multiple times.
	accumulatedTime += dt
	for accumulatedTime >= FixedTimestep {
		accumulatedTime -= FixedTimestep
		game.Update(FixedTimestep)
	}
}

func draw() {
	game.Draw()

	if state == StateTitle {
		if demoResetTimer < 1 || demoStartTimer < 1 {
			value := demoStartTimer
			if demoResetTimer < 1 {
				value = demoResetTimer
			}
			alpha := math.Min(255, 255-value*255)
			if alpha < 0 {
				alpha = 0
			}
			assets.FillRectAlpha(0, 0, Width, Height, RGB{0, 0, 0}, uint8(alpha))
		}

		// '%' is substituted for the controller button image
		text := "PRESS % OR LEFT CONTROL"
		assets.DrawText(text, Width/2, Height-82, true, "font")

		lw, lh := assets.Size("logo")
		assets.Blit("logo", Width/2-lw/2, Height/3-lh/2)
	}
}

func main() {
	selftest := flag.Bool("selftest", false, "step the game headlessly, then exit")
	flag.Parse()

	// Escape ends a race in-game, so it must not quit the application; the
	// gamepad Start button and the window close button quit instead.
	quitOnEscape := false
	a, err := pgzgo.New(pgzgo.Config{
		Title:          "Leading Edge",
		Width:          Width,
		Height:         Height,
		Images:         imagesFS,
		NearestScaling: true, // pixel-art sprites are scaled up per depth
		QuitOnEscape:   &quitOnEscape,
		// Audio is deliberately nil: leadingedge manages its own mixer so it can
		// keep its engine-speed and skid sound banks (see audio.go).
	})
	if err != nil {
		panic(err)
	}
	defer a.Close()

	app = a
	assets = &Assets{Screen: a.Screen}
	audio = NewAudio()
	defer audio.Destroy()

	if *selftest {
		runSelftest()
		return
	}

	state = StateTitle
	game = NewGame(assets, audio, Controls{}, false)
	demoResetTimer = 60 * 2
	demoStartTimer = 0
	audio.PlayMusic("title_theme")

	// The harness runs the loop; it passes the real frame delta as app.Dt, which
	// update turns into fixed-timestep steps.
	a.Loop(
		func(app *pgzgo.App) { update(app.Dt) },
		func(*pgzgo.App) { draw() },
	)
}

// runSelftest drives the game headlessly to exercise the logic and the full
// render path (track polygons, scenery, HUD text, fade overlay) without input.
func runSelftest() {
	state = StateTitle
	demoResetTimer = 1e9 // keep the demo game stable across the run
	demoStartTimer = 1e9
	game = NewGame(assets, audio, Controls{}, false)
	for i := 0; i < 600; i++ {
		update(FixedTimestep)
	}

	// Render a few frames, forcing the fade overlay on, so the drawing helpers
	// (FillPolygon, BlitScaled, DrawText, FillRectAlpha) all run.
	demoResetTimer, demoStartTimer = 0.5, 0.5
	for i := 0; i < 3; i++ {
		a := app.Renderer()
		a.SetDrawColor(0, 0, 0, 255)
		a.Clear()
		draw()
		a.Present()
	}
	fmt.Printf("demo: %d cars, %d track pieces, timer %.2f\n",
		len(game.cars), len(game.track), game.timer)

	player := NewGame(assets, audio, Controls{}, true)
	for i := 0; i < 300; i++ {
		player.Update(FixedTimestep)
	}
	fmt.Printf("player: playerCar %v, startTimer %.2f, timer %.2f, raceComplete %v\n",
		player.playerCar != nil, player.startTimer, player.timer, player.raceComplete)
	fmt.Println("SELFTEST OK")
}
