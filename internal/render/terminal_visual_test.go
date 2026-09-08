package render

import (
	"strings"
	"testing"

	"github.com/marinocg/goinvaders/internal/game"
)

func TestRepresentativeTerminalSizesSelectReadableEntityTiers(t *testing.T) {
	for _, tc := range []struct {
		name, frame   string
		width, height int
		wantTier      spriteTier
	}{
		{"compact", "", 40, 18, compactTier},
		{"medium", "", 80, 30, mediumTier},
		{"large", "", 120, 50, largeTier},
	} {
		t.Run(tc.name, func(t *testing.T) {
			layout := contentViewport(tc.width, tc.height)
			if got := selectSpriteTier(playfieldProjection(layout)); got != tc.wantTier {
				t.Fatalf("sprite tier at %dx%d = %d, want %d", tc.width, tc.height, got, tc.wantTier)
			}
			snapshot := game.Snapshot{
				ArenaWidth: 40, ArenaHeight: 18,
				Player: game.PlayerSnapshot{X: 20, Y: 16},
				Enemies: [64]game.EnemySnapshot{
					{X: 8, Y: 1, Alive: true}, {X: 16, Y: 1, Alive: true}, {X: 24, Y: 1, Alive: true},
				},
			}
			frame := buildFrame(snapshot, Presentation{}, tc.width, tc.height)
			if tc.wantTier == compactTier {
				if strings.Contains(frame, "/W W\\") || strings.Contains(frame, "/A\\") {
					t.Fatal("compact frame unexpectedly used a rich sprite")
				}
			} else {
				if !strings.Contains(frame, "/W W\\") && !strings.Contains(frame, "/WWWWW\\") {
					t.Fatalf("%s frame lost rich enemy sprite: %q", tc.name, frame)
				}
				if !strings.Contains(frame, "/###\\") {
					t.Fatalf("%s frame lost rich player sprite: %q", tc.name, frame)
				}
			}
		})
	}
}

func TestRichFormationSpritesRemainDistinctAndEntitiesWinOverDecoration(t *testing.T) {
	for _, size := range []struct{ width, height int }{{80, 30}, {120, 50}} {
		snapshot := game.Snapshot{
			ArenaWidth: 40, ArenaHeight: 18,
			Player: game.PlayerSnapshot{X: 20, Y: 16},
			Enemies: [64]game.EnemySnapshot{
				{X: 8, Y: 1, Alive: true}, {X: 16, Y: 1, Alive: true}, {X: 24, Y: 1, Alive: true},
			},
		}
		frame := buildFrame(snapshot, Presentation{}, size.width, size.height)
		lines := splitFrameLines(frame)
		layout := contentViewport(size.width, size.height)
		field := playfieldProjection(layout)
		for _, enemy := range snapshot.Enemies {
			if !enemy.Alive {
				continue
			}
			x, y := projectPoint(layout, snapshot.ArenaWidth, snapshot.ArenaHeight, enemy.X, enemy.Y)
			found := false
			for row := y - 1; row <= y+1 && row < len(lines); row++ {
				if row < 0 {
					continue
				}
				for col := x - 3; col <= x+3 && col < len(lines[row]); col++ {
					if col >= 0 && lines[row][col] == 'W' {
						found = true
					}
				}
			}
			if !found {
				t.Fatalf("enemy at %d,%d lost its visible W glyph near %d,%d", enemy.X, enemy.Y, x, y)
			}
		}
		if strings.Count(strings.Join(lines[field.minY:field.maxY+1], "\n"), "W") < 3 {
			t.Fatal("rich formation collapsed to isolated or missing enemy glyphs")
		}
		if strings.Contains(strings.Join(lines[field.minY:field.maxY+1], "\n"), "...") {
			t.Fatal("decoration formed an intrusive run through the entity arena")
		}
	}
}

func TestCompactVisualFallbackKeepsRequiredComposition(t *testing.T) {
	frame := buildFrame(game.Snapshot{
		ArenaWidth: 40, ArenaHeight: 18,
		Player: game.PlayerSnapshot{X: 20, Y: 16}, Score: 12, Lives: 2,
		Enemies: [64]game.EnemySnapshot{{X: 20, Y: 1, Alive: true}},
	}, Presentation{Screen: ScreenStart}, 40, 18)
	for _, text := range []string{"W", "A", "[ INVADERS ]", "Score: 12   Lives: 2", "Press Enter to start"} {
		if !strings.Contains(frame, text) {
			t.Fatalf("compact frame missing %q", text)
		}
	}
	if strings.Contains(frame, ".") {
		t.Fatal("compact frame contains background decoration")
	}
}
