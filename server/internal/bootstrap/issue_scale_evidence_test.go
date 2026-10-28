package bootstrap

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"

	"github.com/open-nerve/NerveProject/server/internal/platform/config"
)

type scaleSample struct {
	Method, Path, Body string
	Started, Ended     time.Time
	Milliseconds       float64
	Status             int
}

func (w issueWorld) scaleHTTP(t *testing.T, method, path, body, wantID string) scaleSample {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
	defer cancel()
	req := newRequest(t, method, w.base+path, w.tokens["alice"], []byte(body)).WithContext(ctx)
	w.contract.CheckRequest(t, req)
	s := scaleSample{Method: method, Path: path, Started: time.Now()}
	res, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		t.Fatal(err)
	}
	data, err := io.ReadAll(res.Body)
	_ = res.Body.Close()
	s.Ended = time.Now()
	s.Milliseconds = float64(s.Ended.Sub(s.Started)) / float64(time.Millisecond)
	s.Status = res.StatusCode
	s.Body = string(data)
	if err != nil {
		t.Fatal(err)
	}
	res.Body = io.NopCloser(bytes.NewReader(data))
	w.contract.CheckResponse(t, req, res)
	if wantID != "" {
		var item struct {
			ID string `json:"id"`
		}
		if err = json.Unmarshal(data, &item); err != nil || s.Status != 200 || item.ID != wantID {
			t.Errorf("live probe %s %s status=%d id=%s want200/%s: %s", method, path, s.Status, item.ID, wantID, s.Body)
		}
	}
	return s
}

func (w issueWorld) scaleSamples(t *testing.T, probe uuid.UUID, phase, dir string) []scaleSample {
	t.Helper()
	samples := make([]scaleSample, 0, 40)
	path := "/api/v0/issues/" + probe.String()
	for i := 0; i < 20; i++ {
		read := w.scaleHTTP(t, "GET", path, "", probe.String())
		samples = append(samples, read)
		name := fmt.Sprintf("%s-%d", phase, i)
		write := w.scaleHTTP(t, "PATCH", path, fmt.Sprintf(`{"name":%q}`, name), probe.String())
		samples = append(samples, write)
		listEvidence(t, dir, phase+"-samples", samples)
		var item struct {
			Name string `json:"name"`
		}
		decodeAnswer(t, write.Body, &item)
		if item.Name != name {
			t.Fatalf("probe write did not persist: %s want %s", item.Name, name)
		}
		if read.Status != 200 || write.Status != 200 {
			t.Fatal("probe requires read200 and write200")
		}
	}
	final := w.scaleHTTP(t, "GET", path, "", probe.String())
	listEvidence(t, dir, phase+"-readback", final)
	var item struct {
		Name string `json:"name"`
	}
	decodeAnswer(t, final.Body, &item)
	if item.Name != phase+"-19" {
		t.Fatal("probe final readback", item.Name)
	}
	return samples
}

func scaleSummary(samples []scaleSample) map[string]any {
	out := map[string]any{"quantiles": "nearest-rank: sorted[ceil(p*n)-1]; all samples retained"}
	for _, method := range []string{"GET", "PATCH"} {
		values := []float64{}
		for _, s := range samples {
			if s.Method == method {
				values = append(values, s.Milliseconds)
			}
		}
		slices.Sort(values)
		out[method] = map[string]any{"n": len(values), "p50_ms": values[int(math.Ceil(.50*float64(len(values))))-1], "p95_ms": values[int(math.Ceil(.95*float64(len(values))))-1], "max_ms": values[len(values)-1]}
	}
	return out
}

type scaleProgress struct {
	JobID             int64
	Project           uuid.UUID
	DBBefore, DBAfter time.Time
	QueryMilliseconds float64
	State             observedJob
	Transitions       []jobTransition
	Deleted, Pending  int
	DeletedMarks      map[string][]uuid.UUID
}

