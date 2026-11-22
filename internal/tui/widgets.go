package tui

import (
	"fmt"
	"math"
	"strings"

	"github.com/blazar/ds4-tui/internal/color"
	"github.com/blazar/ds4-tui/internal/picker"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

func displayWidth(s string) int { return ansi.StringWidth(s) }

func ansiTruncate(s string, width int) string {
	return ansi.Truncate(s, width, "")
}

func ansiTruncateTail(s string, width int) string {
	return ansi.Truncate(s, width, "…")
}

type cardOpts struct {
	Width  int
	Height int
	Title  string
	Active bool
	Accent string
	BG     string
}

// renderCard draws a rounded card of exactly Width x Height cells.
func renderCard(opts cardOpts, body []string) []string {
	w, h := opts.Width, opts.Height
	if w <= 0 || h <= 0 {
		return nil
	}
	bg := opts.BG
	if bg == "" {
		bg = colPanel
	}
	if w < 4 || h < 3 {
		lines := make([]string, 0, h)
		for range h {
			lines = append(lines, paintBG("", bg, w))
		}
		return lines
	}

	borderColor := colBorder
	titleColor := colMuted
	if opts.Active {
		borderColor = opts.Accent
		if borderColor == "" {
			borderColor = colAccent
		}
		titleColor = opts.Accent
		if titleColor == "" {
			titleColor = colAccent
		}
	}

	inner := w - 2
	title := strings.TrimSpace(opts.Title)
	var top string
	if title == "" || w < 8 {
		top = fg(borderColor).Render("╭" + strings.Repeat("─", inner) + "╮")
	} else {
		maxTitle := inner - 3
		if maxTitle < 1 {
			maxTitle = 1
		}
		if displayWidth(title) > maxTitle {
			title = truncateANSI(title, maxTitle)
		}
		titleW := displayWidth(title)
		// "╭─ " + title + " " + fill + "╮" must be exactly w cells.
		fill := w - 5 - titleW
		if fill < 0 {
			fill = 0
		}
		top = fg(borderColor).Render("╭─ ") + fgBold(titleColor).Render(title) +
			fg(borderColor).Render(" "+strings.Repeat("─", fill)+"╮")
	}
	// Keep the top border exactly w cells even if a glyph was measured oddly.
	lines := make([]string, 0, h)
	lines = append(lines, paintBG(top, bg, w))

	bodyRows := h - 2
	for i := 0; i < bodyRows; i++ {
		content := ""
		if i < len(body) {
			content = body[i]
		}
		innerLine := paintBG(content, bg, inner)
		line := fg(borderColor).Render("│") + innerLine + fg(borderColor).Render("│")
		lines = append(lines, paintBG(line, bg, w))
	}
	lines = append(lines, paintBG(fg(borderColor).Render("╰"+strings.Repeat("─", inner)+"╯"), bg, w))
	return lines
}

// renderGlow draws a soft, capsule-shaped lightbar. Each terminal cell encodes
// two vertical samples with ▀, which keeps the falloff smooth even when the
// preview is only a handful of rows tall.
func renderGlow(width, height int, c color.RGBA, bg string) []string {
	if width <= 0 || height <= 0 {
		return nil
	}
	if !colorEnabled {
		return renderMonoGlow(width, height, c)
	}
	if transparentBackground {
		return renderClearGlow(width, height, c)
	}
	eff := c.Palette()
	br, bgc, bb := rgbOf(bg)
	base := color.Palette{R: br, G: bgc, B: bb}
	dim := color.Palette{R: 70, G: 70, B: 92}
	if c.A < 8 || (eff.R < 8 && eff.G < 8 && eff.B < 8) {
		dim = color.Palette{R: 52, G: 52, B: 68}
	}

	halfLen := math.Max(1, float64(width)*0.31)
	cx := (float64(width) - 1) / 2
	cy := float64(height) // pixel-space height is 2*height
	rows := make([]string, 0, height)
	for y := 0; y < height; y++ {
		top := glowSample(width, y*2, cx, cy, halfLen, eff, base, dim, c.A)
		bottom := glowSample(width, y*2+1, cx, cy, halfLen, eff, base, dim, c.A)
		var b strings.Builder
		b.Grow(width * 40)
		for x := 0; x < width; x++ {
			b.WriteString(fgEscape(top[x].Hex()))
			b.WriteString(bgEscape(bottom[x].Hex()))
			b.WriteString("▀")
			b.WriteString("\x1b[0m")
		}
		rows = append(rows, b.String())
	}
	return rows
}

func renderClearGlow(width, height int, c color.RGBA) []string {
	rows := make([]string, height)
	for i := range rows {
		rows[i] = strings.Repeat(" ", width)
	}
	if width < 4 || height < 1 {
		return rows
	}
	eff := c.Palette()
	opaque := color.RGBA{R: eff.R, G: eff.G, B: eff.B, A: 255}
	if c.A < 4 || (eff.R < 8 && eff.G < 8 && eff.B < 8) {
		return rows
	}

	if height <= 3 {
		bar := int(float64(width) * 0.72)
		if bar < 2 {
			bar = 2
		}
		left := (width - bar) / 2
		line := rows[height/2]
		shade := color.ScaleRGB(opaque, 0.9)
		cell := fgEscape(shade.Hex()) + strings.Repeat("━", bar) + "\x1b[0m"
		rows[height/2] = line[:left] + cell + line[left+bar:]
		return rows
	}

	// A transparent background has no color to blend into. Draw a clean
	// rounded lightbar with block glyphs; the terminal background remains
	// visible all around it.
	barH := 5
	if height < 9 {
		barH = 3
	}
	barW := int(float64(width) * 0.78)
	if barW < 6 {
		barW = 6
	}
	if barW > width-2 {
		barW = width - 2
	}
	left := (width - barW) / 2
	top := (height - barH) / 2

	for row := 0; row < barH; row++ {
		glyph := "█"
		factor := 1.0
		switch {
		case row == 0:
			glyph = "▄"
			factor = 0.55
		case row == barH-1:
			glyph = "▀"
			factor = 0.55
		case row == 1 && barH > 3:
			factor = 0.78
		case row == barH-2 && barH > 3:
			factor = 0.78
		}
		shade := color.ScaleRGB(opaque, factor)
		cell := fgEscape(shade.Hex()) + strings.Repeat(glyph, barW) + "\x1b[0m"
		y := top + row
		if y < 0 || y >= height {
			continue
		}
		line := rows[y]
		rows[y] = line[:left] + cell + line[left+barW:]
	}
	return rows
}

func glowSample(width, y int, cx, cy, halfLen float64, eff, base, dim color.Palette, alpha uint8) []color.Palette {
	out := make([]color.Palette, width)
	for x := 0; x < width; x++ {
		dx := math.Abs(float64(x) - cx)
		dx = math.Max(0, dx-halfLen)
		dy := math.Abs(float64(y) - cy)
		d := math.Hypot(dx, dy)
		falloff := math.Max(1.4, cy*0.78)

		glow := 1 - d/falloff
		glow = clampF(glow, 0, 1)
		glow = glow * glow * (3 - 2*glow)

		core := 1 - d/1.65
		core = clampF(core, 0, 1)

		intensity := 0.035 + 0.72*glow + 0.25*core
		if alpha < 24 {
			intensity *= 0.45
		}

		// A barely-visible neutral shape keeps a black/off color readable.
		neutral := color.Mix(base, dim, 0.20+0.55*glow)
		col := color.Mix(neutral, eff, clampF(intensity, 0, 1))
		if core > 0 {
			col = color.Mix(col, color.Mix(eff, color.Palette{R: 255, G: 255, B: 255}, 0.12), 0.18*core)
		}
		out[x] = col
	}
	return out
}

func clampF(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func renderMonoGlow(width, height int, c color.RGBA) []string {
	mono := func(level float64) byte {
		if level <= 0 {
			return ' '
		}
		const ramp = " .:-=+*#%@"
		i := int(level*float64(len(ramp)-1) + 0.5)
		if i < 1 {
			i = 1
		}
		if i >= len(ramp) {
			i = len(ramp) - 1
		}
		return ramp[i]
	}
	halfLen := math.Max(1, float64(width)*0.31)
	cx := (float64(width) - 1) / 2
	cy := float64(height) / 2
	falloff := math.Max(1.4, float64(height)*0.82)
	rows := make([]string, 0, height)
	for y := 0; y < height; y++ {
		var b strings.Builder
		for x := 0; x < width; x++ {
			dx := math.Abs(float64(x) - cx)
			dx = math.Max(0, dx-halfLen)
			dy := math.Abs(float64(y) - cy)
			d := math.Hypot(dx, dy)
			level := 1 - d/falloff
			level = clampF(level, 0, 1)
			level = level * level * (3 - 2*level)
			if c.A < 24 {
				level *= 0.45
			}
			b.WriteByte(mono(level))
		}
		rows = append(rows, b.String())
	}
	return rows
}

func renderMonoSV(width, height int, p picker.Model, focused bool) []string {
	ramp := " .:-=+*#%@"
	cursorX := int(math.Round(p.Sat * float64(width-1)))
	cursorY := int(math.Round((1 - p.Val) * float64(height-1)))
	rows := make([]string, 0, height)
	for y := 0; y < height; y++ {
		var b strings.Builder
		for x := 0; x < width; x++ {
			if x == cursorX && y == cursorY {
				if focused {
					b.WriteString("◉")
				} else {
					b.WriteString("◌")
				}
				continue
			}
			s := float64(x) / math.Max(1, float64(width-1))
			v := 1 - float64(y)/math.Max(1, float64(height-1))
			level := v * (0.28 + 0.72*s)
			i := int(level*float64(len(ramp)-1) + 0.5)
			if i < 0 {
				i = 0
			}
			if i >= len(ramp) {
				i = len(ramp) - 1
			}
			b.WriteByte(ramp[i])
		}
		rows = append(rows, b.String())
	}
	return rows
}

// renderSV draws the saturation/value field and its cursor.
func renderSV(width, height int, p picker.Model, focused bool) []string {
	if width <= 0 || height <= 0 {
		return nil
	}
	if !colorEnabled {
		return renderMonoSV(width, height, p, focused)
	}
	// Half-block sampling gives twice the vertical resolution.
	rows := make([]string, 0, height)
	cursorX := int(math.Round(p.Sat * float64(width-1)))
	cursorY := int(math.Round((1 - p.Val) * float64(height-1)))
	for y := 0; y < height; y++ {
		var b strings.Builder
		for x := 0; x < width; x++ {
			s := (float64(x) + 0.5) / float64(width)
			vTop := 1 - (float64(2*y)+0.5)/float64(2*height)
			vBot := 1 - (float64(2*y+1)+0.5)/float64(2*height)
			topR, topG, topB := color.HSVToRGB(p.Hue, s, vTop)
			botR, botG, botB := color.HSVToRGB(p.Hue, s, vBot)

			if x == cursorX && y == cursorY {
				// A crisp ring that remains visible against any hue. Keep a
				// hollow marker visible when the field is not focused.
				cell := color.RGBA{R: botR, G: botG, B: botB, A: 255}
				b.WriteString(fgEscape(cell.Contrast().Hex()))
				b.WriteString(bgEscape(cell.Hex()))
				if focused {
					b.WriteString("◉")
				} else {
					b.WriteString("◌")
				}
				b.WriteString("\x1b[0m")
				continue
			}
			b.WriteString(fgEscape(color.Palette{R: topR, G: topG, B: topB}.Hex()))
			b.WriteString(bgEscape(color.Palette{R: botR, G: botG, B: botB}.Hex()))
			b.WriteString("▀")
			b.WriteString("\x1b[0m")
		}
		rows = append(rows, b.String())
	}
	return rows
}

// renderHueBar draws a vertical hue strip with a marker on the active row.
func renderHueBar(width, height int, p picker.Model, focused bool) []string {
	if width <= 0 || height <= 0 {
		return nil
	}
	marker := int(math.Round(p.Hue / 360 * float64(height-1)))
	if !colorEnabled {
		rows := make([]string, 0, height)
		for y := 0; y < height; y++ {
			row := strings.Repeat(" ", width)
			at := width / 2
			if y == marker {
				row = row[:at] + "◀" + row[at+1:]
			} else {
				row = row[:at] + "│" + row[at+1:]
			}
			rows = append(rows, row)
		}
		return rows
	}
	rows := make([]string, 0, height)
	for y := 0; y < height; y++ {
		hue := (float64(y) + 0.5) / float64(height) * 360
		var b strings.Builder
		for x := 0; x < width; x++ {
			r, g, bl := color.HSVToRGB(hue, 1, 1)
			if y == marker && x == width/2 {
				cell := color.RGBA{R: r, G: g, B: bl, A: 255}
				b.WriteString(fgEscape(cell.Contrast().Hex()))
				b.WriteString(bgEscape(cell.Hex()))
				b.WriteString("◀")
				b.WriteString("\x1b[0m")
				continue
			}
			b.WriteString(bgEscape(color.Palette{R: r, G: g, B: bl}.Hex()))
			b.WriteString(" ")
			b.WriteString("\x1b[0m")
		}
		rows = append(rows, b.String())
	}
	return rows
}

func renderSliderRow(innerW int, ch picker.Channel, value int, current color.RGBA, active bool) string {
	if innerW <= 0 {
		return ""
	}
	label := "R"
	chColor := color.Palette{R: 255, G: 80, B: 96}
	switch ch {
	case picker.Green:
		label = "G"
		chColor = color.Palette{R: 74, G: 222, B: 128}
	case picker.Blue:
		label = "B"
		chColor = color.Palette{R: 96, G: 165, B: 250}
	case picker.Alpha:
		label = "A"
		chColor = color.Palette{R: 220, G: 220, B: 235}
	}

	valueText := fmt.Sprintf("%3d", value)
	percent := int(math.Round(float64(value) / 255 * 100))
	showPct := ch == picker.Alpha && innerW >= 46
	if showPct {
		valueText += fmt.Sprintf(" %3d%%", percent)
	}
	barW := innerW - 1 - 1 - 1 - displayWidth(valueText)
	if barW < 4 {
		barW = 4
	}

	labelStyle := fgBold(chColor.Hex())
	if active {
		labelStyle = lipglossStyle(chColor, contrastString(chColor))
	}
	labelText := labelStyle.Render(label)

	bar := renderChannelBar(barW, ch, value, current, active)
	valueStyle := fg(colText)
	if !active {
		valueStyle = fg(colMuted)
	}
	line := labelText + " " + bar + " " + valueStyle.Render(valueText)
	return truncateANSI(line, innerW)
}

func lipglossStyle(bg color.Palette, fgHex string) lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(fgHex)).Background(lc(bg)).Bold(true)
}

