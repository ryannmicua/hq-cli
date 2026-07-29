# Recovery Command

## `hq recover inspect`

Inspect a request's state and determine whether recovery is possible:

```
hq recover inspect --request-id <uuid>
```

**Required evidence:**
- request-id must match a known request
- The target file status is checked (exists or missing)
- Backup availability is checked from the receipt
- Intent file presence is checked for crash-recovery state

**Refusal cases:**
- Unknown request-id returns `HQ_NOT_FOUND`

## `hq recover restore`

Restore a target file from a verified backup:

```
hq recover restore --request-id <uuid>
```

The approval reference must be provided via `HQ_APPROVAL_REFERENCE` environment variable or `--approval-reference` flag.

**Required evidence:**
- A receipt must exist for the given request-id
- A backup file must exist at the path recorded in the receipt
- The backup hash must match the receipt's backup hash
- The current target hash must match the receipt's target hash (verify no modification since apply)
- The approval reference must match the original receipt's approval reference

**Refusal cases:**
- Missing backup returns `HQ_NOT_FOUND`
- Hash mismatch returns `HQ_VERSION_CONFLICT`
- Missing or non-matching approval reference returns `HQ_APPROVAL_REQUIRED`
- The target has been modified since the original apply

## Security Notes

- Approval references should be provided via `HQ_APPROVAL_REFERENCE` environment variable or stdin, not as CLI arguments visible to process listing.
- Passing `--approval-reference` on the command line is deprecated and will warn on stderr.
- Backups contain pre-apply content which may include credentials or secrets that were removed by the applied change. Restore may reintroduce previously-removed secrets.
- Backup files use owner-only permissions (0600 POSIX, equivalent Windows DACL) and should be treated as sensitive.
