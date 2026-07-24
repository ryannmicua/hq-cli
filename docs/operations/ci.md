# CI Operations

## Workflow

The `.github/workflows/ci.yml` workflow runs on push and pull request to `main`.

### Jobs

- **Format**: Linux only. Checks `gofmt -l .` is empty and `go vet ./...` passes.
- **Test**: Matrix across `windows-latest`, `ubuntu-latest`, and `macos-latest`. Runs:
  1. `go test ./...` (all platforms)
  2. `go test -race ./...` (Linux and macOS; needs C compiler on Windows)
  3. `go build -o dist/hq` (platform-appropriate binary name)
  4. Smoke tests: `version`, `health`, `context`, `list` against fixtures

### Runner Requirements

- Go version from `go.mod` (currently Go 1.22).
- Git pre-installed (GitHub Actions runners).

### Evidence Expectations

Each CI run produces:
- Format job: `gofmt` empty diff + `go vet` clean.
- Test job per platform: all unit tests pass, race tests pass (Linux/macOS), binary builds, smoke tests return `success: true` JSON for each command.

### Manual Platform Verification

Without CI approval, run equivalent commands locally on each target:

```bash
gofmt -l .          # must be empty
go vet ./...        # must be clean
go test ./...       # all pass
go test -race ./... # Windows requires C compiler (MinGW)
go build -o dist/hq ./cmd/hq
```

## Phase 0 Acceptance

Before declaring Phase 0 acceptance, verify:
- [ ] `gofmt -l .` prints nothing on all three platforms.
- [ ] `go vet ./...` produces no diagnostics on all three platforms.
- [ ] `go test ./...` passes on Windows, Linux, and macOS.
- [ ] `go test -race ./...` passes on Linux and macOS.
- [ ] All six commands (`version`, `health`, `context`, `get`, `list`, `search`) each produce one valid JSON result.
- [ ] Snapshot tests prove no read command modifies the filesystem.
- [ ] Evidence recorded in `implementation/STATUS.md`.
