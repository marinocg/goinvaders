package render

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/marinocg/goinvaders/internal/game"
)

const (
	MinWidth  = 40
	MinHeight = 18
)

var ErrTerminalTooSmall = errors.New("terminal too small")

// Terminal is the small platform boundary needed by the renderer.
type Terminal interface {
	Size() (width, height int, err error)
	io.Writer
}

// Presentation contains UI state that is deliberately separate from gameplay.
type Presentation struct {
	Paused bool
	Screen Screen
	// Styled enables optional ANSI emphasis. The default remains plain ASCII.
	Styled     bool
	Difficulty game.Difficulty
}

type Screen uint8

const (
	ScreenActive Screen = iota
	ScreenStart
)

type Renderer struct {
	terminal Terminal
	active   bool
	last     string
	width    int
	height   int
	tooSmall bool
}

type viewport struct {
	originX, originY int
	width, height    int
}

type projection struct {
	minX, maxX int
	minY, maxY int
}

func NewTerminalRenderer(terminal Terminal) *Renderer { return &Renderer{terminal: terminal} }

func (r *Renderer) Render(snapshot game.Snapshot, presentation Presentation) error {
	width, height, err := r.terminal.Size()
	if err != nil {
		r.cleanup()
		return err
	}
	safeWidth, safeHeight := safeViewport(width, height)
	if safeWidth < MinWidth-1 || safeHeight < MinHeight-1 {
		frame := tooSmallFrame(safeWidth, safeHeight)
		if r.tooSmall && r.width == width && r.height == height && r.last == frame {
			return ErrTerminalTooSmall
		}
		if safeWidth < 1 || safeHeight < 1 {
			// There is no valid viewport in which to address the terminal.
			r.active, r.tooSmall, r.width, r.height, r.last = false, true, width, height, frame
			return ErrTerminalTooSmall
		}
		if err = r.write(frame, width, height, true); err != nil {
			return err
		}
		return ErrTerminalTooSmall
	}
	frame := buildFrame(snapshot, presentation, width, height)
	if !r.tooSmall && r.width == width && r.height == height && r.last == frame {
		return nil
	}
	if err = r.write(frame, width, height, r.tooSmall || r.width != width || r.height != height); err != nil {
		return err
	}
	return nil
}

// Reserve the final reported column and row to avoid implicit autowrap and scroll.
func safeViewport(width, height int) (int, int) { return width - 1, height - 1 }

func contentViewport(width, height int) viewport {
	safeWidth, safeHeight := safeViewport(width, height)
	// Keep the minimum presentation intact, then leave a deliberate margin when
	// the terminal has room so the grown presentation is visibly centered.
	contentWidth := safeWidth
	contentHeight := safeHeight
	if contentWidth > MinWidth-1 {
		contentWidth -= 2
	}
	if contentHeight > MinHeight-1 {
		contentHeight -= 2
	}
	return viewport{
		originX: (safeWidth - contentWidth) / 2,
		originY: (safeHeight - contentHeight) / 2,
		width:   contentWidth,
		height:  contentHeight,
	}
}

func playfieldProjection(content viewport) projection {
	// Keep a generous ceiling and a distinct player apron. Extra height is
	// allocated to the arena rather than stretching the information block.
	playfieldBottom := content.height - 8
	return projection{
		minX: content.originX + 1,
		maxX: content.originX + content.width - 2,
		minY: content.originY + 1,
		maxY: content.originY + playfieldBottom,
	}
}

func projectPoint(content viewport, arenaWidth, arenaHeight, x, y int) (int, int) {
	field := playfieldProjection(content)
	if arenaWidth <= 1 {
		x = field.minX
	} else {
		x = field.minX + roundedInterpolation(x, field.maxX-field.minX, arenaWidth-1)
	}
	if arenaHeight <= 1 {
		y = field.minY
	} else {
		y = field.minY + roundedInterpolation(y, field.maxY-field.minY, arenaHeight-1)
	}
	return x, y
}

func roundedInterpolation(value, span, divisor int) int {
	return (value*span + divisor/2) / divisor
}

