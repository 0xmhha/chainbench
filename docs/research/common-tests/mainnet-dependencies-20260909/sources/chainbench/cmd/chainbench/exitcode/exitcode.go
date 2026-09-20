// Package exitcode carries a process exit status from the command that decided
// it to the main function that applies it.
//
// It is its own package because those two are no longer in the same one: a
// suite run decides the status (0 all pass, 1 a test failed, 2 blocked or an
// infrastructure error, F16-O5) while main is what calls os.Exit. Any surface
// package that can reach a CI-meaningful verdict returns one of these.
package exitcode

import "errors"

// Error carries a process exit code alongside an error.
type Error struct {
	Code int
	Err  error
}

func (e *Error) Error() string { return e.Err.Error() }
func (e *Error) Unwrap() error { return e.Err }

// Of returns the exit code an error asks for: 0 for nil, the carried code for
// an *Error, and 1 for anything else, since an unclassified failure is still a
// failure.
func Of(err error) int {
	if err == nil {
		return 0
	}
	var ee *Error
	if errors.As(err, &ee) {
		return ee.Code
	}
	return 1
}
