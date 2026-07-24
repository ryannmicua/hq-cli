# HQ CLI Delivery Documents

## Start Here

1. `architecture.md` - system boundary, components, platform contract, and rollback.
2. `requirements.md` - MVP use cases, functional requirements, and definition of done.
3. `contracts/cli.md` - commands, result envelope, errors, and compatibility.
4. `contracts/results.md` - command data, error, warning, and lifecycle shapes.
5. `contracts/hq-markdown.md` - canonical target document formats and collection mapping.
6. `contracts/write-operations.md` - typed request operations and policy classes.
7. `contracts/policy.md` - embedded policy shape, defaults, and approval semantics.
8. `security/threat-model.md` - assets, threats, controls, and review gates.
9. `testing/strategy.md` - test levels, failure injection, and platform matrix.
10. `operations/release-and-rollback.md` - install, upgrade, recovery, and rollback.
11. `operations/agent-skill.md` - boundary and evaluation contract for `hq-io`.
12. `traceability.md` - requirements mapped to plan tasks and release evidence.
13. `docs/plans/2026-07-24-hq-cli-master.md` - execution sequence and acceptance gates.

## Decisions

- `decisions/0001-standalone-go-cli.md`
- `decisions/0002-transactional-filesystem-writes.md`

## Phase Plans

- `docs/plans/2026-07-24-phase-0-contracts-and-reads.md`
- `docs/plans/2026-07-24-phase-1-submit-and-status.md`
- `docs/plans/2026-07-24-phase-2-apply-and-changes.md`
- `docs/plans/2026-07-24-phase-3-skill-and-release.md`

## Documents Produced During Execution

- `operations/ci.md` - created in Phase 0 and extended in Phase 3.
- `operations/recovery-command.md` - created and exercised in Phase 2.
- `operations/install.md` - created and exercised in Phase 3.
- `implementation/ACCEPTANCE.md` - final requirement and evidence index created in Phase 3.

## Source Brief

- `source/hq-interface-design-prompt.md`

Repository documents are canonical for implementation. HQ tracks project state at `C:\Users\rmicua\hq\projects\hq-cli\`.
