# Source Record: Original HQ External Interface Design Prompt

The prompt below is preserved verbatim from its original HQ location. It was written before cross-platform support was added. Repository-owned architecture, requirements, and contracts supersede its Windows-only assumptions.

---

# HQ External Interface — Design Prompt

You are designing a bidirectional data exchange interface for **HQ**, a structured markdown-based knowledge workspace. External projects, agent sessions, and automated processes need two capabilities:

1. **Read from HQ** — retrieve information about ongoing work, decisions, project state, people, references, and records.
2. **Write to HQ** — submit status updates, new records, session entries, or project check-ins.

## Background

HQ lives at a known filesystem path on a Windows machine. Its structure is entirely markdown files organized into these folders:

| Folder | Contents |
|---|---|
| `projects/<name>/` | Per-project `README.md` and `STATE.md` |
| `decisions/` | Decision records |
| `work-types/<type>/` | Recurring operational records |
| `references/` | Source-system notes and source-map |
| `identity/` | Role, preferences, review standards |
| `people/` | Contact context |
| `inbox/` | Unclassified raw material |
| `playbooks/` | Process playbooks |
| `templates/` | Approved templates |

Key files at the root: `CURRENT-WORK.md` (active/warm work tracking), `AGENTS.md` (workspace instructions), `SESSION-LOG.md` (topic-switch log), `safety-boundaries.md` (action constraints).

External consumers include: other agent sessions (OpenCode, ChatGPT, etc.), CI/CD pipelines, automated scripts, and other repositories. HQ is not a running service — it is a passive file tree read and written by agents and by Ryann directly.

## Task

Propose the best architecture for this bidirectional interface. Your output must cover:

### 1. Architecture options (2–3, ranked by fit)
For each option, describe:
- The interface mechanism (filesystem, git, lightweight API, MCP tool, agent-to-agent protocol, etc.)
- How reads work (one-shot queries, subscriptions, patterns)
- How writes work (validation, conflict resolution, provenance)
- What each option assumes about the environment

### 2. Recommended option with full specification
For your top pick:
- **Interface boundary and contract** — what the external caller sees
- **Access mechanism** — protocol, transport, call pattern
- **Auth/authorization model** (if any)
- **Read capabilities** — query patterns, search, filtering, lookup by path, semantic or full-text
- **Write rules** — validation, required fields, immutability or versioning conventions
- **Error handling** — what happens on conflict, missing target, invalid format, permission denial
- **Change notification** — how external consumers learn about new or changed HQ content
- **Minimum implementation steps** — ordered, with evidence of completion at each step
- **Implementation rollback plan**

### 3. Trade-off table
Compare all options on: implementation effort, security surface, consumer adoption effort, query expressiveness, write integrity, maintainability, and ability to evolve.

## Success criteria

You are done when:

1. You have produced 2–3 distinct architecture options with clear rationale for each.
2. The recommended option includes a full specification covering all required fields (interface boundary, access mechanism, read/write rules, error handling, rollback, phased implementation).
3. Every option is evaluated against the same trade-off dimensions.
4. Each assumption is explicitly labeled and the risk of it being wrong is stated.
5. The recommendation is self-consistent — reads, writes, errors, and notifications form a coherent system.
6. You cannot identify a plausible requirement that your recommended option fails to address.

If any of these cannot be met, state the gap and why you stopped.

## Constraints

- **No running server or daemon** unless you can demonstrate a requirement that filesystem-only mechanisms cannot meet.
- Prefer filesystem-native mechanisms where sufficient (file reads, git operations, file watchers, symlinks, named pipes, git hooks).
- The interface must not require changes to HQ's internal file structure or conventions unless the migration cost is explicitly justified.
- Must support Windows natively (NTFS paths, PowerShell, Windows git).

## Validation

Before returning your final document, check:

1. **Feasibility** — can each step in the implementation roadmap be executed on a standard Windows 11 machine without undisclosed prerequisites?
2. **Traceability** — does every component in the recommended option have a clear mapping to a real filesystem or OS construct?
3. **Integrity** — does the write path prevent data loss on concurrent or interrupted writes?
4. **Rollback** — can every implementation step be undone without data loss?
5. **Question check** — for any assumption you are unsure about, did you flag it as an open question instead of guessing?

If a check fails, fix the issue or document it as an open question with the impact.

## Output format

Return a single markdown document with these sections:

1. **Executive summary** (< 200 words)
2. **Architecture options** (with rationale per option)
3. **Recommended option — full specification**
4. **Implementation roadmap** (phased, with completion checks)
5. **Trade-off comparison table**
6. **Open questions and assumptions**

Note any assumptions you make about the environment (e.g., git availability, filesystem access permissions, concurrent write handling, network topology).
