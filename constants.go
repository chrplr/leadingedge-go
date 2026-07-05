package main

const (
	Width  = 960
	Height = 540

	HalfWidth  = Width / 2
	HalfHeight = Height / 2

	// Rendering options (mirrors the non-performance-mode branch of the Python game)
	ShowScenery      = true
	ShowTrackside    = true
	ShowRumbleStrips = true
	ShowYellowLines  = true
	ViewDistance     = 200

	ClippingPlane     = -0.25
	ClippingPlaneCars = -0.08

	MaxSceneryScaledWidth = Width * 2
	MaxCarScaledWidth     = Width * 1

	// Track geometry
	Spacing                    = 1.0
	TrackW                     = 3000.0
	HalfStripeW                = 25.0
	HalfRumbleStripW           = 250.0
	HalfYellowLineW            = 80.0
	YellowLineDistanceFromEdge = 150.0

	SectionVeryShort = 25
	SectionShort     = 50
	SectionMedium    = 100
	SectionLong      = 200

	LampX      = TrackW/2 + 300
	BillboardX = TrackW/2 + 600

	CameraFollowDistance = 2.0

	// Player car gameplay settings
	LoseGripSpeed          = 50.0
	ZeroGripSpeed          = 100.0
	PlayerAccelerationMax  = 20.0
	PlayerAccelerationMin  = 10.0
	HighAccelThreshold     = 30.0
	CornerOffsetMultiplier = 5.8
	SteeringStrength       = 72.0

	CPUCarMinTargetSpeed = 40.0
	CPUCarMaxTargetSpeed = 65.0

	NumLaps        = 5
	NumCars        = 20
	GridCarSpacing = 0.55

	SkidSoundStartGrip = 0.8

	FixedTimestep = 1.0 / 60.0
)

// RGB colours used for the track polygons.
type RGB struct{ R, G, B uint8 }

var (
	TrackColour      = RGB{35, 96, 198}
	TracksideColour1 = RGB{0, 77, 180}
	TracksideColour2 = RGB{50, 77, 170}
	StripeColour     = RGB{70, 192, 255}
	YellowLineCol    = RGB{0, 161, 88}
	RumbleColour1    = RGB{0, 116, 255}
	RumbleColour2    = RGB{0, 58, 135}
	StartLineColour  = RGB{255, 255, 255}
)
