package domain

import "uuid"

// AttachmentActivity identifies the file without copying private metadata.
func AttachmentActivity(id uuid.UUID, added bool) Activity {
	if added {
		return Activity{Field: ptr("attachment"), Verb: VerbCreated, NewIdentifier: &id, Comment: "added an attachment"}
	}
	return Activity{Field: ptr("attachment"), Verb: VerbDeleted, OldIdentifier: &id, Comment: "deleted an attachment"}
}
