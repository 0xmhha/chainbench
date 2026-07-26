package probe

import "errors"

// Sentinel errors returned by Detect. Callers (e.g. the chainbench-net handler
// layer) use errors.Is to classify input-validation failures into INVALID_ARGS
// vs upstream/endpoint failures into UPSTREAM_ERROR without fragile substring
// matching on error messages.
var (
	ErrMissingURL      = errors.New("rpc_url required")
	ErrInvalidURL      = errors.New("rpc_url must be http(s)")
	ErrUnknownOverride = errors.New("unknown override")
)

// IsInputError reports whether err is a probe-level input-validation failure
// (ErrMissingURL / ErrInvalidURL / ErrUnknownOverride). Everything else is an
// upstream failure.
func IsInputError(err error) bool {
	return errors.Is(err, ErrMissingURL) ||
		errors.Is(err, ErrInvalidURL) ||
		errors.Is(err, ErrUnknownOverride)
}
