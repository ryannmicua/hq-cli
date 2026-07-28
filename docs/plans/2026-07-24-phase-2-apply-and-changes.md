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
- Produces: `Renderer.Render(ctx context.Context, request contract.Request, current []byte) (RenderedTarget, error)`.
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
- Produces: `FS.Lock(ctx context.Context, target string, timeout time.Duration) (UnlockFunc, error)`.
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
- Modify: `internal/assets/assets.go`
- Modify: `internal/assets/assets_test.go`
- Create: `internal/contract/receipt.go`
- Create: `schemas/receipt-v1.json`
- Create: `internal/assets/schemas/receipt-v1.json`
- Create: `internal/write/transaction.go`
- Create: `internal/write/receipt_store.go`
- Create: `internal/write/transaction_test.go`
- Create: `internal/write/failure_injection_test.go`

**Interfaces:**
- Produces: `TransactionEngine.Apply(ctx context.Context, requestID, approvalReference string) (Receipt, error)`.
- Produces: immutable `contract.Receipt` with cursor, hashes, changed path, approval reference, backup path, and mutation state.

**Rollback:** Disable and remove Task 2.3 apply code; retain pending requests and all receipts/backups produced during disposable tests for inspection until the fixture is removed.

- [ ] **Step 1: Write a failing success-path test that checks lock, expected hash, backup, rendered hash, replacement, receipt, and applied state.**
- [ ] **Step 2: Write failing tests for approval-required submission followed by missing apply approval, mismatched request/command approval references, submit-only and denied policy, already-applied idempotency, conflict, lock timeout, tampered request, create-only collision, and same-account procedural evidence.**
- [ ] **Step 3: Add injectable failure tests at every boundary listed in `docs/testing/strategy.md`.**
- [ ] **Step 4: Run `go test ./internal/write`; verify failures.**
- [ ] **Step 5: Implement the transaction sequence exactly as documented, embed the receipt schema with source-byte parity tests, and use deferred unlock with explicit recovery-state calculation.**
- [ ] **Step 6: Run `go test -race ./internal/write` including two concurrent writers; verify only one conflicting writer succeeds.**
- [ ] **Step 7: Verify no injected failure reports success or leaves an unverified mutation.**
- [ ] **Step 8: If commits are authorized, commit with `feat: apply HQ requests transactionally`.**

### Task 2.4: Apply, Recovery Status, And Changes Commands

**Files:**
- Modify: `internal/app/app.go`
- Modify: `internal/app/app_test.go`
- Modify: `internal/app/write_commands.go`
- Modify: `internal/read/service.go`
- Modify: `internal/read/service_test.go`
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

## Deferred / Open Questions

### From 2026-07-26 review

- **Direct editors bypass the no-loss guarantee** — Goal and Architecture (P1, feasibility, adversarial, confidence 100)

  A direct editor can change a target after validation and have that edit overwritten because only cooperating CLI processes honor the lock. The current acceptance test proves serialization between CLI writers, not protection for ordinary direct edits.

- **Typed payload contracts conflict with renderer assumptions** — Task 2.1: Typed Markdown Renderers (P1, feasibility, security-lens, confidence 100)

  Canonical writes can be rejected or rendered incorrectly because current payload types permit the wrong draft classifications, omit the section required for a new current-work entry, and do not strictly reject unknown or unsafe fields. Implementers would otherwise redesign public contracts during renderer work.

- **Draft policy rejects canonical destinations** — Task 2.1: Typed Markdown Renderers (P1, feasibility, confidence 100)

  Every documented draft destination is currently denied while a noncanonical project path is allowed. Draft application cannot succeed until policy rules align with the canonical inbox, project-report, and project-source destinations.

- **Cursor publication ordering is undefined** — Tasks 2.3-2.4: Receipts and Changes (P1, coherence, adversarial, confidence 100)

  Consumers can permanently miss a receipt when cursor allocation is separate from durable receipt publication. The plan also schedules cursor allocation after the task that must already create immutable receipts carrying cursors.

- **Stored requests lack a trusted tamper baseline** — Task 2.3: Transaction Engine (P1, feasibility, security-lens, confidence 100)

  A modified pending request can be applied as though it were the original because no immutable submit-time digest is persisted for comparison. The current request hash also excludes approval evidence, so authorization-relevant modification cannot be detected reliably.

