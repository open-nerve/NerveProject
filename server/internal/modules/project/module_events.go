package project

// EventTypes declares deletion events; the request publisher is wired separately.
func EventTypes() []string { return []string{"project.deleted"} }
