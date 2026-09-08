package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/marinocg/goinvaders/internal/game"
	"github.com/marinocg/goinvaders/internal/input"
	"github.com/marinocg/goinvaders/internal/loop"
	"github.com/marinocg/goinvaders/internal/render"
	"github.com/marinocg/goinvaders/internal/ui"
	"golang.org/x/term"
)

const (
	startupArenaWidth  = 40
	startupArenaHeight = 18
	startupTickRate    = time.Second / 60
	startupPollRate    = 50 * time.Millisecond
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	ticks := loop.NewTimeTicker(startupTickRate)
	defer ticks.Stop()
	polls := loop.NewTimeTicker(startupPollRate)
	defer polls.Stop()

	terminal := fileTerminal{File: os.Stdout}
	source := input.NewTerminal(os.Stdin)
	if err := source.Setup(); err != nil {
		log.Fatalf("initialize terminal input: %v", err)
	}
	defer source.Shutdown()

	flow := ui.Flow{
		Ticks:  ticks,
		Polls:  polls,
		Input:  source,
		Render: render.NewTerminalRenderer(terminal),
		NewGame: func() (game.State, error) {
			return game.NewState(startupArenaWidth, startupArenaHeight)
		},
		NewGameWithDifficulty: func(d game.Difficulty) (game.State, error) {
			return game.NewStateWithDifficulty(startupArenaWidth, d, startupArenaHeight)
		},
	}
	_, err := flow.Run(ctx)
	if err != nil {
		log.Fatalf("run game: %v", err)
	}
}

type fileTerminal struct{ *os.File }

func (t fileTerminal) Size() (int, int, error) {
	return term.GetSize(int(t.Fd()))
}
