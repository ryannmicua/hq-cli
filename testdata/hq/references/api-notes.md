# API Design Notes

## Result Envelope

Every command emits one JSON result object to stdout with schemaVersion, command, success, timestamp, data, warnings, error, and mutation fields. Diagnostics go to stderr.

## Exit Codes

Exit codes follow the convention: 0 for success, 2 for invalid argument, 3 for not found, 5 for path denied, 70 for internal error.
