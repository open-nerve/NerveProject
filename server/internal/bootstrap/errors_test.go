package bootstrap

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
	"time"

	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// shared.Error reaches the platform only by structure (M2 design 3.11): each
// Kind becomes its status, with the error's code, detail, fields and
// Retry-After, even wrapped.
func TestEveryKindBecomesItsProblem(t *testing.T) {
	tests := []struct {
		err        error
		want       string
		retryAfter string
	}{
		{shared.Invalid(shared.FieldError{Field: "email", Code: shared.FieldRequired, Message: "is required"}),
			`{"status":422,"code":"validation_failed","title":"Unprocessable Entity","detail":"The request has invalid values.",` +
				`"errors":[{"field":"email","code":"required","message":"is required"}]}`, ""},
		{shared.NewError(shared.KindBadRequest, "things.bad_cursor", "d"), `{"status":400,"code":"things.bad_cursor","title":"Bad Request","detail":"d"}`, ""},
		{shared.Unauthenticated(), `{"status":401,"code":"unauthorized","title":"Unauthorized","detail":"Authentication is required."}`, ""},
		{shared.NewError(shared.KindForbidden, "things.forbidden", "d"), `{"status":403,"code":"things.forbidden","title":"Forbidden","detail":"d"}`, ""},
		{shared.NewError(shared.KindNotFound, "things.missing", "d"), `{"status":404,"code":"things.missing","title":"Not Found","detail":"d"}`, ""},
		{shared.NewError(shared.KindConflict, "things.taken", "d"), `{"status":409,"code":"things.taken","title":"Conflict","detail":"d"}`, ""},
		{&shared.Error{Kind: shared.KindRateLimited, Code: "rate_limited", Detail: "d", RetryDelay: 1500 * time.Millisecond},
			`{"status":429,"code":"rate_limited","title":"Too Many Requests","detail":"d"}`, "2"},
		{fmt.Errorf("register: %w", shared.ServerBusy(time.Second)),
			`{"status":503,"code":"server_busy","title":"Service Unavailable","detail":"The server is busy; retry shortly."}`, "1"},
	}
	errs := httpserver.NewAPIErrors(slog.New(slog.DiscardHandler))
	for _, tt := range tests {
		rec := httptest.NewRecorder()

		errs.Write(rec, httptest.NewRequest(http.MethodGet, "/api/v0/things", nil), tt.err)

		if got := rec.Body.String(); got != tt.want+"\n" || rec.Header().Get("Retry-After") != tt.retryAfter {
			t.Errorf("Write(%v) = %s Retry-After %q, want %s Retry-After %q", tt.err, got, rec.Header().Get("Retry-After"), tt.want, tt.retryAfter)
		}
	}
}

// The field codes of internal/shared and the contract's FieldError.code enum
// are one closed set (M2 design 3.11): a code on only one side fails.
func TestFieldCodesAreTheContractsEnum(t *testing.T) {
	enum := slices.Sorted(slices.Values(apitest.Load(t).Enum(t, "FieldError", "code")))
	codes := slices.Sorted(slices.Values(shared.FieldCodes()))

	if !slices.Equal(codes, enum) {
		t.Errorf("shared.FieldCodes() = %q, want the contract's FieldError.code enum %q", codes, enum)
	}
}
