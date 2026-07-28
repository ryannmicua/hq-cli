---
title: HQ CLI Phase 0 — Contracts and Read CLI - Plan
type: feat
date: 2026-07-24
artifact_contract: ce-unified-plan/v1
artifact_readiness: implementation-ready
product_contract_source: legacy-requirements
execution: code
origin: docs/plans/2026-07-24-phase-0-contracts-and-reads.md
---

# HQ CLI Phase 0 — Contracts and Read CLI - Plan

## Goal Capsule

- **Objective:** Produce a compiled, side-effect-free `hq` CLI implementing `version`, `health`, `context`, `get`, `list`, and `search` against disposable HQ fixtures.
- **Authority hierarchy:**
  - `docs/architecture.md` is canonical for component boundaries and runtime layout.
  - `docs/requirements.md` is canonical for product behavior and use-case acceptance.
  - `docs/contracts/cli.md` and `docs/contracts/results.md` are canonical for CLI syntax and JSON shapes.
  - `docs/contracts/hq-markdown.md` is canonical for collection mapping and record projections.
  - `implementation/STATUS.md` is authoritative for task evidence.
  - `docs/traceability.md` declares the requirement-to-task-to-evidence mapping.
- **Stop conditions:** All five tasks verified with `go test ./...`, `go test -race ./...`, `go vet ./...`, `gofmt -l .` clean; every read command leaves a byte-identical filesystem snapshot; native CI passes on Windows, Linux, and macOS.
- **Execution profile:** code — Go standard library only; phase-scoped implementation units with file-level rollback; commit authorization required per task.
- **Tail ownership:** Phase 0.5 (CI bootstrap) gates the phase-to-phase handoff; Phase 1 may not begin until Phase 0 acceptance is verified on all three platforms.

---

## Product Contract

### Summary

Phase 0 delivers a read-only HQ CLI with six commands. The binary resolves an HQ root from `--root` or `HQ_ROOT`, canonicalizes paths with containment checks, reads structured records from allowlisted markdown collections, and returns versioned JSON results. No daemon, persistent index, runtime state initialization, or write command exists in this phase. Every read command must leave the filesystem snapshot byte-identical before and after.

### Requirements

**R1.** Every command emits exactly one JSON result object to stdout with `schemaVersion`, `command`, `success`, `timestamp`, `data`, `warnings`, `error`, and `mutation` fields. Diagnostics go to stderr. Governs FR-01, FR-02.

**R2.** The `version` command returns semantic version, source commit, build time, Go version, OS, and arch without requiring an HQ root. Governs UC-12.

**R3.** The `health` command checks executable presence, embedded contract assets, HQ root readability, Git availability, and read-path containment. It never creates runtime state. Phase 0 implements binary, root, Git, contract assets, and read-path checks only. Governs UC-12.

**R4.** The `context` command returns current work, selected project STATE.md and README.md, the newest session entries (default 20), and operating-rule paths. Explicit `--project` selects a project; otherwise the most recent active item is selected. Ambiguous or missing selection returns an error. It does not load unrelated projects. Governs FR-15.

**R5.** The `get` command retrieves one record by collection and ID or by allowlisted relative path. Returns `path`, `content`, `sha256`, `gitCommit` (null when unavailable), and collection metadata. Governs UC-02.

**R6.** The `list` command enumerates records from an allowed collection (`current-work`, `projects`, `decisions`, `work-types`, `people`, `references`) with optional filters. Returns deterministic projections. Governs UC-03.

**R7.** The `search` command performs literal UTF-8 substring matching (case-insensitive by default, case-sensitive with `--case-sensitive`) within allowlisted HQ paths. Results sort by normalized path, line, column. Does not follow links outside HQ or inspect `.git/` or `.hq-interface/`. Governs UC-04.

**R8.** Read commands never create or modify HQ files, `.hq-interface/` files, or any filesystem artifact. This is verified by snapshot hashing before and after every read command. Governs FR-03.

**R9.** Canonical path checks reject traversal (`..`), absolute paths targeting outside HQ, symlink escape, and platform path hazards. Collection lookups scan only allowlisted directories. `.git/`, `.hq-interface/`, hidden paths, and templates are never readable through `--path` or search. Governs FR-04, FR-05.

