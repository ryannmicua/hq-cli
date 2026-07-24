# ADR 0001: Use A Standalone Go CLI

- Status: Accepted
- Date: 2026-07-24

## Context

HQ needs one local interface usable by agents and scripts on Windows, Linux, and macOS. The interface must not require a daemon and should avoid runtime-installation assumptions.

## Decision

Implement a shell-neutral `hq` command in Go and publish native binaries for Windows, Linux, and macOS on `amd64` and `arm64`.

## Consequences

- Consumers receive one stable command and JSON contract.
- Platform-specific filesystem behavior is isolated behind Go interfaces and build constraints.
- Release automation must build and checksum six target binaries.
- Locking and atomic-replacement behavior must be tested on real operating-system runners.
- Python, PowerShell, Node.js, and .NET runtimes are not required for normal use.

## Rejected Alternatives

- Git exchange is retained for history but not used as the request protocol.
- MCP may later wrap the CLI but is not the canonical implementation.
- Python and PowerShell were rejected because normal use would require additional runtimes.
- Self-contained .NET was rejected because artifact and packaging size exceed the current need.
