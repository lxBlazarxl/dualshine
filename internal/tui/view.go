package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

const (
	minWidth       = 36
	minHeight      = 13
	maxFrameWidth  = 128
	maxFrameHeight = 44
)

type layoutInfo struct {
	screenW, screenH int
	uiW, uiH         int
	padX, padY       int
	gutter           int
	contentW         int
	headerH          int
	bodyH            int
	full             bool
	tiny             bool
}

func (m Model) layout() layoutInfo {
	l := layoutInfo{
		screenW: m.width,
		screenH: m.height,
	}
	if m.width <= 0 || m.height <= 0 {
		return l
	}
	l.tiny = m.width < minWidth || m.height < minHeight
	l.uiW = minInt(m.width, maxFrameWidth)
	l.uiH = minInt(m.height, maxFrameHeight)
	l.padX = (m.width - l.uiW) / 2
	l.padY = (m.height - l.uiH) / 2
	l.gutter = 2
	if l.uiW < 60 {
		l.gutter = 1
	}
	l.contentW = maxInt(1, l.uiW-2*l.gutter)
	l.full = !l.tiny && l.uiH >= 28 && l.uiW >= 72
	l.headerH = 1
	if l.full {
		l.headerH = 2
	}
	if l.full {
		l.bodyH = l.uiH - l.headerH - 6
	} else {
		l.bodyH = l.uiH - l.headerH - 3
	}
	if l.bodyH < 1 {
		l.bodyH = 1
	}
	return l
}

func (m Model) bodySize() (int, int) {
	l := m.layout()
	return l.contentW, l.bodyH
}

// View implements tea.Model. Every path returns exactly m.height lines and
// no line wider than m.width, so resizing cannot leave the UI wrapped.
func (m Model) View() string {
	if m.width <= 0 || m.height <= 0 {
		return ""
	}
	l := m.layout()
	if l.tiny {
		return m.tinyView(l)
	}

	var frame []string
	header := m.header(l.contentW, l.full)
	frame = append(frame, header...)

	if l.full {
		frame = append(frame, blankLine(l.contentW, colBG))
	}

	frame = append(frame, m.tabBar(l.contentW))

	if l.full {
		frame = append(frame, blankLine(l.contentW, colBG))
	}

	body := m.bodyView(l.contentW, l.bodyH)
	if len(body) < l.bodyH {
		for len(body) < l.bodyH {
			body = append(body, blankLine(l.contentW, colBG))
		}
	}
	if len(body) > l.bodyH {
		body = body[:l.bodyH]
	}
	frame = append(frame, body...)

	if l.full {
		frame = append(frame, blankLine(l.contentW, colBG))
	}
	frame = append(frame, m.statusLine(l.contentW))
	frame = append(frame, m.keyBar(l.contentW))

	// Normalize the frame height before compositing.
	if len(frame) < l.uiH {
		for len(frame) < l.uiH {
			frame = append(frame, blankLine(l.contentW, colBG))
		}
	}
	if len(frame) > l.uiH {
		frame = frame[:l.uiH]
	}

	lines := make([]string, 0, m.height)
	for i := 0; i < l.padY; i++ {
		lines = append(lines, blankLine(m.width, colBG))
	}
	left := l.padX + l.gutter
	right := m.width - left - l.contentW
	if right < 0 {
		right = 0
	}
	for _, line := range frame {
		screenLine := strings.Repeat(" ", left) + line + strings.Repeat(" ", right)
		lines = append(lines, paintBG(screenLine, colBG, m.width))
	}
	for len(lines) < m.height {
		lines = append(lines, blankLine(m.width, colBG))
	}
	if len(lines) > m.height {
		lines = lines[:m.height]
	}
	return strings.Join(lines, "\n")
}

func (m Model) bodyView(width, height int) []string {
	if m.help {
		return m.helpView(width, height)
	}
	switch m.screen {
	case screenPresets:
		return m.presetsView(width, height)
	case screenRainbow:
		return m.rainbowView(width, height)
	default:
		return m.colorView(width, height)
	}
}

