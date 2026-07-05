package main

import "github.com/Zyko0/go-sdl3/sdl"

// Keyboard/gamepad snapshotting lives in the pgzgo harness; keyDown,
// keyJustPressed and the pad* helpers are thin wrappers over it (see harness.go).

// Controls models the player's steering axis and the two action buttons.
type Controls struct{}

// gamepadDeadZone matches the Python JoystickControls dead-zone.
const gamepadDeadZone = 0.6

// getX returns the steering axis. The keyboard and d-pad give a digital -1/1;
// the left analogue stick gives a proportional value past the dead-zone (as the
// Python JoystickControls did), for finer steering.
func (c Controls) getX() float64 {
	if keyDown(sdl.SCANCODE_LEFT) {
		return -1
	} else if keyDown(sdl.SCANCODE_RIGHT) {
		return 1
	}
	if padLeft() {
		return -1
	} else if padRight() {
		return 1
	}
	if ax := padAxisX(); ax <= -gamepadDeadZone || ax >= gamepadDeadZone {
		return ax
	}
	return 0
}

// buttonDown reports whether action button 0 (accelerate) or 1 (brake) is held.
// On the gamepad, A (South) accelerates and B (East) brakes.
func (c Controls) buttonDown(button int) bool {
	switch button {
	case 0:
		return keyDown(sdl.SCANCODE_LCTRL) || keyDown(sdl.SCANCODE_Z) || padButton0()
	case 1:
		return keyDown(sdl.SCANCODE_LSHIFT) || keyDown(sdl.SCANCODE_X) || padButton1()
	}
	return false
}

// buttonPressed reports the rising edge of an action button this frame.
func (c Controls) buttonPressed(button int) bool {
	switch button {
	case 0:
		return keyJustPressed(sdl.SCANCODE_LCTRL) || keyJustPressed(sdl.SCANCODE_Z) || padButton0Pressed()
	case 1:
		return keyJustPressed(sdl.SCANCODE_LSHIFT) || keyJustPressed(sdl.SCANCODE_X) || padButton1Pressed()
	}
	return false
}

func escapeDown() bool {
	return keyDown(sdl.SCANCODE_ESCAPE)
}