func (r *Renderer) write(frame string, width, height int, fullClear bool) error {
	prefix := "\x1b[H"
	if fullClear || !r.active {
		prefix = "\x1b[2J\x1b[H"
	}
	if err := writeFrame(r.terminal, prefix+"\x1b[?25l"+frame); err != nil {
		r.cleanup()
		// A failed write may have emitted part of the frame, including the
		// cursor-hide sequence, before returning. Do not retain a frame that
		// was not completely written and allow the next render to reinitialize.
		r.active, r.tooSmall, r.width, r.height, r.last = false, false, 0, 0, ""
		return err
	}
	r.active, r.tooSmall, r.width, r.height, r.last = true, width < MinWidth || height < MinHeight, width, height, frame
	return nil
}

func (r *Renderer) cleanup() error {
	return writeFrame(r.terminal, "\x1b[?25h\x1b[0m")
}

func (r *Renderer) Shutdown() error {
	if !r.active {
		return nil
	}
	err := writeFrame(r.terminal, "\x1b[?25h\x1b[0m")
	r.active = false
	return err
}

func writeFrame(w io.Writer, frame string) error {
	n, err := io.WriteString(w, frame)
	if err != nil {
		return err
	}
	if n != len(frame) {
		return io.ErrShortWrite
	}
	return nil
}

