package main

import "math"

// CPUCar is an AI-controlled car.
type CPUCar struct {
	BaseCar
	Accel            float64
	TargetSpeed      float64
	TargetX          float64
	Steering         float64
	ChangeSpeedTimer float64
}

func NewCPUCar(pos Vec3, accel, speed float64) *CPUCar {
	c := &CPUCar{
		Accel:            PlayerAccelerationMax * accel,
		TargetSpeed:      speed,
		TargetX:          pos.X,
		ChangeSpeedTimer: uniform(2, 4),
	}
	letter := choiceStr([]string{"b", "c", "d", "e"})
	c.BaseCar = BaseCar{Pos: pos, Image: "car_" + letter + "_0_0", Grip: 1, CarLetter: letter}
	c.self = c
	return c
}

func (c *CPUCar) IsCPU() bool { return true }

func (c *CPUCar) Update(g *Game, dt float64) {
	if g.raceComplete && g.playerCar != nil {
		c.TargetSpeed = g.playerCar.Speed
	}

	c.Speed = moveTowards(c.Speed, c.TargetSpeed, c.Accel*dt)
	c.Pos.X = moveTowards(c.Pos.X, c.TargetX, 400*dt)

	c.baseUpdate(g, dt)

	idx, _, ok := g.getFirstTrackPieceAhead(c.Pos.Z)
	valid := ok && idx >= 0 && idx < len(g.track)
	if valid {
		c.Steering = g.track[idx].OffsetX
	}

	// Every few seconds change target speed by a random amount (upwards on average), so slower
	// cars can catch up and we see CPU cars overtaking each other.
	c.ChangeSpeedTimer -= dt
	if c.ChangeSpeedTimer <= 0 && !g.raceComplete {
		c.TargetSpeed += uniform(-4, 6)
		c.TargetSpeed = min(max(c.TargetSpeed, CPUCarMinTargetSpeed), CPUCarMaxTargetSpeed)

		// On a sharp corner above a certain speed, reduce target speed
		if valid && g.track[idx].HasCPUMax {
			override := g.track[idx].CPUMaxTargetSpeed
			if c.TargetSpeed > override {
				c.TargetSpeed = uniform(override-3, override)
			}
		}

		// Change target X to a random value, avoiding values too close to nearby cars
		tooClose := func() bool {
			for _, other := range g.cars {
				if other != Car(c) && math.Abs(c.Pos.Z-other.Base().Pos.Z) < 20 && math.Abs(c.TargetX-other.Base().Pos.X) < 300 {
					return true
				}
			}
			return false
		}
		for attempt := 0; attempt < 20; attempt++ {
			c.TargetX = uniform(-1000, 1000)
			if !tooClose() {
				break
			}
		}

		c.ChangeSpeedTimer = uniform(2, 4)
	}
}
