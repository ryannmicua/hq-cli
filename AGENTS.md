# HQ CLI Agent Instructions

## Start Here

1. Read `README.md`.
2. Read `implementation/STATUS.md`.
3. Read `docs/README.md`, `docs/architecture.md`, and `docs/requirements.md`.
4. Read the master plan and the one active phase plan referenced in the status ledger.
5. Update the status ledger as work progresses; do not reconstruct status at the end.

## Delivery Rules

- Keep the `hq` command contract shell-neutral and consistent across Windows, Linux, and macOS.
- Keep platform-specific path, permission, locking, flush, and replacement behavior behind explicit filesystem adapters.
- Fail closed when the filesystem cannot satisfy the documented integrity contract.
- Do not add unrestricted arbitrary-file writes.
- Do not weaken approval, expected-hash, backup, or receipt requirements for convenience.
- Keep the global `hq-io` skill procedural; parsing, validation, locking, and mutation belong in the CLI.
- Use small, independently verifiable tasks from the approved implementation plan.
- Add tests for every behavior change and run the applicable cross-platform checks before marking a task verified.

## Progress Evidence

Each implementation task must record:

- state: `not-started`, `in-progress`, `blocked`, or `verified`;
- files changed;
- exact verification command;
- observed result and evidence;
- rollback action;
- blocker or follow-up, when applicable.

A task is not complete until fresh verification passes. A phase is not complete until every required task and phase acceptance check is verified.

## Safety

- Never store credentials or secrets in source, fixtures, requests, receipts, logs, or backups.
- Never reset, clean, checkout, stage, commit, push, or modify unrelated work unless Ryann explicitly requests it.
- Do not publish releases, modify external systems, or change access controls without explicit approval.
