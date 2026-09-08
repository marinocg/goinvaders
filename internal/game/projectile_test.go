package game

import "testing"

func TestFireCreatesCenteredProjectileAbovePlayer(t *testing.T) {
	state, err := NewState(9, 6)
	if err != nil {
		t.Fatal(err)
	}

	snap := Step(state, Input{Fire: true}).Snapshot()
	if snap.PlayerProjectile == nil {
		t.Fatal("expected player projectile")
	}
	if got := *snap.PlayerProjectile; got != (ProjectileSnapshot{X: 4, Y: 3}) {
		t.Fatalf("unexpected projectile spawn: %#v", got)
	}
}

func TestActiveProjectileMovesAndFireDoesNotReplaceIt(t *testing.T) {
	state, err := NewState(7, 5)
	if err != nil {
		t.Fatal(err)
	}

	state = Step(state, Input{Fire: true})
	first := *state.Snapshot().PlayerProjectile
	state = Step(state, Input{MoveX: 1, Fire: true})
	got := state.Snapshot()
	if got.PlayerProjectile == nil || got.PlayerProjectile.X != first.X || got.PlayerProjectile.Y != first.Y-projectileStep {
		t.Fatalf("expected active projectile to move without replacement, got %#v", got.PlayerProjectile)
	}
}

func TestProjectileExpiresAtTopBoundary(t *testing.T) {
	state, err := NewState(5, 5)
	if err != nil {
		t.Fatal(err)
	}
	state = Step(state, Input{Fire: true})
	if got := state.Snapshot().PlayerProjectile; got == nil || got.Y != 2 {
		t.Fatalf("expected projectile to move on firing tick, got %#v", got)
	}
	state = Step(state, Input{})
	if state.Snapshot().PlayerProjectile == nil {
		t.Fatal("expected projectile before boundary")
	}
	state = Step(state, Input{})
	if state.Snapshot().PlayerProjectile != nil {
		t.Fatal("expected projectile to expire at top boundary")
	}
}

func TestNewPlayerProjectile(t *testing.T) {
	projectile := NewPlayerProjectile(2, 3)
	if got := projectile.Snapshot(); got != (ProjectileSnapshot{X: 2, Y: 3}) {
		t.Fatalf("unexpected projectile: %#v", got)
	}
}

func TestFireAtTopRowDoesNotCreateOutOfBoundsProjectile(t *testing.T) {
	state, err := NewState(9, 1)
	if err != nil {
		t.Fatal(err)
	}

	if got := Step(state, Input{Fire: true}).Snapshot().PlayerProjectile; got != nil {
		t.Fatalf("expected no out-of-bounds projectile, got %#v", got)
	}
}
