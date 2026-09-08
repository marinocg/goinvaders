package game

import "testing"

func TestBonusTargetSpawnsMovesAndExpires(t *testing.T) {
	state, err := NewState(40, 18)
	if err != nil {
		t.Fatal(err)
	}

	state.waveTick = 599
	state.advanceBonusTarget()
	if state.bonusTarget != nil {
		t.Fatal("bonus target spawned too early")
	}

	state.waveTick = 600
	state.advanceBonusTarget()
	if state.bonusTarget == nil {
		t.Fatal("bonus target did not spawn")
	}
	if state.bonusTarget.Direction != 1 {
		t.Fatalf("bonus target direction = %d, want 1", state.bonusTarget.Direction)
	}

	initialX := state.bonusTarget.X
	state.advanceBonusTarget()
	if state.bonusTarget.X != initialX+1 || state.bonusTarget.Age != 1 {
		t.Fatalf("bonus target did not advance: %+v", *state.bonusTarget)
	}

	state.bonusTarget.Age = bonusTargetLifetime - 1
	state.advanceBonusTarget()
	if state.bonusTarget != nil {
		t.Fatal("bonus target did not expire")
	}
}

func TestBonusTargetSnapshotContainsRendererState(t *testing.T) {
	target := BonusTarget{X: 7, Y: 3, Width: 2, Height: 1, Direction: -1, Age: 4}
	if got := target.Snapshot(); got != (BonusTargetSnapshot{X: 7, Y: 3, Width: 2, Height: 1}) {
		t.Fatalf("unexpected bonus snapshot: %#v", got)
	}
}

func TestBonusTargetLifeLossClearsTargetButPreservesSchedule(t *testing.T) {
	state, err := NewState(40, 18)
	if err != nil {
		t.Fatal(err)
	}
	state.waveTick = 600
	state.advanceBonusTarget()
	state.loseLife()
	if state.bonusTarget != nil || state.waveTick != 600 {
		t.Fatalf("life loss did not reset target while preserving schedule: %+v", state)
	}
	state.advanceBonusTarget()
	if state.bonusTarget == nil {
		t.Fatal("target did not respawn at the deterministic scheduled tick")
	}
}

func TestBonusTargetWaveTransitionResetsSchedule(t *testing.T) {
	state, err := NewState(40, 18)
	if err != nil {
		t.Fatal(err)
	}
	state.waveTick = 600
	state.advanceBonusTarget()
	state.waveAdvancePending = true
	if !state.installWave() {
		t.Fatal("failed to install next wave")
	}
	if state.bonusTarget != nil || state.waveTick != 0 {
		t.Fatalf("wave transition did not reset bonus schedule: %+v", state)
	}
	state.waveTick = 600
	state.advanceBonusTarget()
	if state.bonusTarget == nil {
		t.Fatal("target did not spawn on the next wave schedule")
	}
}

func TestBonusTargetSpawnsOnRepeatedWaveSchedule(t *testing.T) {
	state, err := NewState(40, 18)
	if err != nil {
		t.Fatal(err)
	}
	for _, tick := range []Tick{600, 1500, 2400} {
		state.waveTick = tick
		state.advanceBonusTarget()
		if state.bonusTarget == nil {
			t.Fatalf("target did not spawn at scheduled tick %d", tick)
		}
		state.bonusTarget = nil
	}
}

func TestBonusTargetHitScoresAndMissDoesNotScore(t *testing.T) {
	state, err := NewState(40, 18)
	if err != nil {
		t.Fatal(err)
	}
	state.bonusTarget = &BonusTarget{X: 5, Y: 5, Width: 1, Height: 1, Direction: 1}
	state.playerProjectile = NewPlayerProjectile(5, 6)
	state = Step(state, Input{})
	if state.score != bonusTargetScore || state.bonusTarget != nil || state.playerProjectile != nil {
		t.Fatalf("bonus hit did not score and consume target: %#v", state.Snapshot())
	}
	state.bonusTarget = &BonusTarget{X: 5, Y: 5, Width: 1, Height: 1, Direction: 1}
	state.playerProjectile = NewPlayerProjectile(4, 6)
	state = Step(state, Input{})
	if state.score != bonusTargetScore || state.bonusTarget == nil {
		t.Fatalf("bonus miss changed score or target: %#v", state.Snapshot())
	}
}
