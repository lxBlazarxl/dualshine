package tui

import (
	"errors"
	"os/exec"
	"testing"

	"github.com/blazar/ds4-tui/internal/color"
	"github.com/blazar/ds4-tui/internal/ds4"
	"github.com/blazar/ds4-tui/internal/picker"
	tea "github.com/charmbracelet/bubbletea"
)

func send(t *testing.T, m Model, key string) (Model, tea.Cmd) {
	t.Helper()
	next, cmd := m.Update(keyMsg(key))
	nm, ok := next.(Model)
	if !ok {
		t.Fatalf("Update returned %T, want Model", next)
	}
	return nm, cmd
}

func TestPickerAdjustments(t *testing.T) {
	m := newTestModel(100, 30)
	m.focus = focusRed
	m.picker = picker.FromRGBA(color.RGBA{R: 10, G: 20, B: 30, A: 128})

	m, _ = send(t, m, "right")
	if got := m.picker.ChannelValue(picker.Red); got != 11 {
		t.Fatalf("red after right = %d", got)
	}
	m, _ = send(t, m, "left")
	if got := m.picker.ChannelValue(picker.Red); got != 10 {
		t.Fatalf("red after left = %d", got)
	}

	m.focus = focusGreen
	m, _ = send(t, m, "pgup")
	if got := m.picker.ChannelValue(picker.Green); got != 30 {
		t.Fatalf("green after pgup = %d", got)
	}

	m.focus = focusHue
	before := m.picker.Hue
	m, _ = send(t, m, "]")
	if m.picker.Hue != before+1 {
		t.Fatalf("hue = %v, want %v", m.picker.Hue, before+1)
	}

	m.focus = focusAlpha
	m, _ = send(t, m, "=")
	if got := m.picker.ChannelValue(picker.Alpha); got != 129 {
		t.Fatalf("alpha = %d", got)
	}

	m.focus = focusSV
	val := m.picker.Val
	m, _ = send(t, m, "up")
	if m.picker.Val <= val {
		t.Fatalf("value did not increase: %v -> %v", val, m.picker.Val)
	}
}

func TestTabCyclesFocus(t *testing.T) {
	m := newTestModel(100, 30)
	if m.focus != focusSV {
		t.Fatalf("focus = %v", m.focus)
	}
	for i, want := range []focus{focusHue, focusRed, focusGreen, focusBlue, focusAlpha, focusSV} {
		m, _ = send(t, m, "tab")
		if m.focus != want {
			t.Fatalf("tab %d: focus = %v, want %v", i, m.focus, want)
		}
	}
	m, _ = send(t, m, "shift+tab")
	if m.focus != focusAlpha {
		t.Fatalf("shift+tab focus = %v", m.focus)
	}
}

func TestHexEditor(t *testing.T) {
	m := newTestModel(100, 30)
	m, _ = send(t, m, "e")
	if !m.hexEditing {
		t.Fatal("hex editor did not open")
	}
	m, _ = send(t, m, "FF8000")
	m, _ = send(t, m, "enter")
	if m.hexEditing {
		t.Fatal("hex editor did not close")
	}
	if got := m.picker.Hex(); got != "#FF8000" {
		t.Fatalf("hex = %s", got)
	}
	if m.hexErr != "" {
		t.Fatalf("unexpected error %q", m.hexErr)
	}

	// Invalid input keeps the editor open.
	m, _ = send(t, m, "e")
	m, _ = send(t, m, "GG")
	m, _ = send(t, m, "enter")
	if !m.hexEditing {
		t.Fatal("invalid hex should keep editor open")
	}
	if m.hexErr == "" {
		t.Fatal("expected an error")
	}
	m, _ = send(t, m, "esc")
	if m.hexEditing {
		t.Fatal("esc should close editor")
	}

	// q must be typed as a character while editing, not quit the app.
	m, _ = send(t, m, "e")
	m, cmd := send(t, m, "q")
	if cmd != nil || !m.hexEditing {
		t.Fatalf("q in hex editor should be ignored (cmd=%v editing=%v)", cmd, m.hexEditing)
	}
}

func TestApplyColor(t *testing.T) {
	m := newTestModel(100, 30)
	m.picker = picker.FromRGBA(color.RGBA{R: 12, G: 34, B: 56, A: 200})
	m, cmd := send(t, m, "a")
	if cmd == nil {
		t.Fatal("expected apply command")
	}
	if !m.applying {
		t.Fatal("model should be applying")
	}
	msg := cmd()
	m = updateMsg(t, m, msg)
	if m.applying {
		t.Fatal("still applying after result")
	}
	if m.noticeKind != noticeSuccess {
		t.Fatalf("notice kind = %v (%s)", m.noticeKind, m.notice)
	}
	if got := m.svc.Detect().Current; got != m.picker.RGBA() {
		t.Fatalf("demo controller = %+v, picker = %+v", got, m.picker.RGBA())
	}
}

