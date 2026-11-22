package tui

import (
	"context"
	"time"

	"github.com/blazar/ds4-tui/internal/color"
	"github.com/blazar/ds4-tui/internal/ds4"
	"github.com/blazar/ds4-tui/internal/picker"
	tea "github.com/charmbracelet/bubbletea"
)

type controllerMsg struct{ status ds4.Status }

type statusPollMsg struct{}

type applyDoneMsg struct {
	color  color.RGBA
	output string
	err    error
}

type execApplyDoneMsg struct {
	color color.RGBA
	err   error
}

type rainbowStartedMsg struct {
	handle ds4.Rainbow
	err    error
}

type rainbowExitedMsg struct {
	pid    int
	err    error
	output string
}

type rainbowAnimMsg struct{}

type rainbowAuthMsg struct{ err error }

func refreshStatusCmd(svc ds4.Service) tea.Cmd {
	return func() tea.Msg { return controllerMsg{status: svc.Detect()} }
}

func waitRainbowCmd(h ds4.Rainbow) tea.Cmd {
	return func() tea.Msg {
		err := h.Wait()
		return rainbowExitedMsg{pid: h.PID(), err: err, output: h.Output()}
	}
}

func startRainbowCmd(svc ds4.Service) tea.Cmd {
	return func() tea.Msg {
		h, err := svc.StartRainbow()
		return rainbowStartedMsg{handle: h, err: err}
	}
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	if key == "ctrl+c" {
		return m.quit()
	}

	if m.hexEditing {
		m.handleHexKey(msg)
		return m, nil
	}

	if m.help {
		switch key {
		case "?", "esc", "enter":
			m.help = false
		case "c":
			m.switchScreen(screenColor)
		case "p":
			m.switchScreen(screenPresets)
		case "w":
			m.switchScreen(screenRainbow)
		case "q":
			return m.quit()
		case "up":
			m.helpOffset--
			m.clampHelpOffset()
		case "down":
			m.helpOffset++
			m.clampHelpOffset()
		case "pgup":
			_, visible, _, _ := m.helpViewport()
			m.helpOffset -= maxInt(1, visible-2)
			m.clampHelpOffset()
		case "pgdown":
			_, visible, _, _ := m.helpViewport()
			m.helpOffset += maxInt(1, visible-2)
			m.clampHelpOffset()
		case "home":
			m.helpOffset = 0
		case "end":
			_, _, _, maxOffset := m.helpViewport()
			m.helpOffset = maxOffset
		}
		return m, nil
	}

	switch key {
	case "q":
		return m.quit()
	case "?":
		m.help = true
		m.helpOffset = 0
		return m, nil
	case "r":
		return m, m.rescan()
	case "esc":
		if m.screen != screenColor {
			m.switchScreen(screenColor)
		}
		return m, nil
	case "c":
		m.switchScreen(screenColor)
		return m, nil
	case "p":
		m.switchScreen(screenPresets)
		return m, nil
	case "w":
		m.switchScreen(screenRainbow)
		return m, nil
	}

	switch m.screen {
	case screenColor:
		return m.updateColor(key, msg)
	case screenPresets:
		return m.updatePresets(key)
	case screenRainbow:
		return m.updateRainbow(key)
	}
	return m, nil
}

func (m *Model) switchScreen(s screen) {
	m.screen = s
	m.help = false
	if s == screenPresets && !m.presetInit {
		m.presetCursor = m.closestPreset()
		m.presetInit = true
		m.clampPresetWindow()
	}
}

func (m Model) updateColor(key string, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key {
	case "tab":
		m.cycleFocus(1)
		return m, nil
	case "shift+tab":
		m.cycleFocus(-1)
		return m, nil
	case "1":
		m.focus = focusRed
		return m, nil
	case "2":
		m.focus = focusGreen
		return m, nil
	case "3":
		m.focus = focusBlue
		return m, nil
	case "4":
		m.focus = focusAlpha
		return m, nil
	case "h":
		m.focus = focusHue
		return m, nil
	case "s":
		m.focus = focusSV
		return m, nil
	case "e":
		m.beginHexEdit()
		return m, nil
	case "enter", "a":
		return m.applyCurrent()
	case "[":
		m.picker.NudgeHue(-1)
		m.colorTouched = true
		return m, nil
	case "]":
		m.picker.NudgeHue(1)
		m.colorTouched = true
		return m, nil
	case "-", "_":
		m.picker.NudgeAlpha(-1)
		m.colorTouched = true
		return m, nil
	case "=", "+":
		m.picker.NudgeAlpha(1)
		m.colorTouched = true
		return m, nil
	}

	switch key {
	case "left", "right", "up", "down":
		m.adjust(key, 1)
	case "pgup":
		m.adjust("right", 10)
	case "pgdown":
		m.adjust("left", 10)
	}
	return m, nil
}

func (m *Model) cycleFocus(delta int) {
	n := 6
	f := (int(m.focus) + delta) % n
	if f < 0 {
		f += n
	}
	m.focus = focus(f)
}

