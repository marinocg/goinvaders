package main

import (
	"os"
	"testing"

	"github.com/marinocg/goinvaders/internal/game"
)

func TestFileTerminalDelegatesWriteAndReportsNonTerminalSize(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "terminal")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	terminal := fileTerminal{File: file}
	if _, err := terminal.Write([]byte("frame")); err != nil {
		t.Fatal(err)
	}
	if _, _, err := terminal.Size(); err == nil {
		t.Fatal("Size() unexpectedly succeeded for a non-terminal file")
	}
}

func TestStartupCompositionConstants(t *testing.T) {
	if startupArenaWidth != 40 {
		t.Fatalf("startup arena width=%d, want 40", startupArenaWidth)
	}
	if startupArenaHeight != 18 {
		t.Fatalf("startup arena height=%d, want 18", startupArenaHeight)
	}
	if startupTickRate <= 0 {
		t.Fatal("startup tick rate must be positive")
	}
}

func TestDefaultStartupGameIsPlayableAndUsesFullFormation(t *testing.T) {
	state, err := game.NewState(startupArenaWidth, startupArenaHeight)
	if err != nil {
		t.Fatal(err)
	}
	snapshot := state.Snapshot()
	if snapshot.ArenaWidth != 40 || snapshot.ArenaHeight != 18 {
		t.Fatalf("arena = %dx%d, want 40x18", snapshot.ArenaWidth, snapshot.ArenaHeight)
	}
	const formationSize = 4 * 8
	alive := 0
	for i, enemy := range snapshot.Enemies {
		if !enemy.Alive {
			continue
		}
		alive++
		row, column := i/8, i%8
		wantX, wantY := 9+column*3, row*2
		if enemy.X != wantX || enemy.Y != wantY || enemy.Width != 1 || enemy.Height != 1 {
			t.Fatalf("enemy %d = %+v, want formation position (%d,%d)", i, enemy, wantX, wantY)
		}
	}
	if alive != formationSize {
		t.Fatalf("alive enemies = %d, want complete %d-enemy formation", alive, formationSize)
	}
	if snapshot.Player.Y <= 0 {
		t.Fatalf("player y = %d, want room to fire", snapshot.Player.Y)
	}
}

func TestStartupDifficultyFactoryBuildsRequestedPreset(t *testing.T) {
	for _, difficulty := range []game.Difficulty{game.DifficultyEasy, game.DifficultyNormal, game.DifficultyHard} {
		state, err := game.NewStateWithDifficulty(startupArenaWidth, difficulty, startupArenaHeight)
		if err != nil {
			t.Fatal(err)
		}
		if got := state.Snapshot().Difficulty; got != difficulty {
			t.Fatalf("difficulty=%v constructed as %v", difficulty, got)
		}
	}
}
