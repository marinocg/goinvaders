package ui

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/marinocg/goinvaders/internal/game"
	"github.com/marinocg/goinvaders/internal/input"
	"github.com/marinocg/goinvaders/internal/render"
)

type integrationTerminal struct {
	width, height int
	bytes.Buffer
}

func (t *integrationTerminal) Size() (int, int, error) { return t.width, t.height, nil }

func TestPlayableFlowChangesRenderedTerminalFrame(t *testing.T) {
	terminal := &integrationTerminal{width: render.MinWidth, height: render.MinHeight}
	inputSource := &flowInput{steps: [][]input.Event{
		{{Command: input.CommandEnter}},
		{{Command: input.CommandMoveRight}, {Command: input.CommandFire}},
		{{Command: input.CommandQuit}},
	}}
	renderer := render.NewTerminalRenderer(terminal)
	flow := Flow{
		Ticks: &flowTicks{}, Polls: &flowPolls{}, Input: inputSource, Render: renderer,
		NewGame: func() (game.State, error) { return game.NewState(render.MinWidth, render.MinHeight) },
	}
	if _, err := flow.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	output := terminal.String()
	if !strings.Contains(output, "Press Enter to start") || !strings.Contains(output, "A/D move") {
		t.Fatalf("output omitted start/active contract: %q", output)
	}
	if strings.Count(output, "\n|") < 2 || strings.Count(output, "\x1b[H") < 2 {
		t.Fatalf("output omitted visible fired state or active redraw: %q", output)
	}
	if strings.Count(output, "\x1b[2J") > 1 {
		t.Fatalf("normal flow repeatedly cleared the screen: %q", output)
	}
}

func TestPlayableFlowRendersFiredProjectileAsVisibleChange(t *testing.T) {
	terminal := &integrationTerminal{width: render.MinWidth, height: render.MinHeight}
	flow := Flow{
		Ticks: &flowTicks{}, Polls: &flowPolls{},
		Input: &flowInput{steps: [][]input.Event{
			{{Command: input.CommandEnter}},
			{{Command: input.CommandFire}},
			{{Command: input.CommandQuit}},
		}},
		Render: render.NewTerminalRenderer(terminal),
		NewGame: func() (game.State, error) {
			return game.NewState(render.MinWidth, render.MinHeight)
		},
	}
	if _, err := flow.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	output := terminal.String()
	if !strings.Contains(output, "|") {
		t.Fatalf("fired projectile was not visible in output: %q", output)
	}
}