func buildFrame(s game.Snapshot, p Presentation, width, height int) string {
	safeWidth, safeHeight := safeViewport(width, height)
	lines := make([][]byte, safeHeight)
	for y := range lines {
		lines[y] = blankLine(safeWidth)
	}
	put := func(x, y int, value byte) {
		if x >= 0 && x < safeWidth && y >= 0 && y < safeHeight {
			lines[y][x] = value
		}
	}
	content := contentViewport(width, height)
	putContent := func(x, y int, value byte) { put(content.originX+x, content.originY+y, value) }
	// Keep the playfield separate from the information block. At the minimum
	// size this produces the prescribed rows 1-11 and leaves two padding rows.
	playfieldBottom := content.height - 7
	drawArenaBorder(putContent, content.width, playfieldBottom, defaultArenaTheme)
	field := playfieldProjection(content)
	tier := selectSpriteTier(field)
	mapPoint := func(x, y int) (int, int) { return projectPoint(content, s.ArenaWidth, s.ArenaHeight, x, y) }
	reserved := func(x, y int) bool {
		for _, enemy := range s.Enemies {
			if enemy.Alive {
				px, py := mapPoint(enemy.X, enemy.Y)
				if themeAbsInt(x-px) <= 2 && themeAbsInt(y-py) <= 1 {
					return true
				}
			}
		}
		px, py := mapPoint(s.Player.X, s.Player.Y)
		if themeAbsInt(x-px) <= 2 && themeAbsInt(y-py) <= 1 {
			return true
		}
		if s.PlayerProjectile != nil {
			projectileX, projectileY := mapPoint(s.PlayerProjectile.X, s.PlayerProjectile.Y)
			// Reserve the projectile's complete remaining upward travel path,
			// not just the cells near its current position.
			if x == projectileX && y <= projectileY {
				return true
			}
		}
		for i := 0; i < s.EnemyProjectileCount; i++ {
			px, py := mapPoint(s.EnemyProjectiles[i].X, s.EnemyProjectiles[i].Y)
			if x == px && y == py {
				return true
			}
		}
		for i := 0; i < s.ShieldCount; i++ {
			shield := s.Shields[i]
			for row, cells := range shield.Cells {
				for col, durability := range cells {
					if durability == 0 {
						continue
					}
					px, py := mapPoint(shield.X+col, shield.Y+row)
					if x == px && y == py {
						return true
					}
				}
			}
		}
		return false
	}
	decorateArena(lines, field, reserved)
	formationCenters := make([]int, 0, len(s.Enemies))
	for _, enemy := range s.Enemies {
		if enemy.Alive {
			px, _ := mapPoint(enemy.X, enemy.Y)
			formationCenters = append(formationCenters, px)
		}
	}
	// Enemies are stored in formation order, so projected centers are ordered.
	tier = formationSpriteTier(field, tier, formationCenters)
	place := func(x, y int, glyph byte) {
		px, py := mapPoint(x, y)
		art, ok := spriteFor(glyph, tier, px, py, field)
		if !ok {
			return
		}
		for row, text := range art.rows {
			for col := range text {
				if text[col] != ' ' {
					put(px-art.anchorX+col, py-art.anchorY+row, text[col])
				}
			}
		}
	}
	for i := 0; i < s.ShieldCount; i++ {
		for y := range s.Shields[i].Cells {
			for x, durability := range s.Shields[i].Cells[y] {
				if durability > 0 {
					place(s.Shields[i].X+x, s.Shields[i].Y+y, byte('0'+durability))
				}
			}
		}
	}
	if s.BonusTarget != nil {
		place(s.BonusTarget.X, s.BonusTarget.Y, 'B')
	}
	for _, enemy := range s.Enemies {
		if enemy.Alive {
			place(enemy.X, enemy.Y, 'W')
		}
	}
	if s.PlayerProjectile != nil {
		place(s.PlayerProjectile.X, s.PlayerProjectile.Y, '|')
	}
	place(s.Player.X, s.Player.Y, 'A')
	// Gameplay entities are drawn after decoration and actors so they cannot
	// disappear under another presentation layer at projected-cell conflicts.
	for i := 0; i < s.EnemyProjectileCount; i++ {
		place(s.EnemyProjectiles[i].X, s.EnemyProjectiles[i].Y, 'v')
	}
	center := func(row int, text string) {
		row += content.originY
		if row < 0 || row >= safeHeight {
			return
		}
		if len(text) > content.width {
			text = text[:content.width]
		}
		start := content.originX + (content.width-len(text))/2
		for x := range text {
			lines[row][start+x] = text[x]
		}
	}
	// The footer is a compact information console. Its six rows are fixed so
	// larger terminals give the arena more room instead of making the HUD tall.
	// Brackets and banners provide hierarchy even when ANSI is unavailable.
	title := "INVADERS"
	hud := fmt.Sprintf("Score: %d   Lives: %d", s.Score, s.Lives)
	controls := "A/D move|Space/F fire|P pause|Q quit"
	status := fmt.Sprintf("Wave: %d   Difficulty: %s", s.WaveNumber, difficultyName(s.Difficulty))
	statusRole := roleStatus
	if p.Screen == ScreenStart {
		controls = "1 Easy  2 Normal  3 Hard"
		status = "Selected: " + difficultyName(p.Difficulty) + " | Press Enter to start"
	} else if s.WaveAdvancePending {
		status = fmt.Sprintf("Wave %d clear - next wave ready", s.WaveNumber)
	}
	if p.Paused {
		status = "Paused - Press P to resume"
	}
	switch s.Outcome {
	case game.OutcomeWin:
		status = "You win - Press Enter to play again"
		statusRole = roleWin
	case game.OutcomeGameOver:
		status = "Game over - Press Enter to try again"
		statusRole = roleGameOver
	}
	center(playfieldBottom+2, "[ "+title+" ]")
	center(playfieldBottom+3, "[ "+hud+" ]")
	center(playfieldBottom+4, controls)
	statusText := status
	statusFrame := "== " + status + " =="
	if len(statusFrame) <= content.width {
		statusText = statusFrame
	}
	center(playfieldBottom+5, statusText)
	var b strings.Builder
	for y, line := range lines {
		b.Write(line)
		if y+1 < safeHeight {
			// LF alone preserves the current column on common terminals.
			b.WriteString("\r\n")
		}
	}
	frame := b.String()
	if !p.Styled {
		return frame
	}
	// Styling is applied after layout so escape bytes never participate in
	// viewport or centering calculations. Each replacement has its own reset.
	for _, item := range []struct {
		text string
		role emphasisRole
	}{
		{"[ " + title + " ]", roleTitle},
		{"[ " + hud + " ]", roleHUD},
		{controls, roleControls},
		{statusText, statusRole},
	} {
		if item.text != "" {
			frame = strings.Replace(frame, item.text, styleFor(item.role)+item.text+ansiReset, 1)
		}
	}
	return frame
}

func difficultyName(d game.Difficulty) string {
	switch d {
	case game.DifficultyEasy:
		return "Easy"
	case game.DifficultyHard:
		return "Hard"
	default:
		return "Normal"
	}
}

func tooSmallFrame(width, height int) string {
	if width < 1 || height < 1 {
		return ""
	}
	lines := make([][]byte, height)
	for y := range lines {
		lines[y] = []byte(strings.Repeat(" ", width))
	}
	message := "Terminal too small (required: 40 x 18)"
	y := height / 2
	start := (width - len(message)) / 2
	if start < 0 {
		start = 0
	}
	for i := 0; i < len(message) && start+i < width; i++ {
		lines[y][start+i] = message[i]
	}
	var b strings.Builder
	for i, line := range lines {
		b.Write(line)
		if i+1 < height {
			b.WriteString("\r\n")
		}
	}
	return b.String()
}
