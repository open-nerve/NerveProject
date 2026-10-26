package app

import (
	"cmp"
	"context"
	"slices"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

func (l *Listing) subgroups(ctx context.Context, input ListingInput, rows []ListRecord) (IssueListPage, error) {
	page := IssueListPage{Total: rows[0].Total, Groups: make([]IssueListGroup, 0), SubgroupTotals: make([]ListTotal, 0)}
	lanes := make(map[string]bool)
	var emitted []ListRecord
	var sizes []int
	for start := 0; start < len(rows); {
		row := rows[start]
		if len(page.Groups) == 0 || page.Groups[len(page.Groups)-1].Key != row.GroupKey {
			page.Groups = append(page.Groups, IssueListGroup{Key: row.GroupKey, Total: row.GroupTotal, Subgroups: make([]IssueListGroup, 0)})
		}
		end := start + 1
		for end < len(rows) && rows[end].GroupKey == row.GroupKey && rows[end].SubGroupKey == row.SubGroupKey {
			end++
		}
		part := rows[start:end]
		cursor, err := pageCursor(part, input.Query.Limit)
		if err != nil {
			return IssueListPage{}, err
		}
		group := &page.Groups[len(page.Groups)-1]
		group.Subgroups = append(group.Subgroups, IssueListGroup{Key: row.SubGroupKey, Total: row.CellTotal, NextCursor: cursor})
		if !lanes[row.SubGroupKey] {
			lanes[row.SubGroupKey] = true
			page.SubgroupTotals = append(page.SubgroupTotals, ListTotal{Key: row.SubGroupKey, Total: row.SubgroupTotal})
		}
		size := min(len(part), input.Query.Limit)
		emitted = append(emitted, part[:size]...)
		sizes = append(sizes, size)
		start = end
	}
	items, err := l.issues(ctx, input, emitted)
	if err != nil {
		return IssueListPage{}, err
	}
	for g := range page.Groups {
		for c := range page.Groups[g].Subgroups {
			size := sizes[0]
			sizes = sizes[1:]
			page.Groups[g].Subgroups[c].Data = items[:size:size]
			items = items[size:]
		}
	}
	// The first group may lack an earlier lane, so first-seen order is not
	// the top-level lane order. Keep the same field order as SQL's cell order.
	slices.SortFunc(page.SubgroupTotals, func(a, b ListTotal) int {
		return compareListGroup(input.Query.SubGroupBy, a.Key, b.Key)
	})
	return page, nil
}

func compareListGroup(field domain.ListGroup, a, b string) int {
	if a == b {
		return 0
	}
	if a == "none" {
		return 1
	}
	if b == "none" {
		return -1
	}
	var order []string
	switch field {
	case "priority":
		order = []string{"urgent", "high", "medium", "low"}
	case "state_group":
		order = []string{"backlog", "unstarted", "started", "completed", "cancelled"}
	default:
		return cmp.Compare(a, b)
	}
	return cmp.Compare(slices.Index(order, a), slices.Index(order, b))
}
