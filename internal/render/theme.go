package render

import "strings"

const ansiReset = "\x1b[0m"

type emphasisRole uint8

const (
	roleTitle emphasisRole = iota
	roleHUD
	roleControls
	roleStatus
	roleWin
	roleGameOver
)

func styleFor(role emphasisRole) string {
	switch role {
	case roleTitle:
		return "\x1b[1;36m"
	case roleHUD:
		return "\x1b[1;33m"
	case roleControls:
		return "\x1b[2;37m"
	case roleStatus:
		return "\x1b[1;35m"
	case roleWin:
		return "\x1b[1;32m"
	case roleGameOver:
		return "\x1b[1;31m"
	default:
		return ""
	}
}

// arenaTheme contains only disposable presentation characters. It deliberately
// uses ASCII so that the frame remains byte-width predictable.
type arenaTheme struct {
	borderTop, borderBottom, borderSide, corner byte
}

// A heavier rail makes the arena read as a stage, while ASCII keeps every
// glyph one column wide on terminals with inconsistent Unicode support.
var defaultArenaTheme = arenaTheme{'=', '=', '|', '+'}

func drawArenaBorder(put func(int, int, byte), width, bottom int, theme arenaTheme) {
	for x := 0; x < width; x++ {
		put(x, 0, theme.borderTop)
		put(x, bottom, theme.borderBottom)
	}
	for y := 0; y <= bottom; y++ {
		put(0, y, theme.borderSide)
		put(width-1, y, theme.borderSide)
	}
	put(0, 0, theme.corner)
	put(width-1, 0, theme.corner)
	put(0, bottom, theme.corner)
	put(width-1, bottom, theme.corner)
}

func themeDecorationEnabled(field projection) bool {
	return field.maxX-field.minX+1 >= 50 && field.maxY-field.minY+1 >= 14
}

// decorateArena places a stable, deliberately sparse star field. reserved
// returns true for cells which should remain visually quiet for gameplay.
func decorateArena(lines [][]byte, field projection, reserved func(int, int) bool) {
	if !themeDecorationEnabled(field) {
		return
	}
	for y := field.minY + 1; y < field.maxY; y++ {
		for x := field.minX + 1; x < field.maxX; x++ {
			if reserved(x, y) || lines[y][x] != ' ' || starAt(x, y, field) == false {
				continue
			}
			lines[y][x] = '.'
		}
	}
}

func starAt(x, y int, field projection) bool {
	// A coordinate hash gives stable positions while changing the viewport
	// naturally changes the available atmosphere.
	h := uint32(x*1103515245 + y*12345 + (field.maxX-field.minX)*2654435761)
	return h%37 == 7
}

func blankLine(width int) []byte { return []byte(strings.Repeat(" ", width)) }

func themeAbsInt(value int) int {
	if value < 0 {
		return -value
	}
	return value
}
