# Phase 2: Apply And Changes Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Apply authorized typed requests without silent data loss, emit immutable receipts, support recovery, and expose durable cursor-based change discovery.

**Architecture:** Renderers transform validated typed payloads into complete target content. The transaction engine runs under a target lock and owns hash recheck, backup, replacement, verification, receipt, and state transition. Change polling reads monotonic receipt cursors.

**Tech Stack:** Go standard library, platform-specific locking and replace primitives behind `fsx.FS`, existing request and policy packages, Go race detector and injectable filesystem failures.

## Global Constraints

- Phase 1 acceptance must be verified.
- Apply never performs an automatic merge or Git mutation.
- A result may claim success only after target and receipt hashes verify.
- Commit steps require explicit commit authorization.

---

### Task 2.1: Typed Markdown Renderers

**Files:**
- Create: `internal/write/render.go`
- Create: `internal/write/render_project_checkin.go`
- Create: `internal/write/render_session_entry.go`
- Create: `internal/write/render_draft_record.go`
- Create: `internal/write/render_current_work.go`
- Create: `internal/write/render_test.go`

**Interfaces:**
- Produces: `Renderer.Render(ctx, request, current []byte) (RenderedTarget, error)`.
- Produces: `RenderedTarget{Path string, Content []byte, SHA256 string, CreateOnly bool}`.

**Rollback:** Remove Task 2.1 renderers and golden outputs; the submit-only Phase 1 CLI remains operational.

- [ ] **Step 1: Write golden failing tests from `docs/contracts/hq-markdown.md` for each typed operation and preservation tests for unrelated current-work entries and project-state sections.**
- [ ] **Step 2: Add rejection tests for embedded newlines in session fields, existing draft targets, malformed current records, and target-operation mismatch.**
- [ ] **Step 3: Run `go test ./internal/write`; verify failures identify missing renderers.**
- [ ] **Step 4: Implement deterministic complete-file rendering without executing markdown content.**
- [ ] **Step 5: Run renderer tests and inspect golden diffs; expect exact matches.**
- [ ] **Step 6: If commits are authorized, commit with `feat: render typed HQ updates`.**

### Task 2.2: Cross-Platform Lock, Backup, And Replace Contract

**Files:**
- Modify: `internal/fsx/fs.go`
- Create: `internal/fsx/lock_windows.go`
- Create: `internal/fsx/lock_posix.go`
- Create: `internal/fsx/replace_windows.go`
- Create: `internal/fsx/replace_posix.go`
- Create: `internal/fsx/transaction_contract_test.go`

**Interfaces:**
- Produces: `FS.Lock(ctx, target string, timeout time.Duration) (UnlockFunc, error)`.
- Produces: `FS.Backup(target, backup string) (sha256 string, error)`.
- Produces: `FS.ReplaceDurable(temp, target string) error`.
- Produces: `FS.Capabilities(root string) (Capabilities, error)`.

**Rollback:** Remove Task 2.2 adapter extensions and restore the Phase 1 durable-request filesystem interface; do not touch real runtime data.

- [ ] **Step 1: Write one adapter contract suite for exclusivity, timeout, backup hash, durable replace, cleanup, and unsupported filesystem behavior.**
- [ ] **Step 2: Run the suite on the current platform and verify failures for absent methods.**
- [ ] **Step 3: Implement POSIX locking and rename semantics behind build constraints, including file and parent-directory synchronization.**
- [ ] **Step 4: Implement Windows locking and replacement using Windows APIs rather than assuming POSIX rename semantics.**
- [ ] **Step 5: Add network/unknown filesystem detection that returns `HQ_UNSUPPORTED_FILESYSTEM` before mutation.**
- [ ] **Step 6: Run native NTFS, ext4, and APFS jobs; record filesystem and OS evidence separately.**
- [ ] **Step 7: If commits are authorized, commit with `feat: add cross-platform transaction primitives`.**

### Task 2.3: Transaction Engine And Receipts

