package tui

import (
	"fmt"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/blazar/ds4-tui/internal/ds4"
	"github.com/blazar/ds4-tui/internal/picker"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"
)

func newTestModel(w, h int) Model {
	m := New(ds4.NewDemo())
	m.width = w
	m.height = h
	st := m.svc.Detect()
	m.status = st
	m.statusKnown = true
	m.picker = picker.FromRGBA(st.Current)
	m.lastApplied = st.Current
	m.hasApplied = true
	return m
}

func TestViewFitsEveryScreenSize(t *testing.T) {
	sizes := [][2]int{
		{1, 1}, {2, 1}, {5, 2}, {20, 8}, {35, 12},
		{36, 13}, {40, 14}, {48, 16}, {60, 20}, {80, 24},
		{100, 30}, {120, 36}, {160, 50}, {200, 60}, {300, 100},
	}

	for _, size := range sizes {
		for _, screen := range []screen{screenColor, screenPresets, screenRainbow} {
			for _, help := range []bool{false, true} {
				m := newTestModel(size[0], size[1])
				m.screen = screen
				m.help = help
				m.clampPresetWindow()
				view := m.View()
				assertCanvas(t, view, size[0], size[1],
					fmt.Sprintf("screen=%d help=%v", screen, help))
			}
		}
	}
}

func TestViewHexModeFits(t *testing.T) {
	for _, size := range [][2]int{{36, 13}, {40, 14}, {60, 20}, {80, 24}, {120, 36}} {
		m := newTestModel(size[0], size[1])
		if m.screen != screenColor {
			t.Fatal("expected color screen")
		}
		m.beginHexEdit()
		m.hexInput = "#FF8000"
		assertCanvas(t, m.View(), size[0], size[1], "hex")
		m.hexErr = "expected #RGB, #RGBA, #RRGGBB or #RRGGBBAA"
		assertCanvas(t, m.View(), size[0], size[1], "hex error")
	}
}

func TestViewWithoutWindowSize(t *testing.T) {
	m := New(ds4.NewDemo())
	if got := m.View(); got != "" {
		t.Fatalf("expected empty view before window size, got %q", got)
	}
	m.width = 120
	if got := m.View(); got != "" {
		t.Fatalf("expected empty view without height, got %q", got)
	}
}

func TestCardIsExactlySized(t *testing.T) {
	for _, width := range []int{4, 5, 8, 9, 20, 31} {
		lines := renderCard(cardOpts{
			Width:  width,
			Height: 6,
			Title:  "TOUCHPAD LIGHTBAR",
			Active: true,
			Accent: colAccent,
			BG:     colPanel,
		}, []string{"hello", "world"})
		if len(lines) != 6 {
			t.Fatalf("width %d: got %d lines", width, len(lines))
		}
		for i, line := range lines {
			if got := ansi.StringWidth(line); got != width {
				t.Fatalf("width %d line %d: width %d", width, i, got)
			}
		}
		if width >= 9 {
			last := lines[len(lines)-1]
			if !strings.HasSuffix(last, "╯\x1b[0m") && !strings.Contains(last, "╯") {
				t.Fatalf("width %d bottom border = %q", width, last)
			}
		}
	}
}

func TestPresetGridScrollsLargeGallery(t *testing.T) {
	if len(defaultPresets) != 62 {
		t.Fatalf("preset count = %d, want 62", len(defaultPresets))
	}
	m := newTestModel(120, 36)
	m.screen = screenPresets
	m.clampPresetWindow()
	cols, cellW, cellH, rows := m.presetGrid()
	if cols != 4 {
		t.Fatalf("cols = %d, want 4", cols)
	}
	if cellH < 4 {
		t.Fatalf("cellH = %d", cellH)
	}
	if got := cols * cellW; got > m.contentWidth() {
		t.Fatalf("grid width %d exceeds content %d", got, m.contentWidth())
	}
	if rows*cellH+(rows-1) > m.bodyHeight() {
		t.Fatalf("grid height exceeds the body")
	}
	if cols*rows >= len(defaultPresets) {
		t.Fatalf("gallery should require scrolling: %d visible of %d", cols*rows, len(defaultPresets))
	}

	// The last page must be reachable and fully clamped.
	m, _ = send(t, m, "p")
	m, _ = send(t, m, "end")
	if m.presetCursor != len(defaultPresets)-1 {
		t.Fatalf("cursor = %d", m.presetCursor)
	}
	if m.presetOffset == 0 {
		t.Fatal("expected a scrolled offset at the end of the gallery")
	}
	assertCanvas(t, m.View(), 120, 36, "large gallery at end")
}

