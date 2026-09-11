package common

import "errors"

// ErrNotFound is shared: every repository reports a missing row the same way,
// so a use case can react to it without knowing which table was empty.
var ErrNotFound = errors.New("not found")
