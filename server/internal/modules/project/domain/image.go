package domain

import "time"

// ImageURL is an ephemeral response value, never a stored field.
type ImageURL struct {
	URL       string
	ExpiresAt time.Time
}
