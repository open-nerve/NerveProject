package bootstrap

import (
	"net/http"
	"reflect"
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

func TestLinkCrawlURLInterleaving(t *testing.T) {
	for _, crawlFirst := range []bool{false, true} {
		name := "URL commits before old crawl"
		if crawlFirst {
			name = "old crawl commits before URL"
		}
		t.Run(name, func(t *testing.T) {
			w, pages := newLinkRaceWorld(t)
			item := w.newWorkItemBy(t, "bob", w.web, `{"name":"Crawl race"}`)
			a := createLink(t, w, "bob", item, pages.url+"/A")
			first := pages.wait(t, "/A")
			jobA := linkRaceJobs(t, w, a.ID, a.URL)[0]
			if crawlFirst {
				activity, audit := descriptionRows(t, w, "issue_activities", item), linkRaceAudit(t, w, item)
				first.reply <- "Preview A"
				waitLinkRaceJob(t, w, jobA, a.ID, a.URL)
				wantLinkRaceMetadata(t, w, a.ID, linkPageMetadata(a.URL, "Preview A"))
				if activity != descriptionRows(t, w, "issue_activities", item) || audit != linkRaceAudit(t, w, item) {
					t.Fatal("first crawl changed business audit or activity")
				}
			}
			b := changeLinkRaceURL(t, w, a, pages.url+"/B")
			second := pages.wait(t, "/B")
			jobs := linkRaceJobs(t, w, a.ID, a.URL, b.URL)
			if jobs[0] != jobA || jobs[1] <= jobA {
				t.Fatal("URL change replaced the first job", jobs)
			}
			activity := descriptionRows(t, w, "issue_activities", item)
			audit := linkRaceAudit(t, w, item)
			if !crawlFirst {
				first.reply <- "Stale preview A"
				waitLinkRaceJob(t, w, jobA, a.ID, a.URL)
				wantLinkRaceMetadata(t, w, a.ID, domain.LinkMetadata{})
			}
			second.reply <- "Preview B"
			waitLinkRaceJob(t, w, jobs[1], a.ID, b.URL)
			wantLinkRaceMetadata(t, w, a.ID, linkPageMetadata(b.URL, "Preview B"))
			if activity != descriptionRows(t, w, "issue_activities", item) || audit != linkRaceAudit(t, w, item) {
				t.Fatal("crawl changed business audit or activity")
			}
			if got := w.activityBy(t, item, w.ids["alice"]); !reflect.DeepEqual(got, []string{"link: " + a.URL + " → " + b.URL}) {
				t.Fatal("URL change activity", got)
			}
			if !reflect.DeepEqual(jobs, linkRaceJobs(t, w, a.ID, a.URL, b.URL)) {
				t.Fatal("crawl changed job identities")
			}
		})
	}
	t.Run("A to B to A accepts old A as approved", testLinkCrawlReturningURL)
}

// id+URL intentionally allows the old A result after A -> B -> A. The later
// completion wins for A, while B can never write metadata belonging to B into A.
func testLinkCrawlReturningURL(t *testing.T) {
	w, pages := newLinkRaceWorld(t)
	item := w.newWorkItem(t)
	a := createLink(t, w, "bob", item, pages.url+"/A")
	oldA := pages.wait(t, "/A")
	firstID := linkRaceJobs(t, w, a.ID, a.URL)[0]
	b := changeLinkRaceURL(t, w, a, pages.url+"/B")
	heldB := pages.wait(t, "/B")
	before := linkRaceJobs(t, w, a.ID, a.URL, b.URL)
	changeLinkRaceURL(t, w, a, a.URL)
	newA := pages.wait(t, "/A")
	ids := linkRaceJobs(t, w, a.ID, a.URL, b.URL, a.URL)
	if ids[0] != firstID || !reflect.DeepEqual(ids[:2], before) || ids[2] <= ids[1] {
		t.Fatal("A to B to A lost distinct job identities", ids)
	}
	activity, audit := descriptionRows(t, w, "issue_activities", item), linkRaceAudit(t, w, item)
	oldA.reply <- "Old A accepted"
	waitLinkRaceJob(t, w, ids[0], a.ID, a.URL)
	wantLinkRaceMetadata(t, w, a.ID, linkPageMetadata(a.URL, "Old A accepted"))
	heldB.reply <- "Stale B rejected"
	waitLinkRaceJob(t, w, ids[1], a.ID, b.URL)
	wantLinkRaceMetadata(t, w, a.ID, linkPageMetadata(a.URL, "Old A accepted"))
	newA.reply <- "Last A wins"
	waitLinkRaceJob(t, w, ids[2], a.ID, a.URL)
	wantLinkRaceMetadata(t, w, a.ID, linkPageMetadata(a.URL, "Last A wins"))
	if activity != descriptionRows(t, w, "issue_activities", item) || audit != linkRaceAudit(t, w, item) {
		t.Fatal("returning-URL crawl changed business audit or activity")
	}
	if !reflect.DeepEqual(ids, linkRaceJobs(t, w, a.ID, a.URL, b.URL, a.URL)) {
		t.Fatal("repeated URL confused historical and current jobs")
	}
}

func TestLinkTitlePreservesConcurrentMetadata(t *testing.T) {
	for _, crawlFirst := range []bool{true, false} {
		name := "crawl commits after title read before title write"
		if !crawlFirst {
			name = "title commits before crawl"
		}
		t.Run(name, func(t *testing.T) {
			w, pages := newLinkRaceWorld(t)
			item := w.newWorkItem(t)
			link := createLink(t, w, "bob", item, pages.url+"/title")
			page := pages.wait(t, "/title")
			jobs := linkRaceJobs(t, w, link.ID, link.URL)
			activity := descriptionRows(t, w, "issue_activities", item)
			clock, finish := startLinkRaceTitle(t, w, link)
			want := linkPageMetadata(link.URL, "Crawled preview")
			if crawlFirst {
				pendingAudit := linkRaceAudit(t, w, item)
				page.reply <- "Crawled preview"
				waitLinkRaceJob(t, w, jobs[0], link.ID, link.URL)
				// The worker committed on a second connection while the title transaction
				// still holds the project lock and has not reached its link UPDATE.
				assertLinkProjectLock(t, w)
				select {
				case <-clock.gate.open:
					t.Fatal("title gate opened before crawl committed")
				default:
				}
				wantLinkRaceMetadata(t, w, link.ID, want)
				if pendingAudit != linkRaceAudit(t, w, item) {
					t.Fatal("crawl changed audit while title request was held")
				}
			}
			got := finish()
			wantAnswer := domain.LinkMetadata{}
			if crawlFirst {
				wantAnswer = want
			}
			if got.Title == nil || *got.Title != "User title" || !reflect.DeepEqual(got.Metadata, wantAnswer) {
				t.Fatalf("title response lost concurrent metadata: %+v; want %+v", got, wantAnswer)
			}
			wantLinkRaceMetadata(t, w, link.ID, wantAnswer)
			audit := linkRaceAudit(t, w, item)
			if !crawlFirst {
				page.reply <- "Crawled preview"
				waitLinkRaceJob(t, w, jobs[0], link.ID, link.URL)
			}
			wantLinkRaceMetadata(t, w, link.ID, want)
			if activity != descriptionRows(t, w, "issue_activities", item) || audit != linkRaceAudit(t, w, item) {
				t.Fatal("title/crawl added URL activity or crawl changed audit")
			}
			if !reflect.DeepEqual(jobs, linkRaceJobs(t, w, link.ID, link.URL)) {
				t.Fatal("title-only change enqueued a crawl")
			}
			listed := listIn[answeredLink](t, w.send(t, "alice", http.MethodGet, linksOf(item), "", 200))
			if len(listed) != 1 || listed[0].Title == nil || *listed[0].Title != "User title" || !reflect.DeepEqual(listed[0].Metadata, want) {
				t.Fatal("final API lost user title or preview", listed)
			}
		})
	}
}

func TestLinkCrawlAfterDeletion(t *testing.T) {
	for _, rebuild := range []bool{false, true} {
		name := "deleted link rejects old result"
		if rebuild {
			name = "same URL new ID rejects old result"
		}
		t.Run(name, func(t *testing.T) {
			w, pages := newLinkRaceWorld(t)
			item := w.newWorkItem(t)
			old := createLink(t, w, "bob", item, pages.url+"/same")
			oldPage := pages.wait(t, "/same")
			oldJob := linkRaceJobs(t, w, old.ID, old.URL)[0]
			w.send(t, "alice", http.MethodDelete, linkPath(old.ID), "", 204)
			var fresh answeredLink
			var freshPage heldLinkPage
			var freshJob int64
			if rebuild {
				fresh = createLink(t, w, "bob", item, old.URL)
				if fresh.ID == old.ID {
					t.Fatal("recreated link reused deleted ID")
				}
				freshPage = pages.wait(t, "/same")
				freshJob = linkRaceJobs(t, w, fresh.ID, fresh.URL)[0]
				if freshJob == oldJob {
					t.Fatal("new link reused old job")
				}
			}
			before := descriptionRows(t, w, "issue_links", item)
			activity, audit := descriptionRows(t, w, "issue_activities", item), linkRaceAudit(t, w, item)
			oldPage.reply <- "Deleted link preview"
			waitLinkRaceJob(t, w, oldJob, old.ID, old.URL)
			if before != descriptionRows(t, w, "issue_links", item) {
				t.Fatal("old crawl changed deleted link or same-URL replacement")
			}
			wantLinkRaceMetadata(t, w, old.ID, domain.LinkMetadata{})
			if rebuild {
				wantLinkRaceMetadata(t, w, fresh.ID, domain.LinkMetadata{})
				freshPage.reply <- "New link preview"
				waitLinkRaceJob(t, w, freshJob, fresh.ID, fresh.URL)
				wantLinkRaceMetadata(t, w, fresh.ID, linkPageMetadata(fresh.URL, "New link preview"))
				wantLinkRaceMetadata(t, w, old.ID, domain.LinkMetadata{})
				if ids := linkRaceJobs(t, w, fresh.ID, fresh.URL); ids[0] != freshJob {
					t.Fatal("replacement job identity changed", ids)
				}
			}
			if activity != descriptionRows(t, w, "issue_activities", item) || audit != linkRaceAudit(t, w, item) {
				t.Fatal("post-deletion crawl changed business audit or activity")
			}
			if ids := linkRaceJobs(t, w, old.ID, old.URL); ids[0] != oldJob {
				t.Fatal("deletion enqueued or replaced a job", ids)
			}
		})
	}
}
