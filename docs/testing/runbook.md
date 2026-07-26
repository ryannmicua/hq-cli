# Testing Runbook

### Phase 2 Plan Review Integrity

**Last tested**: 2026-07-26
**Tested by**: OpenCode `ce-doc-review` session

**How to test (step by step)**:
1. Run the whitespace and conflict-marker check from the repository root.
2. Locate the Phase 2 Open Questions heading, dated review subsection, and finding entries.
3. Verify: the search returns 19 lines comprising one section heading, one dated subsection, and 17 findings.

**Safe actions**:
- Both verification commands are read-only.
- The commands do not run builds, tests, generators, or formatters.

**Destructive actions**:
- None.

**Setup / seed data required**:
- Git must be available.
- The Phase 2 plan must be present in the checked-out worktree.

**Cleanup steps**:
- None.

**Verification commands**:
```text
git diff --check
# Expected: no output and exit code 0.

git grep -n -E "^## Deferred / Open Questions$|^### From 2026-07-26 review$|^- \*\*" -- "docs/plans/2026-07-24-phase-2-apply-and-changes.md"
# Expected: 19 lines: one Open Questions heading, one dated subsection, and 17 finding entries.
```

**Gotchas**:
- The standalone `rg` executable is not available on every project shell `PATH`; use `git grep` for this repository check.
