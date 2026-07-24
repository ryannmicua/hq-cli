# HQ CLI Architecture

## Executive Summary

HQ CLI is a standalone Go executable named `hq`. It gives local agents and scripts a shell-neutral JSON interface to an HQ markdown workspace on Windows, Linux, and macOS. Reads are immediate and side-effect free. Writes use typed requests, explicit policy, expected hashes, target locks, recoverable backups, same-filesystem atomic replacement, and immutable receipts.

The CLI is the canonical mechanics boundary. The global `hq-io` skill teaches agents when and how to invoke it, but does not parse markdown, validate requests, lock files, or mutate HQ. No daemon, network service, language runtime, semantic index, or unrestricted file-write command is part of the first release.

## Architecture Decision

### Selected: Standalone Go CLI

A native `hq` binary is compiled for the supported operating-system and CPU matrix. Shared business logic depends on a filesystem interface. Windows and POSIX adapters implement path containment, permission inspection, locking, durable flush, atomic replacement, and cleanup using platform-appropriate primitives.

This option provides one stable contract to agents, scripts, CI, and future adapters without requiring PowerShell, Python, Node.js, or a resident process.

### Alternatives

| Option | Strength | Reason Not Selected |
|---|---|---|
| Git exchange | Strong provenance and remote collaboration | Too cumbersome for routine local reads and status submissions |
| MCP process | Excellent typed agent tools | Adds process lifecycle and excludes ordinary scripts without another adapter |
| Python CLI | Low initial implementation effort | Requires interpreter and environment management |
| PowerShell 7 | Small change from the original Windows concept | Not installed by default on Linux or macOS and still needs OS adapters |
| Self-contained .NET CLI | Strong platform APIs | Larger artifacts and packaging surface than needed |

## Public Boundary

```text
hq <command> [arguments]
```

Every invocation writes one versioned JSON result to stdout. Diagnostics go to stderr. Exit codes and stable error codes distinguish success, invalid input, missing data, conflicts, authorization failures, lock timeouts, interrupted writes, and internal failures.

Initial commands:

- `version`
- `health`
- `context`
- `get`
- `list`
- `search`
- `submit`
- `apply`
- `status`
- `changes`
- `recover`

The exact syntax and result shapes are defined in `docs/contracts/cli.md`. Typed write operations are defined in `docs/contracts/write-operations.md`.

## Components

| Component | Responsibility |
|---|---|
| CLI dispatcher | Parse commands, invoke one use case, emit JSON and exit status |
| Configuration | Resolve `HQ_ROOT`, policy, and supported runtime settings |
| Embedded assets | Supply versioned schemas and default policy from the executable |
| Safe path resolver | Canonicalize paths and enforce root and collection containment |
| Markdown record adapters | Read known HQ collections and return stable record projections |
| Search service | Full-text search allowlisted roots without creating an index |
| Request validator | Validate envelopes, operations, targets, payloads, hashes, and policy |
| Request store | Persist immutable requests and idempotent state transitions |
| Transaction engine | Lock, recheck, back up, render, replace, verify, receipt, and recover |
| Filesystem adapter | Implement cross-platform path, lock, flush, replace, and permission behavior |
| Receipt store | Persist immutable outcomes and monotonic change cursors |
| `hq-io` skill | Compose CLI operations into safe agent workflows |

## Source Layout

The implementation plan will create this structure:

```text
cmd/hq/                     CLI entry point
internal/app/               command use cases and result mapping
internal/config/            HQ root and policy configuration
internal/assets/            embedded schemas and default policy
internal/contract/          result, request, receipt, and error types
internal/fsx/               shared interface plus `*_windows.go` and `*_posix.go` adapters
internal/hq/                safe paths and markdown collection adapters
internal/read/              context, get, list, and search services
internal/write/             submit, status, apply, and changes services
schemas/                    versioned JSON schemas
testdata/hq/                disposable representative HQ fixtures
skills/hq-io/               canonical global skill source
scripts/                    install and verification helpers
docs/                       canonical delivery documentation
implementation/STATUS.md    live execution status and evidence
```

