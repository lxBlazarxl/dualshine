package tui

import "strings"

type helpEntry struct {
	key   string
	label string
}

type helpSection struct {
	title   string
	entries []helpEntry
}

func helpSections() []helpSection {
	return []helpSection{
		{
			title: "NAVIGATION",
			entries: []helpEntry{
				{"c", "color picker"},
				{"p", "preset gallery"},
				{"w", "rainbow mode"},
				{"esc", "back / close help"},
				{"r", "rescan controller"},
				{"q", "quit"},
			},
		},
		{
			title: "PICKER",
			entries: []helpEntry{
				{"tab", "next control"},
				{"shift+tab", "previous control"},
				{"← →", "adjust focused control"},
				{"↑ ↓", "saturation / value"},
				{"1 2 3 4", "focus R / G / B / A"},
				{"h  s", "focus hue / SV field"},
				{"[ ]", "hue -1° / +1°"},
				{"- =", "brightness -1 / +1"},
				{"pgup pgdn", "coarse adjust"},
				{"e", "edit hex value"},
				{"enter a", "apply to lightbar"},
			},
		},
		{
			title: "PRESETS",
			entries: []helpEntry{
				{"← ↑ ↓ →", "move through gallery"},
				{"enter", "apply selected preset"},
				{"space", "preview in picker"},
				{"home end", "first / last preset"},
				{"pgup pgdn", "jump one page"},
			},
		},
		{
			title: "RAINBOW",
			entries: []helpEntry{
				{"space", "start / stop rainbow"},
				{"enter", "start / stop rainbow"},
				{"esc", "back to picker"},
			},
		},
		{
			title: "HEX INPUT",
			entries: []helpEntry{
				{"enter", "preview the typed color"},
				{"esc", "cancel without changing"},
				{"ctrl+u", "clear the input"},
				{"#RGB", "short form"},
				{"#RRGGBBAA", "with brightness"},
			},
		},
		{
			title: "GOOD TO KNOW",
			entries: []helpEntry{
				{"A", "alpha = lightbar brightness"},
				{"a", "apply, not alpha focus"},
				{"enter", "applies from the picker"},
				{"demo", "DS4_TUI_DEMO=1 or --demo"},
			},
		},
	}
}

func (m Model) helpView(width, height int) []string {
	innerW := width - 2
	bodyH := height - 2
	if innerW < 10 || bodyH < 2 {
		return renderCard(cardOpts{
			Width: width, Height: height, Title: "HELP", Active: true, Accent: colAccent2, BG: colPanel,
		}, nil)
	}

	all := m.helpLines(innerW)
	offset := m.helpOffset
	maxOffset := len(all) - bodyH
	if maxOffset < 0 {
		maxOffset = 0
	}
	if offset > maxOffset {
		offset = maxOffset
	}
	if offset < 0 {
		offset = 0
	}

	body := make([]string, 0, bodyH)
	for i := 0; i < bodyH; i++ {
		idx := offset + i
		if idx < len(all) {
			body = append(body, all[idx])
		} else {
			body = append(body, blankLine(innerW, colPanel))
		}
	}

	return renderCard(cardOpts{
		Width:  width,
		Height: height,
		Title:  "KEYBOARD SHORTCUTS",
		Active: true,
		Accent: colAccent2,
		BG:     colPanel,
	}, body)
}

func (m Model) helpLines(innerW int) []string {
	var body []string
	if innerW >= 78 {
		leftW := (innerW - 4) / 2
		rightW := innerW - leftW - 4
		left := renderHelpColumn(helpSections()[:2], leftW)
		right := renderHelpColumn(helpSections()[2:], rightW)
		for i := 0; i < len(left) || i < len(right); i++ {
			l, r := "", ""
			if i < len(left) {
				l = left[i]
			}
			if i < len(right) {
				r = right[i]
			}
			body = append(body, paintBG(l, colPanel, leftW)+
				strings.Repeat(" ", 4)+
				paintBG(r, colPanel, rightW))
		}
	} else {
		body = renderHelpColumn(helpSections(), innerW)
	}
	return body
}

// helpViewport returns the current scroll offset, visible line count, total
// line count and maximum offset for the help panel.
func (m Model) helpViewport() (offset, visible, total, maxOffset int) {
	w, h := m.bodySize()
	innerW := w - 2
	visible = h - 2
	if visible < 1 {
		visible = 1
	}
	if innerW < 10 {
		return 0, visible, 0, 0
	}
	total = len(m.helpLines(innerW))
	maxOffset = total - visible
	if maxOffset < 0 {
		maxOffset = 0
	}
	offset = m.helpOffset
	if offset > maxOffset {
		offset = maxOffset
	}
	if offset < 0 {
		offset = 0
	}
	return offset, visible, total, maxOffset
}

func (m *Model) clampHelpOffset() {
	_, _, _, maxOffset := m.helpViewport()
	if m.helpOffset > maxOffset {
		m.helpOffset = maxOffset
	}
	if m.helpOffset < 0 {
		m.helpOffset = 0
	}
}

func renderHelpColumn(sections []helpSection, width int) []string {
	if width <= 0 {
		return nil
	}
	keyW := 10
	if width < 32 {
		keyW = 7
	}

	lines := make([]string, 0, 32)
	for i, section := range sections {
		if i > 0 {
			lines = append(lines, blankLine(width, colPanel))
		}
		lines = append(lines, fgBold(colAccent2).Render(section.title))
		for _, e := range section.entries {
			line := renderHelpEntry(e, keyW, width)
			lines = append(lines, line)
		}
	}
	return lines
}

func renderHelpEntry(e helpEntry, keyW, width int) string {
	key := e.key
	if displayWidth(key) > keyW-1 {
		key = truncateANSI(key, keyW-1)
	}
	keyCell := fgBold(colText).Render(key)
	keyCell += strings.Repeat(" ", maxInt(1, keyW-displayWidth(key)))
	label := fg(colMuted).Render(e.label)
	line := keyCell + label
	if displayWidth(line) > width {
		line = truncateANSI(line, width)
	}
	return line
}
