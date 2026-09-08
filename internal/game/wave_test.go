package game

import "testing"

func TestWaveTransitionPreservesRunStateAndResetsWaveState(t *testing.T) {
	state, err := NewState(3, 8)
	if err != nil {
		t.Fatal(err)
	}
	state.score, state.lives = 42, 2
	state.playerProjectile = NewPlayerProjectile(1, 2)
	state.enemyProjectiles = []EnemyProjectile{{X: 1, Y: 2}}
	state.enemies = []Enemy{{X: 1, Y: 1, Width: 1, Height: 1, Alive: false}}

	state = Step(state, Input{})
	if state.waveNumber != 1 || !state.waveAdvancePending {
		t.Fatalf("clear must schedule exactly one transition: %+v", state.Snapshot())
	}
	state = Step(state, Input{})
	if state.waveNumber != 2 || state.score != 42 || state.lives != 2 || state.waveAdvancePending {
		t.Fatalf("transition changed run state: %+v", state.Snapshot())
	}
	if state.playerProjectile != nil || len(state.enemyProjectiles) != 0 || state.enemiesAlive() == 0 {
		t.Fatalf("wave-local state was not reset: %+v", state.Snapshot())
	}
	state = Step(state, Input{})
	if state.waveNumber != 2 {
		t.Fatal("wave number incremented without another clear")
	}
}

func TestWavePressureAffectsDifficultyAndCaps(t *testing.T) {
	easy, _ := DifficultyEasy.Config()
	normal, _ := DifficultyNormal.Config()
	hard, _ := DifficultyHard.Config()
	if !(waveFireInterval(easy, 1) > waveFireInterval(normal, 1) && waveFireInterval(normal, 1) > waveFireInterval(hard, 1)) {
		t.Fatal("difficulty presets do not produce distinct firing pressure")
	}
	if waveFireInterval(hard, 100) != minimumFireInterval || wavePressure(100) != maxWavePressure {
		t.Fatal("wave pressure cap/floor not enforced")
	}
}

func TestWaveClearInstallsWaveTwoWithHigherPressure(t *testing.T) {
	state, _ := NewStateWithDifficulty(3, DifficultyNormal, 8)
	state.score, state.lives = 90, 2
	state.enemies[0].Alive = false
	state = Step(state, Input{})
	if !state.waveAdvancePending || state.waveNumber != 1 {
		t.Fatalf("wave clear did not become pending: %+v", state.Snapshot())
	}
	state = Step(state, Input{})
	config := mustConfig(t, DifficultyNormal)
	if state.waveNumber != 2 || state.enemiesAlive() == 0 || state.score != 90 || state.lives != 2 {
		t.Fatalf("wave transition lost run progress: %+v", state.Snapshot())
	}
	if waveFireInterval(config, state.waveNumber) >= waveFireInterval(config, 1) {
		t.Fatal("wave two did not increase firing pressure")
	}
	if enemyMovementIntervalFor(1, config, 2) >= enemyMovementIntervalFor(1, config, 1) {
		t.Fatal("wave two did not increase movement pressure")
	}
}
