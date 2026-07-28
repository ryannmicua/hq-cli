# Phase 2: Apply And Changes Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Apply authorized typed requests without silent data loss, emit immutable receipts, support recovery, and expose durable cursor-based change discovery.

**Architecture:** Renderers transform validated typed payloads into complete target content. The transaction engine runs under a target lock and owns hash recheck, backup, replacement, verification, receipt, and state transition. A durable intent file (write-ahead journal) is written before any mutation and removed after receipt publication; orphaned intents are reconciled on restart. Change polling reads monotonic receipt cursors from cursor-encoded receipt filenames.

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

- [ ] **Step 1: Validate `docs/contracts/hq-markdown.md` assumptions by writing assertion tests against the existing fixture data (in `testdata/hq/`) that confirm section order, owned-section content, and unknown-section preservation behavior. Then write golden failing tests from the same contract for each typed operation and preservation tests for unrelated current-work entries and project-state sections.**
- [ ] **Step 1b: Fix payload type contracts to match hq-markdown.md: restrict `DraftRecord.Classification` to only `"inbox"`, `"project-report"`, `"project-source"` (remove `"draft"`, `"review"`, `"archive"`); add `Section` field to `CurrentWorkUpdate` for distinguishing `"Active"` from `"Warm"` entry creation; use `json.NewDecoder` with `DisallowUnknownFields()` in all payload parsing to reject unknown fields.**
- [ ] **Step 1c: Fix draft-record policy rules in `config/policy-v1.json`, `internal/assets/policy-v1.json`, and `internal/write/policy_test.go`: replace `draft-record` → `projects/*/*.md` (allowed) with three rules — `inbox/*.md` (allowed), `projects/*/reports/*.md` (allowed), and `projects/*/source/*.md` (allowed). Remove the broad `projects/*/*.md` rule.**
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
- Produces: `FS.Lock(ctx context.Context, target string, timeout time.Duration, exclusive bool) (UnlockFunc, error)`. Read commands acquire a shared (non-exclusive) lock to avoid observing partial write state; write transactions acquire an exclusive lock that excludes both other writers and readers.
- Produces: `FS.Backup(target, backup string) (sha256 string, error)`. Backup files must be created with owner-only permissions (0600 POSIX, equivalent Windows DACL). Backup paths must incorporate the request ID (e.g., `.hq-interface/backups/<target-slug>-<request-id>.bak`) so repeated applies never collide; old backups may be cleaned up by subsequent successful transactions on the same target.
- Produces: `FS.ReplaceDurable(temp, target string) error`.
- Produces: `FS.Capabilities(root string) (Capabilities, error)`. `Capabilities` is a struct in `internal/fsx/fs.go` with fields: `SupportAtomicReplace bool`, `SupportFileLocking bool`, `FilesystemType string` (`"ntfs"`, `"ext4"`, `"apfs"`, `"network"`, `"unknown"`), `RootPath string`, and `SupportOwnerOnlyPermissions bool`. All fields are populated during the probe-write step.

**Rollback:** Remove Task 2.2 adapter extensions and restore the Phase 1 durable-request filesystem interface; do not touch real runtime data.

