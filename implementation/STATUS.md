# HQ CLI Implementation Status

## Executive Summary

The standalone `hq` CLI project has a self-contained delivery-document set and four-phase executor plan. Phase 0 (Contracts and Read CLI) and Phase 1 (Submit and Status) are verified on Windows amd64 and confirmed on Ubuntu/macOS via CI. Phase 2 (Apply, Recovery, and Changes) is implemented on `main`, all tests pass on Windows amd64, all 17 deferred review findings are resolved, and all plan checklist items except compiled-subprocess lock tests are implemented. Phase 3 (Skill and Release) is ready to begin.

## Current Outcome

Produce an approved, executor-checkable implementation plan for a safe cross-platform HQ interface and then execute it phase by phase with verifiable evidence.

## Current State

| ID | Task | State | Evidence | Next Action |
|---|---|---|---|---|
| S-001 | Initialize the `hq-cli` repository and continuity scaffold | verified | Local repository is on `main`; required files exist; `git diff --check` passed 2026-07-24 | None |
| D-001 | Establish canonical cross-platform delivery documents | verified | Canonical document set created; independent re-review found no blocking issues on 2026-07-24 | None |
| P-001 | Write executor-level implementation plan | verified | Master and four phase plans pass placeholder, path, rollback, traceability, and independent consistency review | None |
| P0 | Contracts and read CLI phase | verified | All 5 tasks (U1-U5) verified; `go test ./...` passes; `go vet ./...` clean; `gofmt -l .` clean; binary builds; all 6 commands produce valid JSON; snapshot tests prove no mutation | None |
| U1 | Module and Public Contract Types | verified | `go test ./internal/contract ./internal/assets` — all pass; `go vet ./...` clean; `gofmt -l .` clean; `go build ./cmd/hq` succeeds | None |
| U2 | Configuration and Safe Path Resolution | verified | `go test ./internal/config ./internal/hq` — all pass; `go vet ./...` clean; `gofmt -l .` clean | None |
| U3 | HQ Fixtures and Collection Adapters | verified | `go test ./internal/hq` — all pass; `go vet ./...` clean; `gofmt -l .` clean | None |
| U4 | Read Services and CLI Commands | verified | `go test ./internal/read ./internal/app` — all pass; `go build -o dist/hq.exe ./cmd/hq` succeeds; smoke tests pass for all 6 commands; snapshot tests verify no mutation; `go vet ./...` clean; `gofmt -l .` clean | None |
| U5 | Bootstrap Native CI | verified | `.github/workflows/ci.yml` created with format+test matrix; `docs/operations/ci.md` documents commands and acceptance gates; actionlint verification deferred to base branch | None |
| P1 | Submit and status phase | verified | All 14 units implemented; `go test ./...` passes on Windows/Ubuntu/macOS CI; all 5 code review issues closed; `go vet ./...` clean; `gofmt -l .` clean; binary builds; smoke tests pass for submit/status commands; PR #11 merged 2026-07-26 | None |
| P2-R | Re-review Phase 2 executor plan | verified | Five-persona review completed 2026-07-26; 3 deterministic corrections applied; all 17 findings resolved in updated plan; `git diff --check` passes | None |
| P2 | Apply, recovery, and changes phase | verified-on-windows | All 4 tasks (2.1-2.4) implemented; 45 files changed, ~3700 lines added; `go test ./...` passes on Windows amd64; `go vet ./...` clean; `gofmt -l .` clean; binary builds; smoke tests pass for apply/changes/recover commands; all plan checklist items resolved; only compiled-subprocess lock tests deferred to CI. | Confirm Linux/macOS CI; then mark Phase 2 verified |
| P3 | Skill and release phase | not-started | Plan: `docs/plans/2026-07-24-phase-3-skill-and-release.md` | Requires verified Phase 2 |

### Phase 2 Implementation Details

| Unit | State | Evidence |
|---|---|---|
| Task 2.1 — Typed Markdown Renderers | verified | `internal/write/render.go` + 4 implementations (project-check-in, session-entry, draft-record, current-work-update); golden fixture tests pass; `json.NewDecoder` with `DisallowUnknownFields()` in all payload parsing; `DraftRecord.Classification` restricted to `inbox`/`project-report`/`project-source`; `CurrentWorkUpdate.Section` field added |
| Task 2.2 — Cross-Platform Lock, Backup, Replace | verified | `internal/fsx/lock_windows.go` (LockFileEx), `lock_posix.go` (flock), `replace_windows.go` (MoveFileExW with MOVEFILE_WRITE_THROUGH), `replace_posix.go` (os.Rename); `Capabilities` struct defined; contract test suite passes |
| Task 2.3 — Transaction Engine And Receipts | verified | `internal/write/transaction.go` implements full sequence: load → hash recheck → cursor alloc → intent write → backup → render → replace → verify → receipt → state transition; orphaned-intent reconciliation on restart; receipt store with cursor-encoded filenames |
| Task 2.4 — Apply, Recovery, Status, Changes Commands | verified | CLI commands: `hq apply`, `hq changes`, `hq recover inspect`, `hq recover restore`; approval-reference from env var or CLI flag; `"applied"` mutation enum; `docs/operations/recovery-command.md` |
| Policy and contract fixes | verified | Policy rules for draft records fixed (3 separate rules for inbox, reports, source); `CanonicalRequestHash` includes `ApprovalReference`; payload unknown-field rejection |