func (m Model) header(width int, full bool) []string {
	mark := fgBold(colAccent2).Render("◆")
	word := gradient("DUALSHINE", []string{colText, colText, colAccent, colPink, colAccent2})
	left := mark + " " + word
	right := m.statusBadge(full)
	line1 := spread(left, right, width)

	if !full {
		return []string{paintBG(line1, colBG, width)}
	}

	subLeft := fgBold(colMuted).Render("DUALSHOCK 4") +
		fg(colDim).Render("  /  ") +
		fg(colMuted).Render("LIGHTBAR STUDIO")
	subRight := m.headerContext()
	line2 := spread(subLeft, subRight, width)
	return []string{
		paintBG(line1, colBG, width),
		paintBG(line2, colBG, width),
	}
}

func (m Model) statusBadge(full bool) string {
	type badge struct {
		dot   string
		label string
		color string
	}
	var b badge
	switch {
	case !m.statusKnown:
		b = badge{"◌", "SCANNING", colDim}
		if !full {
			b.label = "SCAN"
		}
	case !m.status.LEDClass:
		b = badge{"✕", "NO LED CLASS", colRed}
		if !full {
			b.label = "NO LEDS"
		}
	case m.status.ControllerSeen && !m.status.Found:
		b = badge{"◐", "DS4 · NO LEDS", colAmber}
		if !full {
			b.label = "NO LEDS"
		}
	case m.demo:
		b = badge{"●", "DEMO MODE", colPink}
		if !full {
			b.label = "DEMO"
		}
	case m.status.Found && !m.status.Writable:
		b = badge{"●", "DS4 · NEEDS SUDO", colAmber}
		if !full {
			b.label = "DS4 · SUDO"
		}
	case m.status.Found && !m.status.Sony:
		b = badge{"●", "DS4 · UNVERIFIED", colAmber}
		if !full {
			b.label = "DS4 · ?"
		}
	case m.status.Found:
		b = badge{"●", "DS4 READY", colGreen}
		if !full {
			b.label = "READY"
		}
	default:
		b = badge{"○", "NO DS4", colRed}
	}
	return fgBold(b.color).Render(b.dot) + " " + fgBold(b.color).Render(b.label)
}

func (m Model) headerContext() string {
	switch {
	case !colorEnabled:
		return fg(colDim).Render("NO_COLOR · MONOCHROME")
	case !m.statusKnown:
		return fg(colDim).Render("SCANNING FOR LIGHTBAR")
	case !m.status.LEDClass:
		return fg(colRed).Render("SYSFS LED BACKEND UNAVAILABLE")
	case m.status.ControllerSeen && !m.status.Found:
		return fg(colAmber).Render("DS4 CONNECTED · HID-SONY NEEDED")
	case m.demo:
		return fg(colDim).Render("DEMO · NOTHING IS WRITTEN")
	case m.rainbowRunning:
		return fgBold(colAccent2).Render("RAINBOW ACTIVE")
	case m.applying:
		return fgBold(colAmber).Render("APPLYING")
	case m.statusKnown && !m.status.Found:
		return fg(colAmber).Render("CONNECT A CONTROLLER")
	case m.status.KeeperPID > 0:
		return fg(colMuted).Render(fmt.Sprintf("HOLDING COLOR · PID %d", m.status.KeeperPID))
	default:
		return fg(colDim).Render("LIVE PREVIEW")
	}
}

