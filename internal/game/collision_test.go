package game

import (
	"reflect"
	"testing"
)

func TestResolveProjectileCollisionRemovesNearestEnemy(t *testing.T) {
	state, err := NewState(3, 8)
	if err != nil {
		t.Fatal(err)
	}
	state.playerProjectile = NewPlayerProjectile(1, 6)
	state.enemies[0] = Enemy{X: 1, Y: 5, Width: 1, Height: 1, Alive: true}

	state, result := ResolveProjectileCollisions(state)
	if !result.Hit || result.EnemyIndex != 0 || result.ScoreDelta != enemyPointValue {
		t.Fatalf("unexpected collision result: %#v", result)
	}
	if state.playerProjectile != nil || state.enemies[0].Alive {
		t.Fatal("expected projectile and enemy removal")
	}
}

func TestResolveProjectileCollisionIsDeterministic(t *testing.T) {
	state, err := NewState(3, 8)
	if err != nil {
		t.Fatal(err)
	}
	state.playerProjectile = NewPlayerProjectile(1, 6)
	state.enemies[0] = Enemy{X: 1, Y: 5, Width: 1, Height: 1, Alive: true}
	state, result := ResolveProjectileCollisions(state)
	if result.EnemyIndex != 0 || state.enemies[0].Alive {
		t.Fatalf("expected stable first target, result=%#v enemies=%#v", result, state.enemies)
	}
}

func TestResolveProjectileCollisionUsesClosedEnemyBounds(t *testing.T) {
	state, err := NewState(4, 8)
	if err != nil {
		t.Fatal(err)
	}
	state.playerProjectile = NewPlayerProjectile(2, 5)
	state.enemies[0] = Enemy{X: 1, Y: 4, Width: 1, Height: 1, Alive: true}

	next, result := ResolveProjectileCollisions(state)
	if result != (CollisionResult{}) {
		t.Fatalf("adjacent projectile incorrectly collided: %#v", result)
	}
	if next.playerProjectile == nil || !next.enemies[0].Alive {
		t.Fatal("adjacent projectile should not remove entities")
	}
}

func TestResolveProjectileCollisionIncludesEnemyClosedEdges(t *testing.T) {
	state, err := NewState(4, 8)
	if err != nil {
		t.Fatal(err)
	}
	state.playerProjectile = NewPlayerProjectile(1, 6)
	state.enemies[0] = Enemy{X: 1, Y: 4, Width: 1, Height: 2, Alive: true}

	next, result := ResolveProjectileCollisions(state)
	if !result.Hit || result.EnemyIndex != 0 || next.enemies[0].Alive || next.playerProjectile != nil {
		t.Fatalf("expected collision at closed lower edge, result=%#v state=%#v", result, next)
	}
}

func TestResolveProjectileCollisionRejectsMalformedStateAtomically(t *testing.T) {
	state, err := NewState(3, 8)
	if err != nil {
		t.Fatal(err)
	}
	state.playerProjectile = NewPlayerProjectile(1, 6)
	state.enemies[0] = Enemy{X: 1, Y: 4, Width: 0, Height: 1, Alive: true}

	next, result := ResolveProjectileCollisions(state)
	if result != (CollisionResult{}) {
		t.Fatalf("expected no result for invalid state, got %#v", result)
	}
	if next.playerProjectile != state.playerProjectile || !reflect.DeepEqual(next.enemies, state.enemies) {
		t.Fatalf("invalid collision partially mutated state: %#v", next)
	}
}

func TestResolveProjectileCollisionRejectsPlayerEnemyOverlapAtomically(t *testing.T) {
	state, err := NewState(3, 8)
	if err != nil {
		t.Fatal(err)
	}
	state.playerProjectile = NewPlayerProjectile(1, 6)
	state.enemies[0] = Enemy{X: state.playerX, Y: state.playerY, Width: 1, Height: 1, Alive: true}

	next, result := ResolveProjectileCollisions(state)
	if result != (CollisionResult{}) || next.playerProjectile != state.playerProjectile || !reflect.DeepEqual(next.enemies, state.enemies) {
		t.Fatalf("expected overlapping state to remain unchanged, result=%#v state=%#v", result, next)
	}
}

