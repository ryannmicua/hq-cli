// Package contract defines the public result envelope, error detail, and
// mutation state types for the HQ CLI. Every command emits exactly one
// Result value serialized to stdout as versioned JSON.
package contract

import (
	"encoding/json"
	"io"
	"time"
)

// MutationState describes whether a command changed HQ state.
type MutationState string

const (
	MutationNoMutation       MutationState = "noMutation"
	MutationApplied          MutationState = "applied"
	MutationRolledBack       MutationState = "rolledBack"
	MutationRecoveryRequired MutationState = "recoveryRequired"
)

// Warning is a non-fatal diagnostic message attached to a Result.
type Warning struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// ErrorDetail describes a command error with a stable error code.
type ErrorDetail struct {
	Code      string         `json:"code"`
	Message   string         `json:"message"`
	Details   map[string]any `json:"details,omitempty"`
	Retryable bool           `json:"retryable"`
}

// Result is the versioned envelope for every HQ CLI command output.
type Result struct {
	SchemaVersion string        `json:"schemaVersion"`
	Command       string        `json:"command"`
	Success       bool          `json:"success"`
	Timestamp     string        `json:"timestamp"`
	Data          any           `json:"data"`
	Warnings      []Warning     `json:"warnings"`
	Error         *ErrorDetail  `json:"error"`
	Mutation      MutationState `json:"mutation"`
}

// NewSuccess creates a successful Result for the given command and data
// payload. The mutation is always "noMutation" for read commands.
func NewSuccess(command string, data any) Result {
	return Result{
		SchemaVersion: "1.0",
		Command:       command,
		Success:       true,
		Timestamp:     time.Now().UTC().Format(time.RFC3339),
		Data:          data,
		Warnings:      []Warning{},
		Error:         nil,
		Mutation:      MutationNoMutation,
	}
}

// NewError creates a failed Result carrying the given error detail.
func NewError(command string, errDetail *ErrorDetail) Result {
	return Result{
		SchemaVersion: "1.0",
		Command:       command,
		Success:       false,
		Timestamp:     time.Now().UTC().Format(time.RFC3339),
		Data:          nil,
		Warnings:      []Warning{},
		Error:         errDetail,
		Mutation:      MutationNoMutation,
	}
}

// WriteJSON serializes the Result as indented JSON to the given writer.
// It returns any marshal error.
func WriteJSON(w io.Writer, r Result) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "")
	enc.SetEscapeHTML(false)
	return enc.Encode(r)
}
