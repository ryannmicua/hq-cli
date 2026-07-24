# Release And Rollback

## Release Inputs

- Verified source revision.
- Passing unit, race, integration, filesystem-contract, and agent-skill checks.
- Native evidence for every advertised platform.
- Versioned schemas and matching documentation.
- No unresolved high-risk security finding.

## Artifacts

Produce these archives and SHA-256 checksums:

- `hq_<version>_windows_amd64.zip`
- `hq_<version>_windows_arm64.zip`
- `hq_<version>_linux_amd64.tar.gz`
- `hq_<version>_linux_arm64.tar.gz`
- `hq_<version>_darwin_amd64.tar.gz`
- `hq_<version>_darwin_arm64.tar.gz`

Each executable reports its version, commit, operating system, and architecture through `hq version` before public release. Adding `version` to the implemented CLI is a release requirement even though it is not an HQ data command.

## Installation

- Install the executable into a user-writable directory already on `PATH`, or configure the global skill with an explicit absolute path.
- Install `skills/hq-io/` by copying into a detected global skill root.
- Refuse to overwrite a modified installed skill unless the user approves.
- Verify installed executable and skill checksums.

## Upgrade

1. Run `hq health` with the current version.
2. Back up policy and unexpired runtime metadata.
3. Verify the new artifact checksum.
4. Replace the executable without changing HQ content.
5. Run `hq version`, `hq health`, and read-only smoke tests.
6. Run schema compatibility checks before applying pending requests.

## Disablement

Remove the executable from agent discovery and uninstall the global skill only when its checksum matches the installed managed copy. Leave HQ markdown untouched. Preserve pending requests, receipts, and backups until a human reviews retention needs.

## Transaction Recovery

When a result is `recoveryRequired`:

1. Stop applies for the affected target.
2. Inspect the request, target, backup, temporary file, and recorded hashes.
3. Compare the target with the verified backup and intended rendered output.
4. Inspect with `hq recover inspect --request-id <uuid>`.
5. After human review and explicit approval, restore with `hq recover restore --request-id <uuid> --approval-reference <text>`.
6. Verify the restored hash and recovery receipt returned by the command.
7. Resume applies only after the target lock and request state are reconciled.

## Release Rollback

1. Disable the current executable and skill.
2. Restore the prior checksummed executable and matching skill version.
3. Run `hq version`, `hq health`, and read-only smoke tests.
4. Do not apply requests created under an incompatible schema version.
5. Preserve audit evidence and document the rollback in the status ledger.

No automated rollback may run Git reset, checkout, clean, or modify unrelated working-tree content.
