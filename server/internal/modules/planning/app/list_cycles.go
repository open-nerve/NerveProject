package app

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/planning/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// CycleLister lists a project's undeleted cycles: the unarchived ones by
// sort_order, created_at, id, or, when archived, the archived ones alone by
// archived_at DESC, id (M6 P1 spec 2.3).
type CycleLister interface {
	ListCycles(ctx context.Context, project uuid.UUID, archived bool) ([]domain.Cycle, error)
}

// SnapshotCountsReader reads, of the cycles ids, the undeleted frozen ones'
// counts and moments, by cycle, in one read for the page, in the
// transaction ctx carries; a cycle not frozen is not among them (M6 P4 spec
// R5).
type SnapshotCountsReader interface {
	CycleSnapshotCounts(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]domain.FrozenCounts, error)
}

// ListCyclesDeps are ListCycles's ports.
type ListCyclesDeps struct {
	Reads     Reads
	Cycles    CycleLister
	Groups    StateGroupReader
	Stats     IssueStats
	Snapshots SnapshotCountsReader
	Clock     Clock
}

// ListCycles answers listCycles (M6 P1 spec 2.3, R9).
type ListCycles struct{ d ListCyclesDeps }

// NewListCycles returns the use case over d.
func NewListCycles(d ListCyclesDeps) *ListCycles { return &ListCycles{d: d} }

// Execute lists the project's cycles, archived or not as asked, in the
// store's order, each with its status on the project's day at the clock's
// one reading, after the rest, and the numbers of its work items that the
// caller sees, read for the whole page (rows); an empty list for none, which
// reads nothing more.
func (u *ListCycles) Execute(ctx context.Context, projectID uuid.UUID, archived bool) ([]domain.CycleListItem, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return nil, err
	}
	out := []domain.CycleListItem{}
	err = u.d.Reads.run(ctx, actor, domain.ActionCycleRead, named{project: projectID}, func(ctx context.Context, h held) error {
		cycles, err := u.d.Cycles.ListCycles(ctx, projectID, archived)
		if err != nil || len(cycles) == 0 {
			return err
		}
		ids := make([]uuid.UUID, len(cycles))
		for i, c := range cycles {
			ids[i] = c.ID
		}
		p, err := u.rows(ctx, h.projectID, ids, h.viewer(actor))
		if err != nil {
			return err
		}
		today, err := domain.Today(u.d.Clock.Now(), h.project.Timezone)
		if err != nil {
			return err
		}
		for _, c := range cycles {
			row := domain.CycleListItem{ReadCycle: domain.ReadCycle{Cycle: c, Status: domain.StatusOf(c.Dates, today)}, Live: p.live[c.ID],
				AssigneeIDs: p.assignees[c.ID]}
			if f, ok := p.frozen[c.ID]; ok {
				row.Snapshot = &f
			}
			out = append(out, row)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// cycleRows are the numbers of a page of cycles' rows, by cycle: the
// counts, the assignees and the frozen counts.
type cycleRows struct {
	live      map[uuid.UUID]domain.Counts
	assignees map[uuid.UUID][]uuid.UUID
	frozen    map[uuid.UUID]domain.FrozenCounts
}

// rows reads the numbers of the page of the project's cycles ids, each once
// for the page, whatever its length (M6 P4 spec R2, R9): the groups of the
// project's states, which the counts by state are summed by (domain.CountsOf),
// and the assignees, of the work items viewer sees (held.viewer); and the
// frozen counts, of every work item, unless viewer sees his own alone, whom
// they are not read for (R5).
func (u *ListCycles) rows(ctx context.Context, project uuid.UUID, ids []uuid.UUID, viewer *uuid.UUID) (cycleRows, error) {
	var r cycleRows
	groups, err := u.d.Groups.StateGroups(ctx, project)
	if err != nil {
		return r, err
	}
	counts, err := u.d.Stats.CycleCounts(ctx, project, ids, viewer)
	if err != nil {
		return r, err
	}
	if r.live, err = domain.CountsOf(counts, groups); err != nil {
		return r, err
	}
	assignees, err := u.d.Stats.CycleAssignees(ctx, project, ids, viewer)
	if err != nil {
		return r, err
	}
	r.assignees = map[uuid.UUID][]uuid.UUID{}
	for _, a := range assignees {
		r.assignees[a.CycleID] = append(r.assignees[a.CycleID], a.AssigneeID)
	}
	if viewer == nil {
		r.frozen, err = u.d.Snapshots.CycleSnapshotCounts(ctx, ids)
	}
	return r, err
}
