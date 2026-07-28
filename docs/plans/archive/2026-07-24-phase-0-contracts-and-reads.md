# Phase 0: Contracts And Read CLI Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Produce a compiled, side-effect-free `hq` CLI implementing `version`, `health`, `context`, `get`, `list`, and `search` against disposable HQ fixtures.

**Architecture:** The CLI dispatcher depends on versioned contract types and read services. Read services use a safe path resolver and collection adapters; they do not depend on runtime write state. Platform path tests prove containment before later transaction work begins.

**Tech Stack:** Go standard library, `encoding/json`, `crypto/sha256`, `os`, `path/filepath`, `os/exec`, Go build constraints for platform probes, Go tests.

## Global Constraints

- Follow `docs/architecture.md`, `docs/requirements.md`, and `docs/contracts/cli.md` exactly.
- Read commands must leave a byte-identical filesystem snapshot.
- No persistent index, daemon, runtime-state initialization, or write command is included in this phase.
- Phase 0 requires native `amd64` evidence on Windows, Linux, and macOS; native `arm64` evidence is a release-advertisement gate in Phase 3.
- Commit steps require explicit commit authorization.

---

### Task 0.1: Module And Public Contract Types

**Files:**
- Create: `go.mod`
- Create: `cmd/hq/main.go`
- Create: `internal/contract/result.go`
- Create: `internal/contract/errors.go`
- Create: `internal/contract/result_test.go`
- Create: `internal/assets/assets.go`
- Create: `internal/assets/assets_test.go`
- Create: `schemas/result-v1.json`
- Modify: `implementation/STATUS.md`

**Interfaces:**
- Produces: `contract.Result`, `contract.ErrorDetail`, `contract.MutationState`, `contract.ExitCode(error) int`, and `contract.WriteJSON(io.Writer, Result) error`.

**Rollback:** Remove only Task 0.1 files and its status evidence before any later task depends on them.

- [ ] **Step 1: Record Task 0.1 as `in-progress` in `implementation/STATUS.md`.**
- [ ] **Step 2: Initialize the module as `github.com/ryannmicua/hq-cli` and create a failing result-envelope serialization test.**

```go
func TestResultJSONContract(t *testing.T) {
    got, err := json.Marshal(contract.Success("health", map[string]any{"status": "ok"}))
    if err != nil { t.Fatal(err) }
    for _, field := range []string{`"schemaVersion":"1.0"`, `"command":"health"`, `"success":true`, `"mutation":"noMutation"`} {
        if !bytes.Contains(got, []byte(field)) { t.Fatalf("missing %s in %s", field, got) }
    }
}
```

- [ ] **Step 3: Run `go test ./internal/contract` and verify it fails because the contract package is absent.**
- [ ] **Step 4: Implement the minimal result and error types with stable exit mappings and shapes from `docs/contracts/cli.md` and `docs/contracts/results.md`.**
- [ ] **Step 5: Add `schemas/result-v1.json`, embed it with `//go:embed`, and test that embedded bytes equal the source asset.**
- [ ] **Step 6: Run `gofmt -w cmd internal`, `go test ./internal/contract`, and `go vet ./...`; expect exit 0.**
- [ ] **Step 7: Record files, commands, and observed results; mark Task 0.1 `verified`.**
- [ ] **Step 8: If commits are authorized, commit only Task 0.1 with `feat: establish CLI result contract`.**

### Task 0.2: Configuration And Safe Path Resolution

**Files:**
- Create: `internal/config/config.go`
- Create: `internal/config/config_test.go`
- Create: `internal/hq/path.go`
- Create: `internal/hq/path_test.go`
- Create: `internal/hq/path_windows_test.go`
- Create: `internal/hq/path_posix_test.go`

**Interfaces:**
- Produces: `config.Load(rootFlag string, lookupEnv func(string) string) (Config, error)`.
- Produces: `hq.NewResolver(root string) (*Resolver, error)` and `(*Resolver).Resolve(relative string) (string, error)`.

**Rollback:** Remove Task 0.2 files; Task 0.1 remains a working contract-only CLI foundation.

- [ ] **Step 1: Write table tests for flag-over-environment precedence, missing root, traversal, absolute paths, symlink escape, and valid contained paths.**
- [ ] **Step 2: Run `go test ./internal/config ./internal/hq`; expect compile failures for missing types.**
- [ ] **Step 3: Implement canonical root evaluation and containment using `filepath.Abs`, `filepath.Clean`, `filepath.EvalSymlinks`, and component-aware relative checks.**
- [ ] **Step 4: Add Windows tests for drive changes, UNC input, reserved names, reparse-point escape, and alternate data-stream syntax; add POSIX tests for absolute and symlink escape.**
- [ ] **Step 5: Run `go test ./internal/config ./internal/hq`; expect all tests to pass on the current platform.**
- [ ] **Step 6: Record the current OS evidence and leave other platform rows unverified until native CI runs.**
- [ ] **Step 7: If commits are authorized, commit with `feat: enforce safe HQ path containment`.**

### Task 0.3: HQ Fixtures And Collection Adapters

