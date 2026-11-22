// Package color provides the small amount of color math the UI needs.
package color

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// RGBA is a color as the DualShock 4 lightbar understands it: three channels
// plus an alpha value that is used as brightness (0 = off, 255 = full power).
type RGBA struct {
	R, G, B, A uint8
}

// Palette is an sRGB color without alpha.
type Palette struct {
	R, G, B uint8
}

// New returns an opaque color.
func New(r, g, b uint8) RGBA {
	return RGBA{R: r, G: g, B: b, A: 255}
}

// WithAlpha returns c with its brightness/alpha replaced.
func (c RGBA) WithAlpha(a uint8) RGBA {
	c.A = a
	return c
}

// Effective returns the channels after brightness scaling. The lightbar scales
// all three LEDs by the alpha value, so this is what the hardware will show.
func (c RGBA) Effective() (r, g, b uint8) {
	return scale(c.R, c.A), scale(c.G, c.A), scale(c.B, c.A)
}

// Palette returns the effective, opaque color.
func (c RGBA) Palette() Palette {
	r, g, b := c.Effective()
	return Palette{R: r, G: g, B: b}
}

// Hex returns the RGB value as an uppercase #RRGGBB string.
func (c RGBA) Hex() string {
	return fmt.Sprintf("#%02X%02X%02X", c.R, c.G, c.B)
}

// HexA returns the full RGBA value as an uppercase #RRGGBBAA string.
func (c RGBA) HexA() string {
	return fmt.Sprintf("#%02X%02X%02X%02X", c.R, c.G, c.B, c.A)
}

// AlphaPercent returns the alpha as a rounded percentage.
func (c RGBA) AlphaPercent() int {
	return int(math.Round(float64(c.A) / 255 * 100))
}

// ParseHex accepts #RGB, #RGBA, #RRGGBB, #RRGGBBAA, with or without the '#'.
// The alpha channel is optional and defaults to 255.
func ParseHex(s string) (RGBA, error) {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "#")
	if len(s) != 3 && len(s) != 4 && len(s) != 6 && len(s) != 8 {
		return RGBA{}, fmt.Errorf("expected #RGB, #RGBA, #RRGGBB or #RRGGBBAA")
	}
	for _, r := range s {
		if !isHex(r) {
			return RGBA{}, fmt.Errorf("invalid hex digit %q", r)
		}
	}

	// Expand the shorthand forms (#abc -> #aabbcc).
	if len(s) == 3 || len(s) == 4 {
		var b strings.Builder
		b.Grow(len(s) * 2)
		for _, r := range s {
			b.WriteRune(r)
			b.WriteRune(r)
		}
		s = b.String()
	}

	parse := func(i int) uint8 {
		v, _ := strconv.ParseUint(s[i:i+2], 16, 8)
		return uint8(v)
	}
	c := RGBA{R: parse(0), G: parse(2), B: parse(4), A: 255}
	if len(s) == 8 {
		c.A = parse(6)
	}
	return c, nil
}

func isHex(r rune) bool {
	return (r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')
}

// HSVToRGB converts hue in degrees [0,360), saturation and value in [0,1] to
// 8-bit RGB. Values are clamped.
func HSVToRGB(h, s, v float64) (r, g, b uint8) {
	h = math.Mod(h, 360)
	if h < 0 {
		h += 360
	}
	s = clamp01(s)
	v = clamp01(v)

	c := v * s
	x := c * (1 - math.Abs(math.Mod(h/60, 2)-1))
	m := v - c

	var rf, gf, bf float64
	switch {
	case h < 60:
		rf, gf, bf = c, x, 0
	case h < 120:
		rf, gf, bf = x, c, 0
	case h < 180:
		rf, gf, bf = 0, c, x
	case h < 240:
		rf, gf, bf = 0, x, c
	case h < 300:
		rf, gf, bf = x, 0, c
	default:
		rf, gf, bf = c, 0, x
	}

	return to8(rf + m), to8(gf + m), to8(bf + m)
}

// RGBToHSV converts 8-bit RGB to hue in degrees and saturation/value in [0,1].
// Achromatic colors report a hue of 0; callers that want to preserve a hue for
// gray values should do so themselves.
func RGBToHSV(r, g, b uint8) (h, s, v float64) {
	rf, gf, bf := float64(r)/255, float64(g)/255, float64(b)/255
	max := math.Max(rf, math.Max(gf, bf))
	min := math.Min(rf, math.Min(gf, bf))
	delta := max - min

	v = max
	if max > 0 {
		s = delta / max
	}
	if delta == 0 {
		return 0, 0, v
	}

	switch max {
	case rf:
		h = 60 * math.Mod((gf-bf)/delta, 6)
	case gf:
		h = 60 * ((bf-rf)/delta + 2)
	default:
		h = 60 * ((rf-gf)/delta + 4)
	}
	if h < 0 {
		h += 360
	}
	return h, s, v
}

// Mix linearly interpolates between a and b. t is clamped to [0,1].
func Mix(a, b Palette, t float64) Palette {
	t = clamp01(t)
	return Palette{
		R: uint8(math.Round(float64(a.R) + (float64(b.R)-float64(a.R))*t)),
		G: uint8(math.Round(float64(a.G) + (float64(b.G)-float64(a.G))*t)),
		B: uint8(math.Round(float64(a.B) + (float64(b.B)-float64(a.B))*t)),
	}
}

// MixRGBA interpolates two RGBA values including brightness.
func MixRGBA(a, b RGBA, t float64) RGBA {
	t = clamp01(t)
	return RGBA{
		R: uint8(math.Round(float64(a.R) + (float64(b.R)-float64(a.R))*t)),
		G: uint8(math.Round(float64(a.G) + (float64(b.G)-float64(a.G))*t)),
		B: uint8(math.Round(float64(a.B) + (float64(b.B)-float64(a.B))*t)),
		A: uint8(math.Round(float64(a.A) + (float64(b.A)-float64(a.A))*t)),
	}
}

// ScaleRGB multiplies a color's channels by f, clamping the result.
func ScaleRGB(c RGBA, f float64) Palette {
	return Palette{
		R: to8(float64(c.R) / 255 * f),
		G: to8(float64(c.G) / 255 * f),
		B: to8(float64(c.B) / 255 * f),
	}
}

// Contrast returns black or white, whichever is more readable on c. Very
// transparent colors are treated as dark, because the lightbar is off.
func (c RGBA) Contrast() Palette {
	if c.A < 24 {
		return Palette{R: 255, G: 255, B: 255}
	}
	r, g, b := c.Effective()
	// Relative luminance in sRGB.
	lin := func(v uint8) float64 {
		f := float64(v) / 255
		if f <= 0.04045 {
			return f / 12.92
		}
		return math.Pow((f+0.055)/1.055, 2.4)
	}
	lum := 0.2126*lin(r) + 0.7152*lin(g) + 0.0722*lin(b)
	if lum > 0.45 {
		return Palette{R: 10, G: 10, B: 14}
	}
	return Palette{R: 255, G: 255, B: 255}
}

// Hex returns the palette as an uppercase #RRGGBB string.
func (p Palette) Hex() string {
	return fmt.Sprintf("#%02X%02X%02X", p.R, p.G, p.B)
}

func scale(v, a uint8) uint8 {
	return uint8((uint32(v)*uint32(a) + 127) / 255)
}

func to8(f float64) uint8 {
	return uint8(math.Round(clamp01(f) * 255))
}

func clamp01(v float64) float64 {
	switch {
	case v < 0:
		return 0
	case v > 1:
		return 1
	default:
		return v
	}
}