**R10.** Git metadata is read-only. The CLI never stages, commits, resets, checks out, cleans, pushes, or creates refs. Governs FR-14.

**R11.** Versioned schemas and default policy are embedded in the executable via `//go:embed` and verified to match source assets at build time. Governs FR-17.

### Scope Boundaries

**Deferred for later:**
- Write commands (`submit`, `apply`, `status`, `changes`, `recover`) — Phase 1 and Phase 2.
- Runtime state initialization (`.hq-interface/` directory structure) — Phase 1.
- Filesystem adapter (locking, flush, atomic replacement) — Phase 2.
- Cross-platform native `arm64` CI evidence — Phase 3 release gate (Phase 0 requires `amd64` evidence only).
- Global `hq-io` skill — Phase 3.

**Outside this project's identity:**
- Remote transport or network authentication.
- Semantic or embedding search.
- A daemon, MCP process, or subscription service.
- Automatic Git commits or publication.
- Hard authorization between processes sharing one OS account.

### Acceptance Examples

- AE-R3: `hq --root testdata/hq health` returns `overall: "pass"` with checks for `hq-root`, `git`, and `contract-assets`. Missing `go.mod` file path still reports `pass` if the binary itself is intact — embed checks test the binary, not the source tree.
- AE-R4: `hq --root testdata/hq context` with one active item in CURRENT-WORK.md returns that item's project. With zero active items, returns an error with `HQ_NOT_FOUND`.
- AE-R5: `hq --root testdata/hq get --collection projects --id example` returns one Record. `hq --root testdata/hq get --path projects/example/README.md` returns the same shape. `get --path .git/HEAD` returns `HQ_PATH_DENIED`.
- AE-R6: `hq --root testdata/hq list --collection projects` returns all project slugs. `list --collection current-work --filter section=active` returns only active entries.
- AE-R7: `hq --root testdata/hq search --query "blocker"` returns matches sorted by path then line. `search --query "." --collection decisions` returns only matches under `decisions/`. `search --query "secret" --case-sensitive` matches only exact case.

### Sources

- Architecture: `docs/architecture.md`
- Requirements: `docs/requirements.md`
- CLI contract: `docs/contracts/cli.md`
- Result shapes: `docs/contracts/results.md`
- Markdown contract: `docs/contracts/hq-markdown.md`
- Test strategy: `docs/testing/strategy.md`
- Traceability: `docs/traceability.md`
- Master plan: `docs/plans/2026-07-24-hq-cli-master.md`
- Phase 0 plan: `docs/plans/2026-07-24-phase-0-contracts-and-reads.md`

---

## Planning Contract

### Key Technical Decisions

