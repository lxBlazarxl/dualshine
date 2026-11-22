package tui

import (
	"fmt"
	"strings"
)

func (m Model) rainbowView(width, height int) []string {
	bodyH := height - 2
	innerW := width - 2
	if bodyH < 6 {
		return renderCard(cardOpts{
			Width:  width,
			Height: height,
			Title:  "RAINBOW MODE",
			Active: true,
			Accent: colAccent2,
			BG:     colPanel,
		}, nil)
	}

	lines := make([]string, 0, bodyH)
	lines = append(lines, m.rainbowStatus(innerW))
	lines = append(lines, blankLine(innerW, colPanel))

	ribbonH := bodyH - 7
	if ribbonH < 2 {
		ribbonH = 2
	}
	if ribbonH > 14 {
		ribbonH = 14
	}
	phase := float64(m.rainbowFrame * 3)
	intensity := 1.0
	if !m.rainbowRunning {
		phase = 22
		intensity = 0.52
	}
	ribbon := rainbowRibbon(innerW, ribbonH, phase, intensity)
	for _, line := range ribbon {
		lines = append(lines, line)
	}
	lines = append(lines, blankLine(innerW, colPanel))
	lines = append(lines, center(m.rainbowButton(innerW), innerW))
	lines = append(lines, center(m.rainbowNote1(innerW), innerW))
	lines = append(lines, center(m.rainbowNote2(innerW), innerW))

	// Center the composition vertically when the card is very tall.
	if len(lines) < bodyH {
		top := (bodyH - len(lines)) / 2
		pad := make([]string, top)
		for i := range pad {
			pad[i] = blankLine(innerW, colPanel)
		}
		lines = append(pad, lines...)
	}
	if len(lines) < bodyH {
		for len(lines) < bodyH {
			lines = append(lines, blankLine(innerW, colPanel))
		}
	}
	if len(lines) > bodyH {
		lines = lines[:bodyH]
	}

	return renderCard(cardOpts{
		Width:  width,
		Height: height,
		Title:  "RAINBOW MODE",
		Active: true,
		Accent: colAccent2,
		BG:     colPanel,
	}, lines)
}

func (m Model) rainbowStatus(width int) string {
	var left, right string
	switch {
	case m.rainbowStarting:
		left = fgBold(colAmber).Render("◌ STARTING") +
			fg(colMuted).Render("  the rainbow is starting")
		right = fg(colDim).Render("PLEASE WAIT")
	case m.rainbowRunning:
		left = fgBold(colGreen).Render("● RAINBOW ACTIVE") +
			fg(colMuted).Render("  painting the lightbar")
		right = fg(colDim).Render(fmt.Sprintf("PID %d · rainbow", m.rainbowPID))
	case m.demo:
		left = fgBold(colMuted).Render("○ RAINBOW IDLE")
		right = fg(colPink).Render("DEMO MODE")
	case m.statusKnown && !m.status.Found:
		left = fgBold(colRed).Render("○ NO CONTROLLER")
		right = fg(colDim).Render("CONNECT + PRESS R")
	default:
		left = fgBold(colMuted).Render("○ RAINBOW IDLE")
		right = fg(colDim).Render("PRESS SPACE TO START")
	}
	line := spreadKeepRight(left, right, width)
	return truncateANSI(line, width)
}

func (m Model) rainbowButton(width int) string {
	label := "▶  SPACE   START RAINBOW"
	bg := colAccent
	fgColor := colBG
	switch {
	case m.rainbowStarting:
		label = "◌  STARTING RAINBOW"
		bg = colAmber
	case m.rainbowRunning:
		label = "■  SPACE   STOP RAINBOW"
		bg = colRed
		fgColor = "#1A0A0D"
	case m.statusKnown && !m.status.Found:
		label = "CONNECT A CONTROLLER"
		bg = colPanel2
		fgColor = colMuted
	}
	chipText := chip(label, bg, fgColor)
	if displayWidth(chipText) > width {
		shorter := "SPACE  START"
		if m.rainbowRunning {
			shorter = "SPACE  STOP"
		}
		chipText = chip(shorter, bg, fgColor)
	}
	return chipText
}

func (m Model) rainbowNote1(width int) string {
	if m.rainbowErr != "" && !m.rainbowRunning {
		return fg(colRed).Render(truncateANSI(compactError(m.rainbowErr), width))
	}
	if m.statusKnown && !m.status.Found {
		return fg(colAmber).Render("Connect the DS4 over USB or Bluetooth, then press r.")
	}
	msg := "The rainbow paints continuously and overrides static colors."
	if width < 64 {
		msg = "Rainbow overrides static colors."
	}
	return fg(colDim).Render(msg)
}

func (m Model) rainbowNote2(width int) string {
	msg := "Starting, stopping and quitting this TUI all stop the rainbow process."
	if width < 64 {
		msg = "Quitting this TUI stops the rainbow."
	}
	if m.rainbowRunning {
		return fg(colMuted).Render(msg)
	}
	return fg(colDim).Render(strings.TrimSpace(msg))
}
