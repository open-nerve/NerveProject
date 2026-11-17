package bootstrap

import (
	"context"
	"fmt"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue"
	"github.com/open-nerve/NerveProject/server/internal/modules/notification"
	notificationriver "github.com/open-nerve/NerveProject/server/internal/modules/notification/adapter/river"
	"github.com/open-nerve/NerveProject/server/internal/modules/planning"
	"github.com/open-nerve/NerveProject/server/internal/platform/clock/clocktest"
	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/jobs"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// archiveScaleItems are the work items one round of the auto-archive
// archives in a project of the A/B below: v0's target, a project of at
// most about 20,000 undeleted work items (M4 design, decision 7), every
// one done and inactive past the cutoff, as on the first round after a
// project's auto-archive is turned on. It is the most one project's
// transaction archives, under the project's lock, each work item with its
// event, and since P1b with its event's fan-out enqueued (M7 P1b spec 6).
const archiveScaleItems = 20000

// archiveScaleDay is the round's clock: ahead of the real one, so that the
// app's own runs leave the work items be (archiveAhead).
var archiveScaleDay = time.Date(2044, 6, 15, 12, 0, 0, 0, time.UTC)

// archiveScaleRound is one round of the A/B: its project, whether its
// publisher has the notification module's subscribers, the work items it
// archived, the fan-outs it enqueued, its time and its error.
type archiveScaleRound struct {
	Project            uuid.UUID
	Notifications      bool
	Archived, Fanouts  int
	Seconds, PerItemMS float64
	Err                string
}

// TestTheAutoArchiveRoundWithAndWithoutNotifications times a round of
// issue.auto_archive, wired as autoArchiveOn on a pool with the app's
// planner settings, in one run, with the app's publisher and with it but
// for the notification module's subscribers (M7 P1b spec R9), as the
// freeze's scale times its run. Each round has a project of its own of
// archiveScaleItems work items to archive, the only project whose
// auto-archive is on meanwhile, so that a round is that one project's
// transaction; in the order without, with, with, without, so that a
// round's place in the run weighs on both alike. The fan-outs the rounds
// with notifications enqueue are held (holdFanouts) so that none runs
// beside a later round: the hold's trigger runs at each of their inserts,
// which it makes dearer, if anything.
func TestTheAutoArchiveRoundWithAndWithoutNotifications(t *testing.T) {
	dir := measuredScale(t)
	t.Log("auto-archive scale evidence:", dir)
	db := pgtest.NewDatabase(t)
	pool := openPool(t, db)
	holdFanouts(t, pool)
	var effective config.Config
	w := newIssueWorldOn(t, db, pool, func(c *config.Config) {
		c.Auth.AccessTokenTTL = time.Hour
		effective = *c
	})
	waitForStartupRuns(t, w.pool)
	environment := scaleEnvironment(t, w, effective)
	environment["base"] = "9a3a388a7a3be57df0c4da76d539ed7a5cead34a"
	environment["seed_version"] = fmt.Sprintf("auto-archive-v1: four projects of acme, %d done work items each, inactive 60 days at the round", archiveScaleItems)
	environment["measurement"] = "issue.auto_archive's use case as autoArchiveOn wires it, on a pool with the app's planner settings; the app's publisher with and without notification's subscribers"
	listEvidence(t, dir, "auto-archive-environment", environment)
	with := []bool{false, true, true, false}
	projects := make([]uuid.UUID, len(with))
	for i := range projects {
		projects[i] = createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", fmt.Sprintf("Archive %d", i), fmt.Sprintf("ARC%d", i))
	}
	seeded := seedArchiveScale(t, w, projects)
	listEvidence(t, dir, "auto-archive-fixture", map[string]any{"projects": projects, "items_per_project": archiveScaleItems, "day": archiveScaleDay,
		"seed_seconds": seeded.Seconds()})
	run := w
	run.pool = freezeScalePool(t, db, nil)
	without, notifying := archivePublishers(t, run.pool)
	rounds := make([]archiveScaleRound, len(projects))
	for i, p := range projects {
		publisher := without
		if with[i] {
			publisher = notifying
		}
		turnAutoArchive(t, w.pool, p, 1)
		start := time.Now()
		n, err := autoArchiveOn(run, clocktest.At(archiveScaleDay), publisher).Execute(t.Context())
		r := archiveScaleRound{Project: p, Notifications: with[i], Archived: n, Seconds: time.Since(start).Seconds()}
		turnAutoArchive(t, w.pool, p, 0)
		if err != nil {
			r.Err = err.Error()
		}
		r.PerItemMS = r.Seconds * 1000 / archiveScaleItems
		if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT count(*) FROM river_job WHERE kind = $1 AND args->>'project_id' = $2",
			notificationriver.FanoutKind, p.String()).Scan(&r.Fanouts); err != nil {
			t.Fatal(err)
		}
		rounds[i] = r
		listEvidence(t, dir, "auto-archive-rounds", rounds)
		want := 0
		if with[i] {
			want = archiveScaleItems
		}
		if r.Err != "" || r.Archived != archiveScaleItems || r.Fanouts != want {
			t.Fatalf("round %d: %+v; want %d archived, %d fan-outs", i, r, archiveScaleItems, want)
		}
	}
	mean := func(notifications bool) float64 {
		var sum, n float64
		for _, r := range rounds {
			if r.Notifications == notifications {
				sum, n = sum+r.Seconds, n+1
			}
		}
		return sum / n
	}
	summary := map[string]any{"without_seconds": mean(false), "with_seconds": mean(true),
		"added_ms_per_item": (mean(true) - mean(false)) * 1000 / archiveScaleItems, "rounds": rounds}
	listEvidence(t, dir, "auto-archive-summary", summary)
	t.Logf("auto-archive scale: %d work items a round; without notifications %.3f s and %.3f s, with %.3f s and %.3f s; %.3f ms more an item",
		archiveScaleItems, rounds[0].Seconds, rounds[3].Seconds, rounds[1].Seconds, rounds[2].Seconds, summary["added_ms_per_item"])
}

