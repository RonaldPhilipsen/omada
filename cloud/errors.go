package cloud

import "fmt"

// APIError is a northbound API envelope error: a non-zero errorCode in the
// {errorCode, msg, ...} response envelope.
type APIError struct {
	Code int
	Msg  string
}

// Error implements error.
func (e *APIError) Error() string {
	return fmt.Sprintf("omada: API error %d: %s", e.Code, e.Msg)
}

// NewAPIError creates an APIError from an envelope errorCode and msg.
func NewAPIError(code int, msg string) *APIError {
	return &APIError{Code: code, Msg: msg}
}
