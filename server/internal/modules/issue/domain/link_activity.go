package domain

// LinkCreatedActivity records the URL and new row identity in separate slots.
func LinkCreatedActivity(link Link) Activity {
	return Activity{Field: ptr("link"), Verb: VerbCreated, NewValue: ptr(link.URL), NewIdentifier: ptr(link.ID), Comment: "created a link"}
}

// LinkUpdatedActivity records only a URL change, preserving both row identities.
func LinkUpdatedActivity(old, current Link) Activity {
	return Activity{Field: ptr("link"), Verb: VerbUpdated, OldValue: ptr(old.URL), NewValue: ptr(current.URL), OldIdentifier: ptr(old.ID), NewIdentifier: ptr(current.ID), Comment: "updated a link"}
}

// LinkDeletedActivity clears the value with empty text, without identifiers.
func LinkDeletedActivity(link Link) Activity {
	return Activity{Field: ptr("link"), Verb: VerbDeleted, OldValue: ptr(link.URL), NewValue: ptr(""), Comment: "deleted the link"}
}
