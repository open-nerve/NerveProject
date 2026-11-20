package bootstrap

import (
	"bytes"
	"net/http"
	"net/url"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"
	"uuid"

	"github.com/getkin/kin-openapi/openapi3"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue"
	issueapp "github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	navigationgen "github.com/open-nerve/NerveProject/server/internal/modules/navigation/adapter/http/gen"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// savedViewsRoundTrip is H13's round trip (M7 P3b spec 2.4, 4; re-90
// m10): a view of each kind stores one word at a time, through the API,
// and its words, read back, sent to its kind's list as its query
// (queryOf), are taken and filter the work items the word names. Every
// word is tried: each filter key of the contract's ViewFilters, each
// order and each group the lists' contract names (IssueListOrder,
// IssueListGroupField), and sub_issue both ways. Which of them a kind
// stores, its words say (the issue module's settings the app checks a view
// with: ListSettings, WorkspaceListSettings); those it does not store are
// refused at their field, and those it stores make the round trip. A word
// stored that the list refuses fails it, a new key, order or group too:
// none is listed by hand.
func savedViewsRoundTrip(t *testing.T, w workspaceViewsWorld) {
	t.Helper()
	orders, groups := listWordsOfTheContract(t)
	// One value of each filter key, and what it filters in Web's list and
	// in acme's (newWorkspaceViewsWorld).
	probes := map[string]struct {
		value              string
		project, workspace []uuid.UUID
	}{
		"state_id": {w.done.String(), workItems(w.i2), workItems(w.i2)}, "state_group": {"completed", workItems(w.i2), workItems(w.i2)},
		"priority": {"urgent", workItems(w.i1), workItems(w.i1, w.a1)}, "label_id": {w.bug.String(), workItems(w.i1), workItems(w.i1)},
		"assignee_id": {w.ids["bob"].String(), workItems(w.i1), workItems(w.i1)}, "created_by_id": {w.ids["bob"].String(), workItems(w.i2), workItems(w.i2)},
		"mention_id": {w.ids["bob"].String(), workItems(w.i1), workItems(w.i1)}, "subscriber_id": {w.ids["bob"].String(), workItems(w.i1, w.i2), workItems(w.i1, w.i2)},
		"start_date": {"2026-10-10..2026-10-13", workItems(w.i1), workItems(w.i1)}, "target_date": {"2026-10-14..2026-10-16", workItems(w.i1), workItems(w.i1)},
		"cycle_id": {w.sprint.String(), workItems(w.i1), nil}, "module_id": {w.core.String(), workItems(w.i1), nil}, "project_id": {w.alpha.String(), nil, workItems(w.a1)},
	}
	for _, k := range []struct {
		name, views, list string
		words             issue.ListSettings
		all, parents      []uuid.UUID
		want              func(key string) []uuid.UUID
	}{
		{"workspace view", workspaceViewsPath("acme"), workspaceIssuesPath, issueapp.WorkspaceListSettings{}, workItems(w.i1, w.i2, w.i3, w.a1),
			workItems(w.i1, w.i3, w.a1), func(key string) []uuid.UUID { return probes[key].workspace }},
		{"project view", projectViewsPath(w.web), "/api/v0/projects/" + w.web.String() + "/issues", issueapp.ListSettings{}, workItems(w.i1, w.i2, w.i3),
			workItems(w.i1, w.i3), func(key string) []uuid.UUID { return probes[key].project }},
	} {
		t.Run(k.name, func(t *testing.T) {
			v, _ := w.newView(t, "alice", k.views, `{"name":"Round trip"}`)
			stores := func(filters map[string][]string, d shared.ListDisplayFilters) bool {
				return len(k.words.Validate(filters, &d)) == 0
			}
			// trip stores body in the view, and sends its words, read back,
			// to the list: it takes them, and shows want.
			trip := func(t *testing.T, body string, want []uuid.UUID) {
				t.Helper()
				w.changeView(t, "alice", v.ID, body)
				stored, _ := w.readView(t, "alice", v.ID)
				if got := w.listedIssues(t, "alice", k.list, stored.DisplayFilters, queryOf(stored)); !slices.Equal(got, inIDOrder(want)) {
					t.Errorf("%s lists %v; want %v", k.list, got, inIDOrder(want))
				}
			}
			refused := func(t *testing.T, body, field string) {
				t.Helper()
				w.refusedAs(t, "alice", http.MethodPatch, viewPath(v.ID), body, http.StatusUnprocessableEntity, "validation_failed "+field+" not_allowed")
			}
			d := shared.DefaultListDisplayFilters()
			for _, key := range viewFilterKeys() {
				t.Run("filters."+key, func(t *testing.T) {
					body := `{"filters":{"` + key + `":["` + probes[key].value + `"]},"display_filters":` + displayJSON(t, d) + `}`
					switch {
					case !stores(map[string][]string{key: {}}, d):
						refused(t, body, "filters."+key)
					case probes[key].value == "":
						t.Fatalf("%s stores %s: no value of it is given", k.name, key)
					default:
						// A key the list takes but whose set is not given lists
						// more than none: it fails on its set.
						trip(t, body, k.want(key))
					}
				})
			}
			for _, word := range wordsOf(d, orders, groups, stores) {
				t.Run(word.name, func(t *testing.T) {
					body := `{"filters":{},"display_filters":` + displayJSON(t, word.display) + `}`
					switch {
					case !stores(map[string][]string{}, word.display):
						refused(t, body, "display_filters."+word.field)
					case !word.display.SubIssue:
						trip(t, body, k.parents)
					default:
						trip(t, body, k.all)
					}
				})
			}
		})
	}
}

// displayWord is one word of a view's display filters a list takes as
// its query's: the display filters with it, and its field.
type displayWord struct {
	name, field string
	display     shared.ListDisplayFilters
}

// wordsOf is each word the display filters d may be given that a list
// takes: sub_issue either way; each of orders; each of groups as group_by;
// and each as sub_group_by under the first group_by of groups other than
// it that the kind stores, none when it stores none. The layout, the
// empty groups and the calendar only shape how a page shows its list: no
// list takes them.
func wordsOf(d shared.ListDisplayFilters, orders, groups []string, stores func(map[string][]string, shared.ListDisplayFilters) bool) []displayWord {
	var words []displayWord
	for _, sub := range []bool{true, false} {
		c := d
		c.SubIssue = sub
		words = append(words, displayWord{"sub_issue=" + strconv.FormatBool(sub), "sub_issue", c})
	}
	for _, order := range orders {
		c := d
		c.OrderBy = order
		words = append(words, displayWord{"order_by=" + order, "order_by", c})
	}
	for _, group := range groups {
		c := d
		c.GroupBy = &group
		words = append(words, displayWord{"group_by=" + group, "group_by", c})
	}
	for _, sub := range groups {
		c := d
		for _, group := range groups {
			c.GroupBy = &group
			if group != sub && stores(map[string][]string{}, c) {
				break
			}
			c.GroupBy = nil
		}
		c.SubGroupBy = &sub
		words = append(words, displayWord{"sub_group_by=" + sub, "sub_group_by", c})
	}
	return words
}

// queryOf is the query a list takes for the view v's words: each filter
// key's values, comma-separated but the dates' conditions, each its own
// (listIssues' parameters); its order, whether it shows sub-issues, and
// its groups when it has them.
func queryOf(v viewHTTP) url.Values {
	q := url.Values{}
	for key, values := range v.Filters {
		if key == "start_date" || key == "target_date" {
			q[key] = values
			continue
		}
		q.Set(key, strings.Join(values, ","))
	}
	d := v.DisplayFilters
	q.Set("order_by", d.OrderBy)
	q.Set("sub_issue", strconv.FormatBool(d.SubIssue))
	if d.GroupBy != nil {
		q.Set("group_by", *d.GroupBy)
	}
	if d.SubGroupBy != nil {
		q.Set("sub_group_by", *d.SubGroupBy)
	}
	return q
}

// listedIssues is the ids of the work items actor's list at path shows
// with q, sorted, each once: a flat page's, or each group's and lane's,
// as the display filters d group it.
func (w issueWorld) listedIssues(t *testing.T, actor, path string, d shared.ListDisplayFilters, q url.Values) []uuid.UUID {
	t.Helper()
	p := checkListEnvelope(t, w.send(t, actor, http.MethodGet, path+"?"+q.Encode(), "", http.StatusOK), d.GroupBy != nil, d.SubGroupBy != nil)
	var got []uuid.UUID
	var collect func(p issueListHTTP)
	collect = func(p issueListHTTP) {
		for _, row := range p.Data {
			got = append(got, row.ID)
		}
		for _, g := range slices.Concat(p.Groups, p.Subgroups) {
			collect(g)
		}
	}
	collect(p)
	return slices.Compact(inIDOrder(got))
}

// inIDOrder is ids sorted, a copy.
func inIDOrder(ids []uuid.UUID) []uuid.UUID {
	return slices.SortedFunc(slices.Values(ids), func(a, b uuid.UUID) int { return bytes.Compare(a[:], b[:]) })
}

// workItems is its arguments.
func workItems(id ...uuid.UUID) []uuid.UUID { return id }

// viewFilterKeys are the filter keys a view's body may give: the JSON
// names of the contract's ViewFilters, as its generated type has them.
func viewFilterKeys() []string {
	typ := reflect.TypeFor[navigationgen.ViewFilters]()
	keys := make([]string, typ.NumField())
	for n := range typ.NumField() {
		keys[n], _, _ = strings.Cut(typ.Field(n).Tag.Get("json"), ",")
	}
	return keys
}

// listWordsOfTheContract are the lists' orders and groups as the bundled
// contract names them: the enums IssueListOrder and IssueListGroupField.
func listWordsOfTheContract(t *testing.T) (orders, groups []string) {
	t.Helper()
	doc, err := openapi3.NewLoader().LoadFromFile("../../../api/dist/openapi.yaml")
	if err != nil {
		t.Fatal(err)
	}
	enum := func(name string) []string {
		schema, ok := doc.Components.Schemas[name]
		if !ok || len(schema.Value.Enum) == 0 {
			t.Fatalf("the contract has no enum %s", name)
		}
		values := make([]string, len(schema.Value.Enum))
		for n, v := range schema.Value.Enum {
			values[n] = v.(string)
		}
		return values
	}
	return enum("IssueListOrder"), enum("IssueListGroupField")
}
