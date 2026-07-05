package main

// This file is the glue between the game and the pgzgo harness. It owns the
// embedded images (the //go:embed directive must live in this package, since
// embed can only reach files under the importing package's directory) and adapts
// the game's input helpers onto the harness keyboard/gamepad snapshots.
//
// Sounds and music are handled by the game's own audio.go, which keeps its
// specialised engine/skid sound banks, so they are not embedded here.

import (
	"embed"

	"github.com/Zyko0/go-sdl3/sdl"
	"github.com/chrplr/pgzgo"
)

//go:embed images
var imagesFS embed.FS

// app is the running harness; the input wrappers below read from its per-frame
// keyboard and gamepad snapshots.
var app *pgzgo.App

// Keyboard bindings used by Controls (held and rising-edge).
func keyDown(sc sdl.Scancode) bool        { return app.Keyboard.Held(sc) }
func keyJustPressed(sc sdl.Scancode) bool { return app.Keyboard.Pressed(sc) }

// Gamepad bindings used by Controls.
func padLeft() bool           { return app.Gamepad.Left() }
func padRight() bool          { return app.Gamepad.Right() }
func padAxisX() float64       { return app.Gamepad.AxisX() }
func padButton0() bool        { return app.Gamepad.Button0() }
func padButton1() bool        { return app.Gamepad.Button1() }
func padButton0Pressed() bool { return app.Gamepad.Button0Pressed() }
func padButton1Pressed() bool { return app.Gamepad.Button1Pressed() }
