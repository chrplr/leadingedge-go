package main

import (
	"fmt"
	"math"
	"sort"
)

type Game struct {
	track []*TrackPiece

	playerCar       *PlayerCar
	cameraFollowCar Car
	cars            []Car

	camera   Vec3
	bgOffset Vec2
	bgWidth  float64

	firstFrame   bool
	frameCounter int
	timer        float64
	raceComplete bool
	timeUp       bool
	startTimer   float64

	assets *Assets
	audio  *Audio
}

// NewGame creates a game. If hasPlayer is false the game runs as a title-screen demo.
func NewGame(assets *Assets, audio *Audio, controls Controls, hasPlayer bool) *Game {
	g := &Game{
		assets:     assets,
		audio:      audio,
		firstFrame: true,
	}
	g.track = makeTrack(assets)
	g.setupCars(controls, hasPlayer)

	g.camera = Vec3{X: 0, Y: 400, Z: 0}
	g.bgWidth, _ = assets.Size("background")
	g.bgOffset = Vec2{X: -g.bgWidth / 2, Y: 30}

	if g.playerCar != nil {
		g.startTimer = 3.999
		audio.PlayMusic("engines_startline")
	} else {
		g.startTimer = 0
	}
	return g
}

func (g *Game) setupCars(controls Controls, hasPlayer bool) {
	g.cars = nil
	for i := 0; i < NumCars; i++ {
		z := -3 - float64(i)*GridCarSpacing
		x := -400.0
		if i%2 != 0 {
			x = 400
		}
		if i == 0 && hasPlayer {
			g.playerCar = NewPlayerCar(Vec3{X: x, Y: 0, Z: z}, controls)
			g.cars = append(g.cars, g.playerCar)
		} else {
			targetSpeed := remap(float64(i), 0, NumCars-1, CPUCarMinTargetSpeed, CPUCarMaxTargetSpeed)
			accel := remap(float64(i), 0, NumCars-1, 1.5, 2)
			g.cars = append(g.cars, NewCPUCar(Vec3{X: x, Y: 0, Z: z}, accel, targetSpeed))
		}
	}
	if g.playerCar != nil {
		g.cameraFollowCar = g.playerCar
	} else {
		g.cameraFollowCar = g.cars[0]
	}
}

func (g *Game) playSound(name string, count int) {
	g.audio.PlaySound(name, count)
}

// getTrackPieceForZ returns the index of the track piece occupying z (or ok=false past the end).
func (g *Game) getTrackPieceForZ(z float64) (int, bool) {
	idx := -int(z / Spacing) // truncates toward zero, like Python int()
	if idx < 0 || idx >= len(g.track) {
		return idx, false
	}
	return idx, true
}

// getFirstTrackPieceAhead returns the index and z of the first track piece at or ahead of z.
func (g *Game) getFirstTrackPieceAhead(z float64) (int, float64, bool) {
	idx := -int(math.Floor(z / Spacing))
	firstPieceZ := -float64(idx) * Spacing
	if idx >= len(g.track) {
		return idx, firstPieceZ, false
	}
	return idx, firstPieceZ, true
}

func (g *Game) indexOfCar(c Car) int {
	for i, x := range g.cars {
		if x == c {
			return i
		}
	}
	return -1
}

