package main

import "errors"

// exitError carries a process exit code alongside an error, so the CLI can map a
// session verdict to a CI-meaningful status (F16-O5): 0 all pass, 1 a test
// failed, 2 blocked or an infrastructure error.
type exitError struct {
	code int
	err  error
}

func (e *exitError) Error() string { return e.err.Error() }
func (e *exitError) Unwrap() error { return e.err }

// exitCode returns the exit code carried by err: 0 for nil, the carried code for
// an *exitError, else 1 for any other error.
func exitCode(err error) int {
	if err == nil {
		return 0
	}
	var ee *exitError
	if errors.As(err, &ee) {
		return ee.code
	}
	return 1
}
