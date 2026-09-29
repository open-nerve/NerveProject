package domain

import "strings"

// DisplayNameFromEmail is the display name a new account gets: the part of
// the address before its first @, as Plane's User.save() does
// (plane/apps/api/plane/db/models/user.py:169-187). It is never empty for a
// valid address.
func DisplayNameFromEmail(email string) string {
	name, _, _ := strings.Cut(email, "@")
	return name
}
