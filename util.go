package main

import (
	"fmt"
	"math"
)

// Vec3 and Vec2 mirror pygame's Vector3 / Vector2 with plain value semantics.
type Vec3 struct{ X, Y, Z float64 }
type Vec2 struct{ X, Y float64 }

func (a Vec3) Add(b Vec3) Vec3      { return Vec3{a.X + b.X, a.Y + b.Y, a.Z + b.Z} }
func (a Vec3) Sub(b Vec3) Vec3      { return Vec3{a.X - b.X, a.Y - b.Y, a.Z - b.Z} }
func (a Vec3) Scale(s float64) Vec3 { return Vec3{a.X * s, a.Y * s, a.Z * s} }

func (a Vec2) Add(b Vec2) Vec2      { return Vec2{a.X + b.X, a.Y + b.Y} }
func (a Vec2) Scale(s float64) Vec2 { return Vec2{a.X * s, a.Y * s} }

// remap maps old_val from one range to another (unclamped).
func remap(oldVal, oldMin, oldMax, newMin, newMax float64) float64 {
	return (newMax-newMin)*(oldVal-oldMin)/(oldMax-oldMin) + newMin
}

// remapClamp is remap but constrained to the destination range.
func remapClamp(oldVal, oldMin, oldMax, newMin, newMax float64) float64 {
	lower := math.Min(newMin, newMax)
	upper := math.Max(newMin, newMax)
	return math.Min(upper, math.Max(lower, remap(oldVal, oldMin, oldMax, newMin, newMax)))
}

// inverseLerp returns where value falls between a and b, clamped to [0,1].
func inverseLerp(a, b, value float64) float64 {
	if a != b {
		return math.Min(1, math.Max(0, (value-a)/(b-a)))
	}
	return 0
}

func sign(x float64) float64 {
	if x == 0 {
		return 0
	}
	if x < 0 {
		return -1
	}
	return 1
}

// moveTowards moves n towards target by at most speed.
func moveTowards(n, target, speed float64) float64 {
	if n < target {
		return math.Min(n+speed, target)
	}
	return math.Max(n-speed, target)
}

// formatTime returns a time string of the form "minutes:seconds.milliseconds".
func formatTime(seconds float64) string {
	minutes := int(seconds / 60)
	rem := math.Mod(seconds, 60)
	return fmt.Sprintf("%d:%06.3f", minutes, rem)
}