func TestApplyWithoutController(t *testing.T) {
	m := newTestModel(80, 24)
	m.status.Found = false
	m, cmd := send(t, m, "a")
	if cmd != nil {
		t.Fatal("expected no command without a controller")
	}
	if m.noticeKind != noticeError {
		t.Fatalf("notice kind = %v", m.noticeKind)
	}
	if m.applying {
		t.Fatal("should not enter applying state")
	}
}

func TestPresetNavigation(t *testing.T) {
	m := newTestModel(120, 36)
	m, _ = send(t, m, "p")
	if m.screen != screenPresets {
		t.Fatal("did not switch to presets")
	}
	if got := m.presets[m.presetCursor].Name; got != "Violet" {
		t.Fatalf("closest preset = %q, want Violet", got)
	}
	m, _ = send(t, m, "end")
	if m.presetCursor != len(m.presets)-1 {
		t.Fatalf("cursor = %d", m.presetCursor)
	}
	m, _ = send(t, m, "home")
	if m.presetCursor != 0 {
		t.Fatalf("cursor = %d", m.presetCursor)
	}
	m, _ = send(t, m, "down")
	if m.presetCursor != 4 {
		t.Fatalf("down cursor = %d, want 4", m.presetCursor)
	}

	m, _ = send(t, m, "space")
	if m.screen != screenColor {
		t.Fatal("space should preview in the color screen")
	}
	if got := m.picker.RGBA(); got != m.presets[4].RGBA {
		t.Fatalf("picker = %+v, preset = %+v", got, m.presets[4].RGBA)
	}
}

func TestPresetApply(t *testing.T) {
	m := newTestModel(120, 36)
	m, _ = send(t, m, "p")
	m, _ = send(t, m, "right")
	want := m.presets[m.presetCursor].RGBA
	m, cmd := send(t, m, "enter")
	if cmd == nil {
		t.Fatal("expected apply command")
	}
	if m.screen != screenColor {
		t.Fatal("apply should show the picker")
	}
	if got := m.picker.RGBA(); got != want {
		t.Fatalf("picker = %+v, want %+v", got, want)
	}
}

func TestRainbowLifecycle(t *testing.T) {
	m := newTestModel(120, 36)
	m, _ = send(t, m, "w")
	if m.screen != screenRainbow {
		t.Fatal("did not switch to rainbow")
	}
	m, cmd := send(t, m, "space")
	if cmd == nil || !m.rainbowStarting {
		t.Fatalf("start cmd=%v starting=%v", cmd, m.rainbowStarting)
	}
	startMsg := cmd()
	m = updateMsg(t, m, startMsg)
	if !m.rainbowRunning || m.rainbow == nil {
		t.Fatalf("rainbow did not start: %+v", m.rainbowStartedMsgDebug())
	}

	handle := m.rainbow
	m, _ = send(t, m, "space")
	if !m.rainbowStopping {
		t.Fatal("rainbow should be stopping")
	}
	m = updateMsg(t, m, waitRainbowCmd(handle)())
	if m.rainbowRunning || m.rainbow != nil {
		t.Fatal("rainbow did not stop")
	}
	if m.rainbowStopping {
		t.Fatal("stopping flag not cleared")
	}
}

func (m Model) rainbowStartedMsgDebug() string {
	if m.rainbow == nil {
		return "nil handle"
	}
	return "handle present"
}

func TestHelpAndScreenSwitching(t *testing.T) {
	m := newTestModel(100, 30)
	m, _ = send(t, m, "?")
	if !m.help {
		t.Fatal("help did not open")
	}
	m, _ = send(t, m, "?")
	if m.help {
		t.Fatal("help did not close")
	}
	m, _ = send(t, m, "w")
	if m.screen != screenRainbow {
		t.Fatal("w did not switch")
	}
	m, _ = send(t, m, "esc")
	if m.screen != screenColor {
		t.Fatal("esc did not return to picker")
	}
	m, _ = send(t, m, "p")
	m, _ = send(t, m, "c")
	if m.screen != screenColor {
		t.Fatal("c did not switch back")
	}
}

func TestWindowSizeAndClamping(t *testing.T) {
	m := newTestModel(120, 36)
	m, _ = send(t, m, "p")
	m, _ = send(t, m, "end")
	next, _ := m.Update(tea.WindowSizeMsg{Width: 40, Height: 14})
	m = next.(Model)
	if m.width != 40 || m.height != 14 {
		t.Fatalf("size = %dx%d", m.width, m.height)
	}
	if m.presetOffset < 0 || m.presetCursor >= len(m.presets) {
		t.Fatalf("preset state out of range: cursor=%d offset=%d", m.presetCursor, m.presetOffset)
	}
	assertCanvas(t, m.View(), 40, 14, "after resize")
}

