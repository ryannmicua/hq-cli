# Global `hq-io` Skill Delivery Contract

## Purpose

The global skill helps agent sessions recognize HQ-related work and safely compose `hq` commands. It is not required by human users or scripts that already know the CLI contract.

## Triggers

Load `hq-io` when an agent needs to:

- start or resume work using HQ context;
- inspect current work, project state, decisions, people, references, or records;
- submit a project check-in, session entry, draft record, or current-work update;
- apply an explicitly authorized request;
- inspect request status, conflicts, receipts, or changes.
- inspect and escalate a recovery-required request, or perform an explicitly approved restore.

## Required Workflow

1. Run `hq health` when environment state is unknown.
2. Use `hq context` or a narrower read command.
3. Cite source paths and hashes when reporting important state.
4. Use a typed request for writes; never translate the workflow into arbitrary direct writes.
5. Stop on denied paths, approval-required results, conflicts, and recovery-required results.
6. Use `recover inspect` before recommending any recovery action; never run `recover restore` without explicit approval.
7. Report request ID, state, receipt, and mutation status.

## Placement Boundary

The skill owns triggers, procedural steps, examples, approval reminders, error guidance, and evidence reporting. The CLI owns contracts, markdown projections, validation, path safety, policy, locking, transactions, and receipts.

## Installation Requirements

- Canonical source lives in `skills/hq-io/`.
- `HQ_SKILL_ROOT` is the explicit override.
- Without an override, installers prefer `$HOME/.agents/skills` on Linux/macOS and `%USERPROFILE%\.agents\skills` on Windows.
- If the preferred root is absent but an OpenCode skill root exists, installers use `$XDG_CONFIG_HOME/opencode/skills`, `$HOME/.config/opencode/skills`, or `%USERPROFILE%\.config\opencode\skills` as applicable.
- If no supported root exists, installers create the preferred `.agents/skills` path under the current user's home.
- Copy installation is always available; links are optional.
- Installation and removal are idempotent and checksum-aware.
- Modified user files are never overwritten or removed automatically.

## Evaluation Scenarios

- Session bootstrap returns only relevant active project context.
- Read answer cites path and hash.
- Check-in becomes a pending typed request.
- Missing approval stops before apply.
- Hash conflict triggers reread guidance.
- Recovery-required response escalates instead of retrying blindly.
