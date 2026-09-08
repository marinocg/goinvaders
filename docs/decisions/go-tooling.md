# Go Tooling Decision

## Decisions

- **Target Go version:** Go 1.23. Later changes should use this version (or a
  compatible patch release only when this decision is deliberately revised),
  so local and CI validation have the same language and standard-library
  baseline.
- **Validation runner:** Use plain Go commands and standard POSIX shell tools,
  not a task runner. This keeps the validation surface small and avoids an
  additional tool, lockfile, or task-runner installation requirement.

## Required Commands

These commands are deterministic and non-interactive. They must be run from
the repository root with Go 1.23 available on `PATH`.

### Formatting and static checks

```sh
files=$(git ls-files -- '*.go')
if test -n "$files"; then
  unformatted=$(printf '%s\n' "$files" | xargs gofmt -l)
  test -z "$unformatted"
fi
go vet ./...
```

The formatting portion exits nonzero when any tracked Go file is listed by
`gofmt -l`. It exits successfully when there are no tracked Go files. The
`go vet` command checks all repository packages once Go module and package
scaffolding exists.

### Build validation

```sh
go build ./...
```

### Tests

```sh
go test ./...
```

`go test` uses the repository's package tests and does not enable network,
deployment, browser, credential, or destructive-operation behavior by itself.

## Current Repository Prerequisites

The build, vet, and test commands are immediately runnable only after the
repository has a `go.mod` file and at least one valid Go package. This issue
does not create either, and the current repository state may therefore make
those commands fail with a missing-module or no-package error. That limitation
is intentional rather than a reason to invent scaffolding here.

The formatting check is immediately runnable in the current state and does
not require a Go module. Later quality, CI, testing, and contributor-workflow
issues must use these commands rather than adding a task-runner wrapper.
