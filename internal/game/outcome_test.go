package game

import "testing"

func TestOutcomeTerminal(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		outcome  Outcome
		terminal bool
	}{
		{name: "playing", outcome: OutcomePlaying, terminal: false},
		{name: "win", outcome: OutcomeWin, terminal: true},
		{name: "game over", outcome: OutcomeGameOver, terminal: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.outcome.Terminal(); got != test.terminal {
				t.Fatalf("Outcome(%d).Terminal() = %t, want %t", test.outcome, got, test.terminal)
			}
		})
	}
}
