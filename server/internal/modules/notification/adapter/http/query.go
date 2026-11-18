package httpadapter

import (
	"context"
	"maps"
	"net/http"
	"net/url"
	"slices"

	"github.com/open-nerve/NerveProject/server/internal/modules/notification/adapter/http/gen"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// ownQuery is the query parameters each workspace operation takes (P1c
// spec 2.3, 2.4), by operation: the list's filters, limit and cursor; the
// unread counts and mark-all-read none. A change of a notification has no
// entry: it reads no query, as no other module's change does.
var ownQuery = map[string][]string{
	"ListNotifications":          {"tab", "type", "read", "archived", "snoozed", "limit", "cursor"},
	"GetNotificationUnreadCount": nil,
	"MarkAllNotificationsRead":   nil,
}

// listPattern is the list's route, whose query the generated code binds
// before any middleware runs.
const listPattern = "GET /api/v0/workspaces/{slug}/notifications"

// checkOwnQuery holds the raw query to own, the drafts' list's practice
// (P1c spec 2.3): a query that does not decode is 400 bad_request, as a
// list must never drop a pair it cannot read and run a broader query; a
// parameter not in own is 422 not_allowed on it, and one of own given more
// than once 422 invalid_format, every such parameter in one 422, in name
// order.
func checkOwnQuery(raw string, own []string) error {
	query, err := url.ParseQuery(raw)
	if err != nil {
		return shared.NewError(shared.KindBadRequest, shared.CodeBadRequest, "The request query could not be decoded.")
	}
	var fields []shared.FieldError
	for _, name := range slices.Sorted(maps.Keys(query)) {
		switch {
		case !slices.Contains(own, name):
			fields = append(fields, shared.FieldError{Field: name, Code: shared.FieldNotAllowed, Message: "is not allowed for this operation"})
		case len(query[name]) != 1:
			fields = append(fields, shared.FieldError{Field: name, Code: shared.FieldInvalidFormat, Message: "must be given once"})
		}
	}
	if len(fields) > 0 {
		return shared.Invalid(fields...)
	}
	return nil
}

// withOwnQuery checks a workspace operation's query (checkOwnQuery) after
// authentication and the body's structure, before its handler; the
// change's goes through.
func withOwnQuery(next gen.StrictHandlerFunc, operationID string) gen.StrictHandlerFunc {
	own, ok := ownQuery[operationID]
	if !ok {
		return next
	}
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request, request any) (any, error) {
		if err := checkOwnQuery(r.URL.RawQuery, own); err != nil {
			return nil, err
		}
		return next(ctx, w, r, request)
	}
}

// listBindingError answers a parameter the generated code could not bind:
// on the list, whose binding refuses a parameter given twice before the
// strict middleware runs, the list's query check first, as the drafts'
// list does, so a parameter given twice or not the list's is the same 422
// there as everywhere; else, and on every other route, api.Errors'
// BadRequest.
func listBindingError(api *httpserver.API) func(http.ResponseWriter, *http.Request, error) {
	return func(w http.ResponseWriter, r *http.Request, err error) {
		if r.Pattern == listPattern {
			if queryErr := checkOwnQuery(r.URL.RawQuery, ownQuery["ListNotifications"]); queryErr != nil {
				api.Errors.Write(w, r, queryErr)
				return
			}
		}
		api.Errors.BadRequest(w, r, err)
	}
}
