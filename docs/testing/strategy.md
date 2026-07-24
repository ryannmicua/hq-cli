# Test Strategy

## Test Levels

### Unit Tests

- Contract serialization and strict validation.
- CLI argument and exit-code mapping.
- Current-work and project record parsing.
- Typed operation rendering.
- Request idempotency and state transitions.
- Receipt cursor ordering.

### Filesystem Contract Tests

Run the same suite against Windows and POSIX adapters:

- canonical path containment;
- link and reparse-point escape rejection;
- permission inspection;
- exclusive lock and timeout;
- durable temporary writes;
- same-filesystem atomic replacement;
- backup restoration;
- stale artifact handling.

Validated filesystems are NTFS, ext4, and APFS. Network filesystems are expected to fail capability checks.

### Integration Tests

Use disposable `testdata/hq/` fixtures to execute the compiled CLI and validate stdout, stderr, exit codes, filesystem snapshots, requests, state transitions, and receipts.

### End-To-End Use Cases

Every use case in `docs/requirements.md` receives at least one success test and one relevant failure test. `context`, project check-in, session entry, draft record, current-work update, conflict recovery, recovery inspect/restore, change polling, and receipt audit are release-blocking.

### Agent Skill Evaluation

A clean agent session must:

- discover `hq-io`;
- retrieve session context through `hq context`;
- cite source paths and hashes;
- submit typed requests rather than edit HQ directly;
- stop on approval-required and conflict responses;
- report request and receipt evidence.

## Failure Injection Matrix

Inject failure before and after:

1. request temporary-file creation;
2. request flush;
3. request rename;
4. target lock acquisition;
5. expected-hash recheck;
6. backup creation and verification;
7. rendered temporary-file flush;
8. target replacement;
9. post-write hash verification;
10. receipt creation;
11. request state transition.

Every case must result in `noMutation`, `rolledBack`, or `recoveryRequired` with inspectable evidence. No case may report success with an unverified target.

## Platform Matrix

| Platform | Unit and integration | Race tests | Filesystem contract | Build smoke test |
|---|---|---|---|---|
| Windows `amd64` | Required | Required | NTFS required | Required |
| Windows `arm64` | Required through cross-build plus available native smoke runner | As runner permits | Native evidence required before supported release | Required |
| Linux `amd64` | Required | Required | ext4 required | Required |
| Linux `arm64` | Required through cross-build plus available native smoke runner | As runner permits | Native evidence required before supported release | Required |
| macOS `amd64` | Required | Required | APFS required | Required |
| macOS `arm64` | Required | Required | APFS required | Required |

An architecture is not advertised as supported until native smoke and filesystem-contract evidence exists.

## Standard Commands

The implementation must make these commands authoritative:

```text
go test ./...
go test -race ./...
go vet ./...
gofmt -l .
```

Release automation adds platform build and smoke commands. Exact scripts and expected outputs are specified in the phase plans.

## Evidence

Executors record the command, operating system, filesystem, exit result, and relevant artifact path in `implementation/STATUS.md`. Test claims require fresh output from the current change.
