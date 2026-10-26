package bootstrap

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// A module left on System (or an offset applied only to the worker) writes
// outside the shifted interval. The timezone endpoint makes instance's clock
// observable too: Dublin differs between these two hand-chosen seasons.
func TestBusinessClockServe(t *testing.T) {
	for _, tt := range []struct {
		name      string
		yearDelta int
		month     time.Month
		dublin    string
	}{
		{"future winter", 1, time.January, "UTC+00:00"},
		{"past summer", -1, time.July, "UTC+01:00"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			realStart := time.Now()
			target := time.Date(realStart.Year()+tt.yearDelta, tt.month, 15, 12, 0, 0, 0, time.UTC)
			offset := target.Sub(realStart)
			w := newIssueWorldWith(t, func(cfg *config.Config) { cfg.Clock.Offset = offset })
			id := w.newWorkItem(t)
			end := time.Now().Add(offset)
			for _, row := range []struct {
				name, query string
				id          any
			}{
				{"identity", "SELECT created_at FROM users WHERE id=$1", w.ids["alice"]},
				{"session", "SELECT created_at FROM auth_sessions WHERE user_id=$1", w.ids["alice"]},
				{"workspace", "SELECT created_at FROM workspaces WHERE id=$1", w.acme},
				{"project", "SELECT created_at FROM projects WHERE id=$1", w.web},
				{"issue", "SELECT created_at FROM issues WHERE id=$1", id},
				{"activity", "SELECT created_at FROM issue_activities WHERE issue_id=$1", id},
			} {
				t.Run(row.name, func(t *testing.T) { assertBusinessTime(t, w.pool, row.query, row.id, target, end) })
			}
			body := w.send(t, "alice", http.MethodGet, "/api/v0/timezones", "", http.StatusOK)
			var zones struct {
				Data []struct {
					Value  string `json:"value"`
					Offset string `json:"utc_offset"`
				} `json:"data"`
			}
			if err := json.Unmarshal([]byte(body), &zones); err != nil {
				t.Fatal(err)
			}
			found := false
			for _, zone := range zones.Data {
				if zone.Value == "Europe/Dublin" {
					found = true
					if zone.Offset != tt.dublin {
						t.Errorf("Dublin offset = %s, want %s", zone.Offset, tt.dublin)
					}
				}
			}
			if !found {
				t.Fatal("Dublin missing")
			}
		})
	}
}

func assertBusinessTime(t *testing.T, pool *pgxpool.Pool, query string, id any, before, after time.Time) {
	t.Helper()
	var at time.Time
	if err := pool.QueryRow(pgtest.Soon(t), query, id).Scan(&at); err != nil {
		t.Fatal(err)
	}
	if at.Before(before.Truncate(time.Microsecond)) || at.After(after) {
		t.Errorf("business timestamp %s outside [%s, %s]", at, before, after)
	}
	if at.Nanosecond()%1000 != 0 {
		t.Errorf("timestamp %s has finer than microsecond precision", at)
	}
}

// Both administrator compositions must consume the configured clock, including
// Users' nested workspace deactivator. Changing that one Clock back to System
// leaves the membership's update at the wrong instant.
func TestBusinessClockCommands(t *testing.T) {
	url := pgtest.NewDatabase(t)
	pool := openPool(t, url)
	cfg := testConfig(t, url, false)
	cfg.Clock.Offset = 72 * time.Hour
	before := time.Now().Add(cfg.Clock.Offset)
	if err := Users(context.Background(), cfg, io.Discard, io.Discard, CreateUser("clock@example.com", "Tr0ub4dor&3")); err != nil {
		t.Fatal(err)
	}
	if err := Workspaces(context.Background(), cfg, io.Discard, io.Discard, CreateWorkspace("clock", "Clock", "clock@example.com")); err != nil {
		t.Fatal(err)
	}
	after := time.Now().Add(cfg.Clock.Offset)
	assertBusinessTime(t, pool, "SELECT created_at FROM users WHERE email=$1", "clock@example.com", before, after)
	assertBusinessTime(t, pool, "SELECT created_at FROM workspaces WHERE slug=$1", "clock", before, after)
	cfg.Clock.Offset = -72 * time.Hour
	before = time.Now().Add(cfg.Clock.Offset)
	if err := Users(context.Background(), cfg, io.Discard, io.Discard, DeactivateUser("clock@example.com")); err != nil {
		t.Fatal(err)
	}
	after = time.Now().Add(cfg.Clock.Offset)
	assertBusinessTime(t, pool, "SELECT updated_at FROM users WHERE email=$1", "clock@example.com", before, after)
	assertBusinessTime(t, pool, "SELECT updated_at FROM workspace_members WHERE workspace_id=(SELECT id FROM workspaces WHERE slug=$1)", "clock", before, after)
}
