package render

import (
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/marinocg/goinvaders/internal/game"
)

type fakeTerminal struct {
	width, height     int
	output            strings.Builder
	sizeErr, writeErr error
	shortWrite        bool
}

type partialErrorTerminal struct {
	width, height int
	output        strings.Builder
	err           error
	limit         int
	fail          bool
}

func (t *partialErrorTerminal) Size() (int, int, error) { return t.width, t.height, nil }
func (t *partialErrorTerminal) Write(p []byte) (int, error) {
	n := t.limit
	if n > len(p) {
		n = len(p)
	}
	_, _ = t.output.Write(p[:n])
	if t.fail {
		t.fail = false
		return n, t.err
	}
	return n, nil
}

func (t *fakeTerminal) Size() (int, int, error) { return t.width, t.height, t.sizeErr }
func (t *fakeTerminal) Write(p []byte) (int, error) {
	if t.writeErr != nil {
		return 0, t.writeErr
	}
	if t.shortWrite {
		if len(p) == 0 {
			return 0, nil
		}
		_, _ = t.output.Write(p[:len(p)-1])
		return len(p) - 1, nil
	}
	return t.output.Write(p)
}

func TestRenderStatusFitsMinimumWidth(t *testing.T) {
	terminal := &fakeTerminal{width: 40, height: 18}
	state, err := game.NewState(4, 4)
	if err != nil {
		t.Fatal(err)
	}
	if err := NewTerminalRenderer(terminal).Render(state.Snapshot(), Presentation{Paused: true}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(terminal.output.String(), "Paused - Press P to resume") {
		t.Fatalf("status was clipped: %q", terminal.output.String())
	}
}

func TestSafeViewportReservesRightAndBottomEdges(t *testing.T) {
	for _, size := range []struct{ width, height int }{{40, 18}, {80, 30}} {
		width, height := safeViewport(size.width, size.height)
		if width != size.width-1 || height != size.height-1 {
			t.Fatalf("safeViewport(%d, %d) = %d, %d", size.width, size.height, width, height)
		}
		frame := buildFrame(game.Snapshot{ArenaWidth: 40, ArenaHeight: 18}, Presentation{}, size.width, size.height)
		lines := splitFrameLines(frame)
		if len(lines) != size.height-1 {
			t.Fatalf("frame rows = %d, want %d", len(lines), size.height-1)
		}
		for y, line := range lines {
			if len(line) != size.width-1 {
				t.Fatalf("frame row %d width = %d, want %d", y, len(line), size.width-1)
			}
		}
		if strings.Contains(lines[0], "\x1b") || strings.Contains(frame, "\n\n\n") && len(lines) > size.height {
			t.Fatal("frame escaped its bounded viewport")
		}
	}
}

func TestTooSmallUsesSafeViewportBoundary(t *testing.T) {
	terminal := &fakeTerminal{width: 40, height: 17}
	if err := NewTerminalRenderer(terminal).Render(game.Snapshot{}, Presentation{}); !errors.Is(err, ErrTerminalTooSmall) {
		t.Fatalf("Render() error = %v, want ErrTerminalTooSmall", err)
	}
	if !strings.Contains(terminal.output.String(), "Terminal too small") {
		t.Fatal("missing too-small message")
	}
}

func TestRenderFrameAndHUD(t *testing.T) {
	state, err := game.NewState(4, 4)
	if err != nil {
		t.Fatal(err)
	}
	terminal := &fakeTerminal{width: 40, height: 18}
	renderer := NewTerminalRenderer(terminal)
	if err := renderer.Render(state.Snapshot(), Presentation{Paused: true}); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Score: 0", "Lives: 3", "Paused", "W", "A"} {
		if !strings.Contains(terminal.output.String(), want) {
			t.Fatalf("frame missing %q", want)
		}
	}
}

func TestBuildFrameSnapshotIncludesStableLayoutAndValues(t *testing.T) {
	snapshot := game.Snapshot{
		ArenaWidth:  4,
		ArenaHeight: 4,
		Player:      game.PlayerSnapshot{X: 2, Y: 3},
		Enemies:     [64]game.EnemySnapshot{{X: 0, Y: 0, Alive: true}, {X: 1, Y: 1, Alive: false}},
		Score:       42,
		Lives:       2,
	}
	frame := buildFrame(snapshot, Presentation{}, 40, 18)
	lines := splitFrameLines(frame)
	if len(lines) != 17 {
		t.Fatalf("frame lines = %d, want 17", len(lines))
	}
	if strings.Contains(frame, "\x1b[2J") || strings.Contains(frame, "\x1b[?25l") {
		t.Fatalf("buildFrame contains terminal control sequence: %q", frame)
	}
	if len(lines[0]) != 39 || lines[0] != "+=====================================+" {
		t.Fatalf("top border = %q", lines[0])
	}
	if !strings.Contains(frame, "Score: 42   Lives: 2") {
		t.Fatalf("HUD missing from frame: %q", frame)
	}
	if strings.Count(strings.Join(lines[1:12], "\n"), "W") != 1 || strings.Count(strings.Join(lines[1:12], "\n"), "A") != 1 {
		t.Fatalf("entity glyphs missing or duplicated: %q", frame)
	}
}

