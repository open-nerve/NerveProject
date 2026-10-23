package bootstrap

import (
	"context"
	"reflect"
	"slices"

	issuedomain "github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

// seededRelation is a relation issueSeed stores: its name; its work items,
// each by its project's key and its name (seededItems), the first the end
// stored as issue_id, of its row's project; and its type as stored.
type seededRelation struct {
	name          string
	fromKey, from string
	toKey, to     string
	kind          issuedomain.RelationType
}

// seededRelations are the relations prepareMatrix seeds through the issue
// store (M4 design 9.2), each a case of a list of relations: acme's public
// project's matrixIssue blocked by the private one's (Blocked), which the
// guest rule of the public project hides from PG, who sees every work item
// of the private one; the guest's own work item's relates_to the public
// project's matrixIssue, which the guest rule hides from him; PM's
// duplicate of the member before's in the private project; the workspace
// admin who is a guest's own relates_to the private project's matrixIssue,
// of a project he is none of; and the member before's blocked by it
// (Secret), which PG sees there.
var seededRelations = []seededRelation{
	{"Blocked", "acme/public", matrixIssue, "acme/private", matrixIssue, issuedomain.BlockedBy},
	{"Guest's", "acme/public", issueOf(callerGuestOwn), "acme/public", matrixIssue, issuedomain.RelatesTo},
	{"Member's", "acme/public", issueOf(callerCreator), "acme/private", issueOf(callerBefore), issuedomain.Duplicate},
	{"Admin guest's", "acme/public", issueOf(callerAdminGuestOwn), "acme/private", matrixIssue, issuedomain.RelatesTo},
	{"Secret", "acme/private", issueOf(callerBefore), "acme/private", matrixIssue, issuedomain.BlockedBy},
}

// relationsOf are seededRelations as stored, each with the id sd names, of
// acme, by its admin, at the seed's moment.
func (s issueSeed) relationsOf(sd seeded) []issuedomain.Relation {
	by := s.ids[matrixAdmins["acme"]]
	rs := make([]issuedomain.Relation, len(seededRelations))
	for n, r := range seededRelations {
		rs[n] = issuedomain.Relation{ID: sd.relation(r.fromKey, r.name), WorkspaceID: s.workspaces["acme"], ProjectID: s.projects[r.fromKey],
			IssueID: sd.issue(r.fromKey, r.from), RelatedID: sd.issue(r.toKey, r.to), Type: r.kind, CreatedByID: &by, CreatedAt: s.now}
	}
	return rs
}

// relations stores seededRelations (relationsOf), in one insert, after
// the work items (issues).
func (s issueSeed) relations(sd seeded) {
	s.t.Helper()
	if err := s.store.InsertRelations(context.Background(), s.relationsOf(sd)); err != nil {
		s.t.Fatal(err)
	}
}

// seededRelationRows checks the relations the cells rest on, read back
// through the issue store among the relations of each one's stored
// issue_id end: each as relationsOf has it, but its moment, which the
// database keeps to the microsecond. One missing, or stored otherwise,
// would let a list of relations answer as a cell wants for another reason.
func (s issueSeed) seededRelationRows(sd seeded) {
	s.t.Helper()
	for _, want := range s.relationsOf(sd) {
		rs, err := s.store.Relations(context.Background(), want.IssueID)
		n := slices.IndexFunc(rs, func(r issuedomain.Relation) bool { return r.ID == want.ID })
		if err != nil || n < 0 {
			s.t.Fatalf("the relations of %s = %+v, %v; want %+v among them", want.IssueID, rs, err, want)
		}
		got := rs[n]
		got.CreatedAt = want.CreatedAt
		if !reflect.DeepEqual(got, want) {
			s.t.Fatalf("the relation %s = %+v; want %+v", want.ID, rs[n], want)
		}
	}
}