- [ ] **Step 1: Write one adapter contract suite for exclusivity, timeout, backup hash, durable backup sync (backup file + parent directory must be fsynced before replacement), durable replace, cleanup, and unsupported filesystem behavior.**
- [ ] **Step 2: Run the suite on the current platform and verify failures for absent methods.**
- [ ] **Step 3: Implement POSIX locking and rename semantics behind build constraints, including file and parent-directory synchronization. Support shared (`LOCK_SH`) and exclusive (`LOCK_EX`) lock modes via the `exclusive` parameter exposed in the `FS.Lock` signature. Include the acquiring process hostname, PID, and timestamp in the lock file content. On acquisition attempt, if the lock file exists, check for staleness: a lock older than 5 minutes (configurable via `HQ_LOCK_STALE_TIMEOUT`) is considered stale, the CLI breaks it, and retries acquisition. A non-stale lock waits for the full lock timeout before failing.**
- [ ] **Step 4: Implement Windows locking and replacement using Windows APIs rather than assuming POSIX rename semantics. Support shared (`LOCKFILE_FAIL_IMMEDIATELY` without `LOCKFILE_EXCLUSIVE_LOCK`) and exclusive (`LOCKFILE_EXCLUSIVE_LOCK`) lock modes via the `exclusive` parameter.**
- [ ] **Step 5: Add filesystem capability detection using a layered approach: (1) statfs-based identification of known-safe types (NTFS, ext4, APFS); (2) a probe write on the lock-path parent that tests the atomic replace contract (write temp file, durable sync, rename, verify target) before any real transaction; (3) an `HQ_FS_OVERRIDE` environment variable to bypass detection for known edge cases. Probe results are cached for the process lifetime. Capability validation must be performed after lock acquisition, not before: acquire lock on the user-supplied path first, then resolve ALL mutation paths (target, lock directory, temp directory, backup directory) to real paths (dereferencing symlinks/reparse points), verify all are under the HQ root, validate all are on the same filesystem type, then validate capabilities on the resolved paths. This eliminates TOCTOU windows where an adversary could swap a symlink or reparse point between check and mutation, and prevents mutation escaping the HQ root boundary.**
- [ ] **Step 5b: Enforce owner-only permissions on all runtime artifacts: temp files, receipts, backup files, intent files, and lock files must be created with 0600 (POSIX) or equivalent owner-only DACL (Windows). Target replacement must preserve the original file's permission metadata (use stat+chmod after replacement to copy permissions from the original, or use platform-specific replace APIs that preserve security attributes). Verify this in the contract suite.**
- [ ] **Step 6: Run native NTFS, ext4, and APFS jobs; record filesystem and OS evidence separately. Include compiled-subprocess lock-exclusion tests: build the test binary, spawn two competing writer subprocesses targeting the same file, and verify exactly one succeeds while the other receives `HQ_LOCK_TIMEOUT`. Repeat with a lock-holder crash scenario where the lock file remains and a restarted subprocess detects and breaks it via staleness.**
- [ ] **Step 7: If commits are authorized, commit with `feat: add cross-platform transaction primitives`.**

### Task 2.3: Transaction Engine And Receipts

**Files:**
- Modify: `internal/assets/assets.go` (add `go:embed` directive and `ReceiptSchemaV1` export)
- Modify: `internal/assets/assets_test.go` (add embed-parity test for receipt schema)
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
- [ ] **Step 2: Write failing tests for approval-required submission followed by missing apply approval, mismatched request/command approval references, submit-only and denied policy, already-applied idempotency, conflict, lock timeout, tampered request (modified stored request before apply), create-only collision (file exists at target), and same-account procedural evidence.**
- [ ] **Step 3: Add injectable failure tests at every boundary listed in `docs/testing/strategy.md`.**
- [ ] **Step 4: Run `go test ./internal/write`; verify failures.**
- [ ] **Step 5: Implement the transaction sequence exactly as documented in `docs/contracts/results.md` and this plan, embed the receipt schema with source-byte parity tests, and use deferred unlock with explicit recovery-state calculation. Receipts must be stored in `.hq-interface/receipts/` with zero-padded cursor-encoded filenames (`<cursor-20-digit>.json`) so that filesystem readdir produces deterministic cursor order across all platforms. The transaction sequence must be: (1) load stored request and verify its hash matches the submit-time persisted hash (tamper detection); (2) allocate cursor; (3) write a durable intent file (`.hq-interface/intents/<request-id>.json`) recording request-id, cursor, target, backup-path, pre-hash, rendered-content-hash, and timestamp; (4) recheck target hash against expected-target-hash; (5) create and verify backup; (6) write and durable-sync temp rendered file; (7) replace target and durable-sync parent; (8) write and durable-sync receipt (with embedded cursor) using the cursor-encoded filename; (9) update request state; (10) durable-sync receipt directory; (11) atomically remove intent file. `CanonicalRequestHash` must include `ApprovalReference` in its hashed field set so authorization-relevant modification is detectable. On any subsequent transaction start, scan `.hq-interface/intents/` for orphaned intents and reconcile each: if receipt exists for that request-id, remove intent (receipt is authoritative); if target hash matches rendered-content-hash but no receipt, recreate receipt from intent; if target hash matches pre-hash, remove intent (no mutation occurred); otherwise mark request `recovery-required`. Create-only transactions: use `O_CREAT | O_EXCL` (or equivalent platform atomic-exclusive-create) for the target to prevent racing-writer overwrites; skip backup creation since there is no pre-existing content; if the file already exists at step 4 (target hash recheck), reject with `HQ_VERSION_CONFLICT`. On restart reconciliation of a create-only orphan intent: if the target file exists and its hash matches the expected rendered hash, recreate the receipt (success); otherwise mark `recovery-required`. The cursor must be allocated before the receipt file is written and embedded in the receipt JSON content so the receipt itself is the authoritative cursor publication — no separate cursor-advance step follows receipt creation. The cursor counter file (`.hq-interface/cursor`) is a monotonic allocator only; the authoritative set of published cursors is reconstructed from receipts on restart.**
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
- Create: `internal/write/recovery.go`
- Create: `internal/write/recovery_test.go`
- Create: `docs/operations/recovery-command.md`

