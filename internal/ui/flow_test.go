package ui

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/marinocg/goinvaders/internal/game"
	"github.com/marinocg/goinvaders/internal/input"
	"github.com/marinocg/goinvaders/internal/render"
)

type flowInput struct {
	steps [][]input.Event
	index int
}

func (s *flowInput) Drain(context.Context) ([]input.Event, error) {
	if s.index == len(s.steps) {
		return nil, errors.New("input exhausted")
	}
	events := s.steps[s.index]
	s.index++
	return events, nil
}

type flowTicks struct{ calls int }

func (t *flowTicks) Next(context.Context) (int, error) {
	t.calls++
	return 1, nil
}

type flowPolls struct{ calls int }

func (p *flowPolls) Next(context.Context) (int, error) {
	p.calls++
	return 1, nil
}

type flowRenderer struct {
	presentations []render.Presentation
	snapshots     []game.Snapshot
	outcomes      []game.Outcome
	trace         *[]string
	shutdownErr   error
}

func (r *flowRenderer) Render(snapshot game.Snapshot, presentation render.Presentation) error {
	r.presentations = append(r.presentations, presentation)
	r.snapshots = append(r.snapshots, snapshot)
	r.outcomes = append(r.outcomes, snapshot.Outcome)
	if r.trace != nil {
		*r.trace = append(*r.trace, string(rune('0'+int(snapshot.Tick))))
	}
	return nil
}
func (r *flowRenderer) Shutdown() error { return r.shutdownErr }

type tooSmallFlowRenderer struct{}

func (tooSmallFlowRenderer) Render(game.Snapshot, render.Presentation) error {
	return render.ErrTerminalTooSmall
}
func (tooSmallFlowRenderer) Shutdown() error { return nil }

type recoveringFlowRenderer struct {
	tooSmallCalls int
	renders       int
}

func (r *recoveringFlowRenderer) Render(game.Snapshot, render.Presentation) error {
	r.renders++
	if r.renders == 1 {
		r.tooSmallCalls++
		return render.ErrTerminalTooSmall
	}
	return nil
}

func (r *recoveringFlowRenderer) Shutdown() error { return nil }

func TestFlowDoesNotTickBeforeStart(t *testing.T) {
	ticks := &flowTicks{}
	flow := Flow{
		Ticks: ticks, Polls: &flowPolls{},
		Input:   &flowInput{steps: [][]input.Event{{}}},
		Render:  &flowRenderer{},
		NewGame: func() (game.State, error) { return game.NewState(4, 4) },
	}
	if _, err := flow.Run(context.Background()); err == nil || ticks.calls != 0 {
		t.Fatalf("Run() error=%v, ticks=%d; wanted input exhaustion before any tick", err, ticks.calls)
	}
}

func TestFlowUsesBoundedPollForIdleStart(t *testing.T) {
	polls := &flowPolls{}
	ticks := &flowTicks{}
	renderer := &flowRenderer{}
	flow := Flow{
		Ticks: ticks, Polls: polls,
		Input:  &flowInput{steps: [][]input.Event{{{Command: input.CommandEnter}}, {{Command: input.CommandQuit}}}},
		Render: renderer, NewGame: func() (game.State, error) { return game.NewState(4, 4) },
	}
	if _, err := flow.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if polls.calls != 1 || ticks.calls != 1 || len(renderer.presentations) != 2 {
		t.Fatalf("polls=%d ticks=%d, want one idle poll and one active tick", polls.calls, ticks.calls)
	}
}

func TestFlowRequiresBoundedPollSource(t *testing.T) {
	flow := Flow{
		Ticks: &flowTicks{}, Input: &flowInput{}, Render: &flowRenderer{},
		NewGame: func() (game.State, error) { return game.NewState(4, 4) },
	}
	if err := runFlowForTest(flow); err == nil || err.Error() != "ui: incomplete flow configuration" {
		t.Fatalf("Run() error=%v, want missing poll configuration error", err)
	}
}

func TestFlowDoesNotRepaintUnchangedIdleState(t *testing.T) {
	polls := &flowPolls{}
	renderer := &flowRenderer{}
	flow := Flow{
		Ticks: &flowTicks{}, Polls: polls,
		Input:  &flowInput{steps: [][]input.Event{{}, {{Command: input.CommandQuit}}}},
		Render: renderer, NewGame: func() (game.State, error) { return game.NewState(4, 4) },
	}
	if _, err := flow.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if polls.calls != 2 || len(renderer.presentations) != 1 {
		t.Fatalf("polls=%d, renders=%d, want two polls and one initial render", polls.calls, len(renderer.presentations))
	}
}