func (m Model) tabBar(width int) string {
	type tab struct {
		key    string
		label  string
		screen screen
		color  string
	}
	tabs := []tab{
		{"c", "COLOR", screenColor, colAccent},
		{"p", "PRESETS", screenPresets, colPink},
		{"w", "RAINBOW", screenRainbow, colAccent2},
	}

	build := func(showKeys bool) string {
		parts := make([]string, 0, len(tabs))
		for _, t := range tabs {
			active := m.screen == t.screen
			if active {
				text := " " + t.label + " "
				if showKeys {
					text = " " + t.key + " " + t.label + " "
				}
				if !colorEnabled {
					text = "▸" + text
				}
				parts = append(parts, lipgloss.NewStyle().
					Background(lipgloss.Color(t.color)).
					Foreground(lipgloss.Color(colBG)).
					Bold(true).
					Render(text))
				continue
			}
			if showKeys {
				parts = append(parts,
					fg(t.color).Render(t.key)+" "+fg(colMuted).Render(t.label))
			} else {
				parts = append(parts, fg(colMuted).Render(t.label))
			}
		}
		return strings.Join(parts, "  ")
	}

	bar := build(true)
	if displayWidth(bar) > width {
		bar = build(false)
	}
	if displayWidth(bar) > width {
		bar = truncateANSI(bar, width)
	}

	help := ""
	if width >= 62 {
		help = fg(colDim).Render("?") + " " + fg(colDim).Render("help")
	}
	line := spread(bar, help, width)
	return paintBG(line, colBG, width)
}

func (m Model) statusLine(width int) string {
	if m.hexEditing {
		return m.hexStatusLine(width)
	}

	text, kind := m.contextStatus(width)
	icon, iconColor := noticeIcon(kind)
	if icon == "" {
		icon = "·"
		iconColor = colDim
	}
	left := fgBold(iconColor).Render(icon) + " " + fg(colText).Render(text)
	if kind == noticeError {
		left = fgBold(iconColor).Render(icon) + " " + fg(colRed).Render(text)
	} else if kind == noticeWarn {
		left = fgBold(iconColor).Render(icon) + " " + fg(colAmber).Render(text)
	} else if kind == noticeSuccess {
		left = fgBold(iconColor).Render(icon) + " " + fg(colMuted).Render(text)
	} else {
		left = fgBold(iconColor).Render(icon) + " " + fg(colMuted).Render(text)
	}

	right := ""
	if width >= 92 && !m.hexEditing {
		c := m.picker.RGBA()
		right = miniSwatch(c) + " " + fgBold(colText).Render(c.Hex()) +
			fg(colDim).Render("  A ") + fg(colMuted).Render(fmt.Sprintf("%3d", c.A))
	}
	var line string
	if right == "" {
		if displayWidth(left) > width {
			left = truncateANSIWithTail(left, width)
		}
		line = left
	} else {
		avail := width - displayWidth(right) - 1
		if avail < 0 {
			avail = 0
		}
		if displayWidth(left) > avail {
			left = truncateANSIWithTail(left, avail)
		}
		line = spreadKeepRight(left, right, width)
	}
	return paintBG(line, colBG, width)
}

