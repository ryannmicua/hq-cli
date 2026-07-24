# Phase 3: Skill And Release Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Deliver the global `hq-io` agent workflow, cross-platform installation, native CI evidence, and checksummed release artifacts with rehearsed rollback.

**Architecture:** The skill is a procedural adapter over the compiled CLI. Installers detect supported skill roots and user executable locations, copy managed artifacts idempotently, verify checksums, and preserve modified user files. Release automation builds six targets and blocks advertisement without native evidence.

**Tech Stack:** Markdown skill definition, Go CLI, PowerShell installer for Windows, POSIX shell installer for Linux/macOS, Go release commands, GitHub Actions-compatible workflows.

## Global Constraints

- Phase 2 acceptance must be verified.
- The skill may not contain parsing, validation, locking, mutation, or receipt implementation.
- Installation, release publication, and remote workflow execution require explicit approval.
- Commit steps require explicit commit authorization.

---

### Task 3.1: Canonical `hq-io` Skill

**Files:**
- Create: `skills/hq-io/SKILL.md`
- Create: `skills/hq-io/references/commands.md`
- Create: `skills/hq-io/references/errors.md`
- Create: `skills/hq-io/references/evidence.md`
- Create: `skills/hq-io/evals/scenarios.md`
- Create: `internal/skill/validate.go`
- Create: `internal/skill/validate_test.go`

**Interfaces:**
- Produces: globally discoverable `hq-io` skill using the contract in `docs/operations/agent-skill.md`.

**Rollback:** Remove only the canonical skill source and validation package before installation; do not remove any globally installed skill in this task.

- [ ] **Step 1: Write evaluation scenarios for context bootstrap, evidence-backed lookup, typed submission, approval stop, conflict stop, and recovery escalation.**
- [ ] **Step 2: Author a concise `SKILL.md` with triggers, required workflow, safety gates, and references to command details.**
- [ ] **Step 3: Put command tables, errors, and evidence formats in reference files; do not duplicate CLI mechanics.**
- [ ] **Step 4: Run `go test ./internal/skill -run TestSkillPackage -v`; expect all required frontmatter, trigger, workflow, reference, and evaluation checks to pass.**
- [ ] **Step 5: Run each scenario in a clean agent session against a disposable fixture; compare before/after hashes outside `.hq-interface/` and require zero authoritative changes for reads and submits, then verify apply changes only the receipt-declared target.**
- [ ] **Step 6: If commits are authorized, commit with `feat: add global hq-io skill`.**

### Task 3.2: Idempotent Cross-Platform Installers

**Files:**
- Create: `scripts/install.ps1`
- Create: `scripts/install.sh`
- Create: `scripts/uninstall.ps1`
- Create: `scripts/uninstall.sh`
- Create: `scripts/install_test.go`
- Create: `docs/operations/install.md`

**Interfaces:**
- Install `hq` into a user-writable executable directory and `skills/hq-io/` into a detected global skill root.
- Return nonzero without overwrite or removal when an installed managed file's checksum differs.

**Rollback:** Run the checksum-aware uninstallers only against isolated test profiles; remove Task 3.2 files without touching real user installations.

- [ ] **Step 1: Write installer integration tests using isolated temporary HOME and profile directories.**
- [ ] **Step 2: Test fresh install, repeated install, managed upgrade, modified-file refusal, checksum verification, and safe uninstall.**
- [ ] **Step 3: Implement PowerShell and POSIX installers with equivalent observable results and no elevation requirement.**
- [ ] **Step 4: Run tests on Windows, Linux, and macOS; record detected paths and checksums.**
- [ ] **Step 5: Rehearse uninstall and prove user-modified files remain untouched.**
- [ ] **Step 6: If commits are authorized, commit with `feat: install hq and hq-io safely`.**

### Task 3.3: Extend Native CI And Add Release Matrix

**Files:**
- Modify: `.github/workflows/ci.yml`
- Create: `.github/workflows/release.yml`
- Create: `scripts/build-release.go`
- Create: `scripts/smoke-release.go`
- Modify: `docs/operations/ci.md`

**Interfaces:**
- CI runs tests, race tests where supported, vet, formatting, schema checks, fixture smoke tests, and filesystem contract tests.
- Release builds six named artifacts and `checksums.txt`.

**Rollback:** Restore the Phase 0 CI workflow, remove release-only workflow and scripts, and retain local test commands.

- [ ] **Step 1: Add a matrix definition for native Windows, Linux, and macOS tests and separate cross-build checks for all architectures.**
- [ ] **Step 2: Add artifact naming and embedded version metadata matching `docs/operations/release-and-rollback.md`.**
- [ ] **Step 3: Run `go run github.com/rhysd/actionlint/cmd/actionlint@v1.7.7`; expect no workflow diagnostics.**
- [ ] **Step 4: Run local release build logic and verify six artifacts, checksums, and `hq version` metadata.**
- [ ] **Step 5: Do not trigger remote workflows or publish artifacts without explicit approval.**
- [ ] **Step 6: If commits are authorized, commit with `ci: verify cross-platform hq releases`.**

### Task 3.4: Final Acceptance And Rollback Rehearsal

**Files:**
- Modify: `README.md`
- Modify: `implementation/STATUS.md`
- Modify: `docs/operations/release-and-rollback.md`
- Create: `implementation/ACCEPTANCE.md`

**Interfaces:**
- Produces: one evidence index mapping requirements, threats, tests, platforms, artifacts, and rollback results.

**Rollback:** Revert documentation-only Task 3.4 changes with a targeted patch; retain all test and release evidence for review.

- [ ] **Step 1: Execute every standard test and build command from `docs/testing/strategy.md` on approved runners.**
- [ ] **Step 2: Execute all MVP use cases against disposable fixtures and record command/result evidence.**
- [ ] **Step 3: Rehearse install, upgrade, disablement, skill removal, transaction recovery, and release rollback.**
- [ ] **Step 4: Create `implementation/ACCEPTANCE.md` mapping each requirement and threat to fresh evidence.**
- [ ] **Step 5: Update README installation and usage only with commands exercised successfully.**
- [ ] **Step 6: Mark the project delivery verified only if no high-risk finding or required platform remains unresolved.**
- [ ] **Step 7: If commits are authorized, commit with `docs: record hq-cli delivery evidence`.**

## Phase Acceptance

- [ ] Clean agents discover and correctly use `hq-io` on supported platforms.
- [ ] Installers are idempotent and preserve modified user files.
- [ ] Native CI evidence exists for advertised operating systems and filesystems.
- [ ] Six release artifacts have matching checksums and version metadata.
- [ ] Installation, transaction recovery, disablement, and release rollback are rehearsed.
- [ ] `implementation/ACCEPTANCE.md` maps every MVP requirement to evidence.
