package resource

import (
	"errors"
	"slices"
)

// FieldError names the declared field a validation refusal is about, as the
// keys from the document root (pool, hosts, 1, addr), so an editor can show the
// reason beside that field. Its message is the wrapped refusal unchanged.
type FieldError struct {
	Path []string
	Err  error
}

func (e *FieldError) Error() string { return e.Err.Error() }
func (e *FieldError) Unwrap() error { return e.Err }

func fieldError(err error, path ...string) error {
	if err == nil {
		return nil
	}
	return &FieldError{Path: path, Err: err}
}

// underField places err at prefix: a field error inside it keeps its own field
// below the prefix, and any other refusal belongs to the prefix as a whole.
func underField(err error, prefix ...string) error {
	var inner *FieldError
	if errors.As(err, &inner) {
		return &FieldError{Path: slices.Concat(prefix, inner.Path), Err: err}
	}
	return fieldError(err, prefix...)
}
