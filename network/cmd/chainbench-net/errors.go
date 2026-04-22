package main

import (
	"errors"
	"fmt"

	"github.com/0xmhha/chainbench/network/internal/types"
)

// APIError is a typed error carrying a wire-level result error code plus a
// user-facing message. Handlers return this when a specific code is meaningful;
// any other error is treated as INTERNAL by the dispatcher.
type APIError struct {
	Code    types.ResultErrorCode
	Message string
	Cause   error
}

func (e *APIError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (%v)", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *APIError) Unwrap() error { return e.Cause }

// Constructors for the common codes.

func NewInvalidArgs(message string) *APIError {
	return &APIError{Code: types.ResultErrorCode("INVALID_ARGS"), Message: message}
}

func NewNotSupported(message string) *APIError {
	return &APIError{Code: types.ResultErrorCode("NOT_SUPPORTED"), Message: message}
}

func NewUpstream(message string, cause error) *APIError {
	return &APIError{Code: types.ResultErrorCode("UPSTREAM_ERROR"), Message: message, Cause: cause}
}

func NewProtocolError(message string, cause error) *APIError {
	return &APIError{Code: types.ResultErrorCode("PROTOCOL_ERROR"), Message: message, Cause: cause}
}

func NewInternal(message string, cause error) *APIError {
	return &APIError{Code: types.ResultErrorCode("INTERNAL"), Message: message, Cause: cause}
}

// exitCode maps an error (possibly nil) to an OS exit code per VISION §5.
func exitCode(err error) int {
	if err == nil {
		return 0
	}
	var api *APIError
	if errors.As(err, &api) {
		switch string(api.Code) {
		case "NOT_SUPPORTED":
			return 2
		case "PROTOCOL_ERROR":
			return 3
		case "INVALID_ARGS", "UPSTREAM_ERROR", "INTERNAL":
			return 1
		}
	}
	return 1
}
