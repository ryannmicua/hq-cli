---
date: "2026-07-29"
title: "Go range loop variable shadowing prevents skipping iterations"
tags: [go, range, iteration, loop]
module: internal/write/render_current_work.go
problem_type: bug
---

## Problem

In Go, `for i, line := range lines` creates copies of `i` and `line` for each iteration. Setting `i = value` inside the loop body does NOT affect the next iteration's index — the range loop always advances to the next element.

## Symptoms

When trying to skip an entry's old fields in `render_current_work.go`, setting `i = entryEndIdx - 1; continue` had no effect. The old lines were still written to output because the range loop assigned the next sequential index (7, 8, 9...) regardless of the `i` mutation.

## Solution

Replace `for i, line := range lines` with a C-style `for i := 0; i < len(lines); i++` loop. This gives real control over the loop variable:

```go
// Wrong — i is a copy, range advances independently
for i, line := range lines {
    if i == entryStart {
        i = entryEnd  // no effect
        continue
    }
}

// Correct — real variable control
for i := 0; i < len(lines); i++ {
    if i == entryStart {
        writeNewEntry()
        i = entryEnd - 1
        continue
    }
    writeLine(lines[i])
}
```

## Why This Works

The `range` keyword in Go copies the iteration variables before each loop body execution. Modifying `i` inside the body only changes the local copy — the range mechanism uses its own internal counter. A C-style `for` loop gives the programmer direct control over the counter variable.

## Prevention

When you need to conditionally skip multiple elements within a `range` over a slice, prefer C-style `for` with explicit index control. Use `for range` only when iterating every element unconditionally.
