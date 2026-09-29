package shared

import "time"

// ValidTimezone reports whether time.LoadLocation loads name (M2 design
// 4.2), except "" and "Local", which it takes for UTC and for the host's
// zone. Accounts, workspaces and projects share it (M3 design 3.13).
//
// It is the one rule here that reads outside the process: LoadLocation reads
// the host's zone files first, and the zone database the nerve binary embeds
// (archtest's TestNerveBinaryEmbedsTheTimeZoneDatabase) only for a name the
// host has no file for. So one host may accept a name that another refuses:
// "asia/shanghai" on a case-insensitive file system, "posixrules" where the
// host has that file (spec P3a 3 item 10). Within one process the answer
// for a name does not change (M3 design 11.4).
func ValidTimezone(name string) bool {
	if name == "" || name == "Local" {
		return false
	}
	_, err := time.LoadLocation(name)
	return err == nil
}