- **Abrupt crashes bypass recoverable state** — Task 2.3: Transaction Engine (P1, adversarial, feasibility, confidence 100)

  A process kill after target replacement can leave changed HQ content with no receipt, final request state, or discoverable recovery evidence. Returned-error injection cannot prove restart safety without durable transaction intent and reconciliation.

- **Create-only transaction semantics are incomplete** — Tasks 2.2-2.3: Filesystem Transaction (P1, feasibility, security-lens, adversarial, confidence 100)

  A racing creator can be overwritten, and an interrupted creation can leave an unreceipted file that backup-based restore cannot remove safely. Create-only work needs atomic no-replace installation plus a durable absent-before-state recovery rule.

- **Mutation paths are not race-safe** — Task 2.2: Filesystem Contract (P1, feasibility, security-lens, adversarial, confidence 100)

  A linked ancestor, reparse-point swap, or nested unsupported mount can redirect a mutation beyond the boundary that was validated earlier. Mutation-time containment and capability checks must cover the actual target, temporary, lock, and backup directories.

- **Verified backups are not crash-durable** — Task 2.2: Filesystem Contract (P1, adversarial, confidence 75)

  A power loss after replacement can leave neither the original target nor a usable backup even when the backup hash was read back successfully. Backup completion must include durable file and parent-directory synchronization before replacement begins.

- **Permission invariants are unspecified** — Task 2.2: Filesystem Contract (P1, security-lens, confidence 75)

  Runtime artifacts can inherit access broader than the owner, and target replacement can weaken existing file protections without changing content hashes. The filesystem contract needs owner-only runtime creation and verified preservation of target security metadata.

- **Recovery commands lack a service boundary** — Tasks 2.3-2.4: Recovery (P1, coherence, confidence 75)

  Implementers must invent where read-only inspection and transactional restore belong because only the apply interface is declared. Without an explicit recovery service contract, command code may absorb transaction mechanics or incompatible APIs may emerge.

- **Restore approval lacks a target precondition** — Task 2.4: Recovery (P1, security-lens, confidence 75)

  Restore can erase a legitimate edit made after inspection because approval is not bound to the target state the operator reviewed. The target hash must be rechecked under lock before an approved restore mutates it.

- **Lock acceptance misses independent processes** — Task 2.2 and Phase Acceptance (P1, adversarial, confidence 75)

  Same-process race tests can pass while separate CLI invocations fail to exclude one another or release locks after a crash. Native acceptance must exercise compiled subprocesses on every validated filesystem.

- **Compiled acceptance omits typed journeys** — Task 2.4 and Phase Acceptance (P1, product-lens, confidence 75)

  One or more promised write workflows can fail through the compiled CLI even when renderer and generic transaction tests pass. Each of the four typed operations needs its own submit, apply, status, and change-discovery fixture journey.

- **Change filter interaction is undefined** — Task 2.4: Changes (P2, coherence, feasibility, confidence 100)

  Implementers must choose incompatible behavior because the public command accepts both cursor and time filters while the planned service accepts only a cursor. The contract must decide whether the filters are mutually exclusive or combine conjunctively.

- **Successful apply lacks an envelope state** — Tasks 2.3-2.4: Result Contract (P2, feasibility, confidence 75)

  A successful authoritative mutation cannot be represented accurately because the result envelope permits only no-mutation and failure-recovery states while receipt examples use an applied state. The versioned result contract must resolve that contradiction before command wiring.

- **Backups can retain prohibited secrets** — Tasks 2.2-2.3: Backups (P2, security-lens, confidence 75)

  A credential already present in a target can survive its removal because apply retains the pre-write content as a backup. The plan must reconcile mandatory recoverability with the absolute prohibition on storing secrets in backups.

### From 2026-07-29 review

- **Receipt-cursor atomicity unspecified** — Task 2.3 / Phase Acceptance (P0, adversarial, confidence 100)

  Cursor advancement and receipt creation are independent filesystem operations; crashes between them produce receipt gaps or duplicate processing that violate the no-silent-data-loss guarantee.

- **Crash window during replace unanalyzed** — Task 2.3 Step 5 (P0, adversarial, confidence 100)

  No recovery-state machine distinguishes pre-replace, partial-replace, and post-replace-pre-receipt states after a crash, making it impossible to determine whether to restore from backup or retry.

