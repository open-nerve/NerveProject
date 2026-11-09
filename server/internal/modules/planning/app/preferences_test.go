package app

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/planning/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// The lock world's display settings on the pages (M6 P3b): their store, and the issue module's list settings.

func (w *lockWorld) CyclePreferences(ctx context.Context, cycle, user uuid.UUID) (domain.CyclePreferences, bool, error) {
	w.log(ctx, "CyclePreferences")
	p, ok := w.pagePreferences[[2]uuid.UUID{cycle, user}].(domain.CyclePreferences)
	return p, ok, nil
}

func (w *lockWorld) ModulePreferences(ctx context.Context, module, user uuid.UUID) (domain.ModulePreferences, bool, error) {
	w.log(ctx, "ModulePreferences")
	p, ok := w.pagePreferences[[2]uuid.UUID{module, user}].(domain.ModulePreferences)
	return p, ok, nil
}

func (w *lockWorld) UpsertCyclePreferences(ctx context.Context, c PreferencesChange[domain.CycleFilters]) (domain.CyclePreferences, error) {
	w.log(ctx, "UpsertCyclePreferences")
	return upsertPage(w, c), nil
}

func (w *lockWorld) UpsertModulePreferences(ctx context.Context, c PreferencesChange[domain.ModuleFilters]) (domain.ModulePreferences, error) {
	w.log(ctx, "UpsertModulePreferences")
	return upsertPage(w, c), nil
}

// upsertPage records c, and stores the account's settings on its page: his stored ones, or the defaults, with the
// change applied.
func upsertPage[F domain.PageFilters](w *lockWorld, c PreferencesChange[F]) domain.Preferences[F] {
	w.pageChanges = append(w.pageChanges, c)
	if w.pagePreferences == nil {
		w.pagePreferences = map[[2]uuid.UUID]any{}
	}
	key := [2]uuid.UUID{c.Page, c.UserID}
	p, ok := w.pagePreferences[key].(domain.Preferences[F])
	if !ok {
		p = domain.DefaultPreferences[F]()
	}
	p = p.Apply(c.Patch)
	w.pagePreferences[key] = p
	return p
}

func (w *lockWorld) Validate(filters map[string][]string, display *shared.ListDisplayFilters) []shared.FieldError {
	w.calls = append(w.calls, "Validate")
	w.checkedFilters, w.checkedDisplay = filters, display
	return w.refusedWords
}

// page is the page of the lock world's cycle, or of its module, as the display settings' tests read and change it.
type page[F domain.PageFilters] struct {
	id                  uuid.UUID
	action              shared.Action
	object, read, write string // what the world logs: the read of the page's cycle or module; the settings' read and upsert
	notFound            error
	world               func() *lockWorld // a world with the page, its project's toggle of it on
	remove, archive     func(w *lockWorld)
	get                 func(ctx context.Context, w *lockWorld) (domain.Preferences[F], error)
	update              func(ctx context.Context, w *lockWorld, p domain.PreferencesPatch[F]) (domain.Preferences[F], error)
	filters             F                   // the page's filters, the other page's key among them
	byName              map[string][]string // filters by their JSON names
}

func cyclePage() page[domain.CycleFilters] {
	return page[domain.CycleFilters]{
		id: lockCycle, action: domain.ActionCyclePreferences, object: "GetCycle", read: "CyclePreferences", write: "UpsertCyclePreferences",
		notFound: domain.ErrCycleNotFound, world: newLockWorld,
		remove: func(w *lockWorld) { delete(w.cycles, lockCycle) },
		archive: func(w *lockWorld) {
			c, at := w.cycles[lockCycle], time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
			c.ArchivedAt = &at
			w.cycles[lockCycle] = c
		},
		get: func(ctx context.Context, w *lockWorld) (domain.CyclePreferences, error) {
			return NewGetCyclePreferences(GetPreferencesDeps{Reads: w.reads(), Preferences: w}).Execute(ctx, lockCycle)
		},
		update: func(ctx context.Context, w *lockWorld, p domain.CyclePreferencesPatch) (domain.CyclePreferences, error) {
			return NewUpdateCyclePreferences(UpdatePreferencesDeps{Locks: w.locks(), Settings: w, Clock: w, Preferences: w}).Execute(ctx, lockCycle, p)
		},
		filters: domain.CycleFilters{ListFilters: shared.ListFilters{Priority: []string{"high"}, LabelID: []string{}}, ModuleID: []string{"none"}},
		byName:  map[string][]string{"priority": {"high"}, "label_id": {}, "module_id": {"none"}},
	}
}