func contrastString(p color.Palette) string {
	c := color.RGBA{R: p.R, G: p.G, B: p.B, A: 255}
	return c.Contrast().Hex()
}

func renderChannelBar(width int, ch picker.Channel, value int, current color.RGBA, active bool) string {
	if width <= 0 {
		return ""
	}
	marker := 0
	if width > 1 {
		marker = int(math.Round(float64(value) / 255 * float64(width-1)))
	}
	if !colorEnabled {
		var b strings.Builder
		for x := 0; x < width; x++ {
			switch {
			case x == marker:
				b.WriteString("┃")
			case x < marker:
				b.WriteString("█")
			default:
				b.WriteString("░")
			}
		}
		return b.String()
	}
	var b strings.Builder
	for x := 0; x < width; x++ {
		p := 1.0
		if width > 1 {
			p = float64(x) / float64(width-1)
		}
		var cell color.RGBA
		switch ch {
		case picker.Red:
			cell = color.RGBA{R: uint8(math.Round(p * 255)), G: current.G, B: current.B, A: 255}
		case picker.Green:
			cell = color.RGBA{R: current.R, G: uint8(math.Round(p * 255)), B: current.B, A: 255}
		case picker.Blue:
			cell = color.RGBA{R: current.R, G: current.G, B: uint8(math.Round(p * 255)), A: 255}
		case picker.Alpha:
			cell = current
			cell.A = uint8(math.Round(p * 255))
		}
		r, g, bl := cell.Effective()
		if !active {
			r = uint8(float64(r) * 0.55)
			g = uint8(float64(g) * 0.55)
			bl = uint8(float64(bl) * 0.55)
		}
		bg := color.Palette{R: r, G: g, B: bl}
		b.WriteString(bgEscape(bg.Hex()))
		if x == marker {
			b.WriteString(fgEscape("#FFFFFF"))
			b.WriteString("┃")
		} else {
			b.WriteString(" ")
		}
		b.WriteString("\x1b[0m")
	}
	return b.String()
}