func TestFlowDoesNotTickWhilePaused(t *testing.T) {
	ticks := &flowTicks{}
	flow := Flow{
		Ticks: ticks, Polls: &flowPolls{},
		Input:   &flowInput{steps: [][]input.Event{{{Command: input.CommandEnter}}, {{Command: input.CommandPause}}, {{Command: input.CommandQuit}}}},
		Render:  &flowRenderer{},
		NewGame: func() (game.State, error) { return game.NewState(4, 4) },
	}
	if _, err := flow.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if ticks.calls != 1 {
		t.Fatalf("ticks=%d, want one active-play tick and none while paused", ticks.calls)
	}
}

func TestFlowDoesNotTickOnResume(t *testing.T) {
	ticks := &flowTicks{}
	flow := Flow{
		Ticks: ticks, Polls: &flowPolls{},
		Input:   &flowInput{steps: [][]input.Event{{{Command: input.CommandEnter}}, {{Command: input.CommandPause}}, {{Command: input.CommandPause}}, {{Command: input.CommandQuit}}}},
		Render:  &flowRenderer{},
		NewGame: func() (game.State, error) { return game.NewState(4, 4) },
	}
	if _, err := flow.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if ticks.calls != 1 {
		t.Fatalf("ticks=%d, want one active-play tick and none on pause or resume", ticks.calls)
	}
}

func TestFlowSuppressesGameplayCommandsFromBatchReceivedWhilePaused(t *testing.T) {
	ticks := &flowTicks{}
	var initial game.Snapshot
	flow := Flow{
		Ticks: ticks, Polls: &flowPolls{},
		Input: &flowInput{steps: [][]input.Event{
			{{Command: input.CommandEnter}},
			{{Command: input.CommandPause}},
			{{Command: input.CommandMoveLeft}, {Command: input.CommandPause}, {Command: input.CommandQuit}},
		}},
		Render: &flowRenderer{},
		NewGame: func() (game.State, error) {
			state, err := game.NewState(4, 4)
			initial = state.Snapshot()
			return state, err
		},
	}
	state, err := flow.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if ticks.calls != 1 {
		t.Fatalf("ticks=%d, want one active-play tick", ticks.calls)
	}
	if got := state.Snapshot().Player.X; got != initial.Player.X {
		t.Fatalf("player x=%d, want %d; movement from paused batch was forwarded", got, initial.Player.X)
	}
}

func TestFlowRendersPresentationBoundariesAndShutdownErrors(t *testing.T) {
	renderer := &flowRenderer{shutdownErr: errors.New("shutdown")}
	flow := Flow{
		Ticks: &flowTicks{}, Polls: &flowPolls{},
		Input:   &flowInput{steps: [][]input.Event{{{Command: input.CommandEnter}}, {{Command: input.CommandPause}}, {{Command: input.CommandPause}}, {{Command: input.CommandQuit}}}},
		Render:  renderer,
		NewGame: func() (game.State, error) { return game.NewState(4, 4) },
	}
	if _, err := flow.Run(context.Background()); !errors.Is(err, renderer.shutdownErr) {
		t.Fatalf("Run() error = %v, want shutdown error", err)
	}
	if len(renderer.presentations) != 4 {
		t.Fatalf("renders = %d, want one per state boundary", len(renderer.presentations))
	}
	if renderer.presentations[0].Screen != render.ScreenStart || renderer.presentations[1].Screen != render.ScreenActive {
		t.Fatalf("start presentations = %#v", renderer.presentations[:2])
	}
	if !renderer.presentations[2].Paused || renderer.presentations[3].Paused {
		t.Fatalf("pause/resume presentations = %#v", renderer.presentations[2:])
	}
}

func TestFlowDoesNotTickWhenTerminalIsTooSmall(t *testing.T) {
	ticks := &flowTicks{}
	flow := Flow{
		Ticks: ticks, Polls: &flowPolls{},
		Input:   &flowInput{steps: [][]input.Event{{{Command: input.CommandEnter}}, {{Command: input.CommandQuit}}}},
		Render:  tooSmallFlowRenderer{},
		NewGame: func() (game.State, error) { return game.NewState(4, 4) },
	}
	if _, err := flow.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if ticks.calls != 0 {
		t.Fatalf("ticks=%d, want no ticks while terminal is too small", ticks.calls)
	}
}