func (m Model) contextStatus(width int) (string, noticeKind) {
	if m.notice != "" {
		return m.notice, m.noticeKind
	}
	if m.applying {
		return "Applying color…", noticeInfo
	}
	if !m.statusKnown {
		return "Scanning for a DualShock 4 lightbar…", noticeInfo
	}
	if m.status.LEDBinary == "" {
		return "ds4-led was not found in PATH.", noticeError
	}
	if !m.status.LEDClass {
		if width < 64 {
			return "No /sys/class/leds · kernel LED support missing.", noticeError
		}
		return "Kernel LED class is unavailable. Run --doctor; the sysfs backend cannot work on this kernel.", noticeError
	}
	if !m.status.Found {
		if m.status.ControllerSeen {
			if width < 64 {
				return "DS4 connected · no LED entries · load hid-sony, then press r.", noticeWarn
			}
			return "DS4 is connected, but the kernel exposed no LED entries. Load hid-sony, then press r.", noticeWarn
		}
		if width < 64 {
			return "No DS4 connected · connect it over USB/Bluetooth, then press r.", noticeWarn
		}
		return "No DualShock 4 detected — connect it over USB or Bluetooth, then press r.", noticeWarn
	}
	if !m.status.Writable {
		switch {
		case width >= 90:
			return "Read-only LEDs — apply asks for sudo. Run 'make setup-led' once to remove that.", noticeWarn
		case width >= 60:
			return "Read-only LEDs — apply asks for sudo. See README for the one-time setup.", noticeWarn
		default:
			return "Read-only LEDs · apply asks for sudo.", noticeWarn
		}
	}

	switch {
	case m.help:
		offset, visible, total, _ := m.helpViewport()
		if total > visible {
			if width < 64 {
				return fmt.Sprintf("Help %d-%d of %d · ↑↓ scroll", offset+1, offset+visible, total), noticeInfo
			}
			return fmt.Sprintf("Help %d-%d of %d · ↑↓ scroll · pgup/pgdown page · esc closes.", offset+1, offset+visible, total), noticeInfo
		}
		return "Every shortcut in one place — esc closes this panel.", noticeInfo
	case m.screen == screenPresets:
		cols, _, _, rows := m.presetGrid()
		visible := cols * rows
		page := presetPageLabel(m.presetOffset, len(m.presets), visible)
		switch {
		case width >= 76:
			return fmt.Sprintf("Presets %s — enter applies the selection, space previews it in the picker.", page), noticeInfo
		case width >= 56:
			return fmt.Sprintf("Presets %s · enter apply · space preview", page), noticeInfo
		default:
			return fmt.Sprintf("Presets %s · enter applies", page), noticeInfo
		}
	case m.screen == screenRainbow:
		if m.rainbowRunning {
			if width < 56 {
				return "Rainbow is painting · space stops.", noticeInfo
			}
			return "The rainbow is painting the lightbar — space stops it, esc returns to the picker.", noticeInfo
		}
		if width < 56 {
			return "Space starts the rainbow.", noticeInfo
		}
		return "Press space to start the rainbow. It overrides static colors until stopped.", noticeInfo
	case m.rainbowRunning:
		return "Rainbow active — press w, then space to stop it before applying a static color.", noticeInfo
	case m.status.KeeperPID > 0:
		if width < 64 {
			return fmt.Sprintf("Color keeper running · PID %d.", m.status.KeeperPID), noticeSuccess
		}
		return fmt.Sprintf("Color keeper is running (PID %d) — this color stays applied when you quit.", m.status.KeeperPID), noticeSuccess
	default:
		if width >= 92 {
			return "Live preview only — press a or enter to send this RGBA color to the lightbar.", noticeInfo
		}
		if width >= 56 {
			return "Live preview only — press a to apply this color.", noticeInfo
		}
		return "Live preview · a applies.", noticeInfo
	}
}

func noticeIcon(kind noticeKind) (string, string) {
	switch kind {
	case noticeSuccess:
		return "✓", colGreen
	case noticeWarn:
		return "▲", colAmber
	case noticeError:
		return "!", colRed
	default:
		return "·", colAccent2
	}
}

func (m Model) hexStatusLine(width int) string {
	prompt := fgBold(colAccent2).Render("HEX") + " " +
		fgBold(colText).Render(m.hexInput)
	if len([]rune(m.hexInput)) < 9 {
		prompt += fgBold(colAccent2).Render("▏")
	}
	if m.hexErr != "" {
		prompt += fg(colRed).Render("  " + m.hexErr)
	}
	right := ""
	if width >= 74 {
		right = fg(colDim).Render("#RRGGBB or #RRGGBBAA · enter previews · esc cancels")
	}
	var line string
	if right == "" {
		if displayWidth(prompt) > width {
			prompt = truncateANSIWithTail(prompt, width)
		}
		line = prompt
	} else {
		avail := width - displayWidth(right) - 1
		if avail < 0 {
			avail = 0
		}
		if displayWidth(prompt) > avail {
			prompt = truncateANSIWithTail(prompt, avail)
		}
		line = spreadKeepRight(prompt, right, width)
	}
	return paintBG(line, colBG, width)
}

type keyHint struct {
	key   string
	label string
}