func (g *Game) Update(dt float64) {
	g.timer += dt
	g.frameCounter++

	// Race start sequence
	if g.startTimer > 0 {
		// Keep cars registered on their track pieces so they're drawn during the countdown
		for _, car := range g.cars {
			car.UpdateCurrentTrackPiece(g)
		}
		timerOld := g.startTimer
		g.startTimer = math.Max(0, g.startTimer-dt)
		if g.startTimer == 0 {
			g.audio.PlayMusic("ambience")
			g.playSound("gobeep", 1)
		} else if int(timerOld) != int(g.startTimer) {
			g.playSound("startbeep", 1)
		}
	}

	oldCameraZ := g.camera.Z
	prevAhead, _, prevOk := g.getFirstTrackPieceAhead(oldCameraZ)

	if g.startTimer == 0 {
		for _, car := range g.cars {
			car.Update(g, dt)
		}
	}

	// Is the race complete?
	if !g.raceComplete && g.playerCar != nil {
		if g.playerCar.lapTime >= 60*4 || escapeDown() {
			g.audio.StopMusic()
			g.timeUp = true
			g.raceComplete = true
		} else if g.playerCar.lap > NumLaps {
			g.audio.StopMusic()
			g.raceComplete = true
			g.playSound("game_complete", 1)
		}

		// Sort cars by race position (lower z = further ahead = leading)
		sort.SliceStable(g.cars, func(i, j int) bool {
			return g.cars[i].Base().Pos.Z < g.cars[j].Base().Pos.Z
		})
	}

	// Update camera to follow the target car
	g.camera.X = g.cameraFollowCar.Base().Pos.X
	g.camera.Z = g.cameraFollowCar.Base().Pos.Z + CameraFollowDistance

	newCameraZ := g.camera.Z
	newAhead, _, newOk := g.getFirstTrackPieceAhead(newCameraZ)

	distance := oldCameraZ - newCameraZ
	offsetChange := Vec2{}
	if distance > 0 && !g.firstFrame && prevOk && newOk && prevAhead >= 0 && newAhead >= 0 {
		oldZNextBoundary := math.Floor(oldCameraZ/Spacing) * Spacing
		newZPrevBoundary := math.Floor(newCameraZ/Spacing)*Spacing + Spacing
		prevTrack := g.track[prevAhead]
		newTrack := g.track[newAhead]
		if newAhead > prevAhead {
			distanceFirst := oldCameraZ - oldZNextBoundary
			distanceLast := newZPrevBoundary - newCameraZ
			fractionFirst := distanceFirst / Spacing
			fractionLast := distanceLast / Spacing
			offsetChange = Vec2{X: prevTrack.OffsetX, Y: prevTrack.OffsetY}.Scale(fractionFirst).
				Add(Vec2{X: newTrack.OffsetX, Y: newTrack.OffsetY}.Scale(fractionLast))
			if newAhead-prevAhead > 1 {
				for i := prevAhead + 1; i < newAhead; i++ {
					piece := g.track[i]
					offsetChange = offsetChange.Add(Vec2{X: piece.OffsetX, Y: piece.OffsetY})
				}
			}
		} else {
			fraction := distance / Spacing
			offsetChange = Vec2{X: prevTrack.OffsetX, Y: prevTrack.OffsetY}.Scale(fraction)
		}

		g.bgOffset = g.bgOffset.Add(offsetChange)
		for g.bgOffset.X < -g.bgWidth {
			g.bgOffset.X += g.bgWidth
		}
		for g.bgOffset.X > g.bgWidth {
			g.bgOffset.X -= g.bgWidth
		}
	}

	// Shift player car X offset so corners require steering
	if g.playerCar != nil {
		g.playerCar.setOffsetXChange(offsetChange.X)
	}

	// Move background when the camera moves backwards (debug only)
	if prevOk && newOk && newAhead < prevAhead {
		g.bgOffset.X -= g.track[prevAhead].OffsetX
		g.bgOffset.Y -= g.track[prevAhead].OffsetY
	}

	g.firstFrame = false
}

