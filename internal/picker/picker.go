// Package picker holds the HSV + alpha editing state used by the UI.
package picker

import (
	"math"

	"github.com/blazar/ds4-tui/internal/color"
)

// Channel identifies one editable RGBA channel.
type Channel int

const (
	Red Channel = iota
	Green
	Blue
	Alpha
)

// Model is an HSV color with a separate alpha/brightness channel. HSV is kept
// as the source of truth so the hue survives moves through black and gray.
type Model struct {
	Hue   float64 // degrees, [0,360)
	Sat   float64 // [0,1]
	Val   float64 // [0,1]
	Alpha uint8   // [0,255]
}

// New returns the default hero color.
func New() Model {
	m := Model{Alpha: 255}
	m.SetRGB(139, 92, 246)
	return m
}

// FromRGBA creates a picker model from an RGBA value.
func FromRGBA(c color.RGBA) Model {
	m := New()
	m.SetRGBA(c)
	return m
}

// RGBA returns the current color.
func (m Model) RGBA() color.RGBA {
	r, g, b := color.HSVToRGB(m.Hue, m.Sat, m.Val)
	return color.RGBA{R: r, G: g, B: b, A: m.Alpha}
}

// Effective returns the displayed lightbar color after brightness scaling.
func (m Model) Effective() color.Palette {
	return m.RGBA().Palette()
}

// SetRGBA updates the color. The existing hue is retained for achromatic
// colors so dragging a slider to black does not lose the selected hue.
func (m *Model) SetRGBA(c color.RGBA) {
	h, s, v := color.RGBToHSV(c.R, c.G, c.B)
	if s == 0 {
		h = m.Hue
	}
	m.Hue = normalizeHue(h)
	m.Sat = clamp01(s)
	m.Val = clamp01(v)
	m.Alpha = c.A
}

// SetRGB updates only the color channels.
func (m *Model) SetRGB(r, g, b uint8) {
	h, s, v := color.RGBToHSV(r, g, b)
	if s == 0 {
		h = m.Hue
	}
	m.Hue = normalizeHue(h)
	m.Sat = clamp01(s)
	m.Val = clamp01(v)
}

// SetHue sets and wraps the hue.
func (m *Model) SetHue(h float64) {
	m.Hue = normalizeHue(h)
}

// NudgeHue rotates the hue by delta degrees.
func (m *Model) NudgeHue(delta float64) {
	m.SetHue(m.Hue + delta)
}

// SetSV sets saturation and value, clamping both.
func (m *Model) SetSV(s, v float64) {
	m.Sat = clamp01(s)
	m.Val = clamp01(v)
}

// NudgeSV adjusts saturation and value by the supplied deltas.
func (m *Model) NudgeSV(ds, dv float64) {
	m.SetSV(m.Sat+ds, m.Val+dv)
}

// SetAlpha sets brightness.
func (m *Model) SetAlpha(a int) {
	m.Alpha = clampU8(a)
}

// NudgeAlpha adjusts brightness.
func (m *Model) NudgeAlpha(delta int) {
	m.SetAlpha(int(m.Alpha) + delta)
}

// ChannelValue reads one channel.
func (m Model) ChannelValue(ch Channel) int {
	c := m.RGBA()
	switch ch {
	case Red:
		return int(c.R)
	case Green:
		return int(c.G)
	case Blue:
		return int(c.B)
	case Alpha:
		return int(c.A)
	default:
		return 0
	}
}

// SetChannel sets one channel while preserving hue when possible.
func (m *Model) SetChannel(ch Channel, value int) {
	if ch == Alpha {
		m.SetAlpha(value)
		return
	}
	c := m.RGBA()
	switch ch {
	case Red:
		c.R = clampU8(value)
	case Green:
		c.G = clampU8(value)
	case Blue:
		c.B = clampU8(value)
	}
	m.SetRGB(c.R, c.G, c.B)
}

// NudgeChannel adjusts one channel, clamping at the ends.
func (m *Model) NudgeChannel(ch Channel, delta int) {
	if delta == 0 {
		return
	}
	m.SetChannel(ch, m.ChannelValue(ch)+delta)
}

// Hex returns the current #RRGGBB value.
func (m Model) Hex() string {
	return m.RGBA().Hex()
}

// HexA returns the current #RRGGBBAA value.
func (m Model) HexA() string {
	return m.RGBA().HexA()
}

// AlphaPercent returns brightness as a percentage.
func (m Model) AlphaPercent() int {
	return m.RGBA().AlphaPercent()
}

// SetHex parses a hex color (with optional alpha) and applies it.
func (m *Model) SetHex(s string) error {
	c, err := color.ParseHex(s)
	if err != nil {
		return err
	}
	m.SetRGBA(c)
	return nil
}

func normalizeHue(h float64) float64 {
	if math.IsNaN(h) || math.IsInf(h, 0) {
		return 0
	}
	h = math.Mod(h, 360)
	if h < 0 {
		h += 360
	}
	return h
}

func clamp01(v float64) float64 {
	switch {
	case math.IsNaN(v), v < 0:
		return 0
	case v > 1:
		return 1
	default:
		return v
	}
}

func clampU8(v int) uint8 {
	switch {
	case v < 0:
		return 0
	case v > 255:
		return 255
	default:
		return uint8(v)
	}
}
