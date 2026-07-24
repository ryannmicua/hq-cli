# Threat Model

## Protected Assets

- HQ markdown content and history.
- Private context contained in requests, receipts, and backups.
- Approval provenance and caller attribution.
- Integrity of the `hq` executable, schemas, and policy.

## Trust Boundaries

- Command arguments and request JSON are untrusted input.
- Paths and links inside HQ may change concurrently.
- The local operating-system identity is trusted only to the extent enforced by the host.
- Git metadata is informative and not an authorization source.
- The global `hq-io` skill is guidance, not a security boundary.

## Threats And Controls

| Threat | Control | Required Evidence |
|---|---|---|
| Path traversal or link escape | Canonicalize after resolving symlinks/reparse points; prove root containment | Platform tests for traversal, symlink, junction, reparse point, and Windows ADS cases |
| Lost update | Required expected hash, lock, and recheck immediately before mutation | Two-writer test where only one apply succeeds |
| Partial or torn write | Same-directory temporary file, durable flush, atomic replacement, post-write hash | Failure injection at every transaction boundary |
| Unsafe recovery restore | Read-only inspection first; verified backup, explicit approval, target lock, transactional restore, and recovery receipt | Recovery tests reject missing approval or hash mismatch and verify restored target |
| Request tampering | Immutable request hash and byte-identical idempotency check | Tampered stored request is rejected |
| Policy elevation in payload | Policy is loaded independently; unknown fields and operations fail strict validation | Request cannot change its policy class |
| Approval spoofing under same account | Explicitly documented procedural boundary; receipt records approval reference | Tests enforce presence but documentation denies hard-isolation claim |
| Sensitive runtime leakage | Restrictive ACLs or POSIX modes; no secrets; minimal content; no automatic publication | Permission checks and fixture secret scan |
| Lock denial of service | Bounded timeout and documented stale-lock inspection | Lock-timeout test with no mutation |
| Unsupported filesystem semantics | Capability check and fail-closed behavior | Network/unknown filesystem returns `HQ_UNSUPPORTED_FILESYSTEM` |
| Malicious markdown content | Treat content as data; no command evaluation or template execution | Fixtures with shell and markup payloads remain inert |
| Supply-chain substitution | Checksummed release artifacts and reproducible version metadata | Checksum verification in release smoke tests |

## Security Non-Claims

- The CLI does not provide hard isolation between processes running as the same user.
- Approval references are audit evidence, not cryptographic signatures.
- The first release does not secure remote transport because it provides none.
- Git history does not prove that a write was authorized.

## Required Review Gates

- Review all changes to path resolution, filesystem adapters, policy, schemas, apply flow, backups, and receipts as security-sensitive.
- Run race, failure-injection, and platform filesystem tests before release.
- Require explicit human approval before changing access controls, retention deletion, or remote transport.
