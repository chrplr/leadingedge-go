package main

import "math"

// TrackPiece is a cross-section line of the track. A polygon is drawn connecting each
// piece to the next, so "being on" a piece means being between it and the following one.
type TrackPiece struct {
	Scenery           []*Scenery
	OffsetX           float64
	OffsetY           float64
	CPUMaxTargetSpeed float64
	HasCPUMax         bool
	Col               RGB
	Width             float64
	Cars              []Car
	IsStartLine       bool
}

func tp(scenery []*Scenery, offsetX, offsetY float64) *TrackPiece {
	return &TrackPiece{Scenery: scenery, OffsetX: offsetX, OffsetY: offsetY, Col: TrackColour, Width: TrackW}
}

func tpCPU(scenery []*Scenery, offsetX, offsetY, cpuMax float64) *TrackPiece {
	p := tp(scenery, offsetX, offsetY)
	p.CPUMaxTargetSpeed = cpuMax
	p.HasCPUMax = true
	return p
}

func newStartLinePiece() *TrackPiece {
	return &TrackPiece{
		Scenery: []*Scenery{newStartGantry()}, Col: StartLineColour, Width: TrackW, IsStartLine: true,
	}
}

// makeTrack builds the full multi-lap track as a list of pieces.
func makeTrack(a *Assets) []*TrackPiece {
	var track []*TrackPiece

	add := func(n int, f func(i int) *TrackPiece) {
		for i := 0; i < n; i++ {
			track = append(track, f(i))
		}
	}
	gs := func(i int, image string, interval int, lamps bool) []*Scenery {
		return genScenery(a, i, image, interval, lamps)
	}

	for lap := 0; lap < NumLaps+1; lap++ {
		add(15, func(i int) *TrackPiece { return tp(gs(i, "billboard02", 40, true), 0, 0) })

		// Start gantry / checkpoint line
		track = append(track, newStartLinePiece())

		add(SectionShort, func(i int) *TrackPiece { return tp(nil, 0, 0) })

		// Mild right turn followed by short straight
		add(SectionMedium, func(i int) *TrackPiece { return tp(gs(i, "billboard00", 40, true), -4, 0) })
		add(SectionShort, func(i int) *TrackPiece { return tp(gs(i, "billboard01", 40, true), 0, 0) })

		// Slight downward slope into moderate right hand turn
		add(SectionVeryShort, func(i int) *TrackPiece { return tp(gs(i, "billboard00", 40, true), 0, -1) })
		add(SectionVeryShort, func(i int) *TrackPiece { return tp(gs(i, "billboard00", 40, true), 0, -2) })
		add(SectionVeryShort, func(i int) *TrackPiece { return tp(gs(i, "billboard00", 40, true), -2, -1) })
		add(SectionVeryShort, func(i int) *TrackPiece { return tp(gs(i, "billboard03", 40, true), -5, 0) })
		add(SectionMedium, func(i int) *TrackPiece { return tp(gs(i, "billboard03", 40, true), -10, 0) })

		// Short straight
		add(SectionShort, func(i int) *TrackPiece { return tp(gs(i, "billboard00", 40, true), 0, 0) })

		// Medium-sharp turn left, slight upward slope
		add(SectionMedium, func(i int) *TrackPiece { return tp(gs(i, "arrow_left", 10, true), 13, 1) })

		add(SectionMedium, func(i int) *TrackPiece { return tp(gs(i, "billboard02", 40, true), 0, 0) })

		// Small hill
		add(SectionMedium, func(i int) *TrackPiece { return tp(gs(i, "billboard02", 40, true), 0, 2) })

		// Slightly down and to the right
		add(SectionLong, func(i int) *TrackPiece { return tp(gs(i, "billboard01", 40, true), -3, -1) })

		// Crazy downward curve
		add(SectionMedium, func(i int) *TrackPiece { return tp(gs(i, "billboard00", 40, true), 0, -4) })

		// Upward slope
		add(SectionLong, func(i int) *TrackPiece { return tp(gs(i, "billboard03", 40, true), 0, 2) })

		// Turn to left and up, gradually increasing curve
		for j := 1; j < 10; j++ {
			jj := float64(j)
			add(SectionVeryShort, func(i int) *TrackPiece { return tp(gs(i, "billboard00", 40, true), jj, jj) })
		}

		// Downward curve, increasing then decreasing in intensity
		for j := 1; j < 10; j++ {
			jj := float64(j)
			add(SectionVeryShort, func(i int) *TrackPiece { return tp(gs(i, "billboard00", 40, true), 0, -jj) })
		}

		// Straight with chevron billboards at end, CPU cars slow down here
		add(SectionMedium, func(i int) *TrackPiece { return tpCPU(nil, 0, 0, 60) })
		add(SectionShort, func(i int) *TrackPiece { return tpCPU(gs(i, "arrow_right", 10, false), 0, 0, 58) })
		add(SectionShort, func(i int) *TrackPiece { return tpCPU(gs(i, "arrow_right", 10, false), 0, 0, 58) })

		// Sharp turn right, easing off slightly at end
		add(SectionShort, func(i int) *TrackPiece { return tpCPU(gs(i, "arrow_right", 10, false), -15, 0, 55) })
		add(SectionShort, func(i int) *TrackPiece { return tpCPU(gs(i, "arrow_right", 10, false), -13, 0, 57) })
		add(SectionShort, func(i int) *TrackPiece { return tp(gs(i, "billboard00", 40, true), -11, 0) })
		add(SectionShort, func(i int) *TrackPiece { return tp(gs(i, "billboard00", 40, true), -9, 0) })

		// Straight
		add(SectionMedium, func(i int) *TrackPiece { return tp(gs(i, "billboard00", 40, true), 0, 0) })

		// Cosine hills
		add(SectionLong, func(i int) *TrackPiece {
			return tp(gs(i, "billboard00", 40, true), 0, math.Cos(float64(i)/20)*5)
		})

		// Mild upward slope - resets background Y scrolling to roughly match the lap start
		add(SectionLong, func(i int) *TrackPiece { return tp(gs(i, "billboard03", 40, true), 0, 0.25) })

		// Short straight
		add(SectionShort, func(i int) *TrackPiece { return tp(gs(i, "billboard03", 40, true), 0, 0) })
	}

	return track
}