func TestStatusBadges(t *testing.T) {
	m := newTestModel(100, 30)
	if got := ansi.Strip(m.statusBadge(true)); !strings.Contains(got, "DEMO") {
		t.Fatalf("demo badge = %q", got)
	}
	m.demo = false
	m.status.Found = false
	if got := ansi.Strip(m.statusBadge(true)); !strings.Contains(got, "NO DS4") {
		t.Fatalf("missing badge = %q", got)
	}
	m.status.Found = true
	m.status.Writable = false
	if got := ansi.Strip(m.statusBadge(true)); !strings.Contains(got, "SUDO") {
		t.Fatalf("sudo badge = %q", got)
	}
}

func (m Model) contentWidth() int {
	w, _ := m.bodySize()
	return w
}

func (m Model) bodyHeight() int {
	_, h := m.bodySize()
	return h
}

func assertCanvas(t *testing.T, view string, width, height int, context string) {
	t.Helper()
	if !utf8.ValidString(view) {
		t.Fatalf("%s: view contains invalid UTF-8", context)
	}
	if view == "" {
		t.Fatalf("%s: empty view for %dx%d", context, width, height)
	}
	lines := strings.Split(view, "\n")
	if len(lines) != height {
		t.Fatalf("%s: %dx%d produced %d lines", context, width, height, len(lines))
	}
	for i, line := range lines {
		if got := ansi.StringWidth(line); got != width {
			t.Fatalf("%s: %dx%d line %d width = %d: %q", context, width, height, i, got, ansi.Strip(line))
		}
	}
}

func keyMsg(s string) tea.KeyMsg {
	switch s {
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "tab":
		return tea.KeyMsg{Type: tea.KeyTab}
	case "shift+tab":
		return tea.KeyMsg{Type: tea.KeyShiftTab}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	case "up":
		return tea.KeyMsg{Type: tea.KeyUp}
	case "down":
		return tea.KeyMsg{Type: tea.KeyDown}
	case "left":
		return tea.KeyMsg{Type: tea.KeyLeft}
	case "right":
		return tea.KeyMsg{Type: tea.KeyRight}
	case "pgup":
		return tea.KeyMsg{Type: tea.KeyPgUp}
	case "pgdown":
		return tea.KeyMsg{Type: tea.KeyPgDown}
	case "backspace":
		return tea.KeyMsg{Type: tea.KeyBackspace}
	case "space":
		return tea.KeyMsg{Type: tea.KeySpace}
	default:
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
	}
}

func TestColorRenderingOn(t *testing.T) {
	oldColor := colorEnabled
	oldTransparent := transparentBackground
	colorEnabled = true
	transparentBackground = false
	defer func() {
		colorEnabled = oldColor
		transparentBackground = oldTransparent
		if oldColor {
			lipgloss.SetColorProfile(termenv.TrueColor)
		} else {
			lipgloss.SetColorProfile(termenv.Ascii)
		}
	}()
	lipgloss.SetColorProfile(termenv.TrueColor)

	m := newTestModel(120, 36)
	view := m.View()
	if !strings.Contains(view, "48;2;139;92;246") {
		t.Fatal("expected the selected tab to use the accent background")
	}
	if !strings.Contains(view, "\x1b[48;2;8;8;13m") {
		t.Fatal("expected custom truecolor backgrounds")
	}
	assertCanvas(t, view, 120, 36, "color rendering")
}

func TestTransparentBackground(t *testing.T) {
	oldColor := colorEnabled
	oldTransparent := transparentBackground
	colorEnabled = true
	defer func() {
		colorEnabled = oldColor
		transparentBackground = oldTransparent
		if oldColor {
			lipgloss.SetColorProfile(termenv.TrueColor)
		} else {
			lipgloss.SetColorProfile(termenv.Ascii)
		}
	}()
	lipgloss.SetColorProfile(termenv.TrueColor)

	m := newTestModel(80, 24)
	transparentBackground = true
	view := m.View()
	if strings.Contains(view, "48;2;8;8;13") {
		t.Fatal("transparent view painted the base background color")
	}
	if !strings.Contains(view, "\x1b[49m") {
		t.Fatal("transparent view did not reset cells to the terminal background")
	}

	transparentBackground = false
	view = m.View()
	if !strings.Contains(view, "48;2;8;8;13") {
		t.Fatal("opaque view did not paint the base background color")
	}
}
