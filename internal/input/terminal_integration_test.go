package input

import (
	"bytes"
	"testing"
)

func TestTerminalLifecycleRestoresFakeTerminalExactlyOnce(t *testing.T) {
	restores := 0
	source := NewTerminal(bytes.NewBufferString("\nq"), func() (func() error, error) {
		return func() error { restores++; return nil }, nil
	})
	if err := source.Setup(); err != nil {
		t.Fatal(err)
	}
	if err := source.Shutdown(); err != nil {
		t.Fatal(err)
	}
	if err := source.Shutdown(); err != nil || restores != 1 {
		t.Fatalf("shutdown error=%v restores=%d, want one restore", err, restores)
	}
}