func TestBuildFrameRendersEnemyProjectilesAndShieldDurability(t *testing.T) {
	snapshot := game.Snapshot{
		ArenaWidth:           40,
		ArenaHeight:          18,
		Player:               game.PlayerSnapshot{X: 20, Y: 17},
		EnemyProjectileCount: 1,
		ShieldCount:          1,
	}
	snapshot.EnemyProjectiles[0] = game.EnemyProjectileSnapshot{X: 5, Y: 4}
	snapshot.Shields[0] = game.ShieldSnapshot{X: 10, Y: 10, Cells: [2][3]uint8{{2, 1, 0}, {0, 2, 0}}}

	frame := buildFrame(snapshot, Presentation{}, 120, 50)
	if !strings.Contains(frame, "v") {
		t.Fatalf("enemy projectile is invisible: %q", frame)
	}
	if !strings.Contains(frame, "#") || !strings.Contains(frame, ".") {
		t.Fatalf("intact and damaged shield cells are not visible: %q", frame)
	}
}

func TestBuildFrameProjectsEnemyShotsAndShieldCellsAtEveryTier(t *testing.T) {
	for _, tc := range []struct {
		name          string
		width, height int
	}{
		{"compact", 40, 18},
		{"medium", 80, 30},
		{"large", 120, 50},
	} {
		t.Run(tc.name, func(t *testing.T) {
			snapshot := game.Snapshot{
				ArenaWidth:           40,
				ArenaHeight:          18,
				Player:               game.PlayerSnapshot{X: 39, Y: 17},
				EnemyProjectileCount: 1,
				ShieldCount:          1,
			}
			snapshot.EnemyProjectiles[0] = game.EnemyProjectileSnapshot{X: 3, Y: 3}
			snapshot.Shields[0] = game.ShieldSnapshot{
				X: 10, Y: 10,
				Cells: [2][3]uint8{{2, 1, 0}, {0, 0, 0}},
			}

			frame := buildFrame(snapshot, Presentation{}, tc.width, tc.height)
			lines := splitFrameLines(frame)
			layout := contentViewport(tc.width, tc.height)
			shotX, shotY := projectPoint(layout, snapshot.ArenaWidth, snapshot.ArenaHeight, 3, 3)
			if lines[shotY][shotX] == '|' || lines[shotY][shotX] == ' ' {
				t.Fatalf("enemy projectile projected at %d,%d rendered as %q", shotX, shotY, lines[shotY][shotX])
			}

			for _, cell := range []struct {
				x, y int
				want byte
			}{{10, 10, '#'}, {11, 10, func() byte {
				if tc.name == "compact" {
					return '.'
				}
				return '+'
			}()}} {
				x, y := projectPoint(layout, snapshot.ArenaWidth, snapshot.ArenaHeight, cell.x, cell.y)
				if lines[y][x] != cell.want {
					t.Fatalf("shield cell at %d,%d projected to %d,%d rendered as %q, want %q", cell.x, cell.y, x, y, lines[y][x], cell.want)
				}
			}
			destroyedX, destroyedY := projectPoint(layout, snapshot.ArenaWidth, snapshot.ArenaHeight, 12, 10)
			if lines[destroyedY][destroyedX] == '#' {
				t.Fatalf("destroyed shield cell at %d,%d rendered as shield material", destroyedX, destroyedY)
			}
		})
	}
}

func TestBuildFrameRendersBonusTargetAtEveryTier(t *testing.T) {
	for _, tc := range []struct {
		width, height int
		want          string
	}{{40, 18, "B"}, {80, 30, "-B-"}, {120, 50, "<--->"}} {
		snapshot := game.Snapshot{ArenaWidth: 40, ArenaHeight: 18, BonusTarget: &game.BonusTargetSnapshot{X: 20, Y: 3, Width: 1, Height: 1}}
		frame := buildFrame(snapshot, Presentation{}, tc.width, tc.height)
		if !strings.Contains(frame, tc.want) {
			t.Fatalf("bonus target at %dx%d missing %q: %q", tc.width, tc.height, tc.want, frame)
		}
	}
}

