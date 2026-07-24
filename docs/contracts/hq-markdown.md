# HQ Markdown Contract

## Purpose

This file makes the target HQ document conventions implementation-canonical. Renderers and golden fixtures must follow these shapes exactly. Unknown sections are preserved unless a typed operation explicitly owns them.

## Collection Mapping And Read Allowlist

| CLI Collection | HQ Location | Identifier |
|---|---|---|
| `current-work` | `CURRENT-WORK.md` | fixed ID `current` |
| `projects` | `projects/<slug>/README.md` and `projects/<slug>/STATE.md` | directory slug |
| `decisions` | `decisions/*.md` | filename without `.md` |
| `work-types` | `work-types/<type>/**/*.md` | normalized relative path without `.md` |
| `people` | `people/*.md` | filename without `.md` |
| `references` | `references/**/*.md` | normalized relative path without `.md` |

Allowlisted explicit paths are markdown files under those locations plus `AGENTS.md`, `SESSION-LOG.md`, and `safety-boundaries.md`. `.git/`, `.hq-interface/`, hidden paths, templates, and paths outside HQ are never searchable or retrievable through `--path` in the first release.

## Project `STATE.md`

Canonical section order:

```markdown
# <Project Name> State

## Executive Summary

<summary>

## Current Outcome

<current outcome>

## Current State

<current state>

## Next Action

<next action>

## Open Decisions

- <decision or `None.`>

## Blockers

- <blocker or `None.`>

## Material Risks

- <risk or `None.`>

## Evidence

- <source reference>, verified YYYY-MM-DD.
```

`project-check-in` owns the content of Executive Summary, Current Outcome, Current State, Next Action, Blockers, Material Risks, and Evidence. It preserves Open Decisions and any unknown sections byte-for-byte and in their original positions. It rejects a state file missing any owned section rather than inventing a new format.

## `SESSION-LOG.md`

The file contains a title, explanation, separator, and newest-first entries:

```markdown
# Session Log

Chronological log of topic switches and significant context shifts during HQ work sessions. Newest entries at top. Each entry must include date and time: `YYYY-MM-DD HH:MM`.

---

YYYY-MM-DD HH:MM #tag-one #tag-two summary text
```

`session-entry` prepends exactly one entry after the separator. Input `timestamp` must be RFC 3339 with an explicit offset. Rendering preserves the supplied wall-clock date and time and removes seconds; it does not convert to the executing machine's timezone. Tags match `^[a-z0-9][a-z0-9-]*$`. Summary is one non-empty line without control characters.

## `CURRENT-WORK.md`

The file contains `Active` and `Warm` sections. Each entry uses:

```markdown
### <Work Name>
- Workspace: `<path>`
- Load first: `<path>`
- Supporting context: `<comma-separated paths>`
- Current outcome: <text>
- Current state: <text>
- Next action: <text>
- Last touched: YYYY-MM-DD
```

`current-work-update` matches `Work Name` exactly, updates only that entry's seven fields, and preserves all other entries and section order. It may create a new entry only when the payload includes every field and the target section is explicitly `Active` or `Warm`. It cannot remove, rename, move, archive, or reorder an existing entry.

## Draft Records

Allowed create-only destinations:

- `inbox/YYYY-MM-DD-<slug>.md`
- `projects/<existing-slug>/reports/YYYY-MM-DD-<slug>.md`
- `projects/<existing-slug>/source/YYYY-MM-DD-<slug>.md`

Canonical shape:

```markdown
# <Title>

- Date: YYYY-MM-DD
- Status: Draft
- Source: Submitted through hq-cli request <request-id>

<body>
```

The target path determines the destination. `classification` must agree with it: `inbox`, `project-report`, or `project-source`. Existing files, missing project directories, templates, decisions, people, references, and work-type records are denied.

## Fixture-As-Contract Rule

Phase 0 copies these shapes into `testdata/hq/` and golden expected files. A change to headings, order, or owned-field behavior requires a contract version review, updated golden fixtures, and compatibility assessment before renderer changes.