func TestResolveProjectileCollisionChoosesNearestAlongPath(t *testing.T) {
	state, err := NewState(3, 10)
	if err != nil {
		t.Fatal(err)
	}
	state.enemies = []Enemy{{X: 1, Y: 2, Width: 1, Height: 1, Alive: true}, {X: 1, Y: 5, Width: 1, Height: 1, Alive: true}}
	state.playerProjectile = NewPlayerProjectile(1, 6)
	next, result := ResolveProjectileCollisions(state)
	if result.EnemyIndex != 1 || !next.enemies[0].Alive || next.enemies[1].Alive || next.playerProjectile != nil {
		t.Fatalf("expected nearest target at index 1, result=%#v state=%#v", result, next)
	}
}

func TestResolveProjectileCollisionUsesStableFormationOrderOnTie(t *testing.T) {
	state, err := NewState(3, 10)
	if err != nil {
		t.Fatal(err)
	}
	state.enemies = []Enemy{{X: 1, Y: 4, Width: 1, Height: 2, Alive: true}, {X: 1, Y: 4, Width: 1, Height: 2, Alive: true}}
	state.playerProjectile = NewPlayerProjectile(1, 6)
	next, result := ResolveProjectileCollisions(state)
	if result.EnemyIndex != 0 || next.enemies[0].Alive || !next.enemies[1].Alive {
		t.Fatalf("expected stable first target, result=%#v state=%#v", result, next)
	}
}

func TestResolveProjectileCollisionHitsBonusAndAwardsScoreOnce(t *testing.T) {
	state, err := NewState(5, 10)
	if err != nil {
		t.Fatal(err)
	}
	state.enemies = nil
	state.bonusTarget = &BonusTarget{X: 2, Y: 4, Width: 1, Height: 1, Direction: 1}
	state.playerProjectile = NewPlayerProjectile(2, 5)
	next, result := ResolveProjectileCollisions(state)
	if !result.Hit || !result.BonusHit || result.ScoreDelta != bonusTargetScore || next.bonusTarget != nil || next.playerProjectile != nil {
		t.Fatalf("bonus hit did not resolve correctly: result=%#v state=%+v", result, next)
	}
	next, result = ResolveProjectileCollisions(next)
	if result != (CollisionResult{}) || next.score != state.score {
		t.Fatalf("bonus hit was resolved more than once: result=%#v state=%+v", result, next)
	}
}

func TestStepAwardsBonusScoreExactlyOnce(t *testing.T) {
	state, err := NewState(5, 10)
	if err != nil {
		t.Fatal(err)
	}
	state.enemies = []Enemy{{X: 0, Y: 0, Width: 1, Height: 1, Alive: true}}
	state.bonusTarget = &BonusTarget{X: 2, Y: 4, Width: 1, Height: 1, Direction: 1}
	state.playerProjectile = NewPlayerProjectile(2, 5)
	state = Step(state, Input{})
	if state.score != bonusTargetScore || state.bonusTarget != nil || state.playerProjectile != nil {
		t.Fatalf("step did not award and consume bonus: %+v", state)
	}
	state = Step(state, Input{})
	if state.score != bonusTargetScore {
		t.Fatalf("bonus score changed after target removal: %d", state.score)
	}
}

func TestEnemyCollisionTakesPrecedenceOverBonusCollision(t *testing.T) {
	state, err := NewState(5, 10)
	if err != nil {
		t.Fatal(err)
	}
	state.enemies = []Enemy{{X: 2, Y: 4, Width: 1, Height: 1, Alive: true}}
	state.bonusTarget = &BonusTarget{X: 2, Y: 4, Width: 1, Height: 1, Direction: 1}
	state.playerProjectile = NewPlayerProjectile(2, 5)
	next, result := ResolveProjectileCollisions(state)
	if result.BonusHit || result.ScoreDelta != enemyPointValue || next.bonusTarget == nil || next.enemies[0].Alive {
		t.Fatalf("enemy did not take precedence: result=%#v state=%+v", result, next)
	}
}
