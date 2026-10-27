package app_test

import (
	"reflect"
	"strings"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Removing either endpoint Find makes these refusals authorize or enter a gate.
func TestDeleteRelationDeletedEndpointPrecheck(t *testing.T) {
	for _, reverse := range []bool{false, true} {
		for _, deleted := range []uuid.UUID{web, ops} {
			for _, allowed := range []bool{false, true} {
				t.Run(line("endpoint", reverse, deleted, allowed), func(t *testing.T) {
					_, w, rel := newUnrelate()
					if reverse {
						fromNear(w, rel)
					}
					if deleted == web {
						w.project.found = false
					} else {
						delete(w.project.directory, ops)
					}
					if !allowed {
						w.auth.answers = []error{shared.Forbidden(), shared.Forbidden()}
					}
					gated(w, nil)
					err := unrelateOver(w, rel).Execute(as(alice), relationID)
					if !reflect.DeepEqual(err, domain.ErrRelationNotFound) {
						t.Fatalf("got %#v, want relation 404", err)
					}
					if w.auth.calls != 0 {
						t.Errorf("authorized %d times before existence check", w.auth.calls)
					}
					for _, c := range w.log.calls {
						if strings.HasPrefix(c, "Enter") || c == "Begin" {
							t.Errorf("precheck acquired write resource: %s", c)
						}
					}
					wantNothingWritten(t, w, rel)
				})
			}
		}
	}
	for _, project := range []uuid.UUID{web, ops} {
		t.Run(line("Find error", project), func(t *testing.T) {
			uc, w, _ := newUnrelate()
			w.log.fail = line("Find", project)
			if err := uc.Execute(as(alice), relationID); err != errDisk {
				t.Fatalf("got %v, want original Find error", err)
			}
		})
	}
}

// Removing eitherUnder's map check lets a project disappearing after Find reach authorization.
func TestDeleteRelationChecksBothLockedProjects(t *testing.T) {
	for _, deleted := range []uuid.UUID{web, ops} {
		t.Run(deleted.String(), func(t *testing.T) {
			uc, w, rel := newUnrelate()
			w.project.waited = func() {
				if deleted == web {
					w.project.locked = nil
				} else {
					delete(w.project.across, ops)
				}
			}
			if err := uc.Execute(as(alice), relationID); !reflect.DeepEqual(err, domain.ErrRelationNotFound) {
				t.Fatal(err)
			}
			if w.auth.calls != 1 {
				t.Fatalf("authorization calls %d, want only precheck", w.auth.calls)
			}
			wantNothingWritten(t, w, rel)
		})
	}
}
