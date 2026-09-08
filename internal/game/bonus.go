package game

const (
	bonusTargetScore         = 100
	bonusTargetLifetime Tick = 180
)

type BonusTarget struct {
	X, Y, Width, Height, Direction int
	Age                            Tick
}
type BonusTargetSnapshot struct{ X, Y, Width, Height int }

func (t BonusTarget) Snapshot() BonusTargetSnapshot {
	return BonusTargetSnapshot{t.X, t.Y, t.Width, t.Height}
}
func bonusSpawnTick(t Tick) bool { return t >= 600 && (t == 600 || (t-600)%900 == 0) }
func (s *State) advanceBonusTarget() {
	if s.bonusTarget == nil {
		if bonusSpawnTick(s.waveTick) {
			s.bonusTarget = &BonusTarget{Width: 1, Height: 1, Direction: 1}
		}
		return
	}
	t := s.bonusTarget
	t.Age++
	if t.Age >= bonusTargetLifetime {
		s.bonusTarget = nil
		return
	}
	t.X += t.Direction
	if t.X <= 0 {
		t.X, t.Direction = 0, 1
	}
	if t.X >= s.arenaWidth-t.Width {
		t.X, t.Direction = s.arenaWidth-t.Width, -1
	}
}
