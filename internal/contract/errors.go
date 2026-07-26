package contract

// Stable error codes for the HQ CLI. Each code maps to a specific exit
// code and indicates the category of failure.
const (
	CodeInvalidArgument  = "HQ_INVALID_ARGUMENT"
	CodeNotFound         = "HQ_NOT_FOUND"
	CodeInvalidRequest   = "HQ_INVALID_REQUEST"
	CodePathDenied       = "HQ_PATH_DENIED"
	CodeApprovalRequired = "HQ_APPROVAL_REQUIRED"
	CodePermissionDenied = "HQ_PERMISSION_DENIED"
	CodeVersionConflict  = "HQ_VERSION_CONFLICT"
	CodeLockTimeout      = "HQ_LOCK_TIMEOUT"
	CodeUnsupportedFS    = "HQ_UNSUPPORTED_FILESYSTEM"
	CodeWriteInterrupted = "HQ_WRITE_INTERRUPTED"
	CodePolicyDenied     = "HQ_POLICY_DENIED"
	CodeInternalError    = "HQ_INTERNAL_ERROR"
)

// ErrDetail constructs an ErrorDetail with the given code, message, and
// retryable flag. The details map is optional.
func ErrDetail(code, message string, retryable bool, details map[string]any) *ErrorDetail {
	return &ErrorDetail{
		Code:      code,
		Message:   message,
		Details:   details,
		Retryable: retryable,
	}
}

// ExitCode returns the Unix-style exit code for a given error code string.
// Unknown or empty codes map to exit 1.
func ExitCode(code string) int {
	switch code {
	case CodeInvalidArgument:
		return 2
	case CodeNotFound:
		return 3
	case CodeInvalidRequest:
		return 4
	case CodePathDenied:
		return 5
	case CodeApprovalRequired:
		return 6
	case CodePermissionDenied:
		return 7
	case CodeVersionConflict:
		return 8
	case CodeLockTimeout:
		return 9
	case CodeUnsupportedFS:
		return 10
	case CodeWriteInterrupted:
		return 11
	case CodePolicyDenied:
		return 12
	case CodeInternalError:
		return 70
	default:
		return 1
	}
}