func TestStatusUpdatesUntouchedPicker(t *testing.T) {
	m := New(ds4.NewDemo())
	m.width, m.height = 100, 30
	c := color.New(1, 2, 3)
	m = updateMsg(t, m, controllerMsg{status: ds4.Status{
		Found: true, Writable: true, Current: c, CurrentOK: true,
	}})
	if got := m.picker.RGBA(); got != c {
		t.Fatalf("picker = %+v, want %+v", got, c)
	}
	// After user edits, a poll must not snap the picker back.
	m, _ = send(t, m, "right")
	edited := m.picker.RGBA()
	m = updateMsg(t, m, controllerMsg{status: ds4.Status{
		Found: true, Writable: true, Current: color.New(9, 9, 9), CurrentOK: true,
	}})
	if got := m.picker.RGBA(); got != edited {
		t.Fatalf("picker snapped back to %+v (was %+v)", got, edited)
	}
}

func updateMsg(t *testing.T, m Model, msg tea.Msg) Model {
	t.Helper()
	next, _ := m.Update(msg)
	nm, ok := next.(Model)
	if !ok {
		t.Fatalf("Update returned %T", next)
	}
	return nm
}

func TestHelpScrolling(t *testing.T) {
	m := newTestModel(40, 14)
	m, _ = send(t, m, "?")
	if !m.help {
		t.Fatal("help did not open")
	}
	_, visible, total, maxOffset := m.helpViewport()
	if total <= visible {
		t.Fatalf("expected help to overflow: total=%d visible=%d", total, visible)
	}
	if maxOffset <= 0 {
		t.Fatal("expected a positive max offset")
	}
	m, _ = send(t, m, "down")
	if m.helpOffset != 1 {
		t.Fatalf("offset after down = %d", m.helpOffset)
	}
	m, _ = send(t, m, "end")
	if m.helpOffset != maxOffset {
		t.Fatalf("offset after end = %d, want %d", m.helpOffset, maxOffset)
	}
	m, _ = send(t, m, "home")
	if m.helpOffset != 0 {
		t.Fatalf("offset after home = %d", m.helpOffset)
	}
	m, _ = send(t, m, "esc")
	if m.help {
		t.Fatal("esc did not close help")
	}
}

type authFakeService struct {
	*ds4.Demo
}

func (s *authFakeService) Detect() ds4.Status {
	st := s.Demo.Detect()
	st.Writable = false
	st.KeeperPID = 0
	return st
}

func (s *authFakeService) AuthCommand() *exec.Cmd {
	return exec.Command("true")
}

func TestRainbowAsksForSudoWhenReadOnly(t *testing.T) {
	svc := &authFakeService{Demo: ds4.NewDemo()}
	m := New(svc)
	m.width, m.height = 120, 36
	st := svc.Detect()
	m.status = st
	m.statusKnown = true
	m.picker = picker.FromRGBA(st.Current)

	m, _ = send(t, m, "w")
	m, cmd := send(t, m, "space")
	if cmd == nil {
		t.Fatal("expected an auth command")
	}
	if !m.rainbowStarting || m.rainbowRunning {
		t.Fatalf("starting=%v running=%v", m.rainbowStarting, m.rainbowRunning)
	}

	m = updateMsg(t, m, rainbowAuthMsg{})
	if !m.rainbowStarting {
		t.Fatal("auth success should still be starting")
	}
	start, _ := m.Update(startRainbowCmd(m.svc)())
	next, ok := start.(Model)
	if !ok {
		t.Fatal("bad model type")
	}
	m = next
	if !m.rainbowRunning || m.rainbow == nil {
		t.Fatalf("rainbow did not start after auth: starting=%v", m.rainbowStarting)
	}
	if m.noticeKind != noticeSuccess {
		t.Fatalf("notice kind = %v: %s", m.noticeKind, m.notice)
	}

	// Stop it so the test leaves no live control object.
	handle := m.rainbow
	m, _ = send(t, m, "space")
	m = updateMsg(t, m, waitRainbowCmd(handle)())
	if m.rainbowRunning {
		t.Fatal("rainbow did not stop")
	}
}

func TestRainbowAuthFailure(t *testing.T) {
	svc := &authFakeService{Demo: ds4.NewDemo()}
	m := New(svc)
	m.width, m.height = 100, 30
	m.status = svc.Detect()
	m.statusKnown = true

	m, _ = send(t, m, "w")
	m, _ = send(t, m, "space")
	m = updateMsg(t, m, rainbowAuthMsg{err: errors.New("cancelled")})
	if m.rainbowStarting {
		t.Fatal("rainbow should not be starting after auth failure")
	}
	if m.noticeKind != noticeError {
		t.Fatalf("notice kind = %v", m.noticeKind)
	}
}