func TestRenderTerminalOutcomeStatuses(t *testing.T) {
	for _, test := range []struct {
		name    string
		outcome game.Outcome
		want    string
	}{
		{"win", game.OutcomeWin, "You win - Press Enter to play again"},
		{"game over", game.OutcomeGameOver, "Game over - Press Enter to try again"},
	} {
		t.Run(test.name, func(t *testing.T) {
			terminal := &fakeTerminal{width: 40, height: 18}
			if err := NewTerminalRenderer(terminal).Render(game.Snapshot{Outcome: test.outcome}, Presentation{}); err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(terminal.output.String(), test.want) {
				t.Fatalf("frame missing %q", test.want)
			}
		})
	}
}

func TestOutcomeStatusUsesCompactFallbackAndLargerFrame(t *testing.T) {
	for _, tc := range []struct {
		name, want string
		outcome    game.Outcome
	}{
		{"compact win", "You win - Press Enter to play again", game.OutcomeWin},
		{"compact game over", "Game over - Press Enter to try again", game.OutcomeGameOver},
	} {
		t.Run(tc.name, func(t *testing.T) {
			frame := buildFrame(game.Snapshot{ArenaWidth: 4, ArenaHeight: 4, Score: 123, Lives: 1, Outcome: tc.outcome}, Presentation{}, 40, 18)
			if !strings.Contains(frame, tc.want) || strings.Contains(frame, "== "+tc.want+" ==") {
				t.Fatalf("compact outcome was not complete and width-safe: %q", frame)
			}
		})
	}
	large := buildFrame(game.Snapshot{ArenaWidth: 4, ArenaHeight: 4, Outcome: game.OutcomeWin}, Presentation{}, 80, 30)
	if !strings.Contains(large, "== You win - Press Enter to play again ==") {
		t.Fatalf("larger outcome lost its framing: %q", large)
	}
}

func TestRenderTooSmallIsStableAndNoGameFrame(t *testing.T) {
	terminal := &fakeTerminal{width: 39, height: 10}
	renderer := NewTerminalRenderer(terminal)
	if err := renderer.Render(game.Snapshot{}, Presentation{}); !errors.Is(err, ErrTerminalTooSmall) {
		t.Fatalf("Render() error=%v, want ErrTerminalTooSmall", err)
	}
	if got := terminal.output.String(); !strings.Contains(got, "Terminal too small (required: 40 x 18)") || strings.Contains(got, "Score:") {
		t.Fatalf("unexpected fallback: %q", got)
	}
}

func TestRenderReturnsTerminalErrors(t *testing.T) {
	want := errors.New("size")
	if err := NewTerminalRenderer(&fakeTerminal{sizeErr: want}).Render(game.Snapshot{}, Presentation{}); !errors.Is(err, want) {
		t.Fatal(err)
	}
	want = errors.New("write")
	terminal := &fakeTerminal{width: 40, height: 18, writeErr: want}
	if err := NewTerminalRenderer(terminal).Render(game.Snapshot{}, Presentation{}); !errors.Is(err, want) {
		t.Fatal(err)
	}
}

func TestRenderWriteErrorRestoresCursorAfterPartialFirstFrame(t *testing.T) {
	want := errors.New("partial write")
	terminal := &partialErrorTerminal{width: 40, height: 18, err: want, limit: len("\x1b[2J\x1b[H\x1b[?25l"), fail: true}
	renderer := NewTerminalRenderer(terminal)
	if err := renderer.Render(game.Snapshot{}, Presentation{}); !errors.Is(err, want) {
		t.Fatalf("Render() error=%v, want %v", err, want)
	}
	if got := terminal.output.String(); !strings.HasSuffix(got, "\x1b[?25h\x1b[0m") {
		t.Fatalf("partial render did not attempt terminal cleanup: %q", got)
	}

	terminal.err = nil
	terminal.limit = 100000
	if err := renderer.Render(game.Snapshot{}, Presentation{}); err != nil {
		t.Fatalf("Render() after recovery = %v", err)
	}
}

func TestRenderAndShutdownReturnShortWrite(t *testing.T) {
	terminal := &fakeTerminal{width: 40, height: 18, shortWrite: true}
	renderer := NewTerminalRenderer(terminal)
	if renderer.Render(game.Snapshot{}, Presentation{}) != io.ErrShortWrite {
		t.Fatal("Render did not return io.ErrShortWrite")
	}
	terminal.shortWrite = false
	if err := renderer.Render(game.Snapshot{}, Presentation{}); err != nil {
		t.Fatal(err)
	}
	terminal.shortWrite = true
	if renderer.Shutdown() != io.ErrShortWrite {
		t.Fatal("Shutdown did not return io.ErrShortWrite")
	}
}

