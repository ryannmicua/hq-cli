# HQ CLI Master Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Deliver a standalone cross-platform `hq` CLI and global `hq-io` skill that implement every MVP use case in `docs/requirements.md` without data loss or policy bypass.

**Architecture:** A Go CLI exposes a versioned JSON contract. Shared use cases depend on safe HQ record adapters and an explicit filesystem interface; Windows and POSIX adapters supply platform integrity behavior. Writes are typed requests applied through a lock, hash recheck, backup, atomic replacement, and immutable receipt.

**Tech Stack:** Go, Go standard library, versioned JSON Schema documents, native Windows and POSIX filesystem APIs, Go tests, GitHub Actions-compatible cross-platform automation.

## Global Constraints

- Support Windows, Linux, and macOS on `amd64` and `arm64`; advertise only targets with native smoke evidence.
- Validate filesystem integrity on NTFS, ext4, and APFS; fail closed on network and unsupported filesystems.
- Emit exactly one JSON result to stdout and diagnostics to stderr.
- Provide only typed write operations: `project-check-in`, `session-entry`, `draft-record`, and `current-work-update`.
- Never stage, commit, reset, checkout, clean, push, or modify Git refs from the CLI.
- Never store credentials or secrets.
- Preserve existing HQ markdown conventions; add only `.hq-interface/` runtime state.
- Update `implementation/STATUS.md` immediately as tasks start, block, and verify.
- Commit steps in phase plans require explicit commit authorization for that implementation run.

---

## Plan Order

| Order | Plan | Deliverable | Entry Gate | Exit Evidence |
|---:|---|---|---|---|
| 1 | `2026-07-24-phase-0-contracts-and-reads.md` | Compiled read-only CLI with `version`, `health`, `context`, `get`, `list`, and `search` | Delivery documents approved | Cross-platform read tests pass; filesystem snapshot unchanged |
| 2 | `2026-07-24-phase-1-submit-and-status.md` | Durable typed request queue and `status` | Phase 0 verified | Failure-injection tests leave no partial pending request |
| 3 | `2026-07-24-phase-2-apply-and-changes.md` | Transactional apply, receipts, recovery, and cursor polling | Phase 1 verified | Concurrency and interruption tests prove no silent data loss |
| 4 | `2026-07-24-phase-3-skill-and-release.md` | Global `hq-io` skill, installers, CI matrix, and release artifacts | Phase 2 verified | Clean-agent and native-platform release checks pass |

Do not begin a later plan while a required earlier phase remains `blocked` or lacks its phase acceptance evidence.

## Shared File Map

| Path | Responsibility |
|---|---|
| `cmd/hq/main.go` | Process entry point and dependency assembly |
| `internal/app/` | Command dispatch and use-case coordination |
| `internal/contract/` | Public JSON types, errors, and exit mapping |
| `internal/config/` | Root and policy configuration |
| `internal/assets/` | Embedded schemas and default policy |
| `internal/fsx/` | Cross-platform filesystem contract and adapters |
| `internal/hq/` | Safe paths and HQ markdown projections |
| `internal/read/` | Read-only command services |
| `internal/write/` | Request, transaction, receipt, and change services |
| `schemas/` | Versioned machine-readable contracts |
| `config/policy-v1.json` | Default-deny embedded write policy |
| `testdata/hq/` | Disposable HQ fixtures only |
| `skills/hq-io/` | Canonical global skill source |
| `scripts/` | Install, smoke, and release verification helpers |
| `.github/workflows/ci.yml` | Native cross-platform checks, bootstrapped in Phase 0 |
| `docs/operations/ci.md` | CI commands, runners, and evidence rules |
| `implementation/STATUS.md` | Live state and evidence |

## Master Acceptance Checklist

- [ ] Every requirement in `docs/requirements.md` maps to a verified task.
- [ ] Every command in `docs/contracts/cli.md` has contract, success, and failure tests.
- [ ] Every typed operation has strict schema, renderer, policy, and end-to-end tests.
- [ ] Every threat in `docs/security/threat-model.md` has current evidence.
- [ ] NTFS, ext4, and APFS filesystem suites pass on native runners.
- [ ] Failure injection covers every boundary in `docs/testing/strategy.md`.
- [ ] `hq-io` clean-agent evaluations pass without direct HQ writes.
- [ ] Release and rollback rehearsals match `docs/operations/release-and-rollback.md`.
- [ ] `implementation/STATUS.md` records exact commands and observed results.

## Execution Handoff

Start with `docs/superpowers/plans/2026-07-24-phase-0-contracts-and-reads.md`. Before the first code edit, set that phase and its first task to `in-progress` in `implementation/STATUS.md`. If commits are not explicitly authorized, skip commit steps and retain the verified working tree for review.
