package picker

import (
	"testing"

	"github.com/blazar/ds4-tui/internal/color"
)

func TestRoundTrip(t *testing.T) {
	want := color.RGBA{R: 139, G: 92, B: 246, A: 128}
	m := FromRGBA(want)
	got := m.RGBA()
	if got != want {
		t.Fatalf("roundtrip = %+v, want %+v", got, want)
	}
	if m.Hex() != "#8B5CF6" || m.HexA() != "#8B5CF680" {
		t.Fatalf("hex = %s / %s", m.Hex(), m.HexA())
	}
}

func TestGrayPreservesHue(t *testing.T) {
	m := Model{Hue: 210, Sat: 0.8, Val: 0.9, Alpha: 255}
	m.SetRGBA(color.RGBA{R: 128, G: 128, B: 128, A: 255})
	if m.Hue != 210 {
		t.Fatalf("gray changed hue to %v", m.Hue)
	}
	if m.Sat != 0 {
		t.Fatalf("gray saturation = %v", m.Sat)
	}
	m.SetSV(0, 0.2)
	if got := m.RGBA(); got.R != got.G || got.G != got.B {
		t.Fatalf("raising value on gray should stay gray, got %+v", got)
	}
	if got := m.Hex(); got != "#333333" {
		t.Fatalf("value 0.2 gray = %s", got)
	}
	m.NudgeSV(0, 0.1)
	if got := m.RGBA(); got.R != got.G || got.G != got.B {
		t.Fatalf("nudging gray should stay gray, got %+v", got)
	}
}

func TestSetChannelAndClamp(t *testing.T) {
	m := FromRGBA(color.RGBA{R: 10, G: 20, B: 30, A: 255})
	m.SetChannel(Red, 300)
	if got := m.ChannelValue(Red); got != 255 {
		t.Fatalf("red = %d, want 255", got)
	}
	m.SetChannel(Green, -1)
	if got := m.ChannelValue(Green); got != 0 {
		t.Fatalf("green = %d, want 0", got)
	}
	m.SetAlpha(999)
	if m.Alpha != 255 {
		t.Fatalf("alpha = %d", m.Alpha)
	}
	m.NudgeChannel(Blue, -50)
	if got := m.ChannelValue(Blue); got != 0 {
		t.Fatalf("blue = %d, want 0", got)
	}
}

func TestHueWrap(t *testing.T) {
	m := New()
	m.SetHue(370)
	if m.Hue != 10 {
		t.Fatalf("hue = %v, want 10", m.Hue)
	}
	m.NudgeHue(-20)
	if m.Hue != 350 {
		t.Fatalf("hue = %v, want 350", m.Hue)
	}
}

func TestSetHex(t *testing.T) {
	m := New()
	if err := m.SetHex("#FF8000"); err != nil {
		t.Fatal(err)
	}
	if m.Hex() != "#FF8000" {
		t.Fatalf("hex = %s", m.Hex())
	}
	if err := m.SetHex("nope"); err == nil {
		t.Fatal("expected error")
	}
}
