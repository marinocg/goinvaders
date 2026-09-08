package game

// Outcome is the authoritative terminal result of a game.
type Outcome uint8

const (
	OutcomePlaying Outcome = iota
	OutcomeWin
	OutcomeGameOver
)

func (o Outcome) Terminal() bool { return o != OutcomePlaying }