func TestBuildFrameSnapshotIsDeterministic(t *testing.T) {
	snapshot := game.Snapshot{ArenaWidth: 4, ArenaHeight: 4, Player: game.PlayerSnapshot{X: 2, Y: 3}, Score: 7, Lives: 1}
	want := buildFrame(snapshot, Presentation{Screen: ScreenStart}, 40, 18)
	for i := 0; i < 3; i++ {
		if got := buildFrame(snapshot, Presentation{Screen: ScreenStart}, 40, 18); got != want {
			t.Fatalf("buildFrame() changed between identical snapshots on iteration %d", i)
		}
	}
	if !strings.Contains(want, "Score: 7   Lives: 1") || !strings.Contains(want, "INVADERS") || !strings.Contains(want, "Press Enter to start") {
		t.Fatalf("snapshot frame omitted HUD or start status: %q", want)
	}
}

func TestBuildFrameMatchesSnapshotLayout(t *testing.T) {
	snapshot := game.Snapshot{ArenaWidth: 4, ArenaHeight: 4, Player: game.PlayerSnapshot{X: 2, Y: 3}, Enemies: [64]game.EnemySnapshot{{X: 0, Y: 0, Alive: true}}, Score: 7, Lives: 1}
	got := buildFrame(snapshot, Presentation{Screen: ScreenStart}, 40, 18)
	const want = "+=====================================+\n|W                                    |\n|                                     |\n|                                     |\n|                                     |\n|                                     |\n|                                     |\n|                                     |\n|                                     |\n|                        A            |\n+=====================================+\n                                       \n             [ INVADERS ]              \n        [ Score: 7   Lives: 1 ]        \n       1 Easy  2 Normal  3 Hard        \n Selected: Easy | Press Enter to start \n                                       "
	got = strings.ReplaceAll(got, "\r\n", "\n")
	if strings.TrimSuffix(got, "\n") != strings.TrimSuffix(want, "\n") {
		t.Fatalf("frame snapshot mismatch:\n got %q\nwant %q", got, want)
	}
}

func TestBuildFrameProjectsArenaExtremaMonotonically(t *testing.T) {
	for _, size := range []struct{ width, height int }{{40, 18}, {80, 30}, {79, 29}, {101, 37}} {
		layout := contentViewport(size.width, size.height)
		field := playfieldProjection(layout)
		last := field.minX - 1
		for x := 0; x < 40; x++ {
			got, _ := projectPoint(layout, 40, 18, x, 0)
			if got < last {
				t.Fatalf("width %dx%d projected x backward at %d: %d after %d", size.width, size.height, x, got, last)
			}
			last = got
		}
		if got, _ := projectPoint(layout, 40, 18, 0, 0); got != field.minX {
			t.Fatalf("left extrema for %dx%d = %d, want %d", size.width, size.height, got, field.minX)
		}
		if got, _ := projectPoint(layout, 40, 18, 39, 0); got != field.maxX {
			t.Fatalf("right extrema for %dx%d = %d, want %d", size.width, size.height, got, field.maxX)
		}
		last = field.minY - 1
		for y := 0; y < 18; y++ {
			_, got := projectPoint(layout, 40, 18, 0, y)
			if got < last {
				t.Fatalf("height %dx%d projected y backward at %d: %d after %d", size.width, size.height, y, got, last)
			}
			last = got
		}
		_, top := projectPoint(layout, 40, 18, 0, 0)
		_, bottom := projectPoint(layout, 40, 18, 0, 17)
		if top != field.minY || bottom != field.maxY {
			t.Fatalf("vertical extrema for %dx%d = %d,%d, want %d,%d", size.width, size.height, top, bottom, field.minY, field.maxY)
		}
	}
}

func TestProjectPointRoundsIntermediateCoordinatesWithoutDirectionalBias(t *testing.T) {
	layout := contentViewport(80, 30)
	field := playfieldProjection(layout)
	for _, tc := range []struct {
		logical, want int
	}{
		{1, field.minX + roundedInterpolation(1, field.maxX-field.minX, 39)},
		{20, field.minX + roundedInterpolation(20, field.maxX-field.minX, 39)},
		{38, field.minX + roundedInterpolation(38, field.maxX-field.minX, 39)},
	} {
		got, _ := projectPoint(layout, 40, 18, tc.logical, 0)
		if got != tc.want {
			t.Fatalf("projectPoint x=%d = %d, want nearest display cell %d", tc.logical, got, tc.want)
		}
	}
	for _, tc := range []struct {
		logical, want int
	}{
		{1, field.minY + roundedInterpolation(1, field.maxY-field.minY, 17)},
		{9, field.minY + roundedInterpolation(9, field.maxY-field.minY, 17)},
		{16, field.minY + roundedInterpolation(16, field.maxY-field.minY, 17)},
	} {
		_, got := projectPoint(layout, 40, 18, 0, tc.logical)
		if got != tc.want {
			t.Fatalf("projectPoint y=%d = %d, want nearest display cell %d", tc.logical, got, tc.want)
		}
	}
}