**Files:**
- Create: `internal/contract/receipt.go`
- Create: `schemas/receipt-v1.json`
- Create: `internal/write/transaction.go`
- Create: `internal/write/receipt_store.go`
- Create: `internal/write/transaction_test.go`
- Create: `internal/write/failure_injection_test.go`

**Interfaces:**
- Produces: `TransactionEngine.Apply(ctx, requestID, approvalReference string) (Receipt, error)`.
- Produces: immutable `contract.Receipt` with cursor, hashes, changed path, approval reference, backup path, and mutation state.

**Rollback:** Disable and remove Task 2.3 apply code; retain pending requests and all receipts/backups produced during disposable tests for inspection until the fixture is removed.

- [ ] **Step 1: Write a failing success-path test that checks lock, expected hash, backup, rendered hash, replacement, receipt, and applied state.**
- [ ] **Step 2: Write failing tests for approval-required submission followed by missing apply approval, mismatched request/command approval references, submit-only and denied policy, already-applied idempotency, conflict, lock timeout, tampered request, create-only collision, and same-account procedural evidence.**
- [ ] **Step 3: Add injectable failure tests at every boundary listed in `docs/testing/strategy.md`.**
- [ ] **Step 4: Run `go test ./internal/write`; verify failures.**
- [ ] **Step 5: Implement the transaction sequence exactly as documented, with deferred unlock and explicit recovery-state calculation.**
- [ ] **Step 6: Run `go test -race ./internal/write` including two concurrent writers; verify only one conflicting writer succeeds.**
- [ ] **Step 7: Verify no injected failure reports success or leaves an unverified mutation.**
- [ ] **Step 8: If commits are authorized, commit with `feat: apply HQ requests transactionally`.**

### Task 2.4: Apply, Recovery Status, And Changes Commands

**Files:**
- Modify: `internal/app/write_commands.go`
- Modify: `internal/app/write_commands_test.go`
- Create: `internal/write/changes.go`
- Create: `internal/write/changes_test.go`
- Create: `docs/operations/recovery-command.md`

**Interfaces:**
- Produces: documented `apply`, extended `status`, and `changes` behavior.
- Produces: `Changes.After(cursor uint64, limit int) (Page, error)` with deterministic ordering and `nextCursor`.

**Rollback:** Remove Task 2.4 command wiring and recovery documentation; retain immutable test receipts until disposable fixtures are removed.

- [ ] **Step 1: Write failing CLI tests for apply success, approval required, policy denied, already-applied idempotency, conflict, lock timeout, safely rolled-back interruption, recovery-required interruption, status lifecycle, empty changes, pagination, restart, and concurrent receipts.**
- [ ] **Step 2: Run `go test ./internal/app ./internal/write`; verify failures.**
- [ ] **Step 3: Implement command mapping, Phase 2 health capability checks, and monotonic cursor allocation under an interface-state lock.**
- [ ] **Step 4: Implement `hq recover inspect --request-id <uuid>` and `hq recover restore --request-id <uuid> --approval-reference <text>` exactly as specified in `docs/contracts/cli.md` and `docs/contracts/results.md`.**
- [ ] **Step 5: Write `docs/operations/recovery-command.md` with those exact invocations, required evidence, refusal cases, and exercised fixture output.**
- [ ] **Step 6: Run all tests, race tests, vet, formatting, and a compiled CLI end-to-end fixture workflow.**
- [ ] **Step 7: Record request IDs, receipt hashes, recovery receipt, cursor output, and filesystem snapshot evidence.**
- [ ] **Step 8: If commits are authorized, commit with `feat: expose apply, recovery, and change receipts`.**

## Phase Acceptance

- [ ] All typed operations render deterministic valid markdown.
- [ ] Native filesystem contract tests pass on NTFS, ext4, and APFS.
- [ ] Two-writer race proves one success and one no-mutation conflict.
- [ ] Every failure-injection point produces an inspectable safe state.
- [ ] Cursor polling resumes after restart without missing receipts.
- [ ] Recovery inspect and approved restore match the contract and an exercised fixture recovery.