// Only small, preselected primary-key sets are observed. These read statements
// neither hold row locks nor slow the worker deliberately.
func (w issueWorld) scaleProgress(t *testing.T, job deletionJob, marks []scaleMark, dir string) scaleProgress {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	start := time.Now()
	out := scaleProgress{JobID: job.ID, Project: job.Parent, DeletedMarks: map[string][]uuid.UUID{}}
	if err := w.pool.QueryRow(ctx, "SELECT clock_timestamp()").Scan(&out.DBBefore); err != nil {
		t.Fatal(err)
	}
	out.State = w.scaleSingleRun(t, job, dir)
	for _, table := range []string{"issues", "issue_activities", "issue_description_versions"} {
		ids := []uuid.UUID{}
		for _, m := range marks {
			if m.Project == job.Parent && m.Table == table {
				ids = append(ids, m.ID)
			}
		}
		rows, err := w.pool.Query(ctx, "SELECT id,deleted_at,updated_at,updated_by_id FROM "+table+" WHERE id=ANY($1) ORDER BY id", ids)
		if err != nil {
			t.Fatal(err)
		}
		seen := 0
		for rows.Next() {
			var id uuid.UUID
			var deleted *time.Time
			var updated time.Time
			var by *uuid.UUID
			if err = rows.Scan(&id, &deleted, &updated, &by); err != nil {
				t.Fatal(err)
			}
			seen++
			if deleted == nil {
				out.Pending++
			} else {
				if !deleted.Equal(job.At) || !updated.Equal(job.At) || by == nil || *by != job.By {
					t.Fatal("marker lost original deletion audit", table, id)
				}
				out.Deleted++
				out.DeletedMarks[table] = append(out.DeletedMarks[table], id)
			}
		}
		if err = rows.Err(); err != nil {
			t.Fatal(err)
		}
		rows.Close()
		if seen != len(ids) {
			t.Fatal("missing scale markers", table, seen, len(ids))
		}
	}
	out.Transitions = jobTransitions(t, w.pool, job.ID)
	if err := w.pool.QueryRow(ctx, "SELECT clock_timestamp()").Scan(&out.DBAfter); err != nil {
		t.Fatal(err)
	}
	out.QueryMilliseconds = float64(time.Since(start)) / float64(time.Millisecond)
	return out
}

func (w issueWorld) scaleProcessingSamples(t *testing.T, probe uuid.UUID, name string, jobs []deletionJob, marks []scaleMark, dir string) {
	t.Helper()
	var before scaleProgress
	var selected deletionJob
	var windows []scaleProgress
	requestEventuallyWithin(t, "SP2 running with committed batch and pending candidates", 30*time.Second, func(context.Context) bool {
		for _, job := range jobs {
			state := w.scaleSingleRun(t, job, dir)
			if job.Scope != "project" || state.State != "running" {
				continue
			}
			p := w.scaleProgress(t, job, marks, dir)
			windows = append(windows, p)
			listEvidence(t, dir, name+"-window-search", windows)
			if p.Deleted > 0 && p.Pending > 0 && p.State.State == "running" && len(p.Transitions) == 1 && p.Transitions[0].To == "running" {
				before = p
				selected = job
				return true
			}
		}
		return false
	})
	samples := w.scaleSamples(t, probe, name+"-processing", dir)
	after := w.scaleProgress(t, selected, marks, dir)
	listEvidence(t, dir, name+"-processing-summary", scaleSummary(samples))
	listEvidence(t, dir, name+"-window", map[string]any{"before": before, "after": after, "selected_job_id": selected.ID})
	if after.State.State != "running" || len(after.Transitions) != 1 || after.Pending == 0 {
		t.Fatalf("SP2 probe escaped real processing window: %+v", after)
	}
	var continued scaleProgress
	requestEventuallyWithin(t, "SP2 committed progress after samples", 10*time.Second, func(context.Context) bool {
		continued = w.scaleProgress(t, selected, marks, dir)
		return continued.Deleted > after.Deleted
	})
	listEvidence(t, dir, name+"-window", map[string]any{"before": before, "after": after, "continued": continued, "selected_job_id": selected.ID})
	if !continued.DBBefore.After(after.DBAfter) {
		t.Fatal("invalid processing watermark order")
	}
	t.Logf("SP2 %s real processing job=%d markers committed %d -> %d -> %d pending_after=%d", name, selected.ID, before.Deleted, after.Deleted, continued.Deleted, after.Pending)
}

