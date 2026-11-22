package tui

import (
	"fmt"
	"strings"

	"github.com/blazar/ds4-tui/internal/picker"
)

func (m Model) colorView(width, height int) []string {
	if width >= 96 && height >= 20 {
		leftW := width * 34 / 100
		if leftW < 30 {
			leftW = 30
		}
		if leftW > 42 {
			leftW = 42
		}
		rightW := width - leftW - 2
		if rightW >= 56 {
			left := m.previewCard(leftW, height)
			right := m.pickerCard(rightW, height)
			return joinHorizontal([][]string{left, right}, 2, colBG)
		}
	}

	stripH := 0
	if height >= 16 && width >= 46 {
		stripH = 2
		if height >= 20 {
			stripH = 3
		}
	}
	cardH := height - stripH
	if stripH > 0 {
		cardH--
	}
	if cardH < 3 {
		cardH = height
		stripH = 0
	}

	lines := make([]string, 0, height)
	if stripH > 0 {
		if height >= 16 {
			lines = append(lines, blankLine(width, colBG))
			cardH--
		}
		lines = append(lines, m.previewStrip(width, stripH)...)
		lines = append(lines, blankLine(width, colBG))
	}
	lines = append(lines, m.pickerCard(width, cardH)...)
	if len(lines) < height {
		for len(lines) < height {
			lines = append(lines, blankLine(width, colBG))
		}
	}
	if len(lines) > height {
		lines = lines[:height]
	}
	return lines
}

func (m Model) previewCard(width, height int) []string {
	if width < 8 || height < 4 {
		return renderCard(cardOpts{Width: width, Height: height, BG: colPanel, Active: true, Accent: colAccent}, nil)
	}
	bodyH := height - 2
	innerW := width - 2
	text := m.previewText(innerW)
	glowH := bodyH - len(text) - 1
	if glowH < 1 {
		glowH = 1
	}
	if glowH+len(text)+1 > bodyH {
		// Extremely short card: drop the least important text lines.
		for len(text) > 1 && glowH+len(text)+1 > bodyH {
			text = text[:len(text)-1]
		}
	}
	body := make([]string, 0, bodyH)
	body = append(body, renderGlow(innerW, glowH, m.picker.RGBA(), colPanel)...)
	if len(body) < bodyH {
		body = append(body, blankLine(innerW, colPanel))
	}
	body = append(body, text...)
	if len(body) < bodyH {
		for len(body) < bodyH {
			body = append(body, blankLine(innerW, colPanel))
		}
	}
	if len(body) > bodyH {
		body = body[:bodyH]
	}

	title := "TOUCHPAD LIGHTBAR"
	if width < 30 {
		title = "LIGHTBAR"
	}
	return renderCard(cardOpts{
		Width:  width,
		Height: height,
		Title:  title,
		Active: true,
		Accent: colAccent,
		BG:     colPanel,
	}, body)
}

func (m Model) previewText(width int) []string {
	c := m.picker.RGBA()
	eff := c.Palette()
	hexChip := chip(c.Hex(), eff.Hex(), contrastString(eff))
	aChip := chip(fmt.Sprintf("A %d", c.A), colPanel2, colMuted)
	state := fg(colDim).Render("PREVIEW · NOT APPLIED")
	if m.rainbowRunning {
		state = fgBold(colAccent2).Render("RAINBOW ACTIVE")
	} else if m.hasApplied && c == m.lastApplied {
		state = fgBold(colGreen).Render("● ON LIGHTBAR")
	} else if m.statusKnown && !m.status.Found {
		state = fgBold(colRed).Render("NO HARDWARE")
	}

	rgbLine := fgBold(colRed).Render("R") + " " + fg(colMuted).Render(fmt.Sprintf("%03d", c.R)) + "   " +
		fgBold(colGreen).Render("G") + " " + fg(colMuted).Render(fmt.Sprintf("%03d", c.G)) + "   " +
		fgBold(colBlue).Render("B") + " " + fg(colMuted).Render(fmt.Sprintf("%03d", c.B))

	alphaLine := aChip + "  " + fg(colDim).Render(fmt.Sprintf("%d%% brightness", c.AlphaPercent()))
	lines := []string{
		center(hexChip, width),
		center(rgbLine, width),
		center(alphaLine, width),
		center(state, width),
	}
	return lines
}

