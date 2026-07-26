// Package assets embeds versioned schemas and default policy into the
// standalone executable. Embed parity is verified by the corresponding
// test file.
package assets

import _ "embed"

// ResultSchemaV1 is the embedded bytes of schemas/result-v1.json.
//
//go:embed schemas/result-v1.json
var ResultSchemaV1 []byte

// PolicyV1 is the embedded bytes of the default write policy.
//
//go:embed policy-v1.json
var PolicyV1 []byte
