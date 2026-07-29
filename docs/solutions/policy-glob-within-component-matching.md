---
date: "2026-07-29"
title: "Policy glob matching must use filepath.Match for within-component patterns"
tags: [go, glob, policy, filepath, matchPattern]
module: internal/write/policy.go
problem_type: bug
---

## Problem

Policy patterns like `inbox/*.md` failed to match targets like `inbox/2026-07-29-test-draft.md`. The existing `matchPattern` function only supported exact `*` (any single path component) and `**` (any depth), but not partial filename patterns like `*.md`.

## Symptoms

Policy classification returned `PolicyDenied` for `draft-record` operations targeting `inbox/*.md` destinations, even though the policy explicitly allowed them.

## Solution

Add a `matchComponent` helper that uses Go's `filepath.Match` for within-component globbing. Fall back to exact comparison when `filepath.Match` is not needed:

```go
func matchComponent(pattern, target string) bool {
    if pattern == target {
        return true
    }
    matched, _ := filepath.Match(pattern, target)
    return matched
}
```

Use it in `matchPattern` alongside the existing exact component comparison:

```go
if patternParts[pi] == "*" || matchComponent(patternParts[pi], targetParts[ti]) {
```

## Why This Works

`filepath.Match` implements standard shell-style glob matching within a single path component. It handles `*.md` → `filename.md`, `prefix-*` → `prefix-suffix`, and similar partial patterns correctly.

## Prevention

When writing glob-based policy matchers, test patterns like `dir/*.ext` (partial filename match within a component), not just `dir/*` (whole component match). Add registry tests for each pattern type to the policy test suite.