func TestFlowRechecksTerminalSizeAfterTooSmallRender(t *testing.T) {
	renderer := &recoveringFlowRenderer{}
	flow := Flow{
		Ticks: &flowTicks{}, Polls: &flowPolls{},
		Input:   &flowInput{steps: [][]input.Event{{}, {{Command: input.CommandQuit}}}},
		Render:  renderer,
		NewGame: func() (game.State, error) { return game.NewState(4, 4) },
	}
	if _, err := flow.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if renderer.tooSmallCalls != 1 || renderer.renders != 2 {
		t.Fatalf("renders=%d, too-small renders=%d; want one retry after a poll", renderer.renders, renderer.tooSmallCalls)
	}
}

func TestFlowUsesRenderBoundariesBeforeEachTick(t *testing.T) {
	trace := []string{}
	ticks := &flowTicks{}
	renderer := &flowRenderer{trace: &trace}
	flow := Flow{
		Ticks: ticks, Polls: &flowPolls{},
		Input:   &flowInput{steps: [][]input.Event{{{Command: input.CommandEnter}}, {{Command: input.CommandQuit}}}},
		Render:  renderer,
		NewGame: func() (game.State, error) { return game.NewState(4, 4) },
	}
	if _, err := flow.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if ticks.calls != 1 {
		t.Fatalf("ticks=%d, want one tick between render boundaries", ticks.calls)
	}
	if !reflect.DeepEqual(trace, []string{"0", "1"}) {
		t.Fatalf("render trace=%v, want [0 1]", trace)
	}
}

func TestFlowEndToEndStartMovementFireAndVisibleStateChange(t *testing.T) {
	renderer := &flowRenderer{}
	flow := Flow{
		Ticks: &flowTicks{}, Polls: &flowPolls{},
		Input: &flowInput{steps: [][]input.Event{
			{{Command: input.CommandEnter}},
			{{Command: input.CommandMoveRight}, {Command: input.CommandFire}},
			{{Command: input.CommandQuit}},
		}},
		Render:  renderer,
		NewGame: func() (game.State, error) { return game.NewState(40, 18) },
	}
	state, err := flow.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	snapshot := state.Snapshot()
	if snapshot.Tick == 0 {
		t.Fatal("Enter did not lead to simulation advancement")
	}
	if len(renderer.snapshots) < 2 {
		t.Fatalf("snapshots = %d, want start and active frames", len(renderer.snapshots))
	}
	start := renderer.snapshots[0]
	changed := false
	for _, snapshot := range renderer.snapshots[1:] {
		if snapshot.Player.X != start.Player.X && snapshot.PlayerProjectile != nil {
			changed = true
			break
		}
	}
	if !changed {
		t.Fatalf("visible gameplay snapshots = %#v, want later movement and projectile", renderer.snapshots)
	}
	if len(renderer.presentations) < 2 || renderer.presentations[0].Screen != render.ScreenStart || renderer.presentations[1].Screen != render.ScreenActive {
		t.Fatalf("presentations = %#v, want start then active", renderer.presentations)
	}
}

func TestFlowPresentsGameOverAndDoesNotTickUntilRestart(t *testing.T) {
	steps := [][]input.Event{{{Command: input.CommandEnter}}}
	// A one-column formation reaches the loss line every 18 ticks; three
	// lives therefore require 54 ticks before the game-over boundary.
	for i := 0; i < 179; i++ {
		steps = append(steps, []input.Event{})
	}
	steps = append(steps, []input.Event{{Command: input.CommandEnter}}, []input.Event{{Command: input.CommandQuit}})
	ticks := &flowTicks{}
	renderer := &flowRenderer{}
	flow := Flow{
		Ticks: ticks, Polls: &flowPolls{}, Input: &flowInput{steps: steps}, Render: renderer,
		NewGame: func() (game.State, error) { return game.NewState(1, 4) },
	}
	if _, err := flow.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if ticks.calls != 91 {
		t.Fatalf("ticks=%d, want no tick while game over or on restart", ticks.calls)
	}
	if len(renderer.outcomes) < 2 || renderer.outcomes[len(renderer.outcomes)-2] != game.OutcomeGameOver || renderer.outcomes[len(renderer.outcomes)-1] != game.OutcomePlaying {
		t.Fatalf("outcomes=%v, want game-over then restarted playing state", renderer.outcomes)
	}
}

