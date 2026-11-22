package tui

import (
	"strconv"
	"time"

	"github.com/blazar/ds4-tui/internal/color"
	"github.com/blazar/ds4-tui/internal/ds4"
	"github.com/blazar/ds4-tui/internal/picker"
	tea "github.com/charmbracelet/bubbletea"
)

type screen int

const (
	screenColor screen = iota
	screenPresets
	screenRainbow
)

type focus int

const (
	focusSV focus = iota
	focusHue
	focusRed
	focusGreen
	focusBlue
	focusAlpha
)

func (f focus) channel() (picker.Channel, bool) {
	switch f {
	case focusRed:
		return picker.Red, true
	case focusGreen:
		return picker.Green, true
	case focusBlue:
		return picker.Blue, true
	case focusAlpha:
		return picker.Alpha, true
	default:
		return 0, false
	}
}

type noticeKind int

const (
	noticeInfo noticeKind = iota
	noticeSuccess
	noticeWarn
	noticeError
)

// Model is the Bubble Tea application state.
type Model struct {
	svc  ds4.Service
	demo bool

	width, height int
	screen        screen
	help          bool
	helpOffset    int

	picker       picker.Model
	focus        focus
	colorTouched bool

	hexEditing bool
	hexInput   string
	hexErr     string

	status      ds4.Status
	statusKnown bool

	notice     string
	noticeKind noticeKind
	noticeAuto bool
	applying   bool

	presets      []Preset
	presetCursor int
	presetOffset int
	presetInit   bool

	rainbow         ds4.Rainbow
	rainbowRunning  bool
	rainbowStarting bool
	rainbowStopping bool
	rainbowPID      int
	rainbowFrame    int
	rainbowErr      string

	lastApplied color.RGBA
	hasApplied  bool
}

// New creates the application model. Init starts the first controller scan.
func New(svc ds4.Service) Model {
	_, isDemo := svc.(*ds4.Demo)
	return Model{
		svc:     svc,
		demo:    isDemo,
		picker:  picker.New(),
		focus:   focusSV,
		presets: defaultPresets,
	}
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return tea.Batch(refreshStatusCmd(m.svc), statusPollCmd())
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.clampPresetWindow()
		if m.help {
			m.clampHelpOffset()
		}
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)

	case controllerMsg:
		first := !m.statusKnown
		m.status = msg.status
		m.statusKnown = true
		if m.noticeAuto {
			m.notice = ""
			m.noticeAuto = false
		}
		if msg.status.CurrentOK {
			if first && !m.colorTouched {
				m.picker = picker.FromRGBA(msg.status.Current)
				m.lastApplied = msg.status.Current
				m.hasApplied = true
			}
			if !m.colorTouched {
				m.lastApplied = msg.status.Current
				m.hasApplied = true
			}
		}
		if first && !msg.status.Found && msg.status.Err != "" {
			m.setAutoNotice(noticeInfo, "No lightbar detected. Connect your controller, then press r.")
		}
		m.clampPresetWindow()
		return m, nil

	case statusPollMsg:
		return m, tea.Batch(refreshStatusCmd(m.svc), statusPollCmd())

	case applyDoneMsg:
		m.applying = false
		if msg.err != nil {
			text := msg.err.Error()
			if msg.output != "" {
				text = msg.output
			}
			m.setNotice(noticeError, "Could not apply color: "+compactError(text))
		} else {
			m.lastApplied = msg.color
			m.hasApplied = true
			m.colorTouched = false
			if m.demo {
				m.setNotice(noticeSuccess, "Demo: "+msg.color.Hex()+" · "+
					alphaSummary(msg.color.A)+" · nothing written")
			} else {
				m.setNotice(noticeSuccess, "Applied "+msg.color.Hex()+" · "+
					alphaSummary(msg.color.A)+" · keeping it alive")
			}
		}
		return m, refreshStatusCmd(m.svc)

	case execApplyDoneMsg:
		m.applying = false
		if msg.err != nil {
			m.setNotice(noticeError, "ds4-led did not finish cleanly. Check the terminal output above, then press a to retry.")
		} else {
			m.lastApplied = msg.color
			m.hasApplied = true
			m.colorTouched = false
			m.setNotice(noticeSuccess, "Applied "+msg.color.Hex()+" · "+alphaSummary(msg.color.A))
		}
		return m, refreshStatusCmd(m.svc)

	case rainbowAuthMsg:
		if msg.err != nil {
			m.rainbowStarting = false
			m.setNotice(noticeError, "Sudo authentication failed or was cancelled: "+compactError(msg.err.Error()))
			return m, nil
		}
		m.setNotice(noticeInfo, "Starting rainbow…")
		return m, startRainbowCmd(m.svc)

	case rainbowStartedMsg:
		m.rainbowStarting = false
		if msg.err != nil {
			m.rainbowRunning = false
			m.rainbow = nil
			m.rainbowErr = msg.err.Error()
			m.setNotice(noticeError, "Rainbow failed to start: "+compactError(msg.err.Error()))
			return m, nil
		}
		m.rainbow = msg.handle
		m.rainbowRunning = true
		m.rainbowStopping = false
		m.rainbowPID = msg.handle.PID()
		m.rainbowFrame = 0
		m.rainbowErr = ""
		m.setNotice(noticeSuccess, "Rainbow started · painting the lightbar")
		return m, tea.Batch(waitRainbowCmd(msg.handle), rainbowAnimCmd(), refreshStatusCmd(m.svc))

	case rainbowExitedMsg:
		if m.rainbow != nil && msg.pid != 0 && msg.pid != m.rainbow.PID() {
			return m, nil
		}
		wasStopping := m.rainbowStopping
		m.rainbowRunning = false
		m.rainbowStarting = false
		m.rainbowStopping = false
		m.rainbow = nil
		m.rainbowPID = 0
		if msg.err != nil && !wasStopping {
			out := msg.output
			if out == "" {
				out = msg.err.Error()
			}
			m.rainbowErr = out
			m.setNotice(noticeError, "Rainbow stopped unexpectedly: "+compactError(out))
		} else if wasStopping {
			m.setNotice(noticeInfo, "Rainbow stopped. Static colors are safe to apply again.")
		}
		return m, refreshStatusCmd(m.svc)

	case rainbowAnimMsg:
		if m.rainbowRunning {
			m.rainbowFrame = (m.rainbowFrame + 1) % 360
			return m, rainbowAnimCmd()
		}
		return m, nil
	}

	return m, nil
}

