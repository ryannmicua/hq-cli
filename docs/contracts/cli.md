# CLI Contract

## Invocation

```text
hq [--root <path>] [--output json] <command> [arguments]
```

`--root` overrides `HQ_ROOT`. JSON is the only output format in the first release. Exactly one result object is written to stdout; diagnostics are written to stderr.

## Result Envelope

```json
{
  "schemaVersion": "1.0",
  "command": "get",
  "success": true,
  "timestamp": "2026-07-24T00:00:00Z",
  "data": {},
  "warnings": [],
  "error": null,
  "mutation": "noMutation"
}
```

`mutation` is one of `noMutation`, `rolledBack`, or `recoveryRequired`. Successful read and submit commands return `noMutation` because submit changes interface metadata, not authoritative HQ records.

## Commands

### `version`

```text
hq version
```

Returns semantic version, source commit, build time, Go version, operating system, and architecture. It does not require an HQ root.

### `health`

```text
hq health
```

Checks only capabilities implemented in the current phase. Phase 0 reports executable, embedded read-contract assets, HQ root, Git, and read-path checks. Phase 1 adds lazy runtime-layout capability checks. Phase 2 adds lock, flush, replace, and filesystem-support checks. It never creates runtime state.

### `context`

```text
hq context [--project <slug>] [--session-entries <count>]
```

Returns current work; selected project `STATE.md` and `README.md`; the requested number of newest session entries, default 20; and paths to applicable operating rules. If no project is supplied, the most recent active item is selected. Ambiguous or missing selection returns an error rather than loading multiple projects.

### `get`

```text
hq get --collection <name> --id <identifier>
hq get --path <allowlisted-relative-path>
```

Returns one record with `path`, `content`, `sha256`, `gitCommit`, and collection metadata.

### `list`

```text
hq list --collection <name> [--filter <field=value>] [--limit <n>]
```

Allowed collections are `current-work`, `projects`, `decisions`, `work-types`, `people`, and `references`. Collection-to-path and identifier rules are defined in `hq-markdown.md`; filters and result shapes are defined in `results.md`.

### `search`

```text
hq search --query <text> [--collection <name>] [--path <relative-root>] [--limit <n>]
```

Returns literal path-and-line matches using the semantics in `results.md`. Search does not follow links outside HQ or inspect `.git/` or `.hq-interface/`.

### `submit`

```text
hq submit --request <request.json>
```

Validates and atomically queues one typed request. It returns the request ID, state, request hash, and status lookup command. It never changes an authoritative HQ record.

### `apply`

```text
hq apply --request-id <uuid> [--approval-reference <text>]
```

Applies one pending request after revalidation. An approval reference supplied at apply time is recorded in the receipt but cannot override a denied operation or target.

### `status`

```text
hq status --request-id <uuid>
```

Returns `pending`, `applied`, `rejected`, `conflicted`, or `recovery-required`, plus the immutable request hash, receipt, and recovery evidence when available.

### `changes`

```text
hq changes [--after <uint64>] [--since <RFC3339>] [--limit <n>]
```

Returns ordered receipt summaries and `nextCursor`. Cursor polling is authoritative; watcher integration is outside the CLI contract.

### `recover`

```text
hq recover inspect --request-id <uuid>
hq recover restore --request-id <uuid> --approval-reference <text>
```

`inspect` is read-only and returns target, backup, temporary-file, and observed-hash evidence for a `recovery-required` request. `restore` accepts only a verified backup, acquires the target lock, restores transactionally, verifies the target, emits a recovery receipt, and transitions the request to `rejected`. It returns `HQ_APPROVAL_REQUIRED` without a non-empty approval reference.

## Error Codes

| Code | Exit | Meaning |
|---|---:|---|
| `HQ_INVALID_ARGUMENT` | 2 | Command syntax or argument is invalid |
| `HQ_NOT_FOUND` | 3 | Record, target, or request does not exist |
| `HQ_INVALID_REQUEST` | 4 | Request schema or typed operation validation failed |
| `HQ_PATH_DENIED` | 5 | Target is outside policy or unsafe |
| `HQ_APPROVAL_REQUIRED` | 6 | Required approval evidence is absent |
| `HQ_PERMISSION_DENIED` | 7 | Operating-system or policy permission failed |
| `HQ_VERSION_CONFLICT` | 8 | Expected target state changed |
| `HQ_LOCK_TIMEOUT` | 9 | Target lock was not acquired before timeout |
| `HQ_UNSUPPORTED_FILESYSTEM` | 10 | Filesystem cannot satisfy the integrity contract |
| `HQ_WRITE_INTERRUPTED` | 11 | Write failed and result includes recovery state |
| `HQ_POLICY_DENIED` | 12 | Policy denies submission or application of the operation and target |
| `HQ_INTERNAL_ERROR` | 70 | Unexpected implementation failure |

## Compatibility

Minor contract additions preserve existing fields and meanings. Breaking changes require a new major `schemaVersion`. Unknown major versions fail validation; unknown fields fail strict write schemas and may be ignored only in documented read-response evolution.

Detailed `data`, `error`, `warnings`, ID, filter, search, and lifecycle shapes are normative in `results.md`.