func modulePage() page[domain.ModuleFilters] {
	return page[domain.ModuleFilters]{
		id: lockModule, action: domain.ActionModulePreferences, object: "GetModule", read: "ModulePreferences", write: "UpsertModulePreferences",
		notFound: domain.ErrModuleNotFound, world: newModuleWorld,
		remove: func(w *lockWorld) { delete(w.modules, lockModule) },
		archive: func(w *lockWorld) {
			m, at := w.modules[lockModule], time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
			m.ArchivedAt = &at
			w.modules[lockModule] = m
		},
		get: func(ctx context.Context, w *lockWorld) (domain.ModulePreferences, error) {
			return NewGetModulePreferences(GetPreferencesDeps{Reads: w.reads(), Preferences: w}).Execute(ctx, lockModule)
		},
		update: func(ctx context.Context, w *lockWorld, p domain.ModulePreferencesPatch) (domain.ModulePreferences, error) {
			return NewUpdateModulePreferences(UpdatePreferencesDeps{Locks: w.locks(), Settings: w, Clock: w, Preferences: w}).Execute(ctx, lockModule, p)
		},
		filters: domain.ModuleFilters{ListFilters: shared.ListFilters{Priority: []string{"high"}, LabelID: []string{}}, CycleID: []string{"none"}},
		byName:  map[string][]string{"priority": {"high"}, "label_id": {}, "cycle_id": {"none"}},
	}
}

// onBothPages runs test on the cycle's page and on the module's.
func onBothPages(t *testing.T, cycle func(*testing.T, page[domain.CycleFilters]), module func(*testing.T, page[domain.ModuleFilters])) {
	t.Run("cycle", func(t *testing.T) { cycle(t, cyclePage()) })
	t.Run("module", func(t *testing.T) { module(t, modulePage()) })
}

var prefsCaller = uuid.NewV7()

func asPrefsCaller() context.Context {
	return shared.WithActor(context.Background(), shared.Actor{UserID: prefsCaller})
}

// A read decides the page's action in a read's snapshot, as the page's other reads (Reads): the cycle or module, its
// project's facts, the decision, then the caller's own settings; no gate, no lock, no clock. Without a row of them it
// answers the defaults and writes nothing; with one, his stored settings.
func TestGetPagePreferences(t *testing.T) {
	onBothPages(t, getPagePreferences[domain.CycleFilters], getPagePreferences[domain.ModuleFilters])
}

func getPagePreferences[F domain.PageFilters](t *testing.T, p page[F]) {
	w := p.world()
	got, err := p.get(asPrefsCaller(), w)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"Begin", p.object, "Facts", "Authorize " + string(p.action), p.read, "End"}
	if !slices.Equal(w.calls, want) || !reflect.DeepEqual(got, domain.DefaultPreferences[F]()) || len(w.pageChanges) != 0 {
		t.Fatalf("without a row: %+v, calls\n%q\nwant the defaults, nothing written, calls\n%q", got, w.calls, want)
	}
	stored := domain.Preferences[F]{Filters: p.filters, DisplayProperties: shared.ListDisplayProperties{Key: true}}
	w = p.world()
	w.pagePreferences = map[[2]uuid.UUID]any{{p.id, prefsCaller}: stored, {p.id, uuid.NewV7()}: domain.DefaultPreferences[F]()}
	if got, err := p.get(asPrefsCaller(), w); err != nil || !reflect.DeepEqual(got, stored) || len(w.pageChanges) != 0 {
		t.Fatalf("with his row: %+v, %v; want %+v, nothing written", got, err, stored)
	}
}

