## Residual Review Findings

### Applied Fixes (step 5)

- **P0** `internal/app/write_commands.go:54` — Strict JSON decoding not enforced on submit path. Fixed: replaced `json.Unmarshal` with `json.NewDecoder` + `DisallowUnknownFields`. [Commit 4610ac9]
- **P0** `internal/write/policy.go:57` — Policy `**` glob only matches single-depth paths. Fixed: rewrote `matchPattern` for recursive multi-depth `**` matching. [Commit 4610ac9]
- **P1** `internal/assets/policy-v1.json:8` — Embedded policy missing `draft-record` operation rule. Fixed: added rule to both `config/policy-v1.json` and `internal/assets/policy-v1.json`. [Commit 4610ac9]
- **P2** `internal/assets/assets_test.go:44` — Missing cross-verification between config/ and internal/assets/ policy files. Fixed: added `TestPolicyV1_MatchesConfigSource`. [Commit 4610ac9]

### Deferred to Tracker (confidence 75, not auto-applied)

- **P1** `internal/app/write_commands.go:128` — isErrorCode prefix match causes wrong error routing. Filed: https://github.com/ryannmicua/hq-cli/issues/6
- **P1** `internal/write/request_store.go:179` — Lost visibility: findRequest returns empty string on not-found. Filed: https://github.com/ryannmicua/hq-cli/issues/7
- **P2** `internal/write/request_store.go:152` — Status() validates via synthetic dummy request. Filed: https://github.com/ryannmicua/hq-cli/issues/8
- **P2** `internal/app/write_commands.go:39` — No file size limit before os.ReadFile in submit handler. Filed: https://github.com/ryannmicua/hq-cli/issues/9
- **P2** `internal/app/write_commands.go:60` — Duplicate error-code dispatch chains in both command handlers. Filed: https://github.com/ryannmicua/hq-cli/issues/10

### Not Actionable (advisory / human owner)

- **P1** `internal/write/request_store.go:117` — TOCTOU race in duplicate request detection. Owner: human. Not auto-filed.

### Run Context

- **Review run ID:** `20260726-180101-lRrsxj`
- **Artifact path:** `C:\Users\rmicua\AppData\Local\Temp\compound-engineering/ce-code-review/20260726-180101-lRrsxj`
- **Branch:** `feat/phase-1-submit-and-status`
- **Head commit:** `4610ac9051c6e79b40eaad896ec03b1594cd9b32`
