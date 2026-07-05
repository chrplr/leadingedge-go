package main

import (
	"strconv"

	"github.com/chrplr/pgzgo"
)

// Assets is the game's drawing surface. It embeds the pgzgo Screen — so the
// texture cache and the common helpers (Blit, BlitScaled, Size, SetClip, Fill,
// Destroy) are promoted directly — and adds the game-specific pieces the harness
// doesn't opinionate about: RGB-typed fills, filled track polygons, and the
// sprite-font text used across the HUD and menus.
type Assets struct {
	*pgzgo.Screen
}

// Clear fills the whole frame with a solid RGB colour.
func (a *Assets) Clear(c RGB) { a.Fill(c.R, c.G, c.B) }

// FillRectAlpha draws a translucent rectangle (the title-screen fade overlay).
func (a *Assets) FillRectAlpha(x, y, w, h float64, c RGB, alpha uint8) {
	a.Screen.FillRectAlpha(x, y, w, h, c.R, c.G, c.B, alpha)
}

// FillPolygon fills a convex polygon given as screen-space points.
func (a *Assets) FillPolygon(points []Vec2, c RGB) {
	pts := make([][2]float64, len(points))
	for i, p := range points {
		pts[i] = [2]float64{p.X, p.Y}
	}
	a.Screen.FillPolygon(pts, c.R, c.G, c.B)
}

// fonts maps the game's font names to pgzgo sprite fonts. Each glyph is an image
// named "<font>0<codepoint>"; '%' renders the controller-button image; the
// per-font gap reproduces the original letter spacing.
var fonts = map[string]pgzgo.Font{
	"font":      {Space: 30, GapX: -6, Name: fontGlyph("font")},
	"status1b_": {Space: 30, GapX: 0, Name: fontGlyph("status1b_")},
	"status2_":  {Space: 30, GapX: 0, Name: fontGlyph("status2_")},
}

func fontGlyph(font string) func(rune) string {
	return func(r rune) string {
		if r == '%' {
			return "xb_a"
		}
		return font + "0" + strconv.Itoa(int(r))
	}
}

// DrawText draws sprite-font text, optionally centred horizontally on x.
func (a *Assets) DrawText(text string, x, y float64, centre bool, font string) {
	align := pgzgo.AlignLeft
	if centre {
		align = pgzgo.AlignCentre
	}
	a.Screen.DrawText(text, x, y, align, fonts[font])
}