// A change checks its words first, outside any transaction; then, in one transaction, the page's cycle or module and
// its project's facts read without a lock and the decision, the project's row FOR SHARE (Share) and the page's cycle
// or module read again under it and the decision again; the clock under the lock; the caller's own row last. No write
// gate and no write lock (Lock). The change the store is given is the caller's, on the page of the project's workspace,
// at the clock's moment, with the patch as given; the answer, the settings as stored.
func TestUpdatePagePreferences(t *testing.T) {
	onBothPages(t, updatePagePreferences[domain.CycleFilters], updatePagePreferences[domain.ModuleFilters])
}

func updatePagePreferences[F domain.PageFilters](t *testing.T, p page[F]) {
	w := p.world()
	w.now = time.Date(2026, 11, 9, 8, 0, 0, 0, time.UTC)
	display := shared.ListDisplayFilters{Layout: "kanban", OrderBy: "-cycle_name", Calendar: shared.ListCalendar{Layout: "week"}}
	patch := domain.PreferencesPatch[F]{Filters: &p.filters, DisplayFilters: &display}
	got, err := p.update(asPrefsCaller(), w, patch)
	if err != nil {
		t.Fatal(err)
	}
	authorize := "Authorize " + string(p.action)
	want := []string{"Validate", "Begin", p.object, "Facts", authorize, "Share", p.object, authorize, "Now", p.write, "End"}
	if !slices.Equal(w.calls, want) {
		t.Fatalf("calls\n%q\nwant\n%q", w.calls, want)
	}
	stored := domain.DefaultPreferences[F]().Apply(patch)
	if !reflect.DeepEqual(got, stored) || len(w.pageChanges) != 1 {
		t.Fatalf("answer %+v after %d changes; want %+v after one", got, len(w.pageChanges), stored)
	}
	c := w.pageChanges[0].(PreferencesChange[F])
	if c.ID == (uuid.UUID{}) || c.WorkspaceID != lockWorkspace || c.ProjectID != lockProject || c.Page != p.id || c.UserID != prefsCaller ||
		!c.Now.Equal(w.now) || !reflect.DeepEqual(c.Patch, patch) {
		t.Fatalf("change %+v", c)
	}
	// A second change gives the store its fields alone, whatever is stored.
	properties := shared.ListDisplayProperties{Cycle: true}
	if _, err := p.update(asPrefsCaller(), w, domain.PreferencesPatch[F]{DisplayProperties: &properties}); err != nil {
		t.Fatal(err)
	}
	if c := w.pageChanges[1].(PreferencesChange[F]); !reflect.DeepEqual(c.Patch, domain.PreferencesPatch[F]{DisplayProperties: &properties}) {
		t.Fatalf("the second change's patch %+v", c.Patch)
	}
}

// The words a change gives reach the issue module's list settings by their JSON names, the other page's key among
// them, an empty array kept, with the display filters as given; omitted filters are a nil map, cleared ones an empty
// map, omitted display filters nil.
func TestPagePreferencesWordsReachTheListSettings(t *testing.T) {
	onBothPages(t, pageWordsReachTheListSettings[domain.CycleFilters], pageWordsReachTheListSettings[domain.ModuleFilters])
}

