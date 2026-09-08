# Terminal Space Invaders (Go)

This repository tracks the development of a terminal-based Space Invaders game written in Go.

## Toolchain requirement

- Go 1.23.2 is the required local toolchain version for this repository.

## Core architecture (minimal skeleton)

For the detailed architecture rationale, read `docs/decisions/core-architecture.md`. The contributor-facing package boundaries are:

- `internal/game`: owns deterministic game state and pure state transitions (`NewState`, `Step`, `StepMany`) plus snapshot data for downstream consumers.
- `internal/input`: owns terminal-agnostic logical input commands/events and the `Source` contract used to drain ordered events.
- `internal/loop`: owns fixed-step orchestration (`Run`), tick source integration, input collection per tick, and cancellation/quit handling.
- `cmd/invaders`: owns executable startup wiring (context/signals, initial state, ticker, input source, and loop invocation).

This skeleton currently validates loop/state/input wiring and deterministic stepping. It does not yet claim gameplay systems, rendering behavior, or terminal UX features.

## Run the skeleton

- Supported command: `go run ./cmd/invaders`
- The current entrypoint runs the loop with a fixed tick source and exits on process cancellation or quit input from the wired source.

## Local validation workflow

Use plain Go and shell commands from the repository root (no task runner):

```sh
files=$(git ls-files -- '*.go')
if test -n "$files"; then
  unformatted=$(printf '%s\n' "$files" | xargs gofmt -l)
  test -z "$unformatted"
fi
go vet ./...
go build ./...
go test ./...
```

- The formatting check fails if any tracked Go file is listed by `gofmt -l`.
- `go vet ./...`, `go build ./...`, and `go test ./...` are the repository-approved package checks.

## Pull request CI

- Pull-request CI is configured for pull requests targeting `develop`.
- The current CI validation checks Go module metadata safety rather than package build, vet, or test execution.

## Project goals

- Build a playable Space Invaders experience in the terminal
- Keep gameplay logic modular and testable
- Provide a contributor-friendly workflow for iterative development