func (m Model) keyBar(width int) string {
	var hints []keyHint

	switch {
	case m.hexEditing:
		hints = []keyHint{
			{"enter", "preview"},
			{"esc", "cancel"},
			{"^u", "clear"},
			{"q", "quit"},
		}
	case m.help:
		hints = []keyHint{
			{"esc", "close"},
			{"↑↓", "scroll"},
			{"c p w", "switch view"},
			{"q", "quit"},
		}
	case m.screen == screenPresets:
		hints = []keyHint{
			{"enter", "apply"},
			{"←↑↓→", "move"},
			{"pgup/pgdn", "page"},
			{"space", "preview"},
			{"esc", "back"},
			{"? ", "help"},
			{"q", "quit"},
		}
	case m.screen == screenRainbow:
		hints = []keyHint{
			{"space", "start / stop"},
			{"esc", "back"},
			{"? ", "help"},
			{"q", "quit"},
		}
	default:
		hints = []keyHint{
			{"a", "apply"},
			{"tab", "focus"},
			{"←→", "adjust"},
			{"1-4", "channel"},
			{"e", "hex"},
			{"? ", "help"},
			{"q", "quit"},
		}
	}

	if width < 46 {
		switch {
		case m.hexEditing:
			hints = []keyHint{{"enter", "preview"}, {"esc", "cancel"}, {"q", "quit"}}
		case m.help:
			hints = []keyHint{{"↑↓", "scroll"}, {"esc", "close"}, {"q", "quit"}}
		case m.screen == screenPresets:
			hints = []keyHint{{"enter", "apply"}, {"?", "help"}, {"q", "quit"}}
		case m.screen == screenRainbow:
			hints = []keyHint{{"space", "toggle"}, {"?", "help"}, {"q", "quit"}}
		default:
			hints = []keyHint{{"a", "apply"}, {"?", "help"}, {"q", "quit"}}
		}
	}

	render := func(hints []keyHint) string {
		parts := make([]string, 0, len(hints))
		for _, h := range hints {
			key := strings.TrimSpace(h.key)
			parts = append(parts,
				fgBold(colAccent2).Render(key)+" "+fg(colMuted).Render(h.label))
		}
		return strings.Join(parts, fg(colBorderHi).Render(" · "))
	}

	for len(hints) > 1 && displayWidth(render(hints)) > width {
		// Keep the first hint and the quit hint, drop the least critical
		// middle entry until the bar fits.
		q := hints[len(hints)-1]
		hints = append(hints[:len(hints)-2], q)
	}
	line := render(hints)
	if displayWidth(line) > width {
		line = truncateANSI(line, width)
	}
	return paintBG(line, colBG, width)
}

func (m Model) tinyView(l layoutInfo) string {
	w, h := m.width, m.height
	if w <= 0 || h <= 0 {
		return ""
	}
	if w < 8 || h < 2 {
		label := "◆"
		if w >= 4 {
			label = "ds4"
		}
		if w >= 20 {
			label = "◆ DUALSHINE · small"
		}
		lines := make([]string, h)
		mid := h / 2
		for i := range lines {
			if i == mid {
				lines[i] = paintBG(center(label, w), colBG, w)
			} else {
				lines[i] = blankLine(w, colBG)
			}
		}
		return strings.Join(lines, "\n")
	}

	lines := []string{
		fgBold(colAccent2).Render("◆") + " " +
			gradient("DUALSHINE", []string{colText, colAccent, colPink, colAccent2}),
		fgBold(colAmber).Render("Terminal too small"),
		fg(colMuted).Render(fmt.Sprintf("Need at least %d × %d", minWidth, minHeight)),
		fg(colDim).Render(fmt.Sprintf("Now %d × %d", w, h)),
	}
	if w < 30 {
		lines = []string{
			gradient("DUALSHINE", []string{colText, colAccent, colPink}),
			fgBold(colAmber).Render("too small"),
			fg(colDim).Render(fmt.Sprintf("%dx%d", w, h)),
		}
	}

	content := make([]string, 0, h)
	for _, line := range lines {
		content = append(content, center(line, w))
	}
	top := (h - len(content)) / 2
	for i := 0; i < top; i++ {
		content = append([]string{""}, content...)
	}
	for len(content) < h {
		content = append(content, "")
	}
	if len(content) > h {
		content = content[:h]
	}
	for i := range content {
		content[i] = paintBG(content[i], colBG, w)
	}
	return strings.Join(content, "\n")
}

// min/max helpers used by the layout and widgets.
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
