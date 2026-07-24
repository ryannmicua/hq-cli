# Command Result Data Contracts

## Error Detail

```json
{
  "code": "HQ_NOT_FOUND",
  "message": "project was not found",
  "details": {
    "collection": "projects",
    "id": "example"
  },
  "retryable": false
}
```

`details` contains only documented command-specific scalar fields. It never contains stack traces, credentials, complete private records, or arbitrary Go error text. `warnings` is an array of `{ "code": string, "message": string }`.

## `version` Data

```json
{"version":"0.1.0","commit":"abcdef0","buildTime":"2026-07-24T00:00:00Z","goVersion":"go1.x","os":"linux","arch":"amd64"}
```

## `health` Data

```json
{
  "overall": "pass",
  "checks": [
    {"name":"hq-root","status":"pass","message":"root is readable"},
    {"name":"git","status":"warn","message":"Git metadata unavailable"}
  ]
}
```

Check status is `pass`, `warn`, `fail`, or `not-applicable`. Phase 0 implements binary, HQ root, Git, contract assets, and read-path checks. Phase 1 adds runtime-layout checks after submit exists. Phase 2 adds lock, flush, replace, and filesystem-support checks. Missing later-phase capability checks are absent, not reported as passing.

## Record

```json
{
  "collection":"projects",
  "id":"example",
  "path":"projects/example/STATE.md",
  "content":"# Example State\n",
  "sha256":"hex",
  "gitCommit":"abcdef0",
  "metadata":{"name":"Example","recordType":"project-state"}
}
```

`gitCommit` is `null` when unavailable. Collection IDs follow `hq-markdown.md`.

## `context` Data

```json
{
  "selectedWork":"HQ CLI",
  "currentWork":{"path":"CURRENT-WORK.md","sha256":"hex","content":"..."},
  "project":{"slug":"hq-cli","state":{"path":"projects/hq-cli/STATE.md","sha256":"hex","content":"..."},"readme":{"path":"projects/hq-cli/README.md","sha256":"hex","content":"..."}},
  "sessionEntries":[{"timestamp":"2026-07-24 08:03","tags":["hq-cli"],"summary":"..."}],
  "operatingRules":["AGENTS.md","safety-boundaries.md"]
}
```

## `get` Data

`data` is one Record.

## `list` Data

```json
{"collection":"projects","items":[{"id":"example","path":"projects/example/STATE.md","metadata":{"name":"Example"}}],"count":1}
```

Allowed filters:

- all collections: `id=<literal-prefix>`;
- projects: `name=<case-insensitive-literal>`;
- current-work: `section=active|warm`;
- dated records: `date-from=YYYY-MM-DD`, `date-to=YYYY-MM-DD`.

Multiple filters are ANDed. Unknown fields or malformed values return `HQ_INVALID_ARGUMENT`.

## `search` Data

```json
{"query":"blocker","matches":[{"path":"projects/example/STATE.md","line":12,"column":4,"text":"No blocker."}],"count":1,"truncated":false}
```

Search is literal UTF-8 substring matching, case-insensitive by default and case-sensitive with `--case-sensitive`. It does not accept regular expressions. Results sort by normalized path, line, then column. `text` is the complete matched line with line endings removed.

## `submit` Data

```json
{"requestId":"uuid","state":"pending","requestSha256":"hex","statusCommand":"hq status --request-id uuid"}
```

## `apply` Data

```json
{"requestId":"uuid","state":"applied","receipt":{"receiptId":"uuid","cursor":1,"target":"projects/example/STATE.md","beforeSha256":"hex","afterSha256":"hex","backupPath":".hq-interface/backups/uuid.md","approvalReference":"approved in session","mutation":"applied"}}
```

Reapplying an already-applied request returns the original receipt with success and warning `HQ_ALREADY_APPLIED`. It performs no new mutation.

## `status` Data

```json
{"requestId":"uuid","state":"pending","requestSha256":"hex","receipt":null,"recovery":null}
```

States are `pending`, `applied`, `rejected`, `conflicted`, or `recovery-required`. A safely rolled-back interrupted apply becomes `rejected` with a receipt whose mutation is `rolledBack`. An uncertain apply becomes `recovery-required` and includes paths and observed hashes in `recovery`.

## `changes` Data

```json
{"items":[{"cursor":1,"requestId":"uuid","state":"applied","target":"projects/example/STATE.md","afterSha256":"hex","timestamp":"2026-07-24T00:00:00Z"}],"nextCursor":1,"hasMore":false}
```

## `recover inspect` Data

```json
{"requestId":"uuid","state":"recovery-required","target":{"path":"...","sha256":"hex"},"backup":{"path":"...","sha256":"hex"},"temporary":{"path":"...","sha256":"hex"},"recommendedAction":"restore-backup"}
```

## `recover restore` Data

Returns the request ID, final `rejected` state, restored target hash, and an immutable recovery receipt. Recovery receipts consume normal change cursors.