**KTD1. Module path `github.com/ryannmicua/hq-cli`.** (session-settled: user-approved — chosen as the canonical Go module path matching the owner's GitHub namespace)
The module, all `go.mod` directives, and import paths use this path. No vanity URL or alternative hosting.

**KTD2. Go standard library only in Phase 0.** No external dependencies for Phase 0. `encoding/json`, `crypto/sha256`, `os`, `path/filepath`, and `os/exec` cover all needed capabilities. External packages (e.g., `actionlint` for CI linting) are dev-time only and not vendored. Governs R1-R11.

**KTD3. Embedded schemas via `//go:embed`.** Versioned JSON schemas and default policy live in `schemas/` and `config/` as source-reviewed files. `internal/assets/` embeds them with `//go:embed` so the binary is self-contained. A test proves embedded bytes equal the source file bytes. Governs R11.

**KTD4. Safe path resolution with component-aware containment.** `internal/hq/path.go` resolves the HQ root via `filepath.Abs` → `filepath.EvalSymlinks` → `filepath.Clean`, then checks that every resolved relative target path (after its own eval) starts with the resolved root + separator. Platform-specific tests cover Windows drive changes, UNC input, reserved names, reparse-point escape, and alternate data-stream syntax on Windows; absolute and symlink escape on POSIX. Governs R9.

**KTD5. Explicit collection allowlists, not filesystem walking.** The HQ adapter in `internal/hq/records.go` maps collection names to known directory structures per `docs/contracts/hq-markdown.md`. It never walks `.git/`, `.hq-interface/`, hidden directories, or templates. Unknown collection names return `HQ_INVALID_ARGUMENT`. Governs R9.

**KTD6. Deterministic record ordering.** List and search results sort by normalized path (forward slash separators), then by line/column. No timestamp-based or filesystem-order dependency. This makes test assertions stable across platforms. Governs R6, R7.

**KTD7. Snapshot-based read-side-effect verification.** Every read command test hashes all fixture files before and after execution using SHA-256. Any difference fails the test. This is the Phase 0 contract for FR-03 enforcement; later phases add write-side mutation tracking. Governs R8.

**KTD8. Command dispatch via `internal/app/app.go`.** `app.Run(ctx, args, stdout, stderr) int` parses `--root` and `--output json`, resolves the command, calls the appropriate read service, serializes the `contract.Result` to stdout, and returns the exit code. The `cmd/hq/main.go` entry point is a thin assembly: parse os.Args, call app.Run, call os.Exit. Governs R1.

### Assumptions

1. The executing machine has Go installed at the version pinned in `go.mod` (to be set; minimum Go 1.21 for `//go:embed` stability).
2. HQ fixtures under `testdata/hq/` are disposable and never contain real secrets or credentials.
3. Git metadata lookup via `git log -1 --format=%H` may fail gracefully (no Git installed, not a repo); `gitCommit` is `null` in that case.
4. Native CI runners (`windows-latest`, `ubuntu-latest`, `macos-latest`) have Git and Go pre-installed.
5. `arm64` native CI evidence is deferred to Phase 3; Phase 0 requires `amd64` evidence only.
6. The Phase 0 plan at `docs/plans/2026-07-24-phase-0-contracts-and-reads.md` is the canonical task-level source; this plan enriches it to the unified-plan contract without changing scope.

### Implementation Constraints

- Follow `docs/architecture.md` source layout exactly.
- All file paths in the plan and implementation are repo-relative.
- Each task must record state, files, verification commands, results, and rollback in `implementation/STATUS.md`.
- Commits require explicit authorization per task; without authorization, retain the verified working tree.
- `gofmt -w cmd internal` must produce no changes before verification is claimed.

---

## High-Level Technical Design

```
┌─────────────────────────────────────────────┐
│                 cmd/hq/main.go               │
│  os.Args → app.Run(ctx, args, out, err)     │
└──────────────────┬──────────────────────────┘
                   │
┌──────────────────▼──────────────────────────┐
│              internal/app/app.go             │
│  Parse --root, --output, command            │
│  Dispatch to read service                    │
│  Serialize contract.Result → stdout         │
│  Return exit code                            │
└──────┬──────────────┬──────────────┬────────┘
       │              │              │
┌──────▼──────┐ ┌─────▼──────┐ ┌────▼─────────┐
│  read/      │ │   hq/      │ │  contract/   │
│  service.go │ │  path.go   │ │  result.go   │
│  context.go │ │  records.go│ │  errors.go   │
│  search.go  │ │            │ │              │
└──────┬──────┘ └─────┬──────┘ └────┬─────────┘
       │              │              │
┌──────▼──────────────▼──────────────▼────────┐
│              internal/config/                │
│  config.go: HQ_ROOT, policy settings        │
└─────────────────────────────────────────────┘
```

Read flow:

1. `app.Run` parses `--root` (or reads `HQ_ROOT`) → `config.Load`.
2. Config creates an `hq.Resolver` that canonicalizes the root.
3. Read service resolves the logical target through the resolver → containment check.
4. `hq` adapter reads the collection or allowlisted path → returns `hq.Record`.
5. Read service projects the record into the command's `data` shape.
6. `contract.WriteJSON` serializes the `contract.Result` to stdout.

Snapshot verification wraps steps 3-5: hash all fixture files before, execute command, hash all fixture files after, assert equality.

---

## Implementation Units

### Unit Index

| U-ID | Title | Files | Depends On |
|------|-------|-------|------------|
| U1 | Module and Public Contract Types | `go.mod`, `cmd/hq/main.go`, `internal/contract/`, `internal/assets/`, `schemas/result-v1.json` | — |
| U2 | Configuration and Safe Path Resolution | `internal/config/`, `internal/hq/path*.go` | U1 |
| U3 | HQ Fixtures and Collection Adapters | `testdata/hq/`, `internal/hq/records.go` | U2 |
| U4 | Read Services and CLI Commands | `internal/read/`, `internal/app/`, `cmd/hq/main.go` | U3 |
| U5 | Bootstrap Native CI | `.github/workflows/ci.yml`, `docs/operations/ci.md` | U4 |

---

### U1. Module and Public Contract Types

- **Goal:** Initialize the Go module, define the versioned JSON result contract, embedded asset types, and a minimal `main.go` entry point.
- **Requirements:** R1, R2, R11
- **Files:**
  - Create: `go.mod`
  - Create: `cmd/hq/main.go`
  - Create: `internal/contract/result.go`
  - Create: `internal/contract/errors.go`
  - Create: `internal/contract/result_test.go`
  - Create: `internal/assets/assets.go`
  - Create: `internal/assets/assets_test.go`
  - Create: `schemas/result-v1.json`
  - Modify: `implementation/STATUS.md`
- **Dependencies:** None (first unit).
- **Approach:**
  1. Initialize `go.mod` with module path `github.com/ryannmicua/hq-cli` and Go version ≥1.21.
  2. Define `contract.Result`, `contract.ErrorDetail`, and `contract.MutationState` matching `docs/contracts/results.md`. Include stable exit-code mapping: `contract.ExitCode(error) int`.
  3. Implement `contract.WriteJSON(io.Writer, Result) error` for stdout serialization.
  4. Create `schemas/result-v1.json` with the canonical result envelope schema. Embed it in `internal/assets/assets.go` via `//go:embed`. Test that `assets.ResultSchemaV1` bytes equal the source file bytes.
  5. Write `cmd/hq/main.go` as a placeholder that prints a stub result and exits 0.
  6. Write `internal/contract/result_test.go` with a table-driven serialization test verifying every required field (`schemaVersion`, `command`, `success`, `mutation`, `data`, `error`, `warnings`, `timestamp`) round-trips through JSON.
  7. Run TDD cycle: write failing test → implement → verify.
- **Test Scenarios:**
  - `contract.Success("health", data)` produces valid JSON with `success: true` and `mutation: "noMutation"`.
  - `contract.Error("HQ_NOT_FOUND", ...)` produces valid JSON with `success: false` and the correct error code.
  - `contract.ExitCode(err)` returns correct integer for each error code in `docs/contracts/cli.md`.
  - Embedded `result-v1.json` bytes match the source file on disk.
  - JSON output contains no Go-internal fields, stack traces, or unexpected keys.
- **Verification:**
  - `go test ./internal/contract ./internal/assets` — all pass.
  - `go vet ./...` — no diagnostics.
  - `gofmt -l cmd internal` — prints nothing.
- **Rollback:** Remove U1 files and status evidence; restore empty repository scaffold.

---

### U2. Configuration and Safe Path Resolution

- **Goal:** Implement HQ root resolution from `--root`/`HQ_ROOT` and path containment enforcement that rejects traversal, symlink escape, and platform hazards.
- **Requirements:** R9, R10
- **Files:**
  - Create: `internal/config/config.go`
  - Create: `internal/config/config_test.go`
  - Create: `internal/hq/path.go`
  - Create: `internal/hq/path_test.go`
  - Create: `internal/hq/path_windows_test.go`
  - Create: `internal/hq/path_posix_test.go`
- **Dependencies:** U1 (contract types needed for error returns; `go.mod` must exist).
- **Approach:**
  1. `config.Load(rootFlag string, lookupEnv func(string) string) (Config, error)`: resolves root from flag, then `HQ_ROOT`, then current directory; validates root exists and is a directory.
  2. `hq.NewResolver(root string) (*Resolver, error)`: canonicalizes root via `filepath.Abs` → `filepath.EvalSymlinks` → `filepath.Clean`. Returns error if root is inaccessible or resolves to a non-directory.
  3. `(*Resolver).Resolve(relative string) (string, error)`: resolves the relative target against the canonical root, evaluates symlinks, and checks containment via `strings.HasPrefix(resolved, root+separator)` plus `resolved == root` for the root itself. Rejects `..` segments, absolute paths, empty paths, and paths that escape containment.
  4. Platform-specific tests: Windows tests cover drive letter changes (`C:\` vs `D:\`), UNC paths (`\\server\share`), reserved names (`CON`, `NUL`), reparse-point/junction escape, and ADS syntax (`file:stream`). POSIX tests cover absolute path injection and symlink escape.
- **Test Scenarios:**
  - `config.Load` returns root from `--root` flag when present, overrides `HQ_ROOT`.
  - `config.Load` returns error for missing root directory.
  - Resolver accepts a path within root; rejects `../../etc/passwd`.
  - Resolver rejects absolute paths inside root that symlink outside.
  - Resolver accepts the root itself as a valid target.
  - Windows: rejects drive-relative paths that resolve to a different drive.
  - Windows: rejects NTFS junction pointing outside root.
  - POSIX: rejects symlink `/tmp/hq-link → /etc`.
- **Verification:**
  - `go test ./internal/config ./internal/hq` — all pass on current platform.
  - Platform-conditional tests build correctly (non-applicable tests skip, not fail).
  - Record current OS evidence; leave other platform rows unverified until CI (U5).
- **Rollback:** Remove U2 files; U1 remains a working contract-only foundation.

---

### U3. HQ Fixtures and Collection Adapters

- **Goal:** Create representative disposable HQ fixtures and implement record readers that project markdown collections into structured `hq.Record` values.
- **Requirements:** R5, R6, R9
- **Files:**
  - Create: `testdata/hq/CURRENT-WORK.md` (one active, one warm entry)
  - Create: `testdata/hq/SESSION-LOG.md` (two entries)
  - Create: `testdata/hq/AGENTS.md`
  - Create: `testdata/hq/safety-boundaries.md`
  - Create: `testdata/hq/projects/example/README.md`
  - Create: `testdata/hq/projects/example/STATE.md` (canonical sections)
  - Create: `testdata/hq/decisions/2026-07-23-use-go.md`
  - Create: `testdata/hq/people/ryann.md`
  - Create: `testdata/hq/references/api-notes.md`
  - Create: `testdata/hq/work-types/feature/example-work.md`
  - Create: `internal/hq/records.go`
  - Create: `internal/hq/records_test.go`
- **Dependencies:** U2 (path resolution must be working for containment checks).
- **Approach:**
  1. Create fixtures matching `docs/contracts/hq-markdown.md` exactly: canonical section order in STATE.md, correct entry format in SESSION-LOG.md, correct Active/Warm structure in CURRENT-WORK.md.
  2. Define `hq.Record{Collection, ID, Path, Content, SHA256, GitCommit, Metadata}`.
  3. Implement collection-parsing functions: `parseCurrentWork`, `parseProject`, `parseDecision`, `parseWorkType`, `parsePerson`, `parseReference`. Each walks its allowlisted directory per the collection mapping table.
  4. Implement `hq.Get(ctx, resolver, selector)` for single-record lookup by collection+ID or allowlisted path.
  5. Implement `hq.List(ctx, resolver, query)` with collection filter, deterministic path ordering, and optional `filter`/`limit`.
  6. SHA-256 computation on raw file bytes. Git commit via `git log -1 --format=%H -- <path>` with graceful null on failure.
  7. Collection allowlist: unknown collections return error. `.git/`, `.hq-interface/`, hidden files, and templates are never enumerated.
- **Test Scenarios:**
  - Get project `example` returns correct path, content, and metadata.
  - Get non-existent ID returns `HQ_NOT_FOUND`.
  - List `projects` returns all project slugs with deterministic ordering.
  - List `current-work` with `section=active` returns only active entries.
  - SHA-256 is deterministic for identical content.
  - `gitCommit` is `null` for fixtures in a non-Git `testdata/` directory.
  - Malformed markdown file (missing expected heading) is tolerated and returned with best-effort parsing; does not crash.
  - `.git/` and `.hq-interface/` content never appears in list results.
- **Verification:**
  - `go test ./internal/hq` — all pass.
  - Manual inspection: `go run ./cmd/hq --root testdata/hq list --collection projects` returns expected JSON.
- **Rollback:** Remove U3 adapter and fixture files; retain safe path and contract foundations (U1-U2).

---

### U4. Read Services and CLI Commands

- **Goal:** Wire the dispatcher, read services, and snapshot verification into a complete read-only CLI that passes end-to-end tests for all six commands.
- **Requirements:** R1, R2, R3, R4, R5, R6, R7, R8
- **Files:**
  - Create: `internal/read/service.go`
  - Create: `internal/read/context.go`
  - Create: `internal/read/search.go`
  - Create: `internal/read/service_test.go`
  - Create: `internal/app/app.go`
  - Create: `internal/app/read_commands.go`
  - Create: `internal/app/app_test.go`
  - Modify: `cmd/hq/main.go`
- **Dependencies:** U3 (record adapters must be complete; all fixtures must exist).
- **Approach:**
  1. Define `read.Service` struct holding `*hq.Resolver` and config. Expose methods: `Version()`, `Health()`, `Context(opts)`, `Get(selector)`, `List(query)`, `Search(query)`.
  2. Implement `version`: returns build-time constants from `-ldflags` (empty strings in dev; real values in release) plus `runtime.GOOS` and `runtime.GOARCH`.
  3. Implement `health`: checks binary → pass, root readability → pass/fail, embedded assets present → pass, Git available → pass/warn, read-path within root → pass. Returns `overall: "pass"` or `"fail"`.
  4. Implement `context`: reads CURRENT-WORK.md to find active item, selects project (explicit `--project` or most-recent active), reads that project's STATE.md and README.md, reads newest N session entries from SESSION-LOG.md, collects operating-rule paths (AGENTS.md, safety-boundaries.md). Returns error on ambiguity or absence.
  5. Implement `get`: delegates to `hq.Get` with collection+ID or path selector.
  6. Implement `list`: delegates to `hq.List` with collection and filter.
  7. Implement `search`: literal UTF-8 substring match across allowlisted files, case-insensitive by default, case-sensitive with `--case-sensitive`. Sorts by path, line, column. Returns complete matched line text.
  8. Implement `app.Run`: parse `--root` flag and `--output json` flag, resolve command from `args[0]`, dispatch to read service, serialize result, return exit code. `cmd/hq/main.go` calls `app.Run(ctx, os.Args[1:], os.Stdout, os.Stderr)` and `os.Exit(code)`.
  9. Snapshot helper: walks `testdata/hq/` recursively, SHA-256 hashes every file, stores map. Compares pre-command and post-command maps; any difference fails the test.
- **Test Scenarios:**
  - `version` returns valid JSON with all six fields (version, commit, buildTime, goVersion, os, arch).
  - `health` returns `overall: "pass"` with at least binary, root, and contract-assets checks.
  - `context` with one active item returns that project's state, readme, and session entries.
  - `context` with no active items returns error (`HQ_NOT_FOUND`).
  - `context --project example` selects the explicit project regardless of CURRENT-WORK.md.
  - `get --collection projects --id example` returns one record.
  - `get --collection projects --id none` returns `HQ_NOT_FOUND`.
  - `list --collection projects` returns all project slugs.
  - `list --collection current-work --filter section=active` returns only active items.
  - `search --query "blocker"` returns matches in path+line order.
  - `search --query "BLOCKER"` matches case-insensitively.
  - `search --query "BLOCKER" --case-sensitive` returns no matches if source is lowercase.
  - `search --query "secret"` with no match returns `count: 0, matches: []`.
  - Snapshot: every read command leaves filesystem identical (no new files, no modified files, no deleted files).
  - Invalid command returns `HQ_INVALID_ARGUMENT` and exit code 2.
  - `hq` with no args returns help/usage via stderr and exit code 2.
  - Build: `go build -o dist/hq ./cmd/hq` succeeds.
  - Smoke: `dist/hq --root testdata/hq context` produces one valid JSON object.
- **Verification:**
  - `go test ./internal/read ./internal/app` — all pass.
  - `go test ./...` — all pass (end-to-end across all packages).
  - `go test -race ./...` — no races detected.
  - `go vet ./...` — no diagnostics.
  - `gofmt -l .` — prints nothing (or only non-Go files if any).
  - `go build -o dist/hq ./cmd/hq` — clean build.
  - `dist/hq --root testdata/hq version && dist/hq --root testdata/hq health && dist/hq --root testdata/hq context && dist/hq --root testdata/hq list --collection projects && dist/hq --root testdata/hq search --query example` — all produce valid JSON with `success: true`.
- **Rollback:** Remove U4 service and dispatcher files; `cmd/hq/main.go` reverts to U1 placeholder. U1-U3 remain intact.

---

### U5. Bootstrap Native CI

- **Goal:** Create a GitHub Actions workflow that runs formatting, vet, unit tests, race tests, and filesystem-snapshot checks on native Windows, Linux, and macOS runners.
- **Requirements:** QR-02 (cross-platform test coverage)
- **Files:**
  - Create: `.github/workflows/ci.yml`
  - Create: `docs/operations/ci.md`
  - Modify: `implementation/STATUS.md`
- **Dependencies:** U4 (all Phase 0 code must be in place and passing locally before CI validation).
- **Approach:**
  1. Add native `windows-latest`, `ubuntu-latest`, and `macos-latest` jobs to `.github/workflows/ci.yml`.
  2. Each job: checkout → setup Go (version from `go.mod`) → `gofmt -l .` (assert empty) → `go vet ./...` → `go test ./...` → `go test -race ./...` (where runner supports race; macOS `amd64` and Linux `amd64`).
  3. Use repository-pinned Go version from `go.mod` (parse with a matrix or `go.mod` directive).
  4. Document CI commands, runner requirements, and evidence expectations in `docs/operations/ci.md`.
  5. Run `go run github.com/rhysd/actionlint/cmd/actionlint@v1.7.7` against `.github/workflows/ci.yml`; expect no diagnostics.
- **Test Scenarios:**
  - Actionlint validates the workflow YAML with no errors.
  - All three platform jobs run `go test ./...` and pass.
  - Race tests pass on Linux and macOS `amd64`.
  - `gofmt -l .` is empty on all platforms.
  - `go vet ./...` passes on all platforms.
  - Snapshot tests pass on all platforms (no cross-platform filesystem differences leak into test assertions).
- **Verification:**
  - Local: `go run github.com/rhysd/actionlint/cmd/actionlint@v1.7.7 .github/workflows/ci.yml` — no diagnostics.
  - Remote: With explicit approval, trigger CI run and record job URLs and pass/fail results in `implementation/STATUS.md`.
  - Fallback (no remote approval): Run equivalent commands manually on Windows, Linux, and macOS; record machine evidence per platform.
- **Rollback:** Remove `.github/workflows/ci.yml` and `docs/operations/ci.md`; local Phase 0 verification (U1-U4) remains available.

---

## Verification Contract

### Standard Commands

Every implementation unit must pass these before verification is claimed:

```bash
go test ./...
go test -race ./...
go vet ./...
gofmt -l .
```

`gofmt -l .` must print nothing. `go vet ./...` must produce no diagnostics.

### Unit-Specific Commands

| Unit | Additional Verification |
|------|------------------------|
| U1 | `go test ./internal/contract ./internal/assets` — serialization and embed tests |
| U2 | `go test ./internal/config ./internal/hq` — path containment tests; platform-conditional tests build |
| U3 | `go test ./internal/hq` — record adapter tests; fixtures exist on disk |
| U4 | `go test ./internal/read ./internal/app` — dispatcher and service tests; `go build -o dist/hq ./cmd/hq` — clean compile; smoke test with `dist/hq --root testdata/hq context` |
| U5 | `go run github.com/rhysd/actionlint/cmd/actionlint@v1.7.7 .github/workflows/ci.yml` — no diagnostics; native CI jobs all green or manual platform evidence recorded |

### Phase Acceptance Gates

- [ ] `go test ./...`, `go test -race ./...`, `go vet ./...`, and `gofmt -l .` pass on the development machine.
- [ ] All six commands (`version`, `health`, `context`, `get`, `list`, `search`) produce one JSON result each with documented exit codes.
- [ ] Fixture snapshots prove no read command creates, modifies, or deletes files.
- [ ] Native Windows, Linux, and macOS CI jobs pass formatting, vet, unit, race, path, read, and snapshot checks.
- [ ] `implementation/STATUS.md` contains exact commands, observed results, and evidence for Tasks 0.1–0.5 (U1–U5).

---

## Definition of Done

### Global Done Criteria

1. All five implementation units (U1–U5) verified per their contract.
2. Every R-ID (R1–R11) has test coverage traceable to a passing test.
3. `implementation/STATUS.md` records evidence for every task: command, OS, result, and artifact path.
4. No abandoned code, debug prints, or experimental dead ends remain in the working tree.
5. The `hq` binary is compilable from source with `go build -o dist/hq ./cmd/hq` on all supported platforms.
6. Phase acceptance gates all checked.

### Per-Unit Done Criteria

| Unit | Done When |
|------|-----------|
| U1 | Contract types serialize per `docs/contracts/results.md`; embedded schema matches source; `main.go` placeholder builds |
| U2 | Root and path resolution passes all platform-conditional containment tests on the current OS; other platforms deferred to U5 |
| U3 | All six fixture collections readable; deterministic listing; SHA-256 and Git metadata correct; `.git/`/`.hq-interface/` excluded |
| U4 | All six commands return correct JSON shapes and exit codes; snapshot verification proves no mutation; `dist/hq` smoke test passes |
| U5 | CI workflow validates on `actionlint`; all three platform jobs green (or equivalent manual evidence); CI doc complete |

### Cleanup Criterion

Before declaring any unit verified, remove:
- `fmt.Println` / `log.Println` debug output not routed through the contract pipeline.
- Commented-out code blocks from earlier attempts.
- Temporary test files or helper scripts not tracked in the implementation plan.

---

## Open Questions

None. All product-scope questions resolved. Implementation details deferred to execution-time discovery (exact Go version pin, ldflags plumbing for build-time version injection, Git commit identification edge cases on shallow clones).

---

## Appendix

### File Inventory (Full)

```
cmd/hq/main.go                          # Entry point (placeholder → full dispatch)
internal/app/app.go                     # CLI dispatcher
internal/app/read_commands.go           # Command routing
internal/app/app_test.go                # Integration-level command tests
internal/assets/assets.go               # //go:embed schemas and policy
internal/assets/assets_test.go          # Embed parity tests
internal/config/config.go               # Root and policy config
internal/config/config_test.go          # Config precedence tests
internal/contract/result.go             # Result, ErrorDetail, MutationState
internal/contract/errors.go             # Error codes and exit mapping
internal/contract/result_test.go        # Serialization and exit-code tests
internal/hq/path.go                     # Safe path resolver
internal/hq/path_test.go                # Cross-platform path tests
internal/hq/path_windows_test.go        # Windows-specific path hazards
internal/hq/path_posix_test.go          # POSIX-specific path hazards
internal/hq/records.go                  # Collection adapters
internal/hq/records_test.go             # Record adapter tests
internal/read/service.go                # Read service struct
internal/read/context.go                # Context composition
internal/read/search.go                 # Full-text search
internal/read/service_test.go           # Read service tests
schemas/result-v1.json                  # Version 1.0 result envelope schema
testdata/hq/CURRENT-WORK.md             # Fixture: current work
testdata/hq/SESSION-LOG.md              # Fixture: session log
testdata/hq/AGENTS.md                   # Fixture: agent instructions
testdata/hq/safety-boundaries.md        # Fixture: safety rules
testdata/hq/projects/example/README.md  # Fixture: project readme
testdata/hq/projects/example/STATE.md   # Fixture: project state
testdata/hq/decisions/2026-07-23-use-go.md   # Fixture: decision
testdata/hq/people/ryann.md             # Fixture: person
testdata/hq/references/api-notes.md     # Fixture: reference
testdata/hq/work-types/feature/example-work.md  # Fixture: work-type
.github/workflows/ci.yml                # CI workflow
docs/operations/ci.md                   # CI documentation
implementation/STATUS.md                # Live progress evidence
go.mod                                  # Go module definition
```

### Requirement Traceability

| Req | Covered By |
|-----|-----------|
| R1 | U1 (contract types), U4 (dispatcher serialization) |
| R2 | U1 (result envelope), U4 (version command) |
| R3 | U4 (health command) |
| R4 | U4 (context command) |
| R5 | U3 (record adapters), U4 (get command) |
| R6 | U3 (collection adapters), U4 (list command) |
| R7 | U4 (search command) |
| R8 | U4 (snapshot verification) |
| R9 | U2 (path containment), U3 (allowlist enforcement) |
| R10 | All units (read-only design; no git mutation code exists) |
| R11 | U1 (embedded schemas) |
