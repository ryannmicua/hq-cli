---
lorespec: "0.1"
id: "2026072901"
date: "2026-07-29"
source: "opencode"
topic: "Phase 2 implementation: apply, recovery, and changes for HQ CLI"
tags: [hq-cli, phase-2, go, transaction-engine, cross-platform, apply, recovery]
classification:
  type: technical
  domains: [go, cli, filesystem, transactions]
  value: high
trails: [hq-cli-delivery]
---

## Session Arc

### Started
Executing existing Phase 2 implementation plan for HQ CLI (`docs/plans/2026-07-24-phase-2-apply-and-changes.md`) — four tasks covering typed markdown renderers, cross-platform filesystem primitives, transaction engine with receipts, and CLI commands for apply/changes/recover.

### Pivots
- Discovered Go isn't installed in the execution environment → installed via direct ZIP download and also set up Docker (WSL needed); installed Go 1.22.10 from `go.dev`
- Found that `for i, line := range lines` in Go creates copies — modifying `i` inside a range loop doesn't skip iterations. Switched to C-style `for i := 0; i < len(lines); i++` for the current-work renderer
- Policy glob matching `*.md` didn't match filenames containing dots — `filepath.Match` needed for within-component globbing, not just exact `*` matching

### Ended
Phase 2 fully implemented, tested, and pushed. PR #12 open at `https://github.com/ryannmicua/hq-cli/pull/12`. All 40 files, 3400+ lines, all tests passing.

## ARTIFACTS

### A1: Phase 2 Implementation (4 tasks, 4 commits)
- **Branch:** `feat/phase-2-apply-and-changes`
- **PR:** https://github.com/ryannmicua/hq-cli/pull/12
- **Commits:**
  - `5ff9a30` — `feat: render typed HQ updates` (Task 2.1)
  - `35e75e7` — `feat: add cross-platform transaction primitives` (Task 2.2)
  - `56dd5e3` — `feat: apply HQ requests transactionally` (Task 2.3)
  - `ec5c9d8` — `feat: expose apply, recovery, and change receipts` (Task 2.4)
  - `9c3ec89` — `style: gofmt formatting`

### A2: Renderer Interface and Implementations
- `internal/write/render.go` — `Renderer` interface with `Render(ctx, Request, current) -> RenderedTarget`
- Four implementations: `ProjectCheckInRenderer`, `SessionEntryRenderer`, `DraftRecordRenderer`, `CurrentWorkUpdateRenderer`
- Golden fixture tests in `testdata/golden/` for each operation type

### A3: Cross-Platform Filesystem Adapters
- `internal/fsx/fs.go` — Extended `FS` interface with `Lock`, `Backup`, `ReplaceDurable`, `Capabilities`
- Windows: `LockFileEx`/`UnlockFileEx`, `MoveFileExW` with `MOVEFILE_WRITE_THROUGH`
- POSIX: `flock` with staleness detection (5 min timeout), `os.Rename`
- `Capabilities` struct for filesystem type detection

### A4: Transaction Engine and Receipt Store
- `internal/write/transaction.go` — `TransactionEngine.Apply()`: lock → hash recheck → backup → render → replace → verify → receipt
- Durable intent file (write-ahead journal) written before mutation, reconciled on restart
- Receipt store with cursor-encoded filenames (`%020d.json`) for deterministic ordering

### A5: CLI Commands
- `hq apply --request-id <uuid>` — with `--approval-reference` flag or `HQ_APPROVAL_REFERENCE` env var
- `hq changes --after <cursor> [--limit <n>]` — paginated receipt listing
- `hq recover inspect --request-id <uuid>` and `hq recover restore --request-id <uuid>`
- `docs/operations/recovery-command.md` documenting usage and security notes

## DECISIONS

### D1: Windows Locking via LockFileEx (settled)
- **Issue:** How to implement file locking on Windows
- **Positions:** LockFileEx API vs. flock on a lock file
- **Decision:** Use LockFileEx directly on the lock file handle via `CreateFileW`
- **Warrant:** LockFileEx supports shared/exclusive modes and is the canonical Windows primitive
- **Qualifier:** always
- **Status:** settled

### D2: Within-Component Glob Matching (settled)
- **Issue:** Policy patterns like `inbox/*.md` failed to match because the existing `matchPattern` treated `*` as a full path component match only
- **Decision:** Use `filepath.Match` for within-component matching when pattern doesn't match component exactly
- **Warrant:** `filepath.Match` handles `*.md` → `filename.md` correctly and is the standard Go library for filename globbing
- **Qualifier:** always
- **Status:** settled

### D3: Cursor State from Receipt Files (settled)
- **Issue:** How to make cursor state crash-recoverable
- **Decision:** Scan receipt filenames on startup, initialize counter to `max(cursor) + 1`. `Changes.After` reads receipt files directly, not the in-memory counter
- **Warrant:** Receipt files survive restart; cursor-encoded filenames make readdir produce deterministic order
- **Qualifier:** always
- **Status:** settled

## PATTERNS

### P1: Build-Constrained Platform Adapters
- Two implementations per platform primitive (lock, replace), separated by `//go:build windows` / `//go:build !windows`
- Shared `FS` interface in `fs.go`, platform structs (`windowsFS`, `posixFS`) implement all methods
- NewFS() returns the appropriate implementation transparently
- **Scope:** universal

### P2: Intent-First Transaction Pattern
- Write intent (write-ahead journal) before mutation
- Perform mutation (backup → render → replace)
- Write receipt after verification
- Remove intent atomically
- On restart: scan orphans; if receipt exists → remove intent; if target matches pre-hash → remove intent (no mutation); otherwise mark recovery-required
- **Scope:** universal

## OPEN_QUESTIONS

### Q1: CI Cross-Platform Verification
- Local tests pass on Windows amd64. Requires CI confirmation on Linux/macOS runners for the platform-specific locking and replace implementations.

## NEXT_STEPS

### N1: Babysit PR to Green
- Watch https://github.com/ryannmicua/hq-cli/pull/12 through CI and review
- **Urgency:** now

### N2: Phase 3 Planning
- Master plan defines Phase 3: Skill and Release
- **Urgency:** soon

## Connections
- A1 —[informed_by]→ D1, D2, D3
- D1 —[instance_of]→ P1
- D2 —[instance_of]→ P1
- A4 —[instance_of]→ P2
- A5 —[depends_on]→ A4, A3, A2
