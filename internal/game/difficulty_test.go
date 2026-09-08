package game

import (
	"errors"
	"testing"
)

func TestDifficultyPresets(t *testing.T) {
	want := []DifficultyConfig{{36, 5, 72, 1, 1}, {30, 3, 54, 2, 1}, {24, 2, 42, 3, 2}}
	for d, expected := range want {
		got, err := Difficulty(d).Config()
		if err != nil || got != expected {
			t.Errorf("difficulty %d: got %#v, err %v", d, got, err)
		}
	}
	if _, err := Difficulty(99).Config(); !errors.Is(err, ErrInvalidDifficulty) {
		t.Fatalf("invalid difficulty error = %v", err)
	}
}

func TestWavePressureIsCapped(t *testing.T) {
	if wavePressure(1) != 0 || wavePressure(3) != 2 || wavePressure(99) != maxWavePressure {
		t.Fatal("unexpected pressure")
	}
	c, _ := DifficultyHard.Config()
	if waveFireInterval(c, 99) != minimumFireInterval {
		t.Fatal("fire interval exceeded floor")
	}
}

func TestDifficultyPresetsCompareEquivalentStartingStates(t *testing.T) {
	states := make([]State, 0, 3)
	for _, difficulty := range []Difficulty{DifficultyEasy, DifficultyNormal, DifficultyHard} {
		state, err := NewStateWithDifficulty(40, difficulty, 18)
		if err != nil {
			t.Fatal(err)
		}
		states = append(states, state)
	}
	for i := 1; i < len(states); i++ {
		if states[i].Snapshot().Player != states[0].Snapshot().Player || states[i].enemiesAlive() != states[0].enemiesAlive() {
			t.Fatal("difficulty presets did not start from equivalent state")
		}
	}
	easy, _ := DifficultyEasy.Config()
	normal, _ := DifficultyNormal.Config()
	hard, _ := DifficultyHard.Config()
	if !(easy.BaseMovementInterval > normal.BaseMovementInterval && normal.BaseMovementInterval > hard.BaseMovementInterval &&
		easy.EnemyFireInterval > normal.EnemyFireInterval && normal.EnemyFireInterval > hard.EnemyFireInterval &&
		easy.MaxEnemyProjectiles < normal.MaxEnemyProjectiles && normal.MaxEnemyProjectiles < hard.MaxEnemyProjectiles &&
		hard.EnemyProjectileSpeed > normal.EnemyProjectileSpeed) {
		t.Fatalf("documented difficulty ordering not reflected: easy=%+v normal=%+v hard=%+v", easy, normal, hard)
	}
}