**Interfaces:**
- Produces: documented `apply`, extended `status`, and `changes` behavior.
- Produces: `Changes.After(cursor uint64, limit int) (Page, error)` with deterministic ordering and `nextCursor`.
- Produces: `RecoveryService` interface (`Inspect(ctx, requestID) (RecoveryInspection, error)` and `Restore(ctx, requestID, approvalReference string) (Receipt, error)`) in a new `internal/write/recovery.go` so command handlers call the interface rather than containing transaction mechanics.

**Rollback:** Remove Task 2.4 command wiring and recovery documentation; retain immutable test receipts until disposable fixtures are removed.

- [ ] **Step 0: Add `"applied"` to the mutation enum in `schemas/result-v1.json`, `internal/contract/result.go`, and the `Mutation` property in `docs/contracts/cli.md`. Phase 1's submit command retains `noMutation`; only apply emits `"applied"`.**
- [ ] **Step 2: Run `go test ./internal/app ./internal/write`; verify failures.**
- [ ] **Step 3: Implement command mapping, Phase 2 health capability checks, and monotonic cursor allocation under an interface-state lock. Cursor state must be reconstructed from all existing receipts on first allocation after restart: scan `.hq-interface/receipts/`, collect all cursor values, and initialize the counter to `max(cursor) + 1`. This eliminates gaps between allocation and publication and makes cursor state crash-recoverable without a separate scan pass. `Changes.After` reads receipt files directly — it does not depend on the in-memory counter — so restart never loses published receipts. `status` and `changes` commands must acquire a shared read lock on `.hq-interface/` before reading, preventing observation of partial write state. `--since` and `--after` flags are mutually exclusive: the CLI handler returns `HQ_INVALID_ARGUMENT` if both are provided. `--since` is converted to a cursor by scanning receipts for the first receipt whose timestamp >= the supplied time, then delegating to `Changes.After`.**
- [ ] **Step 4: Implement `hq recover inspect --request-id <uuid>` and `hq recover restore --approval-reference <text> --request-id <uuid>` (approval-reference must be provided via `HQ_APPROVAL_REFERENCE` environment variable or read from stdin, not passed as a CLI argument visible to process listing) exactly as specified in `docs/contracts/cli.md` and `docs/contracts/results.md`. Update the `apply` command to also accept `--approval-reference` from environment variable or stdin. Publish in docs/operations/recovery-command.md that passing approval references via `--approval-reference` on the command line is deprecated and warns on stderr. The restore command must validate the approval reference through the same policy engine as `apply` — a non-empty check alone is insufficient. Additionally, require the restore approval reference to match the one recorded in the original apply receipt when a receipt exists for the request being restored. Before restoring, the command must recheck the target hash against the hash recorded during `recover inspect`; a mismatch after lock acquisition rejects the restore with `HQ_VERSION_CONFLICT`.**
- [ ] **Step 5: Write `docs/operations/recovery-command.md` with those exact invocations, required evidence, refusal cases, and exercised fixture output. Include a warning that backups contain pre-apply content which may include credentials or secrets that were removed by the applied change; restore may reintroduce previously-removed secrets. Document that backup files use owner-only permissions and should be treated as sensitive.**
- [ ] **Step 6: Run all tests, race tests, vet, formatting, and a compiled CLI end-to-end fixture workflow. The compiled E2E must exercise all four typed operations (project-check-in, session-entry, draft-record, current-work-update) each through a complete submit → apply → status → changes journey, verifying receipt content, cursor advancement, and target file state.**
- [ ] **Step 7: Record request IDs, receipt hashes, recovery receipt, cursor output, and filesystem snapshot evidence.**
- [ ] **Step 8: If commits are authorized, commit with `feat: expose apply, recovery, and change receipts`.**

## Phase Acceptance

- [ ] All typed operations render deterministic valid markdown.
- [ ] Native filesystem contract tests pass on NTFS, ext4, and APFS.
- [ ] Two-writer race (in-process goroutines AND compiled subprocesses) proves one success and one no-mutation conflict.
- [ ] Every failure-injection point produces an inspectable safe state.
- [ ] Cursor polling resumes after restart without missing receipts (cursor is reconstructed from receipts on startup, not from a standalone counter).
- [ ] Recovery inspect and approved restore match the contract and an exercised fixture recovery.

