package color

import (
	"math"
	"testing"
)

func TestHSVToRGB(t *testing.T) {
	tests := []struct {
		name    string
		h, s, v float64
		r, g, b uint8
	}{
		{"red", 0, 1, 1, 255, 0, 0},
		{"green", 120, 1, 1, 0, 255, 0},
		{"blue", 240, 1, 1, 0, 0, 255},
		{"cyan", 180, 1, 1, 0, 255, 255},
		{"magenta", 300, 1, 1, 255, 0, 255},
		{"yellow", 60, 1, 1, 255, 255, 0},
		{"gray", 200, 0, 0.5, 128, 128, 128},
		{"black", 100, 1, 0, 0, 0, 0},
		{"wrap", 360, 1, 1, 255, 0, 0},
		{"wrap negative", -120, 1, 1, 0, 0, 255},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, g, b := HSVToRGB(tt.h, tt.s, tt.v)
			if r != tt.r || g != tt.g || b != tt.b {
				t.Fatalf("HSVToRGB(%v,%v,%v) = %d,%d,%d; want %d,%d,%d",
					tt.h, tt.s, tt.v, r, g, b, tt.r, tt.g, tt.b)
			}
		})
	}
}

func TestRGBHSV(t *testing.T) {
	for r := 0; r <= 255; r += 51 {
		for g := 0; g <= 255; g += 51 {
			for b := 0; b <= 255; b += 51 {
				h, s, v := RGBToHSV(uint8(r), uint8(g), uint8(b))
				rr, gg, bb := HSVToRGB(h, s, v)
				if absDiff(rr, uint8(r)) > 1 || absDiff(gg, uint8(g)) > 1 || absDiff(bb, uint8(b)) > 1 {
					t.Fatalf("roundtrip %d,%d,%d -> hsv(%v,%v,%v) -> %d,%d,%d",
						r, g, b, h, s, v, rr, gg, bb)
				}
			}
		}
	}
}

func TestParseHex(t *testing.T) {
	tests := []struct {
		in      string
		want    RGBA
		wantErr bool
	}{
		{"#FF0000", RGBA{255, 0, 0, 255}, false},
		{"FF0000", RGBA{255, 0, 0, 255}, false},
		{"#f0a", RGBA{255, 0, 170, 255}, false},
		{"#f0a8", RGBA{255, 0, 170, 136}, false},
		{"#11223344", RGBA{17, 34, 51, 68}, false},
		{"  #abcdef  ", RGBA{171, 205, 239, 255}, false},
		{"#12345", RGBA{}, true},
		{"#gggggg", RGBA{}, true},
		{"", RGBA{}, true},
	}
	for _, tt := range tests {
		got, err := ParseHex(tt.in)
		if (err != nil) != tt.wantErr {
			t.Fatalf("ParseHex(%q) error = %v, wantErr %v", tt.in, err, tt.wantErr)
		}
		if !tt.wantErr && got != tt.want {
			t.Fatalf("ParseHex(%q) = %+v, want %+v", tt.in, got, tt.want)
		}
	}
}

func TestEffectiveAndAlpha(t *testing.T) {
	c := RGBA{R: 255, G: 128, B: 0, A: 128}
	r, g, b := c.Effective()
	if r != 128 || g != 64 || b != 0 {
		t.Fatalf("Effective = %d,%d,%d; want 128,64,0", r, g, b)
	}
	if got := c.AlphaPercent(); got != 50 {
		t.Fatalf("AlphaPercent = %d, want 50", got)
	}
	if got := (RGBA{R: 10, G: 20, B: 30, A: 255}).Hex(); got != "#0A141E" {
		t.Fatalf("Hex = %q", got)
	}
}

func TestContrast(t *testing.T) {
	dark := (RGBA{R: 4, G: 4, B: 8, A: 255}).Contrast()
	light := (RGBA{R: 240, G: 240, B: 240, A: 255}).Contrast()
	if dark.R != 255 || dark.G != 255 || dark.B != 255 {
		t.Fatalf("dark color should get white text, got %+v", dark)
	}
	if light.R > 20 || light.G > 20 || light.B > 20 {
		t.Fatalf("light color should get dark text, got %+v", light)
	}
}

func absDiff(a, b uint8) int {
	d := int(a) - int(b)
	if d < 0 {
		return -d
	}
	return d
}

func TestMix(t *testing.T) {
	a := Palette{R: 0, G: 0, B: 0}
	b := Palette{R: 100, G: 200, B: 255}
	got := Mix(a, b, 0.5)
	if got.R != 50 || got.G != 100 || math.Abs(float64(got.B)-128) > 1 {
		t.Fatalf("Mix = %+v", got)
	}
}