func scaleEnvironment(t *testing.T, w issueWorld, c config.Config) map[string]any {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
	defer cancel()
	values := map[string]any{
		"seed_version": "SP2-v1:6x660000;80000x1024-byte description activities/project",
		"measurement":  "intact HTTP app, real authentication and production Runner; observer AFTER UPDATE trigger included; no held locks/cache eviction",
		"go":           runtime.Version(), "os": runtime.GOOS, "arch": runtime.GOARCH, "cpus": runtime.NumCPU(), "gomaxprocs": runtime.GOMAXPROCS(0),
		"server": c.Server, "database": map[string]any{"max_conns": c.Database.MaxConns, "commit_timeout": c.Database.CommitTimeout},
		"jobs": c.Jobs, "issue": c.Issue, "cleanup": c.Cleanup, "token_ttl": c.Auth.AccessTokenTTL,
		"assertion_pool_max_conns": w.pool.Config().MaxConns, "assertion_pool_params": w.pool.Config().ConnConfig.RuntimeParams,
		"argv": os.Args, "NERVE_MEASURE_SCALE": os.Getenv("NERVE_MEASURE_SCALE"),
	}
	var raw json.RawMessage
	if err := w.pool.QueryRow(ctx, `SELECT row_to_json(s) FROM (SELECT version() postgres,current_user role,current_setting('shared_buffers') shared_buffers,current_setting('work_mem') work_mem,current_setting('max_connections') max_connections,current_setting('jit') assertion_jit,current_setting('plan_cache_mode') assertion_plan_cache_mode,current_setting('TimeZone') timezone,current_setting('server_encoding') encoding) s`).Scan(&raw); err != nil {
		t.Fatal(err)
	}
	values["database_environment"] = raw
	rows, err := w.pool.Query(ctx, `SELECT indexdef FROM pg_indexes WHERE schemaname='public' ORDER BY tablename,indexname`)
	if err != nil {
		t.Fatal(err)
	}
	indexes, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		t.Fatal(err)
	}
	values["indexes"] = indexes
	// The app's pool sets these in postgres.NewPool; the owner assertion pool is
	// separately identified above, rather than misreported as the runtime pool.
	values["runtime_pool_params"] = map[string]string{"jit": "off", "plan_cache_mode": "force_custom_plan"}
	_, file, _, _ := runtime.Caller(0)
	root, err := filepath.Abs(filepath.Join(filepath.Dir(file), "../../.."))
	if err != nil {
		t.Fatal(err)
	}
	git := func(args ...string) string {
		cmd := exec.CommandContext(ctx, "git", args...)
		cmd.Dir = root
		out, e := cmd.Output()
		if e != nil {
			t.Fatal(e)
		}
		return strings.TrimSpace(string(out))
	}
	values["cwd"], _ = os.Getwd()
	values["base"] = "4c7379f1d4dd53a8af932f2ae148ae2f9d07f859"
	values["head"] = git("rev-parse", "HEAD")
	values["status"] = git("status", "--porcelain")
	files := strings.Split(git("ls-files", "server"), "\n")
	files = append(files, "server/internal/bootstrap/issue_scale_fixture_test.go", "server/internal/bootstrap/issue_scale_evidence_test.go")
	hashes := map[string]string{}
	for _, name := range files {
		ext := filepath.Ext(name)
		if ext != ".go" && ext != ".sql" && ext != ".yaml" && name != "server/go.mod" && name != "server/go.sum" {
			continue
		}
		data, e := os.ReadFile(filepath.Join(root, name))
		if e != nil {
			t.Fatal(e)
		}
		hashes[name] = fmt.Sprintf("%x", sha256.Sum256(data))
	}
	values["source_sha256"] = hashes
	for _, cmd := range [][]string{{"uname", "-a"}, {"sysctl", "-n", "machdep.cpu.brand_string"}} {
		out, e := exec.CommandContext(ctx, cmd[0], cmd[1:]...).CombinedOutput()
		values[strings.Join(cmd, " ")] = map[string]any{"output": strings.TrimSpace(string(out)), "success": e == nil}
	}
	return values
}
