package loop

import (
	"context"
	"errors"
	"time"

	"github.com/marinocg/goinvaders/internal/game"
	"github.com/marinocg/goinvaders/internal/input"
)

// TickSource supplies how many fixed simulation ticks should be processed next.
type TickSource interface {
	Next(ctx context.Context) (int, error)
}

// TimeTicker adapts time.Ticker to a deterministic TickSource contract.
type TimeTicker struct {
	ticker *time.Ticker
}

func NewTimeTicker(interval time.Duration) *TimeTicker {
	return &TimeTicker{ticker: time.NewTicker(interval)}
}

func (t *TimeTicker) Next(ctx context.Context) (int, error) {
	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	case <-t.ticker.C:
		return 1, nil
	}
}

func (t *TimeTicker) Stop() {
	t.ticker.Stop()
}

// Run executes deterministic fixed-step orchestration until quit or cancellation.
func Run(ctx context.Context, state game.State, ticks TickSource, src input.Source) (game.State, error) {
	for {
		if err := ctx.Err(); err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return state, nil
			}
			return state, err
		}

		tickCount, err := ticks.Next(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return state, nil
			}
			return state, err
		}

		if tickCount <= 0 {
			continue
		}

		events, err := src.Drain(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return state, nil
			}
			return state, err
		}

		tickInput, quit := collectTickInput(events)
		if quit {
			return state, nil
		}

		state = game.Step(state, tickInput)

		for i := 1; i < tickCount; i++ {
			if err := ctx.Err(); err != nil {
				if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
					return state, nil
				}
				return state, err
			}

			state = game.Step(state, game.Input{})
		}
	}
}

func collectTickInput(events []input.Event) (game.Input, bool) {
	tickInput := game.Input{}

	for _, event := range events {
		switch event.Normalize().Command {
		case input.CommandMoveLeft:
			tickInput.MoveX = -1
		case input.CommandMoveRight:
			tickInput.MoveX = 1
		case input.CommandFire:
			tickInput.Fire = true
		case input.CommandQuit:
			return game.Input{}, true
		}
	}

	return tickInput, false
}
