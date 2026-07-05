package main

import (
	"math"
	"strconv"
)

// Car is the behaviour shared by the player car and CPU cars.
type Car interface {
	Base() *BaseCar
	Update(g *Game, dt float64)
	UpdateCurrentTrackPiece(g *Game)
	IsCPU() bool
}

// BaseCar holds the state common to all cars.
type BaseCar struct {
	self         Car
	Pos          Vec3
	Image        string
	Speed        float64
	Grip         float64
	CarLetter    string
	trackPiece   *TrackPiece
	TyreRotation float64
}

func (b *BaseCar) Base() *BaseCar { return b }

// baseUpdate applies forward motion and updates the current track piece / tyre rotation.
func (b *BaseCar) baseUpdate(g *Game, dt float64) {
	b.Pos.Z -= b.Speed * dt
	b.UpdateCurrentTrackPiece(g)
	b.TyreRotation += dt * b.Speed * 0.75
}

// UpdateCurrentTrackPiece moves the car between track pieces' car lists as it advances.
func (b *BaseCar) UpdateCurrentTrackPiece(g *Game) {
	idx, ok := g.getTrackPieceForZ(b.Pos.Z)
	if !ok {
		return
	}
	newTP := g.track[idx]
	if newTP != b.trackPiece {
		if b.trackPiece != nil {
			b.trackPiece.Cars = removeCar(b.trackPiece.Cars, b.self)
		}
		b.trackPiece = newTP
		newTP.Cars = append(newTP.Cars, b.self)
	}
}

// updateSprite selects the car sprite based on angle, braking and boost state.
func (b *BaseCar) updateSprite(angle int, braking, boost bool) {
	var frame int
	switch {
	case b.Speed == 0:
		frame = 0
	case braking:
		frame = 3
	case boost:
		frame = int(math.Mod(b.TyreRotation, 2)) + 4
	default:
		frame = int(math.Mod(b.TyreRotation, 2)) + 1
	}
	b.Image = "car_" + b.CarLetter + "_" + strconv.Itoa(angle) + "_" + strconv.Itoa(frame)
}

func removeCar(cars []Car, c Car) []Car {
	for i, x := range cars {
		if x == c {
			return append(cars[:i], cars[i+1:]...)
		}
	}
	return cars
}
