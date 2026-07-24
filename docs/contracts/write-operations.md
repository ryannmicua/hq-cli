# Typed Write Operations

## Request Envelope

```json
{
  "schemaVersion": "1.0",
  "requestId": "018f0000-0000-7000-8000-000000000001",
  "caller": {
    "name": "agent-session",
    "sessionId": "optional-session-id"
  },
  "purpose": "Record the verified project check-in",
  "operation": "project-check-in",
  "target": "projects/example/STATE.md",
  "payload": {},
  "submittedAt": "2026-07-24T00:00:00Z",
  "expectedTargetHash": "sha256-hex",
  "createOnly": false,
  "approvalReference": null
}
```

Exactly one of `expectedTargetHash` or `createOnly: true` is required.

`requestId` is a lowercase canonical RFC 9562 UUID in 8-4-4-4-12 form. Versions 4 and 7 are accepted. `submit --request` may read any local request file accessible to the caller; only the validated target and payload influence HQ access.

## `project-check-in`

Updates an existing project's resumable state using structured fields:

```json
{
  "summary": "Short executive summary",
  "currentOutcome": "The outcome currently being pursued",
  "currentState": "Evidence-based current state",
  "nextAction": "One concrete next action",
  "blockers": [],
  "risks": [],
  "evidence": ["path or source reference"],
  "verifiedAt": "2026-07-24"
}
```

The operation may update only a project `STATE.md`. It cannot modify project scope, safety rules, or official source records.

## `session-entry`

Prepends one validated line to `SESSION-LOG.md`:

```json
{
  "timestamp": "2026-07-24T08:03:00+08:00",
  "tags": ["hq-cli", "planning"],
  "summary": "Prepared canonical delivery documents"
}
```

The renderer follows the newest-first and timezone rules in `hq-markdown.md` and rejects embedded newlines.

## `draft-record`

Creates a new markdown draft under an allowed destination defined in `hq-markdown.md`:

```json
{
  "title": "Draft title",
  "body": "Markdown body",
  "recordDate": "2026-07-24",
  "classification": "inbox"
}
```

The request must use `createOnly: true`. `classification` must match the target path. Existing targets, missing project roots, templates, official records, and source-system files are denied.

## `current-work-update`

Updates one named `CURRENT-WORK.md` entry through structured fields:

```json
{
  "workName": "HQ CLI",
  "workspace": "projects/hq-cli/",
  "loadFirst": "projects/hq-cli/STATE.md",
  "supportingContext": ["projects/hq-cli/README.md"],
  "currentOutcome": "Approved outcome",
  "currentState": "Verified state",
  "nextAction": "Concrete next action",
  "lastTouched": "2026-07-24"
}
```

The renderer preserves unrelated entries and section order. It cannot remove, rename, or archive another entry.

## Policy Classes

| Class | Behavior |
|---|---|
| `allowed` | May be applied when target, hash, and payload validation pass |
| `approval-required` | May be submitted without approval; requires effective non-empty approval evidence before apply |
| `submit-only` | May be queued but cannot be applied by the first-release CLI |
| `denied` | Rejected at submission and apply |

Policy is determined from operation and target rules stored outside the request. Request content cannot elevate its class.

## Retention

- Pending requests remain until applied, rejected, conflicted, or explicitly reviewed.
- Requests and receipts remain immutable for the first release.
- Backups are retained indefinitely in the first release unless a human explicitly reviews and removes them outside the CLI.
- Automatic deletion and retention cleanup commands are excluded until a separate retention policy is approved.
