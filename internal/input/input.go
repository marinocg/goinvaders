package input

import "context"

// Command is a logical, terminal-agnostic input command consumed by the loop.
type Command uint8

const (
	// CommandNoop is a safe no-op command used to normalize invalid values.
	CommandNoop Command = iota
	CommandMoveLeft
	CommandMoveRight
	CommandFire
	CommandQuit
	CommandPause
	CommandEnter
	CommandDifficultyEasy
	CommandDifficultyNormal
	CommandDifficultyHard
)

// Valid reports whether c is one of the closed set of supported commands.
func (c Command) Valid() bool {
	return c <= CommandDifficultyHard
}

// Normalize maps unsupported command values to a safe no-op.
func (c Command) Normalize() Command {
	if !c.Valid() {
		return CommandNoop
	}

	return c
}

// Event is a plain logical input value.
type Event struct {
	Command Command
}

// Valid reports whether e contains a supported logical command.
func (e Event) Valid() bool {
	return e.Command.Valid()
}

// Normalize maps unsupported event values to a safe no-op event.
func (e Event) Normalize() Event {
	e.Command = e.Command.Normalize()
	return e
}

// Source drains pending logical input events in deterministic order.
//
// Implementations must return events in oldest-first order and avoid
// terminal-library types in their public contract.
type Source interface {
	Drain(ctx context.Context) ([]Event, error)
}