func TestFlowContinuesAfterWaveClearAndQuits(t *testing.T) {
	steps := [][]input.Event{{{Command: input.CommandEnter}, {Command: input.CommandFire}}}
	for i := 0; i < 100; i++ {
		steps = append(steps, []input.Event{{Command: input.CommandFire}})
	}
	steps = append(steps, []input.Event{{Command: input.CommandQuit}})
	factoryCalls := 0
	renderer := &flowRenderer{}
	flow := Flow{
		Ticks: &flowTicks{}, Polls: &flowPolls{}, Input: &flowInput{steps: steps}, Render: renderer,
		NewGame: func() (game.State, error) {
			factoryCalls++
			state, err := game.NewState(1, 4)
			return state, err
		},
	}
	if _, err := flow.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if factoryCalls != 1 {
		t.Fatalf("NewGame calls=%d, want no restart after wave clear", factoryCalls)
	}
	for _, outcome := range renderer.outcomes {
		if outcome == game.OutcomeWin {
			t.Fatalf("outcomes=%v, wave clear must not present a terminal win", renderer.outcomes)
		}
	}
}

func TestFlowDifficultySelectionConstructsRequestedGameAndPreservesInputSemantics(t *testing.T) {
	var selected game.Difficulty
	renderer := &flowRenderer{}
	flow := Flow{
		Ticks: &flowTicks{}, Polls: &flowPolls{},
		Input:   &flowInput{steps: [][]input.Event{{{Command: input.CommandDifficultyHard}}, {{Command: input.CommandEnter}}, {{Command: input.CommandMoveRight}}, {{Command: input.CommandQuit}}}},
		Render:  renderer,
		NewGame: func() (game.State, error) { return game.NewState(40, 18) },
		NewGameWithDifficulty: func(difficulty game.Difficulty) (game.State, error) {
			selected = difficulty
			return game.NewStateWithDifficulty(40, difficulty, 18)
		},
	}
	state, err := flow.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if selected != game.DifficultyHard || state.Snapshot().Difficulty != game.DifficultyHard {
		t.Fatalf("selected difficulty=%v, state=%v", selected, state.Snapshot().Difficulty)
	}
	if state.Snapshot().Player.X == (40-1)/2 {
		t.Fatal("movement input was lost after difficulty selection")
	}
}

func TestFlowPropagatesInputAndRenderErrorsAndFactoryErrors(t *testing.T) {
	want := errors.New("fatal")
	for _, test := range []struct {
		name string
		flow Flow
	}{
		{"input", Flow{Ticks: &flowTicks{}, Polls: &flowPolls{}, Input: &flowInput{steps: [][]input.Event{{}}}, Render: &flowRenderer{}, NewGame: func() (game.State, error) { return game.NewState(4, 4) }}},
		{"ticks", Flow{Ticks: errorFlowTicks{err: want}, Polls: &flowPolls{}, Input: &flowInput{steps: [][]input.Event{{{Command: input.CommandEnter}}}}, Render: &flowRenderer{}, NewGame: func() (game.State, error) { return game.NewState(4, 4) }}},
		{"render", Flow{Ticks: &flowTicks{}, Polls: &flowPolls{}, Input: &flowInput{}, Render: errorFlowRenderer{err: want}, NewGame: func() (game.State, error) { return game.NewState(4, 4) }}},
		{"factory", Flow{Ticks: &flowTicks{}, Polls: &flowPolls{}, Input: &flowInput{}, Render: &flowRenderer{}, NewGame: func() (game.State, error) { return game.State{}, want }}},
	} {
		t.Run(test.name, func(t *testing.T) {
			flow := test.flow
			if test.name == "input" {
				flow.Input = errorFlowInput{err: want}
			} else if test.name == "render" {
				flow.Input = &flowInput{}
			}
			if !errors.Is(runFlowForTest(flow), want) {
				t.Fatalf("Run() did not return %v", want)
			}
		})
	}
}

func runFlowForTest(flow Flow) error { _, err := flow.Run(context.Background()); return err }

type errorFlowRenderer struct{ err error }

func (r errorFlowRenderer) Render(game.Snapshot, render.Presentation) error { return r.err }
func (errorFlowRenderer) Shutdown() error                                   { return nil }

type errorFlowTicks struct{ err error }

func (t errorFlowTicks) Next(context.Context) (int, error) { return 0, t.err }

type errorFlowInput struct{ err error }

func (s errorFlowInput) Drain(context.Context) ([]input.Event, error) { return nil, s.err }
