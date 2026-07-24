# Phase 1: Submit And Status Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a durable, idempotent typed request queue and request status lookup without mutating authoritative HQ records.

**Architecture:** Strict request schemas and independent policy classify each typed operation. The request store writes to a private temporary file, flushes it, and atomically renames it into `pending`. Status derives from immutable request and state locations.

**Tech Stack:** Go standard library, `encoding/json`, `crypto/sha256`, existing contract and safe-path packages, platform filesystem adapter.

## Global Constraints

- Phase 0 acceptance must be verified.
- `submit` may change only `.hq-interface/`; it never changes authoritative HQ records.
- Unknown fields, operations, paths, and schema majors fail strict validation.
- Commit steps require explicit commit authorization.

---

### Task 1.1: Request And Operation Contracts

**Files:**
- Create: `internal/contract/request.go`
- Create: `internal/contract/request_test.go`
- Create: `schemas/request-v1.json`
- Create: `schemas/operations/project-check-in-v1.json`
- Create: `schemas/operations/session-entry-v1.json`
- Create: `schemas/operations/draft-record-v1.json`
- Create: `schemas/operations/current-work-update-v1.json`

**Interfaces:**
- Produces: `contract.Request`, typed payload structs, `ValidateRequest(Request) error`, and `CanonicalRequestHash(Request) (string, error)`.

**Rollback:** Remove Task 1.1 request types and schemas; the Phase 0 read CLI remains operational.

- [ ] **Step 1: Write failing tests for every required field, unknown fields, RFC 9562 UUID v4/v7 format, expected-hash/create-only exclusivity, canonical HQ target formats, and all four operation payloads.**
- [ ] **Step 2: Run `go test ./internal/contract`; verify failures for absent request types.**
- [ ] **Step 3: Implement strict decoding with `json.Decoder.DisallowUnknownFields` and operation-specific validation.**
- [ ] **Step 4: Add versioned schemas matching the Go field names and constraints exactly.**
- [ ] **Step 5: Run `go test ./internal/contract`; expect exit 0.**
- [ ] **Step 6: If commits are authorized, commit with `feat: define typed write requests`.**

### Task 1.2: Independent Write Policy

**Files:**
- Create: `internal/write/policy.go`
- Create: `internal/write/policy_test.go`
- Create: `config/policy-v1.json`
- Modify: `internal/assets/assets.go`
- Modify: `internal/assets/assets_test.go`

**Interfaces:**
- Produces: `write.Policy.Classify(operation, target string) (PolicyClass, error)`.
- Policy classes: `allowed`, `approval-required`, `submit-only`, `denied`.

**Rollback:** Remove Task 1.2 policy code and source asset and restore the prior embedded-asset set; request validation remains testable without submission.

- [ ] **Step 1: Write a policy matrix test proving request payloads cannot elevate an operation or target.**
- [ ] **Step 2: Include denied tests for `identity/`, `templates/`, `safety-boundaries.md`, `AGENTS.md`, source-system files, arbitrary paths, and unknown operations.**
- [ ] **Step 3: Run `go test ./internal/write`; verify failure for missing policy.**
- [ ] **Step 4: Implement the exact rule format, initial rules, default-deny behavior, and approval semantics in `docs/contracts/policy.md`.**
- [ ] **Step 5: Embed `config/policy-v1.json` and test embedded bytes against the source asset.**
- [ ] **Step 6: Run `go test ./internal/write ./internal/assets`; expect exit 0.**
- [ ] **Step 7: If commits are authorized, commit with `feat: enforce independent write policy`.**

### Task 1.3: Runtime Layout And Durable Request Store

**Files:**
- Create: `internal/fsx/fs.go`
- Create: `internal/fsx/fs_windows.go`
- Create: `internal/fsx/fs_posix.go`
- Create: `internal/write/request_store.go`
- Create: `internal/write/request_store_test.go`

**Interfaces:**
- Produces: `fsx.FS` methods for private directory creation, durable file write, rename, and sync.
- Produces: `RequestStore.Submit(ctx, request) (RequestStatus, error)` and `RequestStore.Status(ctx, id) (RequestStatus, error)`.

**Rollback:** Remove `.hq-interface/` only from disposable fixtures, then remove Task 1.3 code; never remove runtime state from a real HQ automatically.

- [ ] **Step 1: Write failing tests for runtime permissions, restart durability, byte-identical duplicate submission, mismatched duplicate rejection, and missing request status.**
- [ ] **Step 2: Add injectable failures before temporary write, flush, rename, and parent-directory sync.**
- [ ] **Step 3: Run `go test ./internal/write ./internal/fsx`; verify failures.**
- [ ] **Step 4: Make the first valid `submit` lazily create `.hq-interface/requests/{pending,applied,rejected,conflicted,recovery-required}` with private platform permissions; `health` remains read-only and reports whether creation would be possible.**
- [ ] **Step 5: Implement temporary write, file flush, atomic rename, and parent synchronization through `fsx.FS`.**
- [ ] **Step 6: Run tests and prove injected failures leave no partial pending request.**
- [ ] **Step 7: If commits are authorized, commit with `feat: persist durable write requests`.**

### Task 1.4: Submit And Status Commands

**Files:**
- Create: `internal/app/write_commands.go`
- Create: `internal/app/write_commands_test.go`
- Modify: `internal/app/app.go`
- Modify: `cmd/hq/main.go`

**Interfaces:**
- Consumes: request validation, policy, safe resolver, and request store.
- Produces: documented `submit` and `status` command behavior.

**Rollback:** Remove Task 1.4 command wiring and delete runtime state only from disposable test fixtures; retain Phase 0 commands.

- [ ] **Step 1: Write failing CLI tests for valid submission, arbitrary readable request-file input, invalid JSON, denied target, approval-required submission without evidence, duplicate ID, status by ID, Phase 1 health checks, and authoritative-HQ snapshot equality.**
- [ ] **Step 2: Run `go test ./internal/app`; verify expected failures.**
- [ ] **Step 3: Implement command parsing and result mapping without adding `apply`.**
- [ ] **Step 4: Run `go test ./...`, race tests, vet, and formatting checks.**
- [ ] **Step 5: Build and smoke-test one submission against a copied fixture; verify only `.hq-interface/` changes.**
- [ ] **Step 6: Record evidence and mark Phase 1 verified after native platform jobs pass.**
- [ ] **Step 7: If commits are authorized, commit with `feat: add submit and status commands`.**

## Phase Acceptance

- [ ] All four typed operations validate through strict schemas.
- [ ] Policy cannot be elevated by request content.
- [ ] Duplicate IDs are idempotent only for byte-identical requests.
- [ ] Failure injection leaves no partial pending request.
- [ ] Authoritative HQ fixture content remains byte-identical after submit and status.
- [ ] Native Windows, Linux, and macOS jobs pass runtime-layout and request-store tests.
