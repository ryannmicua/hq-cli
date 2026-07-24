# ADR 0002: Use Typed Transactional Filesystem Writes

- Status: Accepted
- Date: 2026-07-24

## Context

HQ remains directly editable by humans and agents. Concurrent or interrupted writes must not silently lose data. A generic file-write API would bypass workspace rules and make validation unreliable.

## Decision

Writes use versioned typed request envelopes followed by a separate apply operation. Apply acquires a target lock, rechecks the expected hash, creates a verified backup, renders a complete replacement, performs a same-filesystem atomic replacement, verifies the new hash, and writes an immutable receipt.

The first release supports only `project-check-in`, `session-entry`, `draft-record`, and `current-work-update`. It has no arbitrary write operation and performs no automatic merge.

## Consequences

- Conflicts fail without mutation and require reread and resubmission.
- Request and receipt state is durable across process restarts.
- Platform adapters must provide equivalent integrity semantics.
- Unsupported and network filesystems fail closed.
- Same-account authorization remains procedural unless host permissions provide distinct identities.
