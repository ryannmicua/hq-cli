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
