package errs

import "errors"

var (
	ErrNotFound      = errors.New("not found")
	ErrAlreadyExists = errors.New("object already exists")
)

// types RFC 9457
const (
	ProblemValidation      = "https://example.com/problems/validation-error"
	ProblemNotFound        = "https://example.com/problems/not-found"
	ProblemConflict        = "https://example.com/problems/conflict"
	ProblemInvalidJSONBody = "https://example.com/problems/invalid-json-body"
	ProblemInternal        = "https://example.com/problems/internal-error"
)

type ErrorResponse struct {
	Type   string `json:"type"`
	Title  string `json:"title"`
	Status int    `json:"status"`
	Detail string `json:"detail,omitempty"`
}
