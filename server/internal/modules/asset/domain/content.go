package domain

// IsOrdinaryAttachment identifies files that bind on publication to an existing
// content host. It does not grant access or imply that bytes are published.
func (a Asset) IsOrdinaryAttachment() bool {
	if a.Placement.Validate(a.Purpose) != nil {
		return false
	}
	switch a.Placement.Scope.Kind {
	case ScopeIssue:
		return a.Purpose == IssueAttachment
	case ScopeComment:
		return a.Purpose == CommentAttachment
	case ScopeDraft:
		return a.Purpose == DraftIssueAttachment
	default:
		return false
	}
}

// IsBoundContent makes a file eligible for its owner's authorization. Profile
// images and new-comment preparations never gain this eligibility.
func (a Asset) IsBoundContent() bool {
	if a.BoundAt == nil || a.Placement.Validate(a.Purpose) != nil {
		return false
	}
	switch a.Placement.Scope.Kind {
	case ScopeIssue, ScopeComment, ScopeDraft:
		return true
	default:
		return false
	}
}
