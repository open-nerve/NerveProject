package app

import "uuid"

// AccountState is an account's state as another module reads it through
// identity's Accounts, under the FOR SHARE lock the read takes (M3 design
// 6.5): the module that asked decides what a deactivated account may do.
type AccountState struct {
	ID     uuid.UUID
	Email  string
	Active bool
}
