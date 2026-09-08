package input

import (
	"bytes"
	"context"
	"errors"
	"reflect"
	"testing"
)

func TestTerminalDrainMapsKeysInArrivalOrder(t *testing.T) {
	src := NewTerminal(bytes.NewBufferString("xa\x1b[C fdpq\x1b[D"), nil)
	got, err := src.Drain(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	want := []Event{{CommandMoveLeft}, {CommandMoveRight}, {CommandFire}, {CommandFire}, {CommandMoveRight}, {CommandPause}, {CommandQuit}, {CommandMoveLeft}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Drain() = %#v, want %#v", got, want)
	}
}

func TestTerminalDrainMapsStandaloneEscapeToQuit(t *testing.T) {
	src := NewTerminal(bytes.NewBuffer([]byte{0x1b}))
	got, err := src.Drain(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	want := []Event{{CommandQuit}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Drain() = %#v, want %#v", got, want)
	}
}

func TestTerminalDrainIgnoresUIAndUnsupportedKeys(t *testing.T) {
	src := NewTerminal(bytes.NewBufferString("p\r\nxyz\x1b[A"), nil)
	got, err := src.Drain(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, []Event{{CommandPause}, {CommandEnter}, {CommandEnter}}) {
		t.Fatalf("Drain() = %#v, want no gameplay events", got)
	}
}

func TestTerminalDrainMapsEveryDocumentedKey(t *testing.T) {
	data := []byte{'a', 'd', ' ', 'f', 'q', 0x1b, '[', 'D', 0x1b, '[', 'C', 'p', '\n'}
	got, err := NewTerminal(bytes.NewReader(data)).Drain(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	want := []Event{{CommandMoveLeft}, {CommandMoveRight}, {CommandFire}, {CommandFire}, {CommandQuit}, {CommandMoveLeft}, {CommandMoveRight}, {CommandPause}, {CommandEnter}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Drain() = %#v, want %#v", got, want)
	}
}

func TestTerminalDrainPropagatesReaderAndContextErrors(t *testing.T) {
	want := errors.New("read failed")
	_, err := NewTerminal(errorReader{err: want}).Drain(context.Background())
	if !errors.Is(err, want) {
		t.Fatalf("Drain() error = %v, want %v", err, want)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := NewTerminal(bytes.NewBufferString("a")).Drain(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("Drain() error = %v, want context cancellation", err)
	}
}

type errorReader struct{ err error }

func (r errorReader) Read([]byte) (int, error) { return 0, r.err }

func TestTerminalDrainPreservesSplitEscapeSequence(t *testing.T) {
	src := NewTerminal(&chunkReader{chunks: [][]byte{{0x1b}, {'['}, {'C'}}})
	for i, want := range []([]Event){{}, {}, {{CommandMoveRight}}} {
		got, err := src.Drain(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("Drain() call %d = %#v, want %#v", i+1, got, want)
		}
	}
}

type chunkReader struct {
	chunks [][]byte
}

func (r *chunkReader) Read(dst []byte) (int, error) {
	if len(r.chunks) == 0 {
		return 0, errors.New("unexpected read")
	}
	chunk := r.chunks[0]
	r.chunks = r.chunks[1:]
	return copy(dst, chunk), nil
}

func TestTerminalSetupAndRestore(t *testing.T) {
	setupCalled, restoreCalled := false, false
	src := NewTerminal(bytes.NewBuffer(nil), func() (func() error, error) {
		setupCalled = true
		return func() error { restoreCalled = true; return nil }, nil
	})
	if err := src.Setup(); err != nil || !setupCalled {
		t.Fatalf("Setup() error=%v called=%v", err, setupCalled)
	}
	if err := src.Shutdown(); err != nil || !restoreCalled {
		t.Fatalf("Shutdown() error=%v restored=%v", err, restoreCalled)
	}
}

func TestTerminalSetupAndRestoreErrors(t *testing.T) {
	setupErr := errors.New("setup")
	src := NewTerminal(bytes.NewBuffer(nil), func() (func() error, error) { return nil, setupErr })
	if !errors.Is(src.Setup(), setupErr) {
		t.Fatal("Setup() did not propagate error")
	}
	restoreErr := errors.New("restore")
	src = NewTerminal(bytes.NewBuffer(nil), func() (func() error, error) {
		return func() error { return restoreErr }, nil
	})
	if err := src.Setup(); err != nil {
		t.Fatal(err)
	}
	if !errors.Is(src.Restore(), restoreErr) {
		t.Fatal("Restore() did not propagate error")
	}
}

func TestTerminalDrainIgnoresAllUnlistedKeys(t *testing.T) {
	data := "ABCDEFGHIJKLMNOPQRSTUVWXYZ!@#$%^&*()[]{}"
	got, err := NewTerminal(bytes.NewBufferString(data)).Drain(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("Drain() = %#v, want no events", got)
	}
}

func TestTerminalRestoreIsIdempotent(t *testing.T) {
	restores := 0
	src := NewTerminal(bytes.NewBuffer(nil), func() (func() error, error) {
		return func() error { restores++; return nil }, nil
	})
	if err := src.Setup(); err != nil {
		t.Fatal(err)
	}
	if err := src.Restore(); err != nil {
		t.Fatal(err)
	}
	if err := src.Restore(); err != nil || restores != 1 {
		t.Fatalf("second Restore() error=%v calls=%d, want one restore", err, restores)
	}
}

func TestTerminalSetupFailureDoesNotRequireRestore(t *testing.T) {
	setupErr := errors.New("raw mode")
	src := NewTerminal(bytes.NewBuffer(nil), func() (func() error, error) {
		return func() error { t.Fatal("unexpected restore"); return nil }, setupErr
	})
	if !errors.Is(src.Setup(), setupErr) {
		t.Fatal("Setup() did not return setup failure")
	}
	if err := src.Shutdown(); err != nil {
		t.Fatalf("Shutdown() after failed setup = %v", err)
	}
}