func TestBuildFramePositionsAllContentWithinSharedViewport(t *testing.T) {
	const width, height = 101, 37
	snapshot := game.Snapshot{ArenaWidth: 40, ArenaHeight: 18, Player: game.PlayerSnapshot{X: 39, Y: 17}, Score: 1, Lives: 1}
	frame := buildFrame(snapshot, Presentation{Screen: ScreenStart}, width, height)
	lines := splitFrameLines(frame)
	layout := contentViewport(width, height)
	if layout.originX != 1 || layout.originY != 1 || layout.width != width-3 || layout.height != height-3 {
		t.Fatalf("content viewport = %+v, want centered grown viewport", layout)
	}
	if lines[layout.originY+layout.height-5][layout.originX+(layout.width-len("INVADERS"))/2] != 'I' {
		t.Fatal("title did not use the positioned viewport origin")
	}
	if lines[layout.originY+1][layout.originX+1] != ' ' {
		t.Fatal("degenerate projection test setup unexpectedly occupied the top-left interior")
	}

	degenerate := buildFrame(game.Snapshot{ArenaWidth: 1, ArenaHeight: 1, Player: game.PlayerSnapshot{X: 0, Y: 0}}, Presentation{}, width, height)
	degenerateLines := splitFrameLines(degenerate)
	if degenerateLines[layout.originY+1][layout.originX+1] != 'A' {
		t.Fatal("degenerate projection escaped the positioned viewport")
	}
}

func TestContentViewportCentersEvenAndOddRawTerminals(t *testing.T) {
	for _, tc := range []struct{ width, height int }{{80, 30}, {81, 31}} {
		got := contentViewport(tc.width, tc.height)
		left, right := got.originX, tc.width-1-(got.originX+got.width)
		top, bottom := got.originY, tc.height-1-(got.originY+got.height)
		if left < 0 || right < 0 || right-left > 1 || top < 0 || bottom < 0 || bottom-top > 1 {
			t.Fatalf("safe content is not centered: %+v", got)
		}
	}
}

func TestResponsiveViewportAndPlayfieldGrowWithTerminal(t *testing.T) {
	sizes := []struct{ width, height int }{{40, 18}, {80, 30}, {81, 31}, {120, 50}}
	previous := viewport{}
	for _, size := range sizes {
		layout := contentViewport(size.width, size.height)
		field := playfieldProjection(layout)
		if layout.width <= previous.width || layout.height <= previous.height || field.maxX <= playfieldProjection(previous).maxX || field.maxY <= playfieldProjection(previous).maxY {
			t.Fatalf("responsive layout did not grow at %dx%d: layout=%+v field=%+v", size.width, size.height, layout, field)
		}
		previous = layout
	}
}

func TestContentViewportUsesResponsiveSafeGeometry(t *testing.T) {
	for _, tc := range []struct {
		width, height               int
		originX, originY            int
		contentWidth, contentHeight int
	}{
		{40, 18, 0, 0, 39, 17},
		{80, 30, 1, 1, 77, 27},
		{81, 31, 1, 1, 78, 28},
		{120, 50, 1, 1, 117, 47},
	} {
		got := contentViewport(tc.width, tc.height)
		if got.originX != tc.originX || got.originY != tc.originY || got.width != tc.contentWidth || got.height != tc.contentHeight {
			t.Fatalf("contentViewport(%d, %d) = %+v, want origin %d,%d size %d,%d", tc.width, tc.height, got, tc.originX, tc.originY, tc.contentWidth, tc.contentHeight)
		}
		left, right := got.originX, tc.width-1-got.originX-got.width
		top, bottom := got.originY, tc.height-1-got.originY-got.height
		if absInt(left-right) > 1 || absInt(top-bottom) > 1 {
			t.Fatalf("contentViewport(%d, %d) margins are not centered: left/right=%d/%d top/bottom=%d/%d", tc.width, tc.height, left, right, top, bottom)
		}
	}
}