func (g *Game) Draw() {
	// Background colour depends on the vertical scroll of the background image
	if g.bgOffset.Y > 0 {
		g.assets.Clear(RGB{0, 20, 117})
	} else {
		g.assets.Clear(RGB{0, 77, 180})
	}

	// Draw background, with wrap copies as needed
	g.assets.Blit("background", g.bgOffset.X, g.bgOffset.Y)
	if g.bgOffset.X > 0 {
		g.assets.Blit("background", g.bgOffset.X-g.bgWidth, g.bgOffset.Y)
	}
	if g.bgOffset.X+g.bgWidth < Width {
		g.assets.Blit("background", g.bgOffset.X+g.bgWidth, g.bgOffset.Y)
	}

	transform := func(p Vec3, clipping float64) (Vec2, bool) {
		np := p.Sub(g.camera)
		if np.Z > clipping {
			return Vec2{}, false
		}
		return Vec2{X: np.X/np.Z + HalfWidth, Y: np.Y/np.Z + HalfHeight}, true
	}
	transformWH := func(p Vec3, w, h, clipping float64) (Vec2, float64, float64, bool) {
		np := p.Sub(g.camera)
		if np.Z > clipping {
			return Vec2{}, 0, 0, false
		}
		pos := Vec2{X: np.X/np.Z + HalfWidth, Y: np.Y/np.Z + HalfHeight}
		return pos, w / -np.Z, h / -np.Z, true
	}

	var drawList []func()
	add := func(fn func()) { drawList = append(drawList, fn) }

	offset := Vec3{}
	offsetDelta := Vec3{}

	var prevTrackL, prevTrackR Vec2
	var prevStripeL, prevStripeR Vec2
	var prevRumbleLOuter, prevRumbleROuter Vec2
	var prevYLLOuter, prevYLLInner, prevYLROuter, prevYLRInner Vec2
	hasPrev := false

	isFirst := true
	firstIdx, currentPieceZ, _ := g.getFirstTrackPieceAhead(g.camera.Z)
	if firstIdx < 0 { // camera before the track start (shouldn't happen in normal play)
		firstIdx = 0
		currentPieceZ = 0
	}
	trackAheadI := 0
	currentPieceZ += Spacing

	for i := firstIdx; i < len(g.track); i++ {
		trackAheadI++
		if trackAheadI > ViewDistance {
			break
		}
		tp := g.track[i]
		currentPieceZ -= Spacing

		left := Vec3{X: tp.Width / 2, Y: 0, Z: currentPieceZ}
		right := Vec3{X: -tp.Width / 2, Y: 0, Z: currentPieceZ}

		if isFirst {
			adjustedCameraZ := g.camera.Z - Spacing
			fraction := inverseLerp(currentPieceZ-Spacing, currentPieceZ, adjustedCameraZ)
			offsetDelta = Vec3{X: fraction * tp.OffsetX, Y: fraction * tp.OffsetY, Z: 0}
		} else {
			offsetDelta = offsetDelta.Add(Vec3{X: tp.OffsetX, Y: tp.OffsetY, Z: 0})
		}
		isFirst = false

		offset = offset.Add(offsetDelta)
		left = left.Add(offset)
		right = right.Add(offset)

		leftS, leftOk := transform(left, ClippingPlane)
		rightS, rightOk := transform(right, ClippingPlane)

		stripeLS, _ := transform(Vec3{X: HalfStripeW, Y: 0, Z: currentPieceZ}.Add(offset), ClippingPlane)
		stripeRS, _ := transform(Vec3{X: -HalfStripeW, Y: 0, Z: currentPieceZ}.Add(offset), ClippingPlane)

		rumbleLOuterS, _ := transform(left.Add(Vec3{X: HalfRumbleStripW}), ClippingPlane)
		rumbleROuterS, _ := transform(right.Sub(Vec3{X: HalfRumbleStripW}), ClippingPlane)

		ylLOuter := left.Sub(Vec3{X: YellowLineDistanceFromEdge})
		ylLInner := ylLOuter.Sub(Vec3{X: HalfYellowLineW})
		ylROuter := right.Add(Vec3{X: YellowLineDistanceFromEdge})
		ylRInner := ylROuter.Add(Vec3{X: HalfYellowLineW})
		ylLOuterS, _ := transform(ylLOuter, ClippingPlane)
		ylLInnerS, _ := transform(ylLInner, ClippingPlane)
		ylROuterS, _ := transform(ylROuter, ClippingPlane)
		ylRInnerS, _ := transform(ylRInner, ClippingPlane)

		if leftOk && rightOk {
			if hasPrev {
				drawPoints := func(pts []Vec2, col RGB) {
					onScreen := false
					for _, pt := range pts {
						if pt.Y < Height {
							onScreen = true
							break
						}
					}
					if onScreen {
						p := append([]Vec2(nil), pts...)
						c := col
						add(func() { g.assets.FillPolygon(p, c) })
					}
				}

				// Central stripe (3m on / 3m off)
				if (i/3)%2 == 0 {
					drawPoints([]Vec2{stripeLS, stripeRS, prevStripeR, prevStripeL}, StripeColour)
				}

				// Yellow lines (drawn before track so they sit on top after reversal)
				if ShowYellowLines {
					drawPoints([]Vec2{prevYLLOuter, ylLOuterS, ylLInnerS, prevYLLInner}, YellowLineCol)
					drawPoints([]Vec2{prevYLROuter, ylROuterS, ylRInnerS, prevYLRInner}, YellowLineCol)
				}

				// Track surface
				drawPoints([]Vec2{prevTrackL, leftS, rightS, prevTrackR}, tp.Col)

				// Rumble strips
				if ShowRumbleStrips {
					rumbleCol := RumbleColour1
					if (i/2)%2 != 0 {
						rumbleCol = RumbleColour2
					}
					drawPoints([]Vec2{prevRumbleLOuter, prevTrackL, leftS, rumbleLOuterS}, rumbleCol)
					drawPoints([]Vec2{prevRumbleROuter, prevTrackR, rightS, rumbleROuterS}, rumbleCol)
				}

				// Trackside
				if ShowTrackside {
					tsCol := TracksideColour1
					if (i/5)%2 != 0 {
						tsCol = TracksideColour2
					}
					tsLeft := []Vec2{rightS, prevTrackR, {X: 0, Y: prevTrackR.Y}, {X: 0, Y: rightS.Y}}
					tsRight := []Vec2{prevTrackL, leftS, {X: Width - 1, Y: leftS.Y}, {X: Width - 1, Y: prevTrackL.Y}}
					drawPoints(tsLeft, tsCol)
					drawPoints(tsRight, tsCol)
				}
			}

			prevTrackL, prevTrackR = leftS, rightS
			prevStripeL, prevStripeR = stripeLS, stripeRS
			prevRumbleLOuter, prevRumbleROuter = rumbleLOuterS, rumbleROuterS
			prevYLLOuter, prevYLLInner = ylLOuterS, ylLInnerS
			prevYLROuter, prevYLRInner = ylROuterS, ylRInnerS
			hasPrev = true
		}

		// Scenery
		if ShowScenery {
			for _, obj := range tp.Scenery {
				if float64(trackAheadI)*Spacing < obj.MaxDrawDistance {
					posV3 := Vec3{X: obj.X, Y: 0, Z: currentPieceZ}.Add(offset)
					if g.camera.Z-currentPieceZ > obj.MinDrawDistance {
						billboard := obj.GetImage(g)
						bw, bh := g.assets.Size(billboard)
						pos, sw, sh, ok := transformWH(posV3, bw*obj.Scale, bh*obj.Scale, ClippingPlane)
						if ok && sw < MaxSceneryScaledWidth {
							px := pos.X - math.Floor(sw/2)
							py := pos.Y - sh
							name, w2, h2 := billboard, sw, sh
							add(func() { g.assets.BlitScaled(name, px, py, w2, h2) })
						}
					}
				}
			}
		}

		// Cars
		type carDraw struct {
			z  float64
			fn func()
		}
		var carsToDraw []carDraw
		for _, car := range tp.Cars {
			cb := car.Base()
			carOffset := offset
			if math.Mod(cb.Pos.Z, Spacing) != 0 && i+1 < len(g.track) {
				fraction := inverseLerp(currentPieceZ, currentPieceZ-Spacing, cb.Pos.Z)
				nextTP := g.track[i+1]
				carOffset = carOffset.Add(Vec3{X: fraction * nextTP.OffsetX, Y: fraction * nextTP.OffsetY, Z: -fraction * Spacing})
				carOffset = carOffset.Add(offsetDelta.Scale(fraction))
			}
			if car == g.cameraFollowCar {
				carOffset.X = 0
				carOffset.Y = 0
			}
			posV3 := Vec3{X: cb.Pos.X, Y: 0, Z: currentPieceZ}.Add(carOffset)
			scale := 2.0

			if car.IsCPU() {
				cpu := car.(*CPUCar)
				zDistance := math.Max(1, -(posV3.Z - g.camera.Z))
				offsetForAngle := (posV3.X-g.camera.X)/zDistance - cpu.Steering*10
				angleIdx := int(remapClamp(offsetForAngle, -200, 200, -4, 4))
				if car == g.cameraFollowCar {
					angleIdx = min(max(angleIdx, -1), 1)
				}
				cpu.updateSprite(angleIdx, false, false)
			}

			iw, ih := g.assets.Size(cb.Image)
			pos, sw, sh, ok := transformWH(posV3, iw*scale, ih*scale, ClippingPlaneCars)
			if ok && sw < MaxCarScaledWidth {
				px := pos.X - math.Floor(sw/2)
				py := pos.Y - sh
				name, w2, h2 := cb.Image, sw, sh
				carsToDraw = append(carsToDraw, carDraw{z: cb.Pos.Z, fn: func() { g.assets.BlitScaled(name, px, py, w2, h2) }})
			}
		}
		sort.SliceStable(carsToDraw, func(a, b int) bool { return carsToDraw[a].z > carsToDraw[b].z })
		for _, cd := range carsToDraw {
			add(cd.fn)
		}
	}

	// Execute the draw list in reverse so distant items are drawn first
	for k := len(drawList) - 1; k >= 0; k-- {
		drawList[k]()
	}

	g.drawUI()
}

