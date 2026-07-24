# HQ CLI Requirements

## Goal

External agents and scripts can safely retrieve HQ context and propose or apply narrowly defined HQ updates through one cross-platform command contract.

## Actors

- **Reader:** may run side-effect-free queries.
- **Submitter:** may create validated pending requests.
- **Operator:** may apply requests under HQ policy and approval rules.
- **Human owner:** approves protected actions and resolves policy or recovery decisions.

One operating-system identity may perform multiple roles, but role separation is not treated as hard isolation unless the host enforces it.

## MVP Use Cases

| ID | Use Case | Required Commands | Acceptance Result |
|---|---|---|---|
| UC-01 | Start an agent session with relevant HQ context | `context` | Returns current work, selected project state and README, recent session entries, and operating-rule paths |
| UC-02 | Retrieve one known record | `get` | Returns content, source path, hash, and optional Git commit |
| UC-03 | Enumerate work and records | `list` | Returns filtered projects, decisions, work types, people, references, or current work |
| UC-04 | Find text in approved HQ roots | `search` | Returns deterministic matches with path and line evidence |
| UC-05 | Submit a project check-in | `submit`, `status` | Creates one idempotent typed request without changing project files |
| UC-06 | Submit session continuity | `submit`, `status` | Queues a validated session entry with caller and purpose provenance |
| UC-07 | Submit a draft record | `submit`, `status` | Queues a create-only draft under an allowed destination |
| UC-08 | Submit a current-work update | `submit`, `status` | Queues a structured update without arbitrary markdown replacement |
| UC-09 | Apply an authorized request | `apply` | Performs one conflict-safe atomic update and emits a receipt |
| UC-10 | Recover from concurrent editing | `get`, `submit`, `apply` | Returns `HQ_VERSION_CONFLICT` with no mutation; caller can reread and resubmit |
| UC-11 | Resume change processing | `changes` | Returns every receipt after a durable cursor in order |
| UC-12 | Audit an interaction | all | JSON results identify source hashes, request IDs, receipts, and mutation status |
| UC-13 | Inspect and restore an uncertain write | `recover` | Read-only inspection identifies safe action; approved restore verifies backup and emits a recovery receipt |

## Functional Requirements

- FR-01: Every command emits exactly one JSON result object to stdout.
- FR-02: Result, request, receipt, and error contracts are versioned.
- FR-03: Reads never create or modify HQ or interface files.
- FR-04: Caller target paths are relative logical identifiers or allowlisted relative paths.
- FR-05: Canonical path checks reject traversal, links, reparse points, and platform path hazards that escape HQ.
- FR-06: Full-text search is deterministic and requires no persistent index.
- FR-07: Write requests use explicit typed operations; no arbitrary write operation exists.
- FR-08: Request IDs are idempotent and immutable.
- FR-09: Updates require an expected SHA-256 target hash; creates require create-only semantics.
- FR-10: Apply revalidates request, policy, approval, target, and hash under an exclusive lock.
- FR-11: The original target remains recoverable until replacement and receipt verification finish.
- FR-12: Every apply result declares `noMutation`, `rolledBack`, or `recoveryRequired`.
- FR-13: Receipts are immutable and assigned monotonic change cursors.
- FR-14: Git metadata is read-only; the CLI never stages, commits, resets, checks out, cleans, pushes, or creates refs.
- FR-15: `context` does not load unrelated project content.
- FR-16: The global `hq-io` skill calls the CLI and never reimplements core mechanics.
- FR-17: Versioned schemas and default policy are embedded in the standalone executable and verified against source assets.
- FR-18: Recovery-required requests can be inspected without mutation and restored only with approval and a verified backup.

## Quality Requirements

- QR-01: Supported binaries run without a language runtime on Windows, Linux, and macOS.
- QR-02: `go test ./...` passes on each supported operating system.
- QR-03: Filesystem contract tests pass on NTFS, ext4, and APFS before release.
- QR-04: Network and unsupported filesystem writes fail closed.
- QR-05: No credential or secret is written to source, fixtures, logs, requests, receipts, or backups.
- QR-06: A fresh agent can identify current work and verification evidence from `README.md`, `AGENTS.md`, and `implementation/STATUS.md`.
- QR-07: Every implementation task has an exact verification command, expected result, evidence, and rollback.

## Non-Goals

- Remote transport or network authentication.
- Semantic or embedding search.
- Automatic markdown merges.
- A daemon or subscription service.
- Hard authorization between processes sharing one operating-system account.
- Automatic Git commits or publication.

## Definition Of Done

- All MVP use cases pass their contract and end-to-end tests on supported platforms.
- Failure injection demonstrates no silent partial mutation.
- Install, upgrade, disablement, backup restoration, and skill removal are rehearsed.
- Release artifacts and checksums are reproducible from documented commands.
- The status ledger contains fresh evidence for every verified task and phase.
