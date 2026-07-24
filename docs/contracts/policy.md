# Write Policy Contract

## Asset

Canonical policy source is `config/policy-v1.json`. It is embedded into the executable with schemas by `internal/assets` using `//go:embed`. The executable reports embedded policy and schema versions through `health`; installed binaries do not depend on adjacent asset files.

## Rule Shape

```json
{
  "schemaVersion": "1.0",
  "rules": [
    {
      "operation": "project-check-in",
      "targetPattern": "projects/*/STATE.md",
      "class": "allowed"
    }
  ],
  "defaultClass": "denied"
}
```

Patterns match normalized relative path components; `*` matches one component and never crosses `/`. First exact operation and target match wins. Duplicate or overlapping rules fail asset validation. The default is always `denied`.

## Initial Rules

| Operation | Target | Class |
|---|---|---|
| `project-check-in` | `projects/*/STATE.md` | `allowed` |
| `session-entry` | `SESSION-LOG.md` | `allowed` |
| `draft-record` | allowed create-only paths in `hq-markdown.md` | `allowed` |
| `current-work-update` | `CURRENT-WORK.md` | `allowed` |
| any operation | `identity/**`, `templates/**`, `decisions/**`, `people/**`, `references/**`, `work-types/**`, `AGENTS.md`, `safety-boundaries.md` | `denied` |
| unknown operation or target | any | `denied` |

Host policy may make an allowed rule `approval-required` or `submit-only`, but may never broaden a denied target. The first release has no host policy that changes defaults; adding one requires a separate approved contract.

## Approval Semantics

- `submit` accepts an `approval-required` request without approval evidence so it can enter review.
- `apply` enforces approval requirements.
- If the request and command both provide approval references, they must be byte-identical; otherwise apply returns `HQ_INVALID_ARGUMENT`.
- If only one provides a reference, that value is effective and is recorded in the receipt.
- `submit-only` apply returns `HQ_POLICY_DENIED` with exit 12 and no mutation.
- `denied` requests return `HQ_POLICY_DENIED` with exit 12 at submission and are rechecked at apply.
- Same-account evidence is procedural and is not represented as a signature.