## Runtime Layout

The only HQ structural addition is `.hq-interface/`:

```text
.hq-interface/
  requests/
    pending/
    applied/
    rejected/
    conflicted/
    recovery-required/
  receipts/
  backups/
  locks/
  temp/
```

The runtime root is on the same local filesystem as the HQ targets. It is excluded from HQ knowledge queries. Temporary files and locks are never committed. Requests, receipts, and backups receive restrictive platform permissions and explicit retention handling.

## Read Flow

1. Resolve the HQ root from `--root` or `HQ_ROOT`.
2. Canonicalize the root and requested logical target.
3. Resolve links or reparse points and prove containment.
4. Confirm the collection or path is allowlisted.
5. Read and project the markdown record.
6. Return content, source path, SHA-256 hash, and Git commit when available.

`context` composes current work, the selected project's `STATE.md` and `README.md`, the latest requested session entries, and operating-rule paths into one result. It does not silently load unrelated projects.

## Write Flow

1. `submit` validates a versioned typed request.
2. A unique temporary request is written, flushed, and renamed into `pending`.
3. A duplicate request ID returns the existing state only when the request is byte-identical.
4. `apply` revalidates the request, policy, target, expected hash, and approval evidence.
5. The transaction engine acquires a target-scoped exclusive lock.
6. It rereads the target and rejects conflicts without mutation.
7. It creates and verifies a recoverable backup.
8. It renders and validates the complete replacement into a same-directory temporary file.
9. The platform adapter flushes and atomically replaces the target.
10. The engine rereads the target, verifies its hash, writes an immutable receipt, transitions the request, and releases the lock.

No automatic content merge is performed. Conflict recovery means reread, reconcile, and submit a new request.

If an interrupted apply cannot prove the target is original or safely restored, the request enters `recovery-required`. `recover inspect` is read-only. `recover restore` requires explicit approval evidence, restores the verified backup transactionally, writes a recovery receipt, and transitions the request to `rejected`.

## Authorization

The first release uses the authenticated operating-system identity and filesystem permissions. Any process running as a user who can execute `hq` and access HQ may invoke read, submit, and apply. Logical capabilities and approval references enforce workflow policy but do not distinguish processes sharing that identity. Protected operations require an approval reference recorded in the request or apply invocation and receipt.

When all callers run as the same operating-system account, this is a procedural policy boundary, not cryptographic isolation. The CLI states this limitation and does not store shared secrets to simulate stronger authentication.

## Filesystem Contract

Supported adapters must provide equivalent observable behavior for:

- canonical path containment after link resolution;
- platform permission inspection;
- target-scoped exclusive locks with bounded timeout;
- durable file and parent-directory flush where supported;
- same-filesystem atomic replacement;
- stale temporary and lock recovery;
- failure reporting that distinguishes `noMutation`, `rolledBack`, and `recoveryRequired`.

The initial verified filesystems are NTFS on Windows, ext4 on Linux, and APFS on macOS. Other local filesystems are unsupported until the contract suite passes. Network and distributed filesystems fail closed in the first release.

## Distribution

The initial release matrix is:

| OS | Architectures |
|---|---|
| Windows | `amd64`, `arm64` |
| Linux | `amd64`, `arm64` |
| macOS | `amd64`, `arm64` |

Release artifacts include native executables, SHA-256 checksums, version metadata, and installation instructions. Correctness tests for locking and replacement run on real Windows, Linux, and macOS runners rather than emulation alone.

Versioned schemas and the default write policy are embedded in the executable. Source copies remain reviewable under `schemas/` and `config/`; build tests prove embedded bytes match them.

## Change Discovery

`hq changes --after <cursor>` is authoritative. Receipts receive monotonic cursors within one HQ instance. Consumers persist the last processed cursor and deduplicate by request ID. Filesystem watchers may reduce latency but never replace cursor polling because watcher events can be lost or duplicated.

## Rollback

The interface does not migrate existing HQ markdown. Disabling it means removing or withholding the executable and global skill. Applied writes can be restored from verified transaction backups or normal Git history after human review. The first release performs no automatic retention deletion.