// seedArchiveScale writes archiveScaleItems work items in each of
// projects, interleaved as seedAtScale's, each in its project's Done,
// completed and last changed 60 days before archiveScaleDay, then analyzes
// them.
func seedArchiveScale(t *testing.T, w issueWorld, projects []uuid.UUID) time.Duration {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Minute)
	defer cancel()
	start := time.Now()
	inactive := archiveScaleDay.AddDate(0, 0, -60)
	if _, err := w.pool.Exec(ctx, `WITH s AS MATERIALIZED (SELECT p.id AS project_id, p.workspace_id, st.id AS state_id, n
			FROM projects p JOIN states st ON st.project_id = p.id AND st.name = 'Done' AND st.deleted_at IS NULL, generate_series(1, $2::int) n
			WHERE p.id = ANY ($1) ORDER BY n, p.id)
		INSERT INTO issues (id, workspace_id, project_id, state_id, name, sequence_id, completed_at, created_by_id, updated_by_id, created_at, updated_at)
		SELECT uuidv7(), workspace_id, project_id, state_id, 'Archive ' || n, n, $3, $4, $4, $3, $3 FROM s`,
		projects, archiveScaleItems, inactive, w.ids["alice"]); err != nil {
		t.Fatal(err)
	}
	if _, err := w.pool.Exec(ctx, "UPDATE projects SET last_issue_sequence = $2 WHERE id = ANY ($1)", projects, archiveScaleItems); err != nil {
		t.Fatal(err)
	}
	if _, err := w.pool.Exec(ctx, "VACUUM (ANALYZE) issues, projects, states"); err != nil {
		t.Fatal(err)
	}
	var items int
	if err := w.pool.QueryRow(ctx, "SELECT count(*) FROM issues WHERE project_id = ANY ($1) AND archived_at IS NULL", projects).Scan(&items); err != nil ||
		items != len(projects)*archiveScaleItems {
		t.Fatalf("seeded work items %d, %v; want %d", items, err, len(projects)*archiveScaleItems)
	}
	return time.Since(start)
}

// turnAutoArchive sets the project's archive_in, in months: 0 turns its
// auto-archive off.
func turnAutoArchive(t *testing.T, pool *pgxpool.Pool, project uuid.UUID, months int) {
	t.Helper()
	if _, err := pool.Exec(pgtest.Soon(t), "UPDATE projects SET archive_in = $2 WHERE id = $1", project, months); err != nil {
		t.Fatal(err)
	}
}

// archivePublishers are the app's publisher over pool, its subscribers as
// newApp registers them, and the same without the notification module's.
func archivePublishers(t *testing.T, pool *pgxpool.Pool) (without, with shared.EventBatchPublisher) {
	t.Helper()
	client, err := jobs.NewInserter(pool)
	if err != nil {
		t.Fatal(err)
	}
	inserter := jobInserter{client: client}
	others := slices.Concat(issue.Subscribers(issue.Provide(pool, protectedAssetReferences()), inserter), planning.Subscribers(inserter),
		asset.Subscribers(asset.Provide(pool), inserter))
	plain, err := newDispatcher(eventTypes(), others)
	if err != nil {
		t.Fatal(err)
	}
	full, err := newDispatcher(eventTypes(), slices.Concat(others, notification.Subscribers(notification.Provide(pool), inserter)))
	if err != nil {
		t.Fatal(err)
	}
	return plain, full
}
