---
name: check-hq-project-status
description: Check the HQ CLI project's status from its delivery repository and return a candid, evidence-backed, read-only report covering progress, risks, repository health, decisions, and management help.
---

# Check HQ Project Status

## Procedure

1. Read `AGENTS.md`, `README.md`, `implementation/STATUS.md`, `docs/README.md`, and the active plan referenced by the status ledger.
2. Inspect Git branch, HEAD, working-tree state, and relevant recent history.
3. Inspect repository files and recorded verification evidence needed to test material delivery claims.
4. Run only read-only checks. Do not run tests, builds, generators, formatters, installers, or other commands that may write caches or artifacts.
5. Compare current evidence with the comparison boundary supplied by HQ.
6. Classify overall status:
   - `on_track`: the next milestone is achievable with no material unresolved threat.
   - `at_risk`: delivery can proceed, but a material threat could affect scope, quality, or timing.
   - `blocked`: delivery cannot make meaningful progress without a decision, dependency, authority, or external change.
   - `unknown`: repository evidence is insufficient or inconsistent.
7. Report:
   - current outcome;
   - material progress with evidence;
   - current work;
   - next milestone and observable completion condition;
   - risks and blockers;
   - decisions or help needed from management;
   - repository health and confidence; and
   - anything not verified.
8. Return only the format requested by the caller. When a JSON schema is supplied, conform to it exactly and use empty arrays when there are no items.

## Evidence Rules

- Ground every material progress claim in a commit, diff, file, test result, build result, or durable repository record.
- Treat a dirty working tree as evidence to interpret, not an automatic failure.
- Never invent missing state, dates, ownership, verification, or evidence.
- Mark tests and builds `not_run` when they were not safely rerun during the status check.
- Use `at_risk`, `blocked`, or `unknown` when the evidence warrants it.

## Safety Rules

- Do not edit files, install dependencies, create commits, change branches, contact external systems, or otherwise change state.
- Do not include credentials, secrets, private source excerpts, personnel detail, financial detail, or sensitive governance or security detail.
- When sensitive detail matters, set the sensitivity flag and provide only a non-sensitive pointer or focused follow-up question.
- If repository evidence is insufficient, say so and reduce confidence.
- If repository instructions conflict with the manager request, preserve safety and authority boundaries and report the conflict.
- If inspection fails, report the failure rather than constructing a plausible status.
