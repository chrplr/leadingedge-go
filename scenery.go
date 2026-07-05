package main

import "strconv"

// Scenery is a billboard-style sprite positioned beside (or over) the track.
type Scenery struct {
	X               float64
	Image           string
	MinDrawDistance float64
	MaxDrawDistance float64
	Scale           float64
	CollisionZones  [][2]float64
	isStartGantry   bool
}

// GetImage returns the sprite name to draw; the start gantry animates its lights.
func (s *Scenery) GetImage(g *Game) string {
	if s.isStartGantry {
		var index int
		if g.startTimer > 0 {
			index = int(remap(g.startTimer, 4, 0, 0, 4))
		} else if int(g.timer*2)%2 == 0 {
			index = 4
		} else {
			index = 5
		}
		s.Image = "start" + strconv.Itoa(index)
	}
	return s.Image
}

func newStartGantry() *Scenery {
	return &Scenery{
		X: 0, Image: "start0", MinDrawDistance: 1, MaxDrawDistance: ViewDistance,
		Scale: 4, CollisionZones: [][2]float64{{-3000, -2400}, {2400, 3000}},
		isStartGantry: true,
	}
}

func newBillboard(a *Assets, x float64, image string) *Scenery {
	w, _ := a.Size(image)
	halfWidth := w / 2
	scale := 2.0
	return &Scenery{
		X: x, Image: image, MaxDrawDistance: ViewDistance / 2, Scale: scale,
		CollisionZones: [][2]float64{{-halfWidth * scale, halfWidth * scale}},
	}
}

func newLampLeft() *Scenery {
	return &Scenery{
		X: LampX, Image: "left_light", MaxDrawDistance: ViewDistance / 2, Scale: 2,
		CollisionZones: [][2]float64{{350, 1200}},
	}
}

func newLampRight() *Scenery {
	return &Scenery{
		X: -LampX, Image: "right_light", MaxDrawDistance: ViewDistance / 2, Scale: 2,
		CollisionZones: [][2]float64{{-1200, -350}},
	}
}

// genScenery reproduces the Python generate_scenery helper: billboards every
// `interval` pieces, lamps every 30 pieces, nothing otherwise.
func genScenery(a *Assets, trackI int, image string, interval int, lamps bool) []*Scenery {
	if trackI%interval == 0 {
		return []*Scenery{newBillboard(a, BillboardX, image), newBillboard(a, -BillboardX, image)}
	} else if lamps && trackI%30 == 0 {
		return []*Scenery{newLampLeft(), newLampRight()}
	}
	return nil
}
