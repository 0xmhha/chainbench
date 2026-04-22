package main

import (
	"errors"
	"testing"

	"github.com/0xmhha/chainbench/network/internal/types"
)

func TestAPIError_ErrorMessage(t *testing.T) {
	e := &APIError{Code: types.ResultErrorCode("INVALID_ARGS"), Message: "bad name"}
	if e.Error() == "" {
		t.Error("Error() returned empty string")
	}
}

func TestAPIError_Unwrap(t *testing.T) {
	cause := errors.New("underlying")
	e := &APIError{Code: types.ResultErrorCode("UPSTREAM_ERROR"), Message: "upstream failed", Cause: cause}
	if !errors.Is(e, cause) {
		t.Error("errors.Is should find cause")
	}
}

func TestExitCode_NilIsZero(t *testing.T) {
	if got := exitCode(nil); got != 0 {
		t.Errorf("nil: got %d, want 0", got)
	}
}

func TestExitCode_MapsErrorCodes(t *testing.T) {
	cases := map[string]int{
		"NOT_SUPPORTED":  2,
		"PROTOCOL_ERROR": 3,
		"INVALID_ARGS":   1,
		"UPSTREAM_ERROR": 1,
		"INTERNAL":       1,
	}
	for code, want := range cases {
		e := &APIError{Code: types.ResultErrorCode(code), Message: "x"}
		if got := exitCode(e); got != want {
			t.Errorf("code %s: got %d, want %d", code, got, want)
		}
	}
}

func TestExitCode_GenericErrorTreatedAsInternal(t *testing.T) {
	if got := exitCode(errors.New("generic")); got != 1 {
		t.Errorf("generic: got %d, want 1", got)
	}
}

func TestNewInvalidArgs(t *testing.T) {
	e := NewInvalidArgs("bad")
	if string(e.Code) != "INVALID_ARGS" || e.Message != "bad" {
		t.Errorf("got %+v", e)
	}
}

func TestNewUpstream_WrapsCase(t *testing.T) {
	cause := errors.New("x")
	e := NewUpstream("disk gone", cause)
	if string(e.Code) != "UPSTREAM_ERROR" {
		t.Errorf("code: got %q", e.Code)
	}
	if !errors.Is(e, cause) {
		t.Error("should wrap cause")
	}
}