func pageWordsReachTheListSettings[F domain.PageFilters](t *testing.T, p page[F]) {
	w := p.world()
	display := shared.ListDisplayFilters{Layout: "list", GroupBy: new("cycle_id"), OrderBy: "module_name", Calendar: shared.ListCalendar{Layout: "month"}}
	if _, err := p.update(asPrefsCaller(), w, domain.PreferencesPatch[F]{Filters: &p.filters, DisplayFilters: &display}); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(w.checkedFilters, p.byName) || w.checkedFilters["label_id"] == nil || !reflect.DeepEqual(w.checkedDisplay, &display) {
		t.Fatalf("the list settings were given %#v, %+v; want %#v, %+v", w.checkedFilters, w.checkedDisplay, p.byName, display)
	}
	var none F
	for _, patch := range []domain.PreferencesPatch[F]{{}, {Filters: &none}} {
		if _, err := p.update(asPrefsCaller(), w, patch); err != nil {
			t.Fatal(err)
		}
		if (w.checkedFilters == nil) != (patch.Filters == nil) || len(w.checkedFilters) != 0 || w.checkedDisplay != nil {
			t.Fatalf("omitted or cleared filters: given %#v, %+v", w.checkedFilters, w.checkedDisplay)
		}
	}
}

// The refusals of a change in the order of M6 P3b spec 2.3: the words' 422, every field the list settings refuse, in
// their order, before anything is read; then the page's 404 (its cycle or module not there, or its project not there
// or not seen); then the rule's 403. Each case has every later condition too. A read answers the same 404 and 403. A
// refusal before the lock takes no lock, a refusal of either nothing written.
func TestPagePreferencesRefusalsInOrder(t *testing.T) {
	onBothPages(t, pagePreferencesRefusalsInOrder[domain.CycleFilters], pagePreferencesRefusalsInOrder[domain.ModuleFilters])
}

func pagePreferencesRefusalsInOrder[F domain.PageFilters](t *testing.T, p page[F]) {
	words := []shared.FieldError{{Field: "filters.priority[0]", Code: shared.FieldNotAllowed}, {Field: "display_filters.group_by", Code: shared.FieldNotAllowed}}
	forbidden := shared.Forbidden()
	for _, tt := range []struct {
		name string
		set  func(*lockWorld)
		want error
	}{
		{"words", func(w *lockWorld) { w.refusedWords = words; p.remove(w); w.refusal = forbidden }, shared.Invalid(words...)},
		{"no page", func(w *lockWorld) { p.remove(w); w.refusal = forbidden }, p.notFound},
		{"project gone", func(w *lockWorld) { delete(w.facts, lockProject); w.refusal = forbidden }, p.notFound},
		{"project not seen", func(w *lockWorld) { w.refusal = shared.ErrNotVisible }, p.notFound},
		{"role refused", func(w *lockWorld) { w.refusal = forbidden }, forbidden},
	} {
		t.Run(tt.name, func(t *testing.T) {
			w := p.world()
			tt.set(w)
			_, err := p.update(asPrefsCaller(), w, domain.PreferencesPatch[F]{Filters: &p.filters})
			var got *shared.Error
			if !sameRefusal(err, tt.want) || (tt.name == "words" && (!errors.As(err, &got) || !reflect.DeepEqual(got.Fields, words))) {
				t.Fatalf("got %v; want %v", err, tt.want)
			}
			if slices.Contains(w.calls, "Share") || len(w.pageChanges) != 0 || (tt.name == "words" && slices.Contains(w.calls, "Begin")) {
				t.Fatalf("calls %q: a refusal took the lock or wrote, or the words' read on", w.calls)
			}
			if tt.name == "words" {
				return
			}
			w = p.world()
			tt.set(w)
			if _, err := p.get(asPrefsCaller(), w); !sameRefusal(err, tt.want) || slices.Contains(w.calls, p.read) {
				t.Fatalf("a read: %v, calls %q; want %v, his settings not read", err, w.calls, tt.want)
			}
		})
	}
}

// What committed while the share waited is decided again under it: the page's cycle or module deleted, its project
// deleted, the caller removed or demoted. Nothing is written.
func TestPagePreferencesDecideAgainUnderTheShare(t *testing.T) {
	onBothPages(t, pagePreferencesDecideAgain[domain.CycleFilters], pagePreferencesDecideAgain[domain.ModuleFilters])
}

