package outbound

import "errors"

// ErrBlocked identifies an address refused by the outbound policy.
var ErrBlocked = errors.New("outbound address blocked")
