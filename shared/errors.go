package shared

import "errors"

// ErrOutOfRange signals an out of range request.
var ErrOutOfRange = errors.New("out of range")
