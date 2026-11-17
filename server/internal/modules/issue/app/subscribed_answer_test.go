package app_test

import (
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

// answered is a use case's answer, got or err, the world w it ran in, and
// the id of the work item it names.
type answered struct {
	got domain.Issue
	err error
	w   *world
	id  uuid.UUID
}

// detailAnswers are the use cases that answer a work item in full (M7 P1a
// spec 2.3), each over a world of its own, as alice calls them: the work
// item read, by its id and by its identifier; changed, and changed in
// nothing; archived, archived again, unarchived, unarchived again; and
// created, whose id the creation draws. Each answers the work item it
// names and the world it ran in (answered).
var detailAnswers = []struct {
	name   string
	answer func(t *testing.T, subscribers []uuid.UUID, fail string) answered
}{
	{"read", func(_ *testing.T, subscribers []uuid.UUID, fail string) answered {
		get, w, id := newGet()
		w.issues.subscribers, w.log.fail = subscribers, fail
		got, err := get.Execute(as(alice), id)
		return answered{got, err, w, id}
	}},
	{"read by its identifier", func(_ *testing.T, subscribers []uuid.UUID, fail string) answered {
		get, b, id := newGetByIdentifier()
		b.issues.subscribers, b.log.fail = subscribers, fail
		got, err := get.Execute(as(alice), "acme", "WEB-7")
		return answered{got, err, b.world, id}
	}},
	{"changed", func(_ *testing.T, subscribers []uuid.UUID, fail string) answered {
		update, w, id := newUpdate()
		w.issues.subscribers, w.log.fail = subscribers, fail
		got, err := update.Execute(as(alice), id, domain.IssuePatch{Name: name("Mend")})
		return answered{got, err, w, id}
	}},
	{"changed in nothing", func(_ *testing.T, subscribers []uuid.UUID, fail string) answered {
		update, w, id := newUpdate()
		w.issues.subscribers, w.log.fail = subscribers, fail
		got, err := update.Execute(as(alice), id, domain.IssuePatch{Name: name("Fix")})
		return answered{got, err, w, id}
	}},
	{"archived", func(_ *testing.T, subscribers []uuid.UUID, fail string) answered {
		archive, _, w, id := newArchive()
		w.issues.subscribers, w.log.fail = subscribers, fail
		got, err := archive.Execute(as(alice), id)
		return answered{got, err, w, id}
	}},
	{"archived again", func(_ *testing.T, subscribers []uuid.UUID, fail string) answered {
		archive, _, w, id := newArchive()
		w.issues.subscribers, w.issues.archivedAt, w.log.fail = subscribers, &earlier, fail
		got, err := archive.Execute(as(alice), id)
		return answered{got, err, w, id}
	}},
	{"unarchived", func(_ *testing.T, subscribers []uuid.UUID, fail string) answered {
		_, unarchive, w, id := newArchive()
		w.issues.subscribers, w.issues.archivedAt, w.log.fail = subscribers, &earlier, fail
		got, err := unarchive.Execute(as(alice), id)
		return answered{got, err, w, id}
	}},
	{"unarchived again", func(_ *testing.T, subscribers []uuid.UUID, fail string) answered {
		_, unarchive, w, id := newArchive()
		w.issues.subscribers, w.log.fail = subscribers, fail
		got, err := unarchive.Execute(as(alice), id)
		return answered{got, err, w, id}
	}},
	{"created", func(t *testing.T, subscribers []uuid.UUID, fail string) answered {
		create, w := newCreate()
		w.issues.subscribers, w.log.fail = subscribers, fail
		got, err := create.Execute(as(alice), web, domain.NewIssue{Name: "Fix"})
		if w.issues.issue == nil {
			t.Fatalf("Execute() = %v, without a work item written", err)
		}
		return answered{got, err, w, w.issues.issue.ID}
	}},
}

// Each answer of a work item in full says whether its caller subscribes to
// it (M7 P1a spec 2.3, R5): alice's own row among its subscribers, read
// last, for her and the work item it answers, in the answer's transaction
// or read snapshot, after the answer is otherwise whole. Not bob's row:
// another account subscribed is not the caller.
func TestADetailAnswerSaysWhetherTheCallerSubscribes(t *testing.T) {
	for _, tt := range detailAnswers {
		for _, sub := range []struct {
			name        string
			subscribers []uuid.UUID
			want        bool
		}{{"alice subscribed", []uuid.UUID{bob, alice}, true}, {"bob subscribed", []uuid.UUID{bob}, false}, {"none", []uuid.UUID{}, false}} {
			t.Run(tt.name+", "+sub.name, func(t *testing.T) {
				a := tt.answer(t, sub.subscribers, "")
				got, err, w, id := a.got, a.err, a.w, a.id
				n := len(w.log.calls)
				if read := line("IsSubscribed", id, alice); err != nil || got.ID != id || got.Subscribed != sub.want || n == 0 ||
					w.log.calls[n-1] != read {
					t.Errorf("Execute() = %+v, %v after %q; want work item %s subscribed %v, read last as %q", got, err, w.log.calls, id,
						sub.want, read)
				}
			})
		}
		t.Run(tt.name+", the read failing", func(t *testing.T) {
			a := tt.answer(t, []uuid.UUID{alice}, "IsSubscribed")
			wantStoppedAt(t, a.w, a.err, "IsSubscribed")
		})
	}
}
