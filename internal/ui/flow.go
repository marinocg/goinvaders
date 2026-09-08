package ui

import (
	"context"
	"errors"

	"github.com/marinocg/goinvaders/internal/game"
	"github.com/marinocg/goinvaders/internal/input"
	"github.com/marinocg/goinvaders/internal/loop"
	"github.com/marinocg/goinvaders/internal/render"
)

type Renderer interface {
	Render(game.Snapshot, render.Presentation) error
	Shutdown() error
}

type StateFactory func() (game.State, error)
type DifficultyStateFactory func(game.Difficulty) (game.State, error)

// PollSource provides the bounded cadence used by non-simulation states.
type PollSource interface {
	Next(context.Context) (int, error)
}

type Flow struct {
	Ticks                 loop.TickSource
	Polls                 PollSource
	Input                 input.Source
	Render                Renderer
	NewGame               StateFactory
	NewGameWithDifficulty DifficultyStateFactory
}

func (f Flow) Run(ctx context.Context) (state game.State, err error) {
	if f.Ticks == nil || f.Polls == nil || f.Input == nil || f.Render == nil || f.NewGame == nil {
		return state, errors.New("ui: incomplete flow configuration")
	}
	defer func() {
		if shutdownErr := f.Render.Shutdown(); err == nil {
			err = shutdownErr
		}
	}()
	state, err = f.NewGame()
	if err != nil {
		return state, err
	}
	started, paused := false, false
	difficulty := game.DifficultyNormal
	newGame := func() (game.State, error) {
		if f.NewGameWithDifficulty != nil {
			return f.NewGameWithDifficulty(difficulty)
		}
		return f.NewGame()
	}
	renderNeeded := true
	tooSmall := false
	for {
		presentation := render.Presentation{Paused: paused, Screen: screen(started), Difficulty: difficulty}
		if renderNeeded {
			err = f.Render.Render(state.Snapshot(), presentation)
			if errors.Is(err, render.ErrTerminalTooSmall) {
				// Keep the size warning visible, but do not advance gameplay until
				// the renderer can draw a normal frame again.
				err = nil
				tooSmall = true
			} else if err != nil {
				return state, err
			} else {
				tooSmall = false
			}
			renderNeeded = false
		}
		if tooSmall || !started || paused || state.Snapshot().Outcome.Terminal() {
			if _, err = f.Polls.Next(ctx); err != nil {
				return state, contextError(err)
			}
			// A poll is also the resize check while the renderer reports that
			// the terminal is too small. Retry rendering at the bounded poll
			// cadence so the flow can recover after a resize.
			if tooSmall {
				renderNeeded = true
			}
		}
		events, err := f.Input.Drain(ctx)
		if err != nil {
			return state, contextError(err)
		}
		pausedAtDrain := paused
		var in game.Input
		resumed := false
		for _, event := range events {
			switch event.Normalize().Command {
			case input.CommandDifficultyEasy, input.CommandDifficultyNormal, input.CommandDifficultyHard:
				if !started {
					difficulty = map[input.Command]game.Difficulty{input.CommandDifficultyEasy: game.DifficultyEasy, input.CommandDifficultyNormal: game.DifficultyNormal, input.CommandDifficultyHard: game.DifficultyHard}[event.Normalize().Command]
					state, err = newGame()
					if err != nil {
						return state, err
					}
					renderNeeded = true
				}
			case input.CommandQuit:
				return state, nil
			case input.CommandPause:
				if started && !state.Snapshot().Outcome.Terminal() {
					if paused {
						resumed = true
					}
					paused = !paused
					renderNeeded = true
				}
			case input.CommandEnter:
				if !started {
					started = true
					renderNeeded = true
				} else if state.Snapshot().Outcome.Terminal() {
					state, err = newGame()
					if err != nil {
						return state, err
					}
					renderNeeded = true
				}
			case input.CommandMoveLeft, input.CommandMoveRight, input.CommandFire:
				if !started || pausedAtDrain || paused || state.Snapshot().Outcome.Terminal() {
					continue
				}
				switch event.Normalize().Command {
				case input.CommandMoveLeft:
					in.MoveX = -1
				case input.CommandMoveRight:
					in.MoveX = 1
				case input.CommandFire:
					in.Fire = true
				}
			}
		}
		if tooSmall || !started || paused || resumed || state.Snapshot().Outcome.Terminal() {
			continue
		}
		ticks, err := f.Ticks.Next(ctx)
		if err != nil {
			return state, contextError(err)
		}
		for i := 0; i < ticks; i++ {
			state = game.Step(state, in)
			in = game.Input{}
		}
		renderNeeded = true
	}
}

func screen(started bool) render.Screen {
	if started {
		return render.ScreenActive
	}
	return render.ScreenStart
}
func contextError(err error) error {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return nil
	}
	return err
}