func TestResponsiveLayoutSharesOriginAndAllocatesVerticalGrowthToPlayfield(t *testing.T) {
	minimum := contentViewport(40, 18)
	medium := contentViewport(80, 30)
	large := contentViewport(120, 50)
	for _, layout := range []viewport{minimum, medium, large} {
		field := playfieldProjection(layout)
		if field.minX != layout.originX+1 || field.minY != layout.originY+1 || field.maxX != layout.originX+layout.width-2 || field.maxY != layout.originY+layout.height-8 {
			t.Fatalf("playfield=%+v does not share content origin/layout=%+v", field, layout)
		}
	}
	if (playfieldProjection(large).maxY - playfieldProjection(minimum).maxY) <= (large.height-minimum.height)/2 {
		t.Fatalf("playfield did not receive most added vertical capacity: min=%+v large=%+v", playfieldProjection(minimum), playfieldProjection(large))
	}
	// The information block is a bounded suffix derived from the rendered rows;
	// do not reduce this to a tautology about field coordinates.
	for _, size := range []struct{ width, height int }{{40, 18}, {80, 30}, {120, 50}} {
		layout := contentViewport(size.width, size.height)
		field := playfieldProjection(layout)
		frame := buildFrame(game.Snapshot{ArenaWidth: 40, ArenaHeight: 18, Score: 1, Lives: 1}, Presentation{Paused: true}, size.width, size.height)
		lines := splitFrameLines(frame)
		rows := make([]int, 0, 4)
		for _, text := range []string{"INVADERS", "Score: 1   Lives: 1", "A/D move|Space/F fire|P pause|Q quit", "Paused - Press P to resume"} {
			row := -1
			for y, line := range lines {
				if strings.Contains(line, text) {
					row = y
					break
				}
			}
			if row < field.maxY+1 || row > field.maxY+6 {
				t.Fatalf("%q row %d is outside bounded information suffix for field=%+v layout=%+v", text, row, field, layout)
			}
			rows = append(rows, row)
		}
		if rows[len(rows)-1]-rows[0] > 5 {
			t.Fatalf("information block exceeds six-row suffix: rows=%v", rows)
		}
		if rows[0] != field.maxY+3 || rows[1] != field.maxY+4 || rows[2] != field.maxY+5 || rows[3] != field.maxY+6 {
			t.Fatalf("information rows moved from playfield basis: rows=%v field=%+v", rows, field)
		}
	}

	frame := buildFrame(game.Snapshot{ArenaWidth: 40, ArenaHeight: 18, Score: 1, Lives: 1}, Presentation{Paused: true}, 120, 50)
	lines := splitFrameLines(frame)
	contentTop, contentBottom := large.originY, large.originY+large.height-1
	topMargin, bottomMargin := contentTop, 50-1-contentBottom
	if absInt(topMargin-bottomMargin) > 1 {
		t.Fatalf("rendered content is not vertically centered: top/bottom margins=%d/%d", topMargin, bottomMargin)
	}
	for _, text := range []string{"INVADERS", "Score: 1   Lives: 1", "A/D move|Space/F fire|P pause|Q quit", "Paused - Press P to resume"} {
		found := false
		for _, line := range lines[contentTop : contentBottom+1] {
			if strings.Contains(line, text) {
				start := strings.Index(line, text)
				left := start
				right := 120 - 1 - (start + len(text))
				if absInt(left-right) > 1 {
					t.Fatalf("%q is not centered: margins=%d/%d", text, left, right)
				}
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("rendered block %q missing", text)
		}
	}
}

func TestResponsiveRenderedBlocksShareViewportOrigin(t *testing.T) {
	snapshot := game.Snapshot{ArenaWidth: 40, ArenaHeight: 18, Score: 12, Lives: 2}
	sizes := []struct {
		name          string
		width, height int
	}{
		{"minimum", 40, 18},
		{"medium even", 80, 30},
		{"medium odd", 81, 31},
		{"large", 120, 50},
	}
	var minimum viewport
	for _, size := range sizes {
		t.Run(size.name, func(t *testing.T) {
			layout := contentViewport(size.width, size.height)
			field := playfieldProjection(layout)
			if size.name == "minimum" {
				minimum = layout
			} else if layout.width <= minimum.width || layout.height <= minimum.height {
				t.Fatalf("grown terminal retained minimum content size: minimum=%+v current=%+v", minimum, layout)
			}

			frame := buildFrame(snapshot, Presentation{Paused: true}, size.width, size.height)
			lines := splitFrameLines(frame)
			if len(lines) != size.height-1 {
				t.Fatalf("frame rows = %d, want %d", len(lines), size.height-1)
			}
			for y, line := range lines {
				if len(line) != size.width-1 {
					t.Fatalf("row %d width = %d, want %d", y, len(line), size.width-1)
				}
			}

			if lines[field.minY][field.minX-1] != '|' || lines[field.minY][field.maxX+1] != '|' ||
				lines[field.maxY][field.minX-1] != '|' || lines[field.maxY][field.maxX+1] != '|' {
				t.Fatalf("playfield edges do not use layout origin: layout=%+v field=%+v", layout, field)
			}
			blocks := []struct {
				text string
				row  int
			}{
				{"INVADERS", field.maxY + 3},
				{"Score: 12   Lives: 2", field.maxY + 4},
				{"A/D move|Space/F fire|P pause|Q quit", field.maxY + 5},
				{"Paused - Press P to resume", field.maxY + 6},
			}
			for _, block := range blocks {
				if block.row < layout.originY || block.row >= layout.originY+layout.height {
					t.Fatalf("%q row %d escaped layout %+v", block.text, block.row, layout)
				}
				wantStart := layout.originX + (layout.width-len(block.text))/2
				if got := strings.Index(lines[block.row], block.text); got != wantStart {
					t.Fatalf("%q starts at %d, want %d from shared origin %+v", block.text, got, wantStart, layout)
				}
			}
		})
	}
}

func TestResizeSequenceReturnsToIdenticalMinimumGeometry(t *testing.T) {
	sequence := []struct{ width, height int }{{40, 18}, {80, 30}, {81, 31}, {120, 50}, {40, 18}}
	var first viewport
	snapshot := game.Snapshot{ArenaWidth: 40, ArenaHeight: 18, Player: game.PlayerSnapshot{X: 39, Y: 17}, Score: 1, Lives: 1}
	for i, size := range sequence {
		layout := contentViewport(size.width, size.height)
		if i == 0 {
			first = layout
		}
		if i == len(sequence)-1 && layout != first {
			t.Fatalf("minimum geometry drifted after resize sequence: first=%+v final=%+v", first, layout)
		}
		field := playfieldProjection(layout)
		frame := buildFrame(snapshot, Presentation{Paused: true}, size.width, size.height)
		lines := splitFrameLines(frame)
		if len(lines) != size.height-1 {
			t.Fatalf("frame rows at %dx%d = %d, want %d", size.width, size.height, len(lines), size.height-1)
		}
		for y, line := range lines {
			if len(line) != size.width-1 {
				t.Fatalf("stale/clipped frame at %dx%d row %d", size.width, size.height, y)
			}
		}
		for _, y := range []int{field.minY, field.maxY} {
			if lines[y][field.minX-1] != '|' || lines[y][field.maxX+1] != '|' {
				t.Fatalf("stale playfield edge at %dx%d: field=%+v", size.width, size.height, field)
			}
		}
		if got, y := projectPoint(layout, 40, 18, 39, 17); got != field.maxX || y != field.maxY {
			t.Fatalf("right/bottom extrema drifted at %dx%d: got=%d,%d want=%d,%d", size.width, size.height, got, y, field.maxX, field.maxY)
		}
		if strings.Contains(lines[len(lines)-1], "\x1b") || strings.Contains(frame, "\n\n\n") {
			t.Fatalf("resize frame wrapped or scrolled at %dx%d", size.width, size.height)
		}
	}
}

func absInt(value int) int {
	if value < 0 {
		return -value
	}
	return value
}

func splitFrameLines(frame string) []string {
	return strings.Split(strings.ReplaceAll(frame, "\r\n", "\n"), "\n")
}

func TestRenderSuppressesUnchangedFramesAndAvoidsNormalFullClear(t *testing.T) {
	terminal := &fakeTerminal{width: 40, height: 18}
	renderer := NewTerminalRenderer(terminal)
	snapshot := game.Snapshot{ArenaWidth: 4, ArenaHeight: 4}
	if err := renderer.Render(snapshot, Presentation{}); err != nil {
		t.Fatal(err)
	}
	first := terminal.output.Len()
	if err := renderer.Render(snapshot, Presentation{}); err != nil {
		t.Fatal(err)
	}
	if terminal.output.Len() != first {
		t.Fatal("unchanged frame emitted output")
	}
	if err := renderer.Render(snapshot, Presentation{Paused: true}); err != nil {
		t.Fatal(err)
	}
	got := terminal.output.String()[first:]
	if strings.Contains(got, "\x1b[2J") || !strings.HasPrefix(got, "\x1b[H") {
		t.Fatalf("changed normal frame used unstable redraw: %q", got)
	}
}

func TestShutdownPropagatesTerminalWriteError(t *testing.T) {
	want := errors.New("shutdown write")
	terminal := &fakeTerminal{width: 40, height: 18}
	renderer := NewTerminalRenderer(terminal)
	if err := renderer.Render(game.Snapshot{}, Presentation{}); err != nil {
		t.Fatal(err)
	}
	terminal.writeErr = want
	if !errors.Is(renderer.Shutdown(), want) {
		t.Fatal("Shutdown() did not propagate terminal error")
	}
}

func TestRenderSizeErrorAttemptsTerminalCleanup(t *testing.T) {
	want := errors.New("size unavailable")
	terminal := &fakeTerminal{width: 40, height: 18, sizeErr: want}
	renderer := NewTerminalRenderer(terminal)
	if err := renderer.Render(game.Snapshot{}, Presentation{}); !errors.Is(err, want) {
		t.Fatalf("Render() error = %v, want %v", err, want)
	}
	if got := terminal.output.String(); got != "\x1b[?25h\x1b[0m" {
		t.Fatalf("cleanup output = %q, want cursor restore", got)
	}
}

func TestRenderTooSmallAtExactBoundaryAndRecovers(t *testing.T) {
	terminal := &fakeTerminal{width: 40, height: 17}
	renderer := NewTerminalRenderer(terminal)
	if err := renderer.Render(game.Snapshot{}, Presentation{}); !errors.Is(err, ErrTerminalTooSmall) {
		t.Fatalf("Render() error = %v, want ErrTerminalTooSmall", err)
	}
	terminal.width, terminal.height = 40, 18
	if err := renderer.Render(game.Snapshot{}, Presentation{}); err != nil {
		t.Fatalf("Render() after resize = %v", err)
	}
	if !strings.Contains(terminal.output.String(), "Score:") {
		t.Fatal("normal frame was not rendered after resize")
	}
}

func TestRenderPresentationStatesUseStablePlayerVisibleContract(t *testing.T) {
	snapshot := game.Snapshot{ArenaWidth: 40, ArenaHeight: 18, Player: game.PlayerSnapshot{X: 10, Y: 12}, Score: 12, Lives: 2}
	for _, test := range []struct {
		name         string
		presentation Presentation
		want         string
	}{
		{"start", Presentation{Screen: ScreenStart}, "Press Enter to start"},
		{"active", Presentation{Screen: ScreenActive}, "A/D move|Space/F fire|P pause|Q quit"},
		{"pause", Presentation{Screen: ScreenActive, Paused: true}, "Paused - Press P to resume"},
	} {
		t.Run(test.name, func(t *testing.T) {
			frame := buildFrame(snapshot, test.presentation, 40, 18)
			if !strings.Contains(frame, test.want) {
				t.Fatalf("frame missing %q", test.want)
			}
		})
	}
}

func TestStartAndWaveTransitionShowSelectionAndProgress(t *testing.T) {
	start := buildFrame(game.Snapshot{}, Presentation{Screen: ScreenStart, Difficulty: game.DifficultyEasy}, 80, 30)
	if !strings.Contains(start, "Selected: Easy") || !strings.Contains(start, "1 Easy  2 Normal  3 Hard") {
		t.Fatalf("start frame omitted difficulty selection: %q", start)
	}
	transition := buildFrame(game.Snapshot{WaveNumber: 2, WaveAdvancePending: true}, Presentation{Screen: ScreenActive}, 80, 30)
	if !strings.Contains(transition, "Wave 2 clear - next wave ready") {
		t.Fatalf("transition frame omitted wave status: %q", transition)
	}
}

func TestActiveHUDShowsWaveAndDifficulty(t *testing.T) {
	frame := buildFrame(game.Snapshot{WaveNumber: 3, Difficulty: game.DifficultyHard}, Presentation{Screen: ScreenActive}, 80, 30)
	if !strings.Contains(frame, "Wave: 3   Difficulty: Hard") {
		t.Fatalf("active HUD omitted wave and difficulty: %q", frame)
	}
}

func TestStyledFrameKeepsPlainSemanticsAndResetsEachRole(t *testing.T) {
	snapshot := game.Snapshot{ArenaWidth: 4, ArenaHeight: 4, Score: 42, Lives: 2, Outcome: game.OutcomeWin}
	plain := buildFrame(snapshot, Presentation{}, 80, 30)
	styled := buildFrame(snapshot, Presentation{Styled: true}, 80, 30)
	for _, text := range []string{"INVADERS", "Score: 42   Lives: 2", "A/D move|Space/F fire|P pause|Q quit", "You win - Press Enter to play again"} {
		if !strings.Contains(plain, text) || !strings.Contains(styled, text) {
			t.Fatalf("role text %q missing from plain or styled frame", text)
		}
	}
	if strings.Count(styled, ansiReset) != 4 || !strings.Contains(styled, styleFor(roleTitle)+"[ INVADERS ]"+ansiReset) {
		t.Fatalf("styled frame did not reset all roles: %q", styled)
	}
	if len(splitFrameLines(plain)) != len(splitFrameLines(styled)) {
		t.Fatal("styling changed frame row count")
	}
}

func TestPresentationStatesRemainDistinctWithoutANSI(t *testing.T) {
	snapshot := game.Snapshot{ArenaWidth: 4, ArenaHeight: 4, Score: 9, Lives: 1}
	frames := []string{
		buildFrame(snapshot, Presentation{Screen: ScreenStart}, 40, 18),
		buildFrame(snapshot, Presentation{}, 40, 18),
		buildFrame(snapshot, Presentation{Paused: true}, 40, 18),
		buildFrame(game.Snapshot{ArenaWidth: 4, ArenaHeight: 4, Score: 9, Lives: 1, Outcome: game.OutcomeWin}, Presentation{}, 40, 18),
		buildFrame(game.Snapshot{ArenaWidth: 4, ArenaHeight: 4, Score: 9, Lives: 0, Outcome: game.OutcomeGameOver}, Presentation{}, 40, 18),
	}
	for i := range frames {
		for j := i + 1; j < len(frames); j++ {
			if frames[i] == frames[j] {
				t.Fatalf("state frames %d and %d are identical", i, j)
			}
		}
	}
}
