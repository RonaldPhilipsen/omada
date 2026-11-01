package models

import (
	"errors"
	"fmt"
)

// RequestError is returned when the Omada controller answers a request with a
// non-zero errorCode. Code holds the controller error code, Msg the message.
type RequestError struct {
	Code int
	Msg  string
}

func (e *RequestError) Error() string {
	return fmt.Sprintf("Omada controller responded '%s' (%d)", e.Msg, e.Code)
}

// Is matches ErrLoginFailed for controller error code -30109 so callers can
// use errors.Is with the sentinel.
func (e *RequestError) Is(target error) bool {
	return target == ErrLoginFailed && e.Code == -30109
}

// ValidationError is returned when a caller supplies invalid input to a method.
type ValidationError struct {
	Msg string
}

func (e *ValidationError) Error() string {
	return e.Msg
}

var (
	// ErrLoginFailed is returned when the controller rejects the credentials
	// or reports a missing/expired session (controller code -30109).
	ErrLoginFailed = &RequestError{Code: -30109}
	// ErrLoginSessionClosed is returned when the session cookie is no longer
	// accepted, typically because the cookie jar dropped it.
	ErrLoginSessionClosed = errors.New("login session closed")
	// ErrUnsupportedControllerVersion is returned when the controller is older
	// than the minimum supported version.
	ErrUnsupportedControllerVersion = errors.New("unsupported Omada controller version")
	// ErrConnectionFailed wraps network-level failures reaching the controller.
	ErrConnectionFailed = errors.New("connection to Omada controller failed")
	// ErrBadControllerURL wraps failures resolving the controller URL.
	ErrBadControllerURL = errors.New("controller URL could not be resolved")
)