func pagePreferencesDecideAgain[F domain.PageFilters](t *testing.T, p page[F]) {
	for _, tt := range []struct {
		name   string
		change func(*lockWorld)
		want   error
	}{
		{"page deleted", func(w *lockWorld) { p.remove(w) }, p.notFound},
		{"project deleted", func(w *lockWorld) { delete(w.facts, lockProject) }, p.notFound},
		{"caller removed", func(w *lockWorld) { w.refusal = shared.ErrNotVisible }, p.notFound},
		{"caller demoted", func(w *lockWorld) { w.refusal = shared.Forbidden() }, shared.Forbidden()},
	} {
		t.Run(tt.name, func(t *testing.T) {
			w := p.world()
			w.onLock = func() { tt.change(w) }
			if _, err := p.update(asPrefsCaller(), w, domain.PreferencesPatch[F]{}); !sameRefusal(err, tt.want) || len(w.pageChanges) != 0 ||
				slices.Contains(w.calls, "Now") {
				t.Fatalf("got %v, calls %q; want %v, nothing written", err, w.calls, tt.want)
			}
		})
	}
}

// What refuses a planning write does not refuse one's own display settings (M6 P3b spec R6): neither an OwnOnly grant,
// nor an archived project, an archived cycle or module, nor the project's cycles and modules turned off; reading and
// changing them both go through. A busy transaction is the change's answer.
func TestPagePreferencesAllowWhatAWriteRefuses(t *testing.T) {
	onBothPages(t, pagePreferencesAllow[domain.CycleFilters], pagePreferencesAllow[domain.ModuleFilters])
}

func pagePreferencesAllow[F domain.PageFilters](t *testing.T, p page[F]) {
	for _, tt := range []struct {
		name string
		set  func(*lockWorld)
	}{
		{"own only", func(w *lockWorld) { w.grant.OwnOnly = true }},
		{"project archived", func(w *lockWorld) { f := w.facts[lockProject]; f.Archived = true; w.facts[lockProject] = f }},
		{"page archived", func(w *lockWorld) { p.archive(w) }},
		{"toggles off", func(w *lockWorld) {
			f := w.facts[lockProject]
			f.CycleView, f.ModuleView = false, false
			w.facts[lockProject] = f
		}},
		{"all of them", func(w *lockWorld) {
			w.grant.OwnOnly = true
			p.archive(w)
			w.facts[lockProject] = ProjectFacts{WorkspaceID: lockWorkspace, Archived: true, Timezone: "UTC"}
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			w := p.world()
			tt.set(w)
			if _, err := p.get(asPrefsCaller(), w); err != nil {
				t.Fatalf("a read: %v", err)
			}
			if _, err := p.update(asPrefsCaller(), w, domain.PreferencesPatch[F]{Filters: &p.filters}); err != nil || len(w.pageChanges) != 1 {
				t.Fatalf("a change: %v, %d written", err, len(w.pageChanges))
			}
		})
	}
	w := p.world()
	busy := shared.ServerBusy(time.Second)
	w.busy = busy
	if _, err := p.update(asPrefsCaller(), w, domain.PreferencesPatch[F]{}); !errors.Is(err, busy) {
		t.Fatalf("a busy transaction = %v", err)
	}
}

// Without an actor, neither reads nor checks anything.
func TestPagePreferencesNeedAnActor(t *testing.T) {
	p := cyclePage()
	w := p.world()
	if _, err := p.update(context.Background(), w, domain.CyclePreferencesPatch{}); !sameRefusal(err, shared.Unauthenticated()) || len(w.calls) != 0 {
		t.Fatalf("a change without an actor: %v, calls %q", err, w.calls)
	}
	if _, err := p.get(context.Background(), w); !sameRefusal(err, shared.Unauthenticated()) || len(w.calls) != 0 {
		t.Fatalf("a read without an actor: %v, calls %q", err, w.calls)
	}
}
