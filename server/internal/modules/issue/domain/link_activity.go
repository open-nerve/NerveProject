package domain

// LinkCreatedActivity records the URL and new row identity in separate slots.
func LinkCreatedActivity(link Link) Activity {
	return Activity{Field: ptr("link"), Verb: VerbCreated, NewValue: ptr(link.URL), NewIdentifier: ptr(link.ID), Comment: "created a link"}
}
