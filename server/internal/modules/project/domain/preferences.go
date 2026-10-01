package domain

// DefaultSortOrder is a project's place in a member's sidebar when he has
// no other place in the workspace's projects:
// project_user_properties.sort_order's default (M3 design 3.18).
const DefaultSortOrder = 65535

// SortOrderFirst is the place in a member's sidebar of a project he is made
// a member of by its creation: before the others, 10000 less than the least
// of his places in the workspace's projects, lowest, or DefaultSortOrder
// when he has none (Plane's ProjectMember.save, db/models/project.py:226-
// 239; M3 design 3.18).
func SortOrderFirst(lowest *float64) float64 {
	if lowest == nil {
		return DefaultSortOrder
	}
	return *lowest - 10000
}