## Resolved Open Questions

All open questions from the 2026-07-26 and 2026-07-29 reviews have been resolved in the plan above. Concrete resolutions are encoded directly into the task steps and architecture description. See the following for cross-reference:

- **Q1 (receipt-cursor atomicity)**: Task 2.3 Step 5 — cursor allocated before receipt, embedded in receipt JSON, cursor state reconstructed from receipts on restart.
- **Q2, Q26 (crash window, abrupt crashes)**: Task 2.3 Step 5 — durable intent file before mutation, orphaned-intent reconciliation on restart.
- **Q3 (restore authorization)**: Task 2.4 Step 4 — policy-engine validation for restore, approval reference must match original receipt.
- **Q4 (mutation enum)**: Task 2.4 Step 0 — `"applied"` added to schema, Go enum, and CLI contract.
- **Q5 (approval reference exposed)**: Task 2.4 Step 4 — read from env var or stdin, not CLI arg; deprecate CLI flag.
- **Q6 (deterministic ordering)**: Task 2.3 Step 5 — cursor-encoded receipt filenames for stable readdir.
- **Q7 (cursor restart persistence)**: Task 2.4 Step 3 — receipts survive restart, `Changes.After` reads receipt files directly.
- **Q8 (cursor ordering gap)**: Resolved by Q1 migration — cursor allocation defined in Task 2.3 Step 5.
- **Q9 (read-side isolation)**: Task 2.2 Lock interface includes shared/exclusive modes; Task 2.4 Step 3 — read commands acquire shared lock.
- **Q10, Q30 (backup permissions, permission invariants)**: Task 2.2 Backup returns owner-only; Task 2.2 Step 5b — owner-only artifacts + permission preservation on replace.
- **Q11, Q28 (symlink redirection, mutation path safety)**: Task 2.2 Step 5 — validate after lock, resolve all mutation paths to real paths under HQ root.
- **Q12, Q32 (restore target precondition)**: Task 2.4 Step 4 — recheck target hash under lock before restore.
- **Q13 (filesystem detection)**: Task 2.2 Step 5 — layered detection (statfs + probe write + override env var).
- **Q14 (journal pattern)**: Architecture section — intent file IS the journal; Task 2.3 Step 5 intent lifecycle.
- **Q15 (backup collision)**: Task 2.2 Backup — request-ID in backup filename.
- **Q16 (assets.go embedding)**: Task 2.3 files list — embed directive and ReceiptSchemaV1 export.
- **Q17 (Capabilities type)**: Task 2.2 interface — `Capabilities` struct defined in fs.go with all required fields.
- **Q18 (ambiguous reference)**: Task 2.3 Step 5 — names `docs/contracts/results.md` and this plan.
- **Q19 (stale lock)**: Task 2.2 Step 3 — lock file content with PID/timestamp, staleness timeout.
- **Q20 (renderer contract)**: Task 2.1 Step 1 — validate contract assumptions against fixture data before golden tests.
- **Q21 (direct editors)**: Task 2.3 Step 5 — hash recheck at apply time detects any modification (not just CLI).
- **Q22, Q23 (payload contracts, draft policy)**: Task 2.1 Steps 1b-1c — fix DraftRecord classifications, CurrentWorkUpdate Section field, payload unknown-field rejection, policy rules for canonical draft destinations.
- **Q24 (cursor publication)**: Resolved by Q1 — receipt IS publication.
- **Q25 (tamper baseline)**: Task 2.3 Step 5 — verify stored request hash against submit-time persisted hash; `CanonicalRequestHash` includes `ApprovalReference`.
- **Q27 (create-only semantics)**: Task 2.3 Step 5 — `O_CREAT | O_EXCL` for creates, no backup, absent-before recovery rule.
- **Q29 (backup durability)**: Task 2.2 Step 1 — backup file + parent directory fsynced before replacement.
- **Q31 (recovery service boundary)**: Task 2.4 files and interface — `RecoveryService` interface with `Inspect`/`Restore` methods.
- **Q33 (lock subprocesses)**: Task 2.2 Step 6 — compiled-subprocess lock-exclusion tests.
- **Q34 (typed journeys)**: Task 2.4 Step 6 — compiled E2E journeys for all four typed operations.
- **Q35 (change filters)**: Task 2.4 Step 3 — `--since` and `--after` mutually exclusive; `--since` converted to cursor scan.
- **Q36 (envelope state)**: Resolved by Q4 — `"applied"` added.
- **Q37 (backup secrets)**: Task 2.4 Step 5 — recovery docs warn about secrets in backups.
