package tui

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/blazar/ds4-tui/internal/color"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

const (
	colBG       = "#08080D"
	colPanel    = "#101018"
	colPanel2   = "#161622"
	colBorder   = "#44445E"
	colBorderHi = "#52526C"
	colText     = "#F2F2F8"
	colMuted    = "#9595AC"
	colDim      = "#5A5A72"
	colAccent   = "#8B5CF6"
	colAccent2  = "#22D3EE"
	colPink     = "#F472B6"
	colBlue     = "#60A5FA"
	colGreen    = "#34D399"
	colAmber    = "#FBBF24"
	colRed      = "#FB7185"
)

// colorEnabled keeps the hand-built truecolor escapes in sync with Lipgloss
// and with the user's NO_COLOR / TERM=dumb preferences.
var colorEnabled = detectColor()

// transparentBackground lets the terminal's background (and any transparency
// in the terminal emulator) show through chrome, cards and padding. Color
// swatches, picker fields, sliders and the glow still draw their own colors.
var transparentBackground = true

// SetTransparent enables or disables the transparent chrome.
func SetTransparent(enabled bool) {
	transparentBackground = enabled
}

func isChromeBG(hex string) bool {
	return hex == colBG || hex == colPanel || hex == colPanel2
}

func detectColor() bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	if strings.EqualFold(os.Getenv("TERM"), "dumb") {
		return false
	}
	if os.Getenv("CLICOLOR") == "0" {
		return false
	}
	return true
}

func init() {
	if colorEnabled {
		// The preview, picker and gradients are truecolor by design. Forcing the
		// profile keeps Lipgloss chrome consistent with those custom escapes.
		lipgloss.SetColorProfile(termenv.TrueColor)
	} else {
		lipgloss.SetColorProfile(termenv.Ascii)
	}
}

func fg(c string) lipgloss.Style { return lipgloss.NewStyle().Foreground(lipgloss.Color(c)) }

func fgBold(c string) lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(c)).Bold(true)
}

func lc(p color.Palette) lipgloss.Color { return lipgloss.Color(p.Hex()) }

// bgEscape returns an SGR sequence for a truecolor background.
func bgEscape(hex string) string {
	if !colorEnabled {
		return ""
	}
	r, g, b := parseHexRGB(hex)
	return fmt.Sprintf("\x1b[48;2;%d;%d;%dm", r, g, b)
}

// chromeEscape is used for layout backgrounds. In transparent mode those
// cells reset to the terminal's default background instead of painting a
// color. Color fills (swatches, picker cells, sliders) use bgEscape directly.
func chromeEscape(hex string) string {
	if !colorEnabled {
		return ""
	}
	if transparentBackground && isChromeBG(hex) {
		return "\x1b[49m"
	}
	return bgEscape(hex)
}

func fgEscape(hex string) string {
	if !colorEnabled {
		return ""
	}
	r, g, b := parseHexRGB(hex)
	return fmt.Sprintf("\x1b[38;2;%d;%d;%dm", r, g, b)
}

func parseHexRGB(hex string) (int, int, int) {
	hex = strings.TrimPrefix(hex, "#")
	if len(hex) < 6 {
		return 0, 0, 0
	}
	r, _ := strconv.ParseInt(hex[0:2], 16, 32)
	g, _ := strconv.ParseInt(hex[2:4], 16, 32)
	b, _ := strconv.ParseInt(hex[4:6], 16, 32)
	return int(r), int(g), int(b)
}

// paintBG gives every cell in line a background, including cells after nested
// ANSI resets, then pads or truncates the line to exactly width cells.
func paintBG(line, bg string, width int) string {
	if width <= 0 {
		return ""
	}
	seq := chromeEscape(bg)
	line = strings.ReplaceAll(line, "\x1b[0m", "\x1b[0m"+seq)
	w := displayWidth(line)
	if w > width {
		line = truncateANSI(line, width)
		w = displayWidth(line)
	}
	pad := width - w
	if pad < 0 {
		pad = 0
	}
	return seq + line + strings.Repeat(" ", pad) + "\x1b[0m"
}

func blankLine(width int, bg string) string {
	if width <= 0 {
		return ""
	}
	return chromeEscape(bg) + strings.Repeat(" ", width) + "\x1b[0m"
}

// spread puts left and right on one line, with as much space between them as
// possible. If they do not fit, the right side is dropped.
func spread(left, right string, width int) string {
	if width <= 0 {
		return ""
	}
	lw, rw := displayWidth(left), displayWidth(right)
	if lw+rw+1 > width {
		return truncateANSI(left, width)
	}
	gap := width - lw - rw
	return left + strings.Repeat(" ", gap) + right
}

func center(s string, width int) string {
	w := displayWidth(s)
	if w >= width {
		return truncateANSI(s, width)
	}
	left := (width - w) / 2
	return strings.Repeat(" ", left) + s + strings.Repeat(" ", width-w-left)
}

func truncateANSIWithTail(s string, width int) string {
	if width <= 0 {
		return ""
	}
	if displayWidth(s) <= width {
		return s
	}
	return ansiTruncateTail(s, width)
}

func truncateANSI(s string, width int) string {
	if width <= 0 {
		return ""
	}
	if displayWidth(s) <= width {
		return s
	}
	// ansi.Truncate is imported in widgets.go; keep all width logic together.
	return ansiTruncate(s, width)
}

func gradient(text string, colors []string) string {
	runes := []rune(text)
	if len(runes) == 0 {
		return ""
	}
	if len(colors) == 0 {
		colors = []string{colText}
	}
	var b strings.Builder
	for i, r := range runes {
		pos := 0.0
		if len(runes) > 1 {
			pos = float64(i) / float64(len(runes)-1)
		}
		c := interpolateHex(colors, pos)
		b.WriteString(fgBold(c).Render(string(r)))
	}
	return b.String()
}

func interpolateHex(colors []string, t float64) string {
	if len(colors) == 1 {
		return colors[0]
	}
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	scaled := t * float64(len(colors)-1)
	idx := int(scaled)
	if idx >= len(colors)-1 {
		return colors[len(colors)-1]
	}
	local := scaled - float64(idx)
	r1, g1, b1 := rgbOf(colors[idx])
	r2, g2, b2 := rgbOf(colors[idx+1])
	return color.Mix(
		color.Palette{R: r1, G: g1, B: b1},
		color.Palette{R: r2, G: g2, B: b2},
		local,
	).Hex()
}

func rgbOf(hex string) (r, g, b uint8) {
	ri, gi, bi := parseHexRGB(hex)
	return uint8(ri), uint8(gi), uint8(bi)
}

// spreadKeepRight is like spread, but protects the right side when the left
// side is too long. Useful for header-style rows where the value matters more
// than the label.
func spreadKeepRight(left, right string, width int) string {
	if width <= 0 {
		return ""
	}
	rw := displayWidth(right)
	if rw >= width {
		return truncateANSI(right, width)
	}
	avail := width - rw - 1
	if avail < 0 {
		avail = 0
	}
	l := truncateANSI(left, avail)
	lw := displayWidth(l)
	gap := width - lw - rw
	if gap < 0 {
		gap = 0
	}
	return l + strings.Repeat(" ", gap) + right
}
