package main

import (
	"fmt"
	"math"
)

// PlayerCar is the human-controlled car.
type PlayerCar struct {
	BaseCar
	controls Controls

	offsetXChange float64
	resetting     bool

	explodeTimer int
	exploding    bool

	lastCheckpointIdx int
	hasCheckpoint     bool

	lap      int
	lapTime  float64
	raceTime float64

	fastestLap        float64
	hasFastest        bool
	lastLapWasFastest bool
	braking           bool

	grassSoundRepeatTimer float64
	onGrass               bool

	prevPosition int
}

func NewPlayerCar(pos Vec3, controls Controls) *PlayerCar {
	p := &PlayerCar{
		controls:     controls,
		lap:          1,
		prevPosition: NumCars - 1,
	}
	p.BaseCar = BaseCar{Pos: pos, Image: "car_a_0_0", Grip: 1, CarLetter: "a"}
	p.self = p
	return p
}

func (p *PlayerCar) IsCPU() bool { return false }

func (p *PlayerCar) getXInput() float64 { return p.controls.getX() }

func (p *PlayerCar) Update(g *Game, dt float64) {
	if !g.raceComplete {
		p.lapTime += dt
		p.raceTime += dt
	}

	p.grassSoundRepeatTimer -= dt

	g.audio.UpdateEngineSound(p.Speed)

	// Play overtaking sounds when our race position changes
	currentPosition := g.indexOfCar(p.self)
	if currentPosition != p.prevPosition {
		if math.Abs(p.Speed-g.cars[p.prevPosition].Base().Speed) > 4 {
			g.playSound("overtake", 6)
		}
		p.prevPosition = currentPosition
	}

	if p.resetting {
		if p.exploding {
			p.explodeTimer++
			if p.explodeTimer > 31 {
				p.exploding = false
			}
		} else {
			// Reset player to centre of track over about 2 seconds
			p.Pos.X = moveTowards(p.Pos.X, 0, 2000*dt)
			p.resetting = p.Pos.X != 0
		}
	}

	xMove := 0.0
	accel := 0.0

	// track piece the car is on after forward motion - needed by the skid sound logic below
	trackPieceIdx := -1
	trackPieceOk := false

	if !p.resetting {
		p.braking = false

		if !g.raceComplete {
			if p.controls.buttonDown(0) {
				accel = PlayerAccelerationMax
				if p.Speed >= HighAccelThreshold {
					accel = PlayerAccelerationMin
				}
				p.Speed += accel * dt
			} else if p.controls.buttonDown(1) {
				p.braking = true
				p.Speed = math.Max(0, p.Speed-dt*10)
			}
		}

		// Apply drag in a frame-rate independent way
		dragFactor := 0.9975
		if p.onGrass {
			dragFactor -= 0.0025
		}
		p.Speed *= math.Pow(dragFactor, dt/(1.0/60.0))

		// Going round a corner shifts X pos, so failing to steer takes you off the track
		if p.offsetXChange != 0 {
			if p.Speed > LoseGripSpeed && sign(p.getXInput()) == -sign(p.offsetXChange) {
				p.Grip = remapClamp(p.Speed, LoseGripSpeed, ZeroGripSpeed, 1, 0)
			} else {
				p.Grip = 1
			}
			if !g.raceComplete {
				p.Pos.X -= p.offsetXChange * CornerOffsetMultiplier
			}
		} else {
			p.Grip = 1
		}

		// Track piece we were on before forward motion was applied
		previousTrackPieceIdx, _, prevOk := g.getFirstTrackPieceAhead(p.Pos.Z)

		// Apply steering
		if p.Speed > 0 && !g.raceComplete {
			xMove = p.getXInput() * p.Speed * SteeringStrength * p.Grip * dt
			p.Pos.X -= xMove
		}

		// Apply forward motion
		p.baseUpdate(g, dt)

		// Check for collisions with other cars
		for _, other := range g.cars {
			if other == Car(p.self) {
				continue
			}
			ob := other.Base()
			vec := p.Pos.Sub(ob.Pos)
			const collideFrontZ = 0.6
			const collideBackZ = 1.2
			if math.Abs(vec.X) < 260 && vec.Z < collideFrontZ && vec.Z > -collideBackZ {
				midpoint := (p.Pos.Z-ob.Pos.Z)/2 + ob.Pos.Z
				if math.Abs(vec.Z) < 0.2 {
					// Side collision
					p.Pos.X += sign(vec.X) * 50
					ob.Pos.X -= sign(vec.X) * 50
				} else if vec.Z > 0 {
					// Colliding with the back of the car in front
					p.Speed = math.Max(ob.Speed-3, 0)
					ob.Speed = math.Max(ob.Speed, p.Speed+3)
					if cpu, ok := other.(*CPUCar); ok {
						cpu.TargetSpeed = ob.Speed
					}
					p.Pos.Z = midpoint + collideFrontZ*0.6
					ob.Pos.Z = midpoint - collideFrontZ*0.6
					g.playSound("bump", 6)
				} else {
					// Car behind collided with us - get a speed boost
					p.Speed = math.Max(p.Speed, ob.Speed+3)
					ob.Speed = math.Max(p.Speed-3, 0)
					p.Pos.Z = midpoint - collideBackZ*0.6
					ob.Pos.Z = midpoint + collideBackZ*0.6
					g.playSound("bump_behind", 1)
				}
			}
		}

		// Check for scenery collisions, driving on grass and passing a checkpoint
		trackPieceIdx, _, trackPieceOk = g.getFirstTrackPieceAhead(p.Pos.Z)
		if trackPieceOk {
			trackPiece := g.track[trackPieceIdx]

			for _, scenery := range trackPiece.Scenery {
				for _, zone := range scenery.CollisionZones {
					zoneLeft := scenery.X + zone[0]
					zoneRight := scenery.X + zone[1]
					if zoneLeft < p.Pos.X && p.Pos.X < zoneRight {
						p.Speed = 0
						p.resetting = true
						p.exploding = true
						p.explodeTimer = 0
						g.playSound("explosion", 1)
					}
				}
			}

			// Are we on, or have we passed, a checkpoint?
			if prevOk {
				for i := previousTrackPieceIdx; i <= trackPieceIdx; i++ {
					if i < 0 || i >= len(g.track) {
						continue
					}
					if g.track[i].IsStartLine {
						if p.hasCheckpoint && p.lastCheckpointIdx != i {
							p.lap++

							if !p.hasFastest || p.lapTime < p.fastestLap {
								p.fastestLap = p.lapTime
								p.hasFastest = true
								p.lastLapWasFastest = true
								g.playSound("fastlap", 1)
							} else {
								p.lastLapWasFastest = false
							}

							if p.lap == NumLaps {
								g.playSound("final_lap", 1)
							}

							p.lapTime = 0
						}
						p.lastCheckpointIdx = i
						p.hasCheckpoint = true
					}
				}
			}

			// Are we on the grass?
			if math.Abs(p.Pos.X)+100 > trackPiece.Width/2 {
				p.onGrass = true
				if p.grassSoundRepeatTimer <= 0 {
					g.playSound("hit_grass", 1)
					p.grassSoundRepeatTimer = 0.15
				}
				if math.Abs(p.Pos.X) > 6000 {
					p.Speed = 0
					p.resetting = true
				}
			} else {
				p.onGrass = false
			}
		}
	}

	// Depending on grip, turn skid sound on/off or vary volume
	volume := 0.0
	if p.resetting || p.Grip >= SkidSoundStartGrip || p.getXInput() == 0 {
		volume = 0
	} else {
		volume = remapClamp(p.Grip, SkidSoundStartGrip, 0.5, 0, 1)
		if trackPieceOk {
			trackPiece := g.track[trackPieceIdx]
			volume *= remapClamp(math.Abs(trackPiece.OffsetX), 0, 15, 0, 1)
		}
	}
	g.audio.SkidSound(volume)

	// Set sprite
	if p.exploding {
		p.Image = fmt.Sprintf("explode%02d", p.explodeTimer/2)
	} else {
		direction := 0
		if xMove < 0 {
			direction = -1
		} else if xMove > 0 {
			direction = 1
		}
		boost := accel > 0 && p.Speed < HighAccelThreshold && p.Speed > 0
		p.updateSprite(direction, p.braking, boost)
	}
}

// setOffsetXChange records the camera's per-frame X offset change for cornering physics.
func (p *PlayerCar) setOffsetXChange(value float64) { p.offsetXChange = value }
