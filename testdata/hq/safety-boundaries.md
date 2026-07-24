# Safety Boundaries

## Read-Only Principle

Read commands must never create or modify files. Every read command is verified by snapshot hashing before and after execution.

## Path Containment

All file access must remain within the HQ root directory. Traversal (`..`), symlinks, junctions, and absolute paths referencing locations outside HQ are rejected.

## No Credential Storage

Never store credentials or secrets in source, fixtures, requests, receipts, logs, or backups.