- **Restore authorization too weak** — Task 2.4 (P0, security-lens, confidence 100)

  The recover restore command authorizes destructive file overwrite by checking only that the approval-reference is non-empty; any local user who knows the command shape can restore backups without policy-engine evaluation.

- **Result schema mutation enum incomplete** — Task 2.3 (P1, feasibility, confidence 100)

  The result envelope constrains mutation to noMutation/rolledBack/recoveryRequired but the apply receipt contract requires "applied"; a successful authoritative mutation cannot be represented accurately.

- **Approval reference exposed in process listing** — Task 2.4 (P1, security-lens, confidence 75)

  The approval-reference flag passes the reference as a CLI argument visible to all local processes via ps or WMI, leaking audit data to any user on the host.

- **Deterministic ordering lacks enforcement** — Task 2.4 (P1, adversarial, confidence 75)

  Changes.After claims deterministic ordering but filesystem readdir does not guarantee stable order across platforms after fsck, defrag, or restore cycles; consumers experience non-deterministic behavior across reboots.

- **Cursor persistence across restarts underspecified** — Phase Acceptance vs Task 2.4 (P1, coherence, confidence 75)

  Phase Acceptance requires cursor polling to survive process restart without missing receipts, but no implementation step specifies how cursor state is persisted; implementers will produce incompatible designs.

- **Cursor allocation ordering gap** — Tasks 2.3-2.4 (P1, feasibility, confidence 75)

  Receipt carries a cursor field created in Task 2.3 but cursor allocation is not specified until Task 2.4 Step 3, leaving the Task 2.3 implementer without a mechanism for producing valid receipts.

- **Read-side isolation not addressed** — Architecture (P1, adversarial, confidence 75)

  Readers (status, changes, polling consumers) do not acquire the transaction lock, so concurrent reads observe partially written mutation state and produce inconsistent reporting to users.

- **Backup permissions unspecified** — Task 2.2 (P1, security-lens, confidence 75)

  Backups contain complete pre-mutation content with the same sensitivity as production records but have no documented owner-only creation or permission contract.

- **Symlink redirection between check and mutation** — Task 2.2 (P1, security-lens, confidence 75)

  Filesystem capability validation precedes lock acquisition, but an adversary with filesystem write access can swap a symlink or reparse point between check and use, redirecting mutation.

- **Restore lacks target-state precondition** — Task 2.4 (P1, security-lens, confidence 75)

  An approved restore can overwrite legitimate edits made between inspection and restore because approval is not bound to the target hash observed during inspect.

- **Filesystem detection unreliable for mutation gating** — Task 2.2 Step 5 (P1, adversarial, confidence 75)

  Statfs-based filesystem-type detection produces false negatives that silently proceed with non-atomic mutation on network filesystems, and false positives that block legitimate local use.

- **Journal-based atomic write not evaluated** — Architecture (P1, adversarial, confidence 75)

  The lock-backup-replace sequence has asymmetric failure modes; the standard journal-rename pattern used by database engines and package managers is not discussed or compared.

- **Backup path collision on repeated applies** — Task 2.2 (P1, adversarial, confidence 75)

  Apply running twice against the same target file overwrites the first backup before the second transaction completes, destroying the only usable restore point.

- **Assets.go not extended for receipt schema** — Task 2.3 (P1, feasibility, confidence 75)

  The plan creates internal/assets/schemas/receipt-v1.json but does not add the go:embed directive or exported variable following the established Phase-1 embed pattern.

- **FS.Capabilities return type undefined** — Task 2.2 (P1, feasibility, confidence 75)

  The interface method FS.Capabilities returns a Capabilities type that is never defined; implementers will produce incompatible struct definitions that shape the FS adapter contract.

- **Ambiguous implementer reference** — Task 2.3 Step 5 (P2, coherence, confidence 75)

  Other steps name specific contract documents for reference, but Task 2.3 Step 5 says "exactly as documented" without naming which document is authoritative.

- **Stale lock after crash not handled** — Task 2.2 (P3, adversarial, confidence 50)

  Filesystem lock files survive process crashes; without lock-file content conventions and timeout-based staleness checks, a restarted CLI refuses to operate with no documented recovery procedure.

- **Renderer contract unreviewed** — Task 2.1 (P3, adversarial, confidence 50)

  Renderers implement the hq-markdown.md contract, but the contract itself is not validated within this plan's scope; golden tests reify specification errors, making them harder to detect later.
