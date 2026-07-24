# Delivery Traceability

## Use Cases

| Requirement | Primary Plan Task | Release Evidence |
|---|---|---|
| UC-01 session context | Phase 0 Task 0.4 | `hq context` end-to-end fixture result and unchanged snapshot |
| UC-02 record retrieval | Phase 0 Tasks 0.3-0.4 | `get` contract and fixture tests |
| UC-03 enumeration | Phase 0 Tasks 0.3-0.4 | deterministic `list` tests |
| UC-04 full-text search | Phase 0 Task 0.4 | ordered path-and-line search tests |
| UC-05 project check-in submission | Phase 1 Tasks 1.1-1.4 | typed request and pending status evidence |
| UC-06 session continuity submission | Phase 1 Tasks 1.1-1.4 | session-entry schema and pending status evidence |
| UC-07 draft record submission | Phase 1 Tasks 1.1-1.4 | create-only request evidence |
| UC-08 current-work submission | Phase 1 Tasks 1.1-1.4 | structured request evidence |
| UC-09 authorized apply | Phase 2 Tasks 2.1-2.4 | target hash and immutable receipt |
| UC-10 conflict recovery | Phase 2 Tasks 2.2-2.4 | two-writer conflict with no mutation |
| UC-11 change resume | Phase 2 Task 2.4 | restart and cursor pagination evidence |
| UC-12 audit evidence | Phases 0-3 | source hashes, request IDs, receipts, and acceptance index |
| UC-13 recovery inspection and restore | Phase 2 Tasks 2.3-2.4 | inspect result, restored hash, and recovery receipt |

## Functional Requirements

| Requirements | Plan Coverage |
|---|---|
| FR-01 through FR-02 | Phase 0 Task 0.1 |
| FR-03 through FR-06 | Phase 0 Tasks 0.2-0.4 |
| FR-07 through FR-08 | Phase 1 Tasks 1.1-1.4 |
| FR-09 through FR-12 | Phase 2 Tasks 2.1-2.4 |
| FR-13 | Phase 2 Tasks 2.3-2.4 |
| FR-14 | Global constraints and command integration tests in every phase |
| FR-15 | Phase 0 Task 0.4 |
| FR-16 | Phase 3 Task 3.1 |
| FR-17 | Phase 0 Task 0.1 and Phase 1 Task 1.2 |
| FR-18 | Phase 2 Tasks 2.3-2.4 |

## Quality Requirements

| Requirement | Plan Coverage |
|---|---|
| QR-01 native standalone binaries | Phase 3 Task 3.3 |
| QR-02 cross-platform Go tests | Every phase acceptance gate |
| QR-03 NTFS, ext4, and APFS evidence | Phase 2 Task 2.2 and Phase 3 Task 3.3 |
| QR-04 unsupported filesystem failure | Phase 2 Task 2.2 |
| QR-05 no credentials or secrets | Global constraints, threat model, and Phase 3 acceptance scan |
| QR-06 agent continuity | Repository scaffold and Phase 3 Task 3.4 |
| QR-07 task evidence and rollback | Every phase plan and `implementation/STATUS.md` |

## Threat Coverage

Threat controls in `docs/security/threat-model.md` map to Phase 0 Task 0.2, Phase 1 Tasks 1.1-1.3, Phase 2 Tasks 2.1-2.4, and Phase 3 Tasks 3.2-3.4. Final evidence is indexed in `implementation/ACCEPTANCE.md` before delivery can be marked verified.