**Files:**
- Create: `testdata/hq/CURRENT-WORK.md`
- Create: `testdata/hq/SESSION-LOG.md`
- Create: `testdata/hq/AGENTS.md`
- Create: `testdata/hq/safety-boundaries.md`
- Create: `testdata/hq/projects/example/README.md`
- Create: `testdata/hq/projects/example/STATE.md`
- Create: representative fixtures under `testdata/hq/decisions/`, `people/`, `references/`, and `work-types/`
- Create: `internal/hq/records.go`
- Create: `internal/hq/records_test.go`

**Interfaces:**
- Produces: `hq.Record{Collection, ID, Path, Content, SHA256, GitCommit, Metadata}`.
- Produces: `hq.Get(ctx, resolver, selector)`, `hq.List(ctx, resolver, query)`, and collection parsers.

**Rollback:** Remove Task 0.3 adapters and fixture files; retain safe path and contract foundations.

- [ ] **Step 1: Create minimal representative fixtures matching `docs/contracts/hq-markdown.md`, with active and warm work, one project, one decision, one person, one reference, and one work-type record.**
- [ ] **Step 2: Write failing tests for logical lookup, deterministic listing, malformed record tolerance, SHA-256, and absent Git metadata.**
- [ ] **Step 3: Run `go test ./internal/hq`; verify failures identify missing adapters.**
- [ ] **Step 4: Implement explicit collection allowlists and deterministic projections; do not scan `.git/` or `.hq-interface/`.**
- [ ] **Step 5: Run `go test ./internal/hq`; expect exit 0.**
- [ ] **Step 6: If commits are authorized, commit with `feat: read structured HQ records`.**

### Task 0.4: Read Services And CLI Commands

**Files:**
- Create: `internal/read/service.go`
- Create: `internal/read/context.go`
- Create: `internal/read/search.go`
- Create: `internal/read/service_test.go`
- Create: `internal/app/app.go`
- Create: `internal/app/read_commands.go`
- Create: `internal/app/app_test.go`
- Modify: `cmd/hq/main.go`

**Interfaces:**
- Produces: `read.Service.Context`, `Get`, `List`, and `Search`.
- Produces: `app.Run(ctx context.Context, args []string, stdout, stderr io.Writer) int`.

**Rollback:** Remove Task 0.4 service and dispatcher changes; retain tested contracts, paths, and adapters.

- [ ] **Step 1: Write failing command tests for `version`, phased `health`, `context`, `get`, `list`, and `search`, including every `data`, `error`, and warning shape in `docs/contracts/results.md`.**
- [ ] **Step 2: Add a snapshot helper that hashes every fixture file before and after each read command.**
- [ ] **Step 3: Run `go test ./internal/read ./internal/app`; expect compile failures.**
- [ ] **Step 4: Implement the smallest dispatcher and services needed for the documented syntax. Use deterministic path and line ordering for search.**
- [ ] **Step 5: Implement `context` selection: explicit project first, otherwise most-recent active item, and error on ambiguity or absence.**
- [ ] **Step 6: Run `go test ./...`, `go test -race ./...`, `go vet ./...`, and `gofmt -l .`; expect tests and vet to pass and `gofmt -l` to print nothing.**
- [ ] **Step 7: Build with `go build -o dist/hq ./cmd/hq` and smoke-test `dist/hq --root testdata/hq context`; expect one successful JSON object.**
- [ ] **Step 8: Record local evidence; leave Phase 0 in progress until Task 0.5 native jobs pass.**
- [ ] **Step 9: If commits are authorized, commit with `feat: deliver read-only HQ CLI`.**

### Task 0.5: Bootstrap Native CI

**Files:**
- Create: `.github/workflows/ci.yml`
- Create: `docs/operations/ci.md`
- Modify: `implementation/STATUS.md`

**Interfaces:**
- Produces: native Windows, Linux, and macOS jobs running formatting, vet, unit, race where supported, path, read, and filesystem-snapshot checks.

**Rollback:** Remove the Phase 0 CI workflow and CI document; local Phase 0 verification remains available.

- [ ] **Step 1: Add native `windows-latest`, `ubuntu-latest`, and `macos-latest` jobs using the repository-pinned Go version from `go.mod`.**
- [ ] **Step 2: Each job runs `gofmt -l .`, `go vet ./...`, `go test ./...`, and `go test -race ./...` where the runner supports the race detector.**
- [ ] **Step 3: Run `go run github.com/rhysd/actionlint/cmd/actionlint@v1.7.7`; expect no diagnostics. Record that first use requires network access and cache the module for offline reruns.**
- [ ] **Step 4: With explicit approval to run remote CI, execute the workflow and record job URLs and results; otherwise run equivalent commands manually on all three operating systems and record machine evidence.**
- [ ] **Step 5: Mark Phase 0 verified only after all three native operating-system checks pass.**
- [ ] **Step 6: If commits are authorized, commit with `ci: bootstrap cross-platform read checks`.**

## Phase Acceptance

- [ ] `go test ./...`, race tests, vet, and formatting checks pass.
- [ ] All read commands produce one JSON result and documented exit codes.
- [ ] Fixture snapshots prove no read command creates or modifies files.
- [ ] Native Windows, Linux, and macOS jobs pass path and read tests.
- [ ] `implementation/STATUS.md` contains commands and evidence for Tasks 0.1-0.4.