func (m *Model) setNotice(kind noticeKind, text string) {
	m.noticeKind = kind
	m.notice = text
	m.noticeAuto = false
}

func (m *Model) setAutoNotice(kind noticeKind, text string) {
	m.noticeKind = kind
	m.notice = text
	m.noticeAuto = true
}

func (m *Model) quit() (tea.Model, tea.Cmd) {
	if m.rainbow != nil {
		_ = m.rainbow.Stop()
	}
	return *m, tea.Quit
}

func (m *Model) stopRainbow() {
	if m.rainbow != nil {
		_ = m.rainbow.Stop()
	}
}

func (m *Model) rescan() tea.Cmd {
	m.setAutoNotice(noticeInfo, "Scanning for a DualShock 4…")
	return refreshStatusCmd(m.svc)
}

// statusPollCmd keeps the controller badge and keeper PID fresh.
func statusPollCmd() tea.Cmd {
	return tea.Tick(6*time.Second, func(time.Time) tea.Msg { return statusPollMsg{} })
}

func rainbowAnimCmd() tea.Cmd {
	return tea.Tick(85*time.Millisecond, func(time.Time) tea.Msg { return rainbowAnimMsg{} })
}

func compactError(s string) string {
	s = oneLine(s)
	const max = 150
	runes := []rune(s)
	if len(runes) > max {
		return string(runes[:max-1]) + "…"
	}
	return s
}

func oneLine(s string) string {
	b := []byte(s)
	for i := range b {
		if b[i] == '\n' || b[i] == '\r' || b[i] == '\t' {
			b[i] = ' '
		}
	}
	return string(b)
}

func alphaSummary(a uint8) string {
	return "A " + strconv.Itoa(int(a)) + " / " + strconv.Itoa(int(color.RGBA{A: a}.AlphaPercent())) + "%"
}
