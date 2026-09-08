package render

import (
	"strings"
	"testing"

	"github.com/marinocg/goinvaders/internal/game"
)

func TestTerminalRendererVisualStatesAndResize(t *testing.T) {
	terminal := &fakeTerminal{width: MinWidth, height: MinHeight}
	renderer := NewTerminalRenderer(terminal)
	snapshot := game.Snapshot{ArenaWidth: 40, ArenaHeight: 18, Player: game.PlayerSnapshot{X: 10, Y: 12}, Score: 3, Lives: 2}
	for _, test := range []struct {
		name         string
		presentation Presentation
		outcome      game.Outcome
		want         string
	}{
		{"start", Presentation{Screen: ScreenStart}, game.OutcomePlaying, "Press Enter to start"},
		{"active", Presentation{Screen: ScreenActive}, game.OutcomePlaying, "A/D move"},
		{"pause", Presentation{Screen: ScreenActive, Paused: true}, game.OutcomePlaying, "Paused - Press P to resume"},
		{"win", Presentation{}, game.OutcomeWin, "You win - Press Enter to play again"},
		{"game over", Presentation{}, game.OutcomeGameOver, "Game over - Press Enter to try again"},
	} {
		t.Run(test.name, func(t *testing.T) {
			snapshot.Outcome = test.outcome
			if err := renderer.Render(snapshot, test.presentation); err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(terminal.output.String(), test.want) {
				t.Fatalf("rendered output missing %q", test.want)
			}
		})
	}
	terminal.width, terminal.height = MinWidth-1, MinHeight-1
	if err := renderer.Render(snapshot, Presentation{}); err != ErrTerminalTooSmall {
		t.Fatalf("resize error=%v, want ErrTerminalTooSmall", err)
	}
	if !strings.Contains(terminal.output.String(), "Terminal too small") {
		t.Fatal("resize warning was not rendered")
	}
}

func TestFrameSerializationDoesNotWrapOrScroll(t *testing.T) {
	for _, size := range []struct {
		name          string
		width, height int
		tooSmall      bool
	}{{"minimum", 40, 18, false}, {"medium", 80, 30, false}, {"tall", 120, 50, false}, {"just too short", 40, 17, true}} {
		t.Run(size.name, func(t *testing.T) {
			frame := buildFrame(game.Snapshot{ArenaWidth: 40, ArenaHeight: 18}, Presentation{Paused: true}, size.width, size.height)
			if size.tooSmall {
				frame = tooSmallFrame(size.width-1, size.height-1)
			}
			lines := splitFrameLines(frame)
			if len(lines) != size.height-1 {
				t.Fatalf("serialized rows = %d, want %d", len(lines), size.height-1)
			}
			for row, line := range lines {
				if len(line) > size.width-1 {
					t.Fatalf("row %d has %d printable cells, exceeds safe width %d", row, len(line), size.width-1)
				}
			}
			if strings.Contains(frame, "\n") && !strings.Contains(frame, "\r\n") {
				t.Fatal("frame uses LF row transitions without carriage return")
			}
			if !size.tooSmall {
				layout := contentViewport(size.width, size.height)
				if lines[layout.originY][layout.originX] != '+' {
					t.Fatalf("top playfield border is not visible at (%d,%d): %q", layout.originX, layout.originY, lines[layout.originY])
				}
			}
			if !size.tooSmall && lines[len(lines)-1] == "" {
				t.Fatal("bottom presentation row is missing")
			}
			term := modeledTerminal{width: size.width, height: size.height}
			term.write(frame)
			if term.scrolled || term.row >= size.height {
				t.Fatalf("frame scrolled viewport: cursor=(%d,%d), scrolled=%v", term.row, term.column, term.scrolled)
			}
		})
	}
}

type modeledTerminal struct {
	width, height int
	row, column   int
	scrolled      bool
}

func (t *modeledTerminal) write(frame string) {
	for i := 0; i < len(frame); i++ {
		switch frame[i] {
		case '\r':
			t.column = 0
		case '\n':
			t.row++
		case ' ', '+', '-', '|', 'A', 'W':
			t.column++
			if t.column >= t.width {
				t.column = 0
				t.row++
				if t.row >= t.height {
					t.scrolled = true
				}
			}
		}
	}
}