func miniSwatch(c color.RGBA) string {
	p := c.Palette()
	return fg(p.Hex()).Render("██")
}

func chip(text, bg, fgColor string) string {
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color(fgColor)).
		Bold(true)
	if !(transparentBackground && isChromeBG(bg)) {
		style = style.Background(lipgloss.Color(bg))
	}
	return style.Render(" " + text + " ")
}

func rainbowRibbon(width, height int, phase, intensity float64) []string {
	if width <= 0 || height <= 0 {
		return nil
	}
	if !colorEnabled {
		shades := []rune("░▒▓█▓▒")
		rows := make([]string, 0, height)
		off := int(phase) % len(shades)
		for y := 0; y < height; y++ {
			var b strings.Builder
			for x := 0; x < width; x++ {
				b.WriteRune(shades[(x+y*2+off)%len(shades)])
			}
			rows = append(rows, b.String())
		}
		return rows
	}
	rows := make([]string, 0, height)
	for y := 0; y < height; y++ {
		var b strings.Builder
		for x := 0; x < width; x++ {
			hue := math.Mod(phase+float64(x)/float64(width)*360+float64(y)*17, 360)
			r, g, bl := color.HSVToRGB(hue, 0.86, 1)
			r = uint8(float64(r) * intensity)
			g = uint8(float64(g) * intensity)
			bl = uint8(float64(bl) * intensity)
			b.WriteString(bgEscape(color.Palette{R: r, G: g, B: bl}.Hex()))
			b.WriteString(" ")
			b.WriteString("\x1b[0m")
		}
		rows = append(rows, b.String())
	}
	return rows
}

func hsvLine(p picker.Model) string {
	return fmt.Sprintf("H %03.0f°  S %02.0f%%  V %02.0f%%", p.Hue, p.Sat*100, p.Val*100)
}