func (m *Model) adjust(dirKey string, step int) {
	delta := 1
	switch dirKey {
	case "left", "down":
		delta = -1
	}
	delta *= step

	if ch, ok := m.focus.channel(); ok {
		m.picker.NudgeChannel(ch, delta)
		m.colorTouched = true
		return
	}

	switch m.focus {
	case focusHue:
		m.picker.NudgeHue(float64(delta))
	case focusSV:
		if dirKey == "left" || dirKey == "right" || step > 1 {
			m.picker.NudgeSV(float64(delta)*0.01, 0)
		} else {
			m.picker.NudgeSV(0, float64(delta)*0.01)
		}
	}
	m.colorTouched = true
}

func (m *Model) beginHexEdit() {
	m.hexEditing = true
	m.hexInput = "#"
	m.hexErr = ""
}

func (m *Model) handleHexKey(msg tea.KeyMsg) {
	key := msg.String()
	switch key {
	case "esc":
		m.hexEditing = false
		m.hexErr = ""
	case "enter":
		c, err := color.ParseHex(m.hexInput)
		if err != nil {
			m.hexErr = err.Error()
			return
		}
		m.picker.SetRGBA(c)
		m.colorTouched = true
		m.hexEditing = false
		m.hexErr = ""
		m.setNotice(noticeInfo, "Loaded "+c.HexA()+" into the picker — preview only")
	case "backspace":
		runes := []rune(m.hexInput)
		if len(runes) > 1 {
			m.hexInput = string(runes[:len(runes)-1])
		}
	case "ctrl+u":
		m.hexInput = "#"
		m.hexErr = ""
	default:
		if msg.Type != tea.KeyRunes {
			return
		}
		if len([]rune(m.hexInput)) >= 9 {
			return
		}
		for _, r := range msg.Runes {
			switch {
			case r == '#':
				// The input always starts with '#', so ignore extra hashes.
			case (r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F'):
				m.hexInput += string(r)
			}
		}
		if len([]rune(m.hexInput)) > 9 {
			m.hexInput = string([]rune(m.hexInput)[:9])
		}
		m.hexErr = ""
	}
}

func (m Model) applyCurrent() (tea.Model, tea.Cmd) {
	if m.applying {
		m.setNotice(noticeInfo, "Already applying a color…")
		return m, nil
	}
	if !m.statusKnown {
		m.setNotice(noticeWarn, "Still scanning for a controller — try again in a moment.")
		return m, nil
	}
	if !m.status.LEDClass {
		m.setNotice(noticeError, "Kernel LED class unavailable. Run --doctor; the sysfs backend cannot work.")
		return m, nil
	}
	if !m.status.Found {
		m.setNotice(noticeError, m.status.Err+"; press r after fixing.")
		return m, nil
	}
	if m.status.LEDBinary == "" {
		m.setNotice(noticeError, "ds4-led was not found in PATH.")
		return m, nil
	}
	if m.rainbowStarting {
		m.setNotice(noticeWarn, "Rainbow is still starting — stop it before applying a static color.")
		return m, nil
	}

	c := m.picker.RGBA()
	if m.rainbowRunning {
		m.rainbowStopping = true
		m.stopRainbow()
	}
	m.applying = true
	m.setNotice(noticeInfo, "Applying "+c.Hex()+" · "+alphaSummary(c.A)+"…")

	// sysfs LEDs are often root-owned. ds4-led knows how to elevate; running
	// it through ExecProcess gives sudo a real terminal to prompt on.
	if !m.status.Writable && !m.demo {
		cmd := m.svc.ColorCommand(c)
		if cmd == nil {
			m.applying = false
			m.setNotice(noticeError, "ds4-led was not found in PATH.")
			return m, nil
		}
		return m, tea.ExecProcess(cmd, func(err error) tea.Msg {
			return execApplyDoneMsg{color: c, err: err}
		})
	}

	return m, m.applyCmd(c)
}

func (m Model) applyCmd(c color.RGBA) tea.Cmd {
	svc := m.svc
	rainbow := m.rainbow
	return func() tea.Msg {
		if rainbow != nil {
			_ = rainbow.Stop()
			time.Sleep(120 * time.Millisecond)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		out, err := svc.Apply(ctx, c)
		return applyDoneMsg{color: c, output: out, err: err}
	}
}

func (m Model) updatePresets(key string) (tea.Model, tea.Cmd) {
	cols, _, _, rows := m.presetGrid()
	page := cols * rows
	if page < 1 {
		page = 1
	}

	switch key {
	case "left":
		m.presetCursor--
	case "right":
		m.presetCursor++
	case "up":
		m.presetCursor -= cols
	case "down":
		m.presetCursor += cols
	case "pgup":
		m.presetCursor -= page
	case "pgdown":
		m.presetCursor += page
	case "home":
		m.presetCursor = 0
	case "end":
		m.presetCursor = len(m.presets) - 1
	case "enter", "a":
		return m.applyPreset()
	case " ", "space":
		return m.previewPreset()
	default:
		return m, nil
	}

	if m.presetCursor < 0 {
		m.presetCursor = 0
	}
	if m.presetCursor >= len(m.presets) {
		m.presetCursor = len(m.presets) - 1
	}
	m.clampPresetWindow()
	return m, nil
}

func (m Model) applyPreset() (tea.Model, tea.Cmd) {
	if len(m.presets) == 0 {
		return m, nil
	}
	p := m.presets[m.presetCursor]
	m.picker = picker.FromRGBA(p.RGBA)
	m.colorTouched = true
	m.screen = screenColor
	m.setNotice(noticeInfo, "Preset "+p.Name+" selected")
	return m.applyCurrent()
}

func (m Model) previewPreset() (tea.Model, tea.Cmd) {
	if len(m.presets) == 0 {
		return m, nil
	}
	p := m.presets[m.presetCursor]
	m.picker = picker.FromRGBA(p.RGBA)
	m.colorTouched = true
	m.screen = screenColor
	m.setNotice(noticeInfo, "Previewing "+p.Name+" ("+p.RGBA.Hex()+") — press a to apply")
	return m, nil
}

func (m Model) updateRainbow(key string) (tea.Model, tea.Cmd) {
	switch key {
	case " ", "space", "enter":
		return m.toggleRainbow()
	case "a":
		m.switchScreen(screenColor)
		return m.applyCurrent()
	}
	return m, nil
}

func (m Model) toggleRainbow() (tea.Model, tea.Cmd) {
	if m.rainbowStarting {
		return m, nil
	}
	if m.applying {
		m.setNotice(noticeInfo, "Finish applying the static color first.")
		return m, nil
	}
	if m.rainbowRunning {
		m.rainbowStopping = true
		m.stopRainbow()
		m.setNotice(noticeInfo, "Stopping the rainbow…")
		return m, nil
	}
	if !m.statusKnown {
		m.setNotice(noticeWarn, "Still scanning for a controller — try again in a moment.")
		return m, nil
	}
	if !m.status.LEDClass {
		m.setNotice(noticeError, "Kernel LED class unavailable. Run --doctor; the sysfs backend cannot work.")
		return m, nil
	}
	if !m.status.Found {
		m.setNotice(noticeError, m.status.Err+"; press r after fixing.")
		return m, nil
	}
	if m.status.Writable && m.status.RainbowBinary == "" {
		m.setNotice(noticeError, "ds4-rb was not found in PATH.")
		return m, nil
	}
	// Root-owned LEDs need one sudo authentication before the privileged
	// rainbow helper can start. ExecProcess gives sudo the real terminal.
	if !m.status.Writable && !m.demo {
		cmd := m.svc.AuthCommand()
		if cmd == nil {
			m.setNotice(noticeError, "Rainbow needs writable LEDs or sudo; neither is available.")
			return m, nil
		}
		m.rainbowStarting = true
		m.setNotice(noticeInfo, "Sudo is needed for the rainbow. Authenticate in the terminal…")
		return m, tea.ExecProcess(cmd, func(err error) tea.Msg {
			return rainbowAuthMsg{err: err}
		})
	}
	m.rainbowStarting = true
	m.rainbowErr = ""
	m.setNotice(noticeInfo, "Starting rainbow…")
	return m, startRainbowCmd(m.svc)
}

func (m Model) closestPreset() int {
	want := m.picker.RGBA()
	best := 0
	bestD := 1 << 30
	for i, p := range m.presets {
		dr := int(want.R) - int(p.RGBA.R)
		dg := int(want.G) - int(p.RGBA.G)
		db := int(want.B) - int(p.RGBA.B)
		d := dr*dr + dg*dg + db*db
		if d < bestD {
			bestD = d
			best = i
		}
	}
	return best
}

func (m *Model) clampPresetWindow() {
	cols, _, _, rows := m.presetGrid()
	if cols < 1 || rows < 1 || len(m.presets) == 0 {
		m.presetOffset = 0
		m.presetCursor = 0
		return
	}
	page := cols * rows
	if m.presetCursor < 0 {
		m.presetCursor = 0
	}
	if m.presetCursor >= len(m.presets) {
		m.presetCursor = len(m.presets) - 1
	}
	if len(m.presets) <= page {
		m.presetOffset = 0
		return
	}
	if m.presetCursor < m.presetOffset {
		m.presetOffset = m.presetCursor / cols * cols
	}
	for m.presetCursor >= m.presetOffset+page {
		m.presetOffset += cols
	}
	maxOffset := ((len(m.presets)-1)/cols - (rows - 1)) * cols
	if maxOffset < 0 {
		maxOffset = 0
	}
	if m.presetOffset > maxOffset {
		m.presetOffset = maxOffset
	}
	if m.presetOffset < 0 {
		m.presetOffset = 0
	}
	m.presetOffset = m.presetOffset / cols * cols
}