func (m Model) previewStrip(width, height int) []string {
	if height <= 0 {
		return nil
	}
	c := m.picker.RGBA()
	eff := c.Palette()
	textLine := center(
		chip(c.Hex(), eff.Hex(), contrastString(eff))+
			fg(colDim).Render("   A ")+fg(colMuted).Render(fmt.Sprintf("%d", c.A))+
			fg(colDim).Render(fmt.Sprintf(" · %d%%", c.AlphaPercent())),
		width,
	)
	if height == 1 {
		return []string{paintBG(textLine, colBG, width)}
	}
	glowH := height - 1
	lines := renderGlow(width, glowH, c, colBG)
	for i := range lines {
		lines[i] = paintBG(lines[i], colBG, width)
	}
	return append(lines, paintBG(textLine, colBG, width))
}

func (m Model) pickerCard(width, height int) []string {
	if width < 6 || height < 4 {
		return renderCard(cardOpts{Width: width, Height: height, Title: "COLOR PICKER", Active: true, BG: colPanel}, nil)
	}
	bodyH := height - 2
	innerW := width - 2

	const slidersH = 4
	gaps := 0
	if bodyH >= 8 {
		gaps = 1
	}
	if bodyH >= 11 {
		gaps = 2
	}
	svH := bodyH - 1 - slidersH - gaps
	if svH < 2 {
		svH = 2
	}

	body := make([]string, 0, bodyH)
	body = append(body, m.pickerInfo(innerW))
	if gaps >= 2 {
		body = append(body, blankLine(innerW, colPanel))
	}
	body = append(body, m.pickerCanvas(innerW, svH)...)
	if gaps >= 1 {
		body = append(body, blankLine(innerW, colPanel))
	}
	for _, ch := range []picker.Channel{picker.Red, picker.Green, picker.Blue, picker.Alpha} {
		active := false
		if f, ok := m.focus.channel(); ok && f == ch {
			active = true
		}
		body = append(body, renderSliderRow(innerW, ch, m.picker.ChannelValue(ch), m.picker.RGBA(), active))
	}

	if len(body) < bodyH {
		for len(body) < bodyH {
			body = append(body, blankLine(innerW, colPanel))
		}
	}
	if len(body) > bodyH {
		body = body[:bodyH]
	}
	title := "COLOR PICKER"
	if width < 28 {
		title = "PICKER"
	}
	return renderCard(cardOpts{
		Width:  width,
		Height: height,
		Title:  title,
		Active: true,
		Accent: colAccent,
		BG:     colPanel,
	}, body)
}

func (m Model) pickerInfo(width int) string {
	focusLabel := "SV"
	switch m.focus {
	case focusHue:
		focusLabel = "HUE"
	case focusRed:
		focusLabel = "R"
	case focusGreen:
		focusLabel = "G"
	case focusBlue:
		focusLabel = "B"
	case focusAlpha:
		focusLabel = "A"
	}

	focus := chip(focusLabel, colAccent, colBG)
	hsv := fmt.Sprintf("H %03.0f°  S %02.0f%%  V %02.0f%%", m.picker.Hue, m.picker.Sat*100, m.picker.Val*100)
	if width < 52 {
		hsv = fmt.Sprintf("H%03.0f S%02.0f V%02.0f", m.picker.Hue, m.picker.Sat*100, m.picker.Val*100)
	}
	left := focus + " " + fg(colMuted).Render(hsv)

	c := m.picker.RGBA()
	right := fg(c.Palette().Hex()).Render("██") + " " + fgBold(colText).Render(c.Hex())
	if width >= 58 {
		right += fg(colDim).Render("  A ") + fg(colMuted).Render(fmt.Sprintf("%d", c.A))
	}
	line := spreadKeepRight(left, right, width)
	return truncateANSI(line, width)
}

func (m Model) pickerCanvas(width, height int) []string {
	if width <= 0 || height <= 0 {
		return nil
	}
	hueW := 3
	if width < 26 {
		hueW = 2
	}
	gap := 1
	svW := width - hueW - gap
	if svW < 8 {
		hueW = 0
		gap = 0
		svW = width
	}

	sv := renderSV(svW, height, m.picker, m.focus == focusSV)
	if hueW == 0 {
		return sv
	}
	hue := renderHueBar(hueW, height, m.picker, m.focus == focusHue)
	rows := make([]string, 0, height)
	for i := 0; i < height; i++ {
		line := sv[i] + strings.Repeat(" ", gap) + hue[i]
		rows = append(rows, line)
	}
	return rows
}

func joinHorizontal(blocks [][]string, gap int, bg string) []string {
	height := 0
	for _, b := range blocks {
		if len(b) > height {
			height = len(b)
		}
	}
	if height == 0 {
		return nil
	}
	gapLine := strings.Repeat(" ", gap)
	out := make([]string, 0, height)
	for row := 0; row < height; row++ {
		var b strings.Builder
		for i, block := range blocks {
			if i > 0 {
				b.WriteString(gapLine)
			}
			if row < len(block) {
				b.WriteString(block[row])
			}
		}
		out = append(out, b.String())
	}
	return out
}
