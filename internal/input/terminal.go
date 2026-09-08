package input

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"

	"golang.org/x/sys/unix"
	"golang.org/x/term"
)

// Terminal owns terminal input decoding and raw-mode lifecycle without exposing
// terminal-specific values to the input contract.
type Terminal struct {
	reader  io.Reader
	setup   func() (func() error, error)
	restore func() error
	active  bool
	pending []byte
}

// NewTerminal constructs an adapter. setup should put the terminal in raw mode
// and return the operation that restores its previous state.
func NewTerminal(reader io.Reader, setup ...func() (func() error, error)) *Terminal {
	if len(setup) > 1 {
		panic("input: terminal setup configured more than once")
	}
	var configure func() (func() error, error)
	if len(setup) > 0 {
		configure = setup[0]
	} else {
		configure = func() (func() error, error) {
			file, ok := reader.(*os.File)
			if !ok {
				return nil, errors.New("input: terminal reader is not an *os.File")
			}
			state, err := term.MakeRaw(int(file.Fd()))
			if err != nil {
				return nil, err
			}
			return func() error { return term.Restore(int(file.Fd()), state) }, nil
		}
	}
	return &Terminal{reader: reader, setup: configure}
}

// Setup enters raw mode. A successful setup is always paired with Restore.
func (t *Terminal) Setup() error {
	if t.active {
		return nil
	}
	if t.setup == nil {
		return errors.New("input: terminal setup is not configured")
	}
	restore, err := t.setup()
	if err != nil {
		return err
	}
	t.restore = restore
	t.active = true
	return nil
}

// Restore attempts to return the terminal to its previous state.
func (t *Terminal) Restore() error {
	if !t.active {
		return nil
	}
	t.active = false
	if t.restore == nil {
		return nil
	}
	return t.restore()
}

// Shutdown is an alias for Restore for callers coordinating terminal cleanup.
func (t *Terminal) Shutdown() error { return t.Restore() }

// Drain reads available synthetic or terminal bytes and returns mapped events.
// It intentionally does not block waiting for another key after input ends.
func (t *Terminal) Drain(ctx context.Context) ([]Event, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if t.reader == nil {
		return nil, errors.New("input: terminal reader is nil")
	}
	data, ended, err := readAvailable(t.reader)
	if err != nil {
		return nil, err
	}
	data = append(t.pending, data...)
	t.pending = nil
	events := make([]Event, 0, len(data))
	for i := 0; i < len(data); i++ {
		// Keep a lone ESC pending while a terminal may still provide an arrow
		// suffix; EOF makes it an unambiguous quit key.
		if data[i] == 0x1b && len(data[i:]) == 1 && !ended {
			t.pending = append(t.pending, data[i:]...)
			break
		}
		command, consumed := decodeTerminalKey(data[i:])
		if consumed == 0 {
			t.pending = append(t.pending, data[i:]...)
			break
		}
		i += consumed - 1
		if command != CommandNoop {
			events = append(events, Event{Command: command})
		}
	}
	return events, nil
}

// readAvailable never waits for EOF. Terminal files are temporarily switched
// to nonblocking mode so a Drain call only consumes bytes already available.
// A bounded read is also important for synthetic readers, which commonly
// return all their data and EOF in one call.
func readAvailable(reader io.Reader) ([]byte, bool, error) {
	if file, ok := reader.(*os.File); ok {
		fd := int(file.Fd())
		flags, err := unix.FcntlInt(uintptr(fd), unix.F_GETFL, 0)
		if err != nil {
			return nil, false, err
		}
		if err := unix.SetNonblock(fd, true); err != nil {
			return nil, false, err
		}
		defer func() { _ = unix.SetNonblock(fd, flags&unix.O_NONBLOCK != 0) }()

		return readUntilUnavailable(reader)
	}

	buffer := make([]byte, 4096)
	n, err := reader.Read(buffer)
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, false, err
	}
	ended := errors.Is(err, io.EOF)
	if err == nil && n > 0 {
		// bytes.Buffer can report data without EOF; probing its exhausted state
		// lets a synthetic lone ESC be decoded without delaying it forever.
		if bufferReader, ok := reader.(*bytes.Buffer); ok && bufferReader.Len() == 0 {
			ended = true
		}
	}
	return buffer[:n], ended, nil
}

func readUntilUnavailable(reader io.Reader) ([]byte, bool, error) {
	data := make([]byte, 0, 256)
	buffer := make([]byte, 4096)
	for {
		n, err := reader.Read(buffer)
		data = append(data, buffer[:n]...)
		if err != nil {
			if errors.Is(err, unix.EAGAIN) || errors.Is(err, unix.EWOULDBLOCK) || errors.Is(err, io.EOF) {
				return data, errors.Is(err, io.EOF), nil
			}
			return nil, false, err
		}
		if n == 0 {
			return data, false, nil
		}
	}
}

func decodeTerminalKey(data []byte) (Command, int) {
	if len(data) == 0 {
		return CommandNoop, 0
	}
	switch data[0] {
	case 'a':
		return CommandMoveLeft, 1
	case 'd':
		return CommandMoveRight, 1
	case ' ', 'f':
		return CommandFire, 1
	case 'q', 0x1b:
		if data[0] == 0x1b && len(data) == 1 {
			return CommandQuit, 1
		}
		if data[0] == 0x1b && len(data) >= 2 && data[1] == '[' {
			for i := 2; i < len(data); i++ {
				if data[i] >= 0x40 && data[i] <= 0x7e {
					switch data[i] {
					case 'D':
						return CommandMoveLeft, i + 1
					case 'C':
						return CommandMoveRight, i + 1
					default:
						return CommandNoop, i + 1
					}
				}
			}
			return CommandNoop, 0
		}
		return CommandQuit, 1
	case 'p':
		return CommandPause, 1
	case '\r', '\n':
		return CommandEnter, 1
	case '1':
		return CommandDifficultyEasy, 1
	case '2':
		return CommandDifficultyNormal, 1
	case '3':
		return CommandDifficultyHard, 1
	default:
		return CommandNoop, 1
	}
}