func (g *Game) drawUI() {
	if g.playerCar == nil {
		return
	}
	pc := g.playerCar
	playerPos := g.indexOfCar(pc.self) + 1

	if g.timeUp {
		g.assets.DrawText("TIME UP!", Width/2, Height*0.4, true, "font")
	} else if g.raceComplete {
		g.assets.DrawText("RACE COMPLETE!", Width/2, Height*0.15, true, "font")
		g.assets.DrawText("POSITION", Width/2, Height*0.3, true, "font")
		g.assets.DrawText(fmt.Sprintf("%d", playerPos), Width/2, Height*0.42, true, "font")
		g.assets.DrawText("FASTEST LAP", Width*0.25, Height*0.55, true, "font")
		g.assets.DrawText(formatTime(pc.fastestLap), Width*0.25, Height*0.68, true, "font")
		g.assets.DrawText("RACE TIME", Width*0.75, Height*0.55, true, "font")
		g.assets.DrawText(formatTime(pc.raceTime), Width*0.75, Height*0.68, true, "font")
	} else {
		statusX := Width/2.0 - 565.0/2.0
		g.assets.Blit("status", statusX, 0)
		g.assets.DrawText(fmt.Sprintf("%02d", pc.lap), statusX+30, 37, false, "status1b_")
		g.assets.DrawText(fmt.Sprintf("%02d", playerPos), statusX+116, 37, false, "status1b_")
		g.assets.DrawText(fmt.Sprintf("%03d", int(pc.Speed)), statusX+197, 37, false, "status1b_")
		g.assets.DrawText(formatTime(pc.lapTime), statusX+299, 37, false, "status2_")

		if pc.lastLapWasFastest && pc.lapTime < 4 {
			y := Height * 0.4
			g.assets.DrawText("FASTEST LAP!", Width/2, y, true, "font")
			g.assets.DrawText(formatTime(pc.fastestLap), Width/2, y+60, true, "font")
		}

		beginTime, endTime := 0.0, 4.0
		if pc.lastLapWasFastest {
			beginTime, endTime = 4, 8
		}
		if pc.lap == NumLaps && pc.lapTime > beginTime && pc.lapTime < endTime {
			g.assets.DrawText("FINAL LAP!", Width/2, Height*0.4, true, "font")
		}
	}
}
