## Residual Review Findings

### Filed

- **P2** `internal/app/read_commands.go:197-203` — Missing internal/read/context.go per plan → https://github.com/ryannmicua/hq-cli/issues/1
- **P3** `internal/hq/records.go:348-354` — Hardcoded work-type ID suffix in determineID → https://github.com/ryannmicua/hq-cli/issues/2
- **P3** `internal/hq/records_test.go:192-198` — Duplicated module-root walk helper in three test files → https://github.com/ryannmicua/hq-cli/issues/3
- **P3** `internal/app/app_test.go:270-278` — Snapshot test does not run all read commands → https://github.com/ryannmicua/hq-cli/issues/4

### Applied (included for completeness)

- **P1** `internal/app/read_commands.go:180` — Case-sensitive default violates R7 contract (applied)
- **P2** `internal/app/read_commands.go:192` — Empty search query returns CodeInternalError (applied)
- **P2** `internal/read/service_test.go:133` — Search case-sensitive test missing assertion (applied)
- **P2** `internal/read/search.go:200` — Search returns absolute paths in match results (applied)

### Source

- Run: LFG pipeline Phase 4 code review
- Branch: eager-hedgehog
- Base commit: 443b76e
- Plan: docs/plans/lfg-plan.md
