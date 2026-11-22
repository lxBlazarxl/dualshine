package tui

import (
	"fmt"
	"strings"

	"github.com/blazar/ds4-tui/internal/color"
)

func (m Model) presetGrid() (cols, cellW, cellH, rows int) {
	w, h := m.bodySize()
	cols = 1
	switch {
	case w >= 112:
		cols = 4
	case w >= 84:
		cols = 3
	case w >= 48:
		cols = 2
	}
	gapX := 2
	for cols > 1 && (w-gapX*(cols-1))/cols < 18 {
		cols--
	}
	cellW = (w - gapX*(cols-1)) / cols
	if cellW < 1 {
		cellW = 1
	}
	if h < 1 {
		return cols, cellW, 4, 1
	}

	n := len(m.presets)
	if n == 0 {
		return cols, cellW, 4, 1
	}
	wantRows := (n + cols - 1) / cols
	// If every preset can fit at the minimum card height, grow the cards to
	// use the available space instead of leaving a large empty region.
	minTotal := wantRows*4 + (wantRows - 1)
	if h >= minTotal {
		rows = wantRows
		cellH = (h - (rows - 1)) / rows
		if cellH > 10 {
			cellH = 10
		}
		if cellH < 4 {
			cellH = 4
		}
		return cols, cellW, cellH, rows
	}

	cellH = 4
	if h >= 26 {
		cellH = 5
	}
	rows = (h + 1) / (cellH + 1)
	if rows < 1 {
		rows = 1
	}
	return cols, cellW, cellH, rows
}

func (m Model) presetsView(width, height int) []string {
	cols, cellW, cellH, rows := m.presetGrid()
	gapX := 2

	grid := make([]string, 0, height)
	for row := 0; row < rows; row++ {
		blocks := make([][]string, 0, cols)
		for col := 0; col < cols; col++ {
			idx := m.presetOffset + row*cols + col
			if idx < len(m.presets) {
				active := idx == m.presetCursor
				blocks = append(blocks, m.presetCell(m.presets[idx], cellW, cellH, active))
			} else {
				blocks = append(blocks, blankBlock(cellW, cellH, colBG))
			}
		}
		grid = append(grid, joinHorizontal(blocks, gapX, colBG)...)
		if row < rows-1 {
			grid = append(grid, blankLine(width, colBG))
		}
	}

	// Center the gallery vertically; with a full-screen terminal the cards
	// grow, but on shorter terminals this avoids a lopsided gap.
	lines := make([]string, 0, height)
	top := (height - len(grid)) / 2
	if top < 0 {
		top = 0
	}
	for i := 0; i < top; i++ {
		lines = append(lines, blankLine(width, colBG))
	}
	lines = append(lines, grid...)
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

func (m Model) presetCell(p Preset, width, height int, active bool) []string {
	if width < 4 || height < 3 {
		return blankBlock(width, height, colBG)
	}
	innerW := width - 2
	bodyH := height - 2
	body := make([]string, 0, bodyH)

	swatchRows := bodyH - 1
	if swatchRows < 1 {
		swatchRows = 1
	}
	for i := 0; i < swatchRows; i++ {
		factor := 1.0
		if swatchRows > 1 {
			factor = 0.52 + 0.48*float64(i)/float64(swatchRows-1)
		}
		shade := color.ScaleRGB(p.RGBA, factor)
		body = append(body, solidLine(innerW, shade.Hex()))
	}
	hexStyle := fg(colMuted)
	if active {
		hexStyle = fgBold(colText)
	}
	body = append(body, center(hexStyle.Render(p.RGBA.Hex()), innerW))

	if len(body) < bodyH {
		for len(body) < bodyH {
			body = append(body, blankLine(innerW, colPanel))
		}
	}
	if len(body) > bodyH {
		body = body[:bodyH]
	}

	title := p.Name
	if active {
		title = "◆ " + p.Name
	}
	return renderCard(cardOpts{
		Width:  width,
		Height: height,
		Title:  title,
		Active: active,
		Accent: colAccent,
		BG:     colPanel,
	}, body)
}

func solidLine(width int, hex string) string {
	if width <= 0 {
		return ""
	}
	if !colorEnabled {
		return strings.Repeat("▒", width)
	}
	return bgEscape(hex) + strings.Repeat(" ", width) + "\x1b[0m"
}

func blankBlock(width, height int, bg string) []string {
	if width <= 0 || height <= 0 {
		return nil
	}
	lines := make([]string, 0, height)
	for i := 0; i < height; i++ {
		lines = append(lines, blankLine(width, bg))
	}
	return lines
}

func presetPageLabel(offset, total, visible int) string {
	if total == 0 {
		return ""
	}
	last := offset + visible
	if last > total {
		last = total
	}
	return fmt.Sprintf("%d-%d of %d", offset+1, last, total)
}
