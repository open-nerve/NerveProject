// Package clock supplies business time. Each module declares its own Clock
// port (Now() time.Time) in its app layer; System satisfies it by structure.
package clock

import "time"

// System reads the system clock. Its instants are in UTC and truncated to
// microseconds, the precision of PostgreSQL's timestamptz, so a time a use
// case writes reads back from the database unchanged (M2 design 3.13).
type System struct{}

// Now returns the current instant.
func (System) Now() time.Time {
	return time.Now().UTC().Truncate(time.Microsecond)
}

// Offset shifts the business clock while retaining the database time contract.
type Offset struct {
	Base   interface{ Now() time.Time }
	Offset time.Duration
}

// Now returns the shifted business instant.
func (c Offset) Now() time.Time { return c.Base.Now().Add(c.Offset).UTC().Truncate(time.Microsecond) }