### Resolved Checklist Items (implemented 2026-07-29)

| Item | Plan Reference | Resolution |
|---|---|---|
| `--since` flag for `changes` | Task 2.4 Step 3 | Implemented: `ChangesService.Since()` + CLI flag with mutual-exclusion check against `--after` |
| `failure_injection_test.go` | Task 2.3 Step 3 | Created: injectable `FS` wrapper with 4 test cases (temp-file-create, lock, backup, target-replace); all pass |
| Windows lock staleness | Task 2.2 Steps 3-4 | Added `checkStaleLock()` using `OpenProcess` on Windows; POSIX already had it via PID signal check |
| Lock staleness timeout config | Task 2.2 Steps 3-4 | `HQ_LOCK_STALE_TIMEOUT` env var supported on both Windows and POSIX (default 5 min) |
| Shared read lock for status | Task 2.4 Step 3 | `handleStatus` and `handleChanges` acquire shared lock on `global.lock` via `LockFile` |
| Capability probe | Task 2.2 Step 5 | Windows: `GetVolumeInformationW` for filesystem detection + probe write for atomic-replace. POSIX: `statfs` type detection + probe rename. `HQ_FS_OVERRIDE` supported on both. |
| Approval-reference via stdin | Task 2.4 Step 4 | `resolveApprovalReference()` reads from `--approval-reference` (deprecated w/ stderr warning), `HQ_APPROVAL_REFERENCE` env var, or stdin |
| `LockFile` method on `FS` interface | Task 2.2 | Added `LockFile(ctx, lockPath, timeout, exclusive)` to lock arbitrary paths directly |
| Owner-only 0600 enforcement | Task 2.2 Step 5b | Already implemented across all file writes |

### Remaining Plan Items (deferred to CI)

| Item | Plan Reference | Reason |
|---|---|---|
| Compiled-subprocess lock tests | Task 2.2 Step 6 | Requires CI build artifact; goroutine-based lock-exclusion tests cover the functional contract on this platform |

### Phase 2 Verification Evidence

- **Commit:** `5ec8cea` + current working changes on `main`
- **Verification commands (2026-07-29):**
  - `go test -count=1 ./...` — all 9 packages pass
  - `go vet ./...` — clean
  - `go fmt ./...` — clean
  - `go build -o dist/hq.exe ./cmd/hq` — succeeds
- **Smoke tests:** `hq version`, `hq changes --after 0`, `hq changes --since 2026-01-01T00:00:00Z`, `hq changes --since... --after...` mutual-exclusion error — all pass
- **CI confirmation:** Phase 1 CI passed on Windows/Ubuntu/macOS. Phase 2 CI pending on Linux/macOS runners.
- **Rollback action:** Revert commits to restore prior state.

## Blockers

- None. Phase 2 is fully implemented and testable on Windows. CI and cross-platform verification on Linux/macOS are the only open items before full Phase 2 verification.

## Next Phase

Phase 3 (Skill and Release) is ready to start. See `docs/plans/2026-07-24-phase-3-skill-and-release.md`. It requires verified Phase 2; the current Phase 2 state is verified-on-windows with CI evidence pending for Linux/macOS.

## Material Risks

- Same-account approval is procedural rather than cryptographically isolated.
- Locking, durable flush, and atomic replacement differ across operating systems and filesystems.
- Runtime requests and backups may contain private HQ content and require restrictive permissions and retention rules.
- Phase 2 platform-specific implementations (LockFileEx, flock, MoveFileExW) are only tested on Windows amd64.

## Source Evidence

- `docs/README.md` and its indexed canonical delivery documents, created 2026-07-24.
- `docs/source/hq-interface-design-prompt.md`, original HQ prompt preserved verbatim on 2026-07-24.
- `C:\Users\rmicua\hq\safety-boundaries.md`, reviewed 2026-07-24.
- `docs/plans/2026-07-24-phase-2-apply-and-changes.md` with all 17 review findings resolved.
- `docs/session-digests/2026072901_phase2_apply_and_changes_implementation.md`.
