# HQ CLI

Cross-platform command-line interface for safe, structured access to Ryann Micua's HQ markdown workspace.

## Intended Outcome

Provide one standalone `hq` executable for Windows, Linux, and macOS that external agents and scripts can use to:

- load current HQ and project context;
- retrieve, list, and search approved HQ records;
- submit typed project check-ins, session entries, draft records, and current-work updates;
- apply approved requests with conflict detection and atomic writes;
- inspect request status, receipts, and changes after a cursor.

The global `hq-io` skill will guide agent workflows. The CLI remains the only implementation of parsing, validation, locking, mutation, and receipt behavior.

## Current Status

Delivery documents and executor plans are present and awaiting final review. No CLI behavior, dependencies, CI workflow, installer, or release artifact has been implemented.

## Scope

- Standalone Go CLI.
- Native support for Windows, Linux, and macOS.
- Local filesystem access in the initial release.
- Versioned JSON command contracts.
- Typed, approval-aware write operations.
- Platform-specific filesystem adapters behind one integrity contract.
- Executor-checkable plans, status, tests, and evidence.

## Non-Goals For The Initial Release

- A resident server or daemon.
- Remote network access.
- Arbitrary unrestricted file writes.
- Semantic search.
- Replacing Git or HQ's markdown conventions.
- Treating same-account approval as cryptographic isolation.

## Repository Map

- `AGENTS.md` - project rules for agents and contributors.
- `docs/README.md` - canonical delivery-document index.
- `implementation/STATUS.md` - current delivery status and verification evidence.
- `.gitignore` - build and local runtime exclusions.

Source, schemas, tests, installers, CI, and release automation will be added only through the approved phase plans.

## Source Context

- HQ tracking project: `C:\Users\rmicua\hq\projects\hq-cli\`
- Canonical architecture: `docs/architecture.md`
- Canonical requirements: `docs/requirements.md`
- Master plan: `docs/superpowers/plans/2026-07-24-hq-cli-master.md`
- Preserved original prompt: `docs/source/hq-interface-design-prompt.md`
- HQ safety rules: `C:\Users\rmicua\hq\safety-boundaries.md`

Repository-owned documents are canonical for delivery. The local absolute paths identify HQ tracking and source provenance only.
