package bootstrap

import (
	"context"
	"reflect"
	"strings"

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

// goneItem is the second work item of gone's project (relatedItems).
const goneItem = "Second"

// relatedItems are the work items issueSeed stores for a relation alone,
// after the others of seededItems: gone's second, by its admin, so that a
// relation of the deleted workspace's project is there for its column's
// cells to name (relationOf).
var relatedItems = []seededItem{{"gone/project", goneItem, matrixAdmins["gone"]}}

// seededRelations are the relations prepareMatrix seeds through the issue
// store (M4 design 9.2), each a case of a list of relations, or the one a
// column's cells delete (relationOf): acme's public project's matrixIssue
// blocked by the private one's (Blocked), which the guest rule of the
// public project hides from PG, who sees every work item of the private
// one; the guest's own work item's relates_to the public project's
// matrixIssue, which the guest rule hides from him; PM's duplicate of the
// member before's in the private project; the workspace admin who is a
// guest's own relates_to the private project's matrixIssue, of a project
// he is none of; the member before's blocked by it (Secret), which PG sees
// there; the public project's matrixIssue relates_to the workspace admin
// who is a guest's own (Others'), neither the guest's; the archived
// project's matrixIssue a duplicate of the private one's (Shelved); and
// gone's matrixIssue relates_to its second work item, deleted with gone.
var seededRelations = []seededRelation{
	{"Blocked", "acme/public", matrixIssue, "acme/private", matrixIssue, issuedomain.BlockedBy},
	{"Guest's", "acme/public", issueOf(callerGuestOwn), "acme/public", matrixIssue, issuedomain.RelatesTo},
	{"Member's", "acme/public", issueOf(callerCreator), "acme/private", issueOf(callerBefore), issuedomain.Duplicate},
	{"Admin guest's", "acme/public", issueOf(callerAdminGuestOwn), "acme/private", matrixIssue, issuedomain.RelatesTo},
	{"Secret", "acme/private", issueOf(callerBefore), "acme/private", matrixIssue, issuedomain.BlockedBy},
	{"Others'", "acme/public", matrixIssue, "acme/public", issueOf(callerAdminGuestOwn), issuedomain.RelatesTo},
	{"Shelved", "acme/archived", matrixIssue, "acme/private", matrixIssue, issuedomain.Duplicate},
	{"Gone", "gone/project", matrixIssue, "gone/project", goneItem, issuedomain.RelatesTo},
}

// relationsOf are seededRelations as stored, each with the id sd names, of
// its row's workspace, by that workspace's admin, at the seed's moment.
func (s issueSeed) relationsOf(sd seeded) []issuedomain.Relation {
	rs := make([]issuedomain.Relation, len(seededRelations))
	for n, r := range seededRelations {
		slug, _, _ := strings.Cut(r.fromKey, "/")
		by := s.ids[matrixAdmins[slug]]
		rs[n] = issuedomain.Relation{ID: sd.relation(r.fromKey, r.name), WorkspaceID: s.workspaces[slug], ProjectID: s.projects[r.fromKey],
			IssueID: sd.issue(r.fromKey, r.from), RelatedID: sd.issue(r.toKey, r.to), Type: r.kind, CreatedByID: &by, CreatedAt: s.now}
	}
	return rs
}

// relations stores seededRelations (relationsOf), each in an insert of its
// own, after the work items (issues).
func (s issueSeed) relations(sd seeded) {
	s.t.Helper()
	for _, r := range s.relationsOf(sd) {
		if err := s.store.InsertRelations(context.Background(), []issuedomain.Relation{r}); err != nil {
			s.t.Fatal(err)
		}
	}
}

// seededRelationRows checks the relations the cells rest on, read back
// through the issue store by the ids sd names: each as relationsOf has it,
// but its moment, which the database keeps to the microsecond; gone's
// deleted with it. One missing, or stored otherwise, would let a cell
// answer as it wants for another reason.
func (s issueSeed) seededRelationRows(sd seeded) {
	s.t.Helper()
	for _, want := range s.relationsOf(sd) {
		gone := want.WorkspaceID == s.workspaces["gone"]
		got, found, err := s.store.GetRelation(context.Background(), want.ID)
		got.CreatedAt = want.CreatedAt
		if err != nil || found == gone || !gone && !reflect.DeepEqual(got, want) {
			s.t.Fatalf("the relation %s = %+v, %v, %v; want %+v, deleted %v", want.ID, got, found, err, want, gone)
		}
	}
}
