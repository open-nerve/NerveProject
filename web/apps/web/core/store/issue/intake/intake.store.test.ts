/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { IntakeCounts, IntakeIssuePage, Workspace } from "@nerve/api-client";
import { ApiError } from "@/lib/api-error";
import { FakeNerve, answered, json, problem } from "@/lib/auth/fake-nerve";
import { settle, track, until } from "@/lib/auth/fake-time";
import { countsOf, intakeIssueOf, pageOf } from "@/store/issue/intake/fake-intake";
import type { IntakeFilters } from "@/store/issue/intake/filters";
import type { IIntakeStore } from "@/store/issue/intake/intake.store";
import { loadWorkspaces, workspaceOf } from "@/store/workspace/fake-workspaces";

vi.mock("@/lib/auth/api-client", () => ({ tokenManager: {}, publicClient: {} }));
vi.mock("@/lib/store-context", () => ({ rootStore: {} }));

const { RootStore } = await import("@/store/root.store");

// The pending work items of the caller's intakes in each of his workspaces (P5a spec 2.3, R7), and the lists of each
// project's intake (2.3, R6), as the intake store of a session's issue root holds them, against a fake nerve that
// answers each request when the test says. The tab's caller has acme and beta; the store keeps each workspace's counts
// by its id, and its readers name the workspace by the address's slug, as the caller's workspaces list it now (v0
// design 7.7); it keeps each project's lists by the project's id and the filters' key.

const acme = workspaceOf("acme");
const beta = workspaceOf("beta");
const countsPath = (workspace: Pick<Workspace, "slug">) => `/api/v0/workspaces/${workspace.slug}/intake-counts`;

/** The intake store of a session whose caller's workspaces nerve listed as acme and beta; the requests so far forgotten. */
async function intakeTab() {
  const nerve = new FakeNerve();
  const api = nerve.client();
  const app = new RootStore(api);
  await loadWorkspaces(nerve, app.workspaceRoot, [acme, beta]);
  nerve.calls.length = 0;
  return { nerve, api, app, store: app.issue.intake };
}

/** The store fetches the workspace's counts, and nerve gives counts. */
function loadCounts(nerve: FakeNerve, store: IIntakeStore, workspace: Workspace, counts: IntakeCounts) {
  return answered(nerve, () => store.fetchCounts(workspace), ["GET", countsPath(workspace)], counts, "the counts");
}

beforeEach(() => {
  vi.useFakeTimers({ now: 1_000_000 });
});
afterEach(() => {
  vi.useRealTimers();
});

describe("IntakeStore, the pending counts", () => {
  it("keeps each workspace's counts apart, by its id, and gives a project's by the address's slug", async () => {
    const { nerve, store } = await intakeTab();
    expect(store.pendingCount(acme.slug, "p-web")).toBeUndefined();
    const fetched = await loadCounts(nerve, store, acme, countsOf({ "p-web": 3, "p-api": 0 }));
    expect(nerve.calls[0]?.query).toEqual({});
    expect(fetched.value).toEqual(countsOf({ "p-web": 3, "p-api": 0 }));
    expect(store.pendingCount(acme.slug, "p-web")).toBe(3);
    expect(store.pendingCount(acme.slug, "p-api")).toBe(0);
    // a project nerve did not count (intake off, archived, not his) has none
    expect(store.pendingCount(acme.slug, "p-ops")).toBeUndefined();
    expect(store.pendingCount(beta.slug, "p-web")).toBeUndefined();

    await loadCounts(nerve, store, beta, countsOf({ "p-docs": 2 }));
    expect(store.pendingCount(beta.slug, "p-docs")).toBe(2);
    expect(store.pendingCount(acme.slug, "p-docs")).toBeUndefined();
    expect(store.pendingCount(acme.slug, "p-web")).toBe(3);
    // acme's again: its own counts change, beta's stay
    await loadCounts(nerve, store, acme, countsOf({ "p-web": 1 }));
    expect(store.pendingCount(acme.slug, "p-web")).toBe(1);
    expect(store.pendingCount(acme.slug, "p-api")).toBeUndefined();
    expect(store.pendingCount(beta.slug, "p-docs")).toBe(2);
  });

  it("gives nothing for a slug the caller's workspaces do not name now: never his, left, or another workspace's since", async () => {
    const { nerve, app, store } = await intakeTab();
    await loadCounts(nerve, store, acme, countsOf({ "p-web": 3 }));
    expect(store.pendingCount("gamma", "p-web")).toBeUndefined();

    // he leaves acme: the counts the store holds of it no longer show
    await loadWorkspaces(nerve, app.workspaceRoot, [beta]);
    expect(store.pendingCount(acme.slug, "p-web")).toBeUndefined();

    // a new workspace takes the slug acme: it is another workspace, of no counts fetched yet
    const newAcme = workspaceOf("acme", { id: "id-acme-new" });
    await loadWorkspaces(nerve, app.workspaceRoot, [newAcme, beta]);
    expect(store.pendingCount("acme", "p-web")).toBeUndefined();
    await loadCounts(nerve, store, newAcme, countsOf({ "p-web": 5 }));
    expect(store.pendingCount("acme", "p-web")).toBe(5);
  });

  it("writes no answer of a fetch a newer one overtook, whichever nerve answers first", async () => {
    const { nerve, store } = await intakeTab();
    const path = countsPath(acme);
    const older = track(store.fetchCounts(acme));
    const newer = track(store.fetchCounts(acme));
    await until(() => nerve.calls.length === 2, "both fetches");
    expect(nerve.calls.map((call) => call.path)).toEqual([path, path]);
    nerve.calls[0]?.answer(json(200, countsOf({ "p-web": 9 })));
    await until(() => older.settled, "the older answer");
    expect(older.value).toBeUndefined();
    expect(store.pendingCount(acme.slug, "p-web")).toBeUndefined();
    nerve.calls[1]?.answer(json(200, countsOf({ "p-web": 1 })));
    await until(() => newer.settled, "the newer answer");
    expect(newer.value).toEqual(countsOf({ "p-web": 1 }));
    expect(store.pendingCount(acme.slug, "p-web")).toBe(1);

    // the other order: the newer answers first, and the older, answered after it, writes nothing
    const late = track(store.fetchCounts(acme));
    const latest = track(store.fetchCounts(acme));
    await until(() => nerve.calls.length === 4, "both fetches again");
    nerve.calls[3]?.answer(json(200, countsOf({ "p-web": 2 })));
    await until(() => latest.settled, "the newest answer");
    nerve.calls[2]?.answer(json(200, countsOf({ "p-web": 7 })));
    await until(() => late.settled, "the late answer");
    expect(late.value).toBeUndefined();
    expect(store.pendingCount(acme.slug, "p-web")).toBe(2);
  });

  it("hands nerve's refusal to the caller, the counts as they were; gives nothing once the session changed", async () => {
    const { nerve, api, store } = await intakeTab();
    await loadCounts(nerve, store, acme, countsOf({ "p-web": 3 }));
    const refused = track(store.fetchCounts(acme));
    await until(() => nerve.calls.length === 2, "the fetch");
    nerve.calls[1]?.answer(problem(404, "workspace.not_found"));
    await until(() => refused.settled, "the refusal");
    expect(refused.error).toBeInstanceOf(ApiError);
    expect(refused.error).toMatchObject({ status: 404, problem: { code: "workspace.not_found" } });
    expect(store.pendingCount(acme.slug, "p-web")).toBe(3);

    // another session took the tab: the fetch is no failure, it gives nothing, and nothing changes
    FakeNerve.replaceSession(api);
    const cut = track(store.fetchCounts(acme));
    await until(() => cut.settled, "the cut fetch");
    expect(cut).toEqual({ settled: true, value: undefined });
    expect(nerve.calls).toHaveLength(2);
    expect(store.pendingCount(acme.slug, "p-web")).toBe(3);
  });
});

/** The open tab as the intake page opens it: the pending, newest first. */
const open: IntakeFilters = {
  tab: "open",
  statuses: ["pending"],
  priorities: [],
  assignees: [],
  creators: [],
  labels: [],
  order: "-created_at",
};
const snoozedToo: IntakeFilters = { ...open, statuses: ["pending", "snoozed"] };
const [one, two, three, four, five] = ["one", "two", "three", "four", "five"].map((name) => intakeIssueOf(name));
const listPath = (projectId: string) => `/api/v0/projects/${projectId}/intake-issues`;

/** The store fetches the first page of the project's intake the filters select, in acme, and nerve gives page. */
function loadList(
  nerve: FakeNerve,
  store: IIntakeStore,
  projectId: string,
  filters: IntakeFilters,
  page: IntakeIssuePage
) {
  const fetch = () => store.fetchList(acme, projectId, filters);
  return answered(nerve, fetch, ["GET", listPath(projectId)], page, "the list");
}

describe("IntakeStore, the lists", () => {
  it("keeps each first page by the project's id and the filters' key, whatever the order of their values", async () => {
    const { nerve, store } = await intakeTab();
    expect(store.getList("p-web", open)).toBeUndefined();
    const page = pageOf([one, two], "c-1", 3);
    const fetched = await loadList(nerve, store, "p-web", open, page);
    expect(nerve.calls[0]?.query).toEqual({ status: "pending", order_by: "-created_at", limit: "50" });
    expect(fetched.value).toEqual(page);
    expect(store.getList("p-web", open)).toEqual(page);
    expect(store.getList("p-web", snoozedToo)).toBeUndefined();
    expect(store.getList("p-api", open)).toBeUndefined();

    // another project's list of the same filters is its own
    const theirs = pageOf([intakeIssueOf("api-one", { project_id: "p-api" })]);
    await loadList(nerve, store, "p-api", open, theirs);
    expect(store.getList("p-api", open)).toEqual(theirs);
    expect(store.getList("p-web", open)).toEqual(page);

    // the key names what is fetched: the statuses in another order, one named twice, name the same list, asked in order
    const both = pageOf([three]);
    await loadList(nerve, store, "p-web", { ...open, statuses: ["snoozed", "pending", "snoozed"] }, both);
    expect(nerve.calls[2]?.query).toMatchObject({ status: "pending,snoozed" });
    expect(store.getList("p-web", snoozedToo)).toEqual(both);
    expect(store.getList("p-web", open)).toEqual(page);
  });

  it("reads the page after the held page's cursor: adds the rows it lacks, the answer's total and cursor", async () => {
    const { nerve, store } = await intakeTab();
    const path = listPath("p-web");
    await loadList(nerve, store, "p-web", snoozedToo, pageOf([one, two], "c-1", 4));
    // a work item submitted since pushed two onto the next page: it shows once
    const next = pageOf([two, three], "c-2", 5);
    const more = await answered(
      nerve,
      () => store.fetchMore("p-web", snoozedToo),
      ["GET", path],
      next,
      "the next page"
    );
    expect(nerve.calls[1]?.query).toEqual({
      status: "pending,snoozed",
      order_by: "-created_at",
      limit: "50",
      cursor: "c-1",
    });
    expect(more.value).toEqual(pageOf([one, two, three], "c-2", 5));
    expect(store.getList("p-web", snoozedToo)).toEqual(pageOf([one, two, three], "c-2", 5));

    const last = pageOf([four], null, 5);
    await answered(nerve, () => store.fetchMore("p-web", snoozedToo), ["GET", path], last, "the last page");
    expect(nerve.calls[2]?.query.cursor).toBe("c-2");
    expect(store.getList("p-web", snoozedToo)).toEqual(pageOf([one, two, three, four], null, 5));
    // the last page held, or a list not fetched: there is nothing to ask
    expect(await settle(store.fetchMore("p-web", snoozedToo), "past the last")).toEqual({
      settled: true,
      value: undefined,
    });
    expect(await settle(store.fetchMore("p-api", snoozedToo), "a list not fetched")).toEqual({
      settled: true,
      value: undefined,
    });
    expect(nerve.calls).toHaveLength(3);
  });

  it("a refresh reads the first page in place of every page held; a next page out then writes nothing", async () => {
    const { nerve, store } = await intakeTab();
    const path = listPath("p-web");
    await loadList(nerve, store, "p-web", open, pageOf([one, two], "c-1", 5));
    await answered(nerve, () => store.fetchMore("p-web", open), ["GET", path], pageOf([three], "c-2", 5), "page two");
    const more = track(store.fetchMore("p-web", open));
    await until(() => nerve.calls.length === 3, "the page after");
    expect(nerve.calls[2]?.query.cursor).toBe("c-2");

    // two and three were triaged in another tab: the refresh lists neither, nor any row of the pages after the first
    const refreshed = pageOf([four, one], "c-9", 3);
    const refresh = track(store.fetchList(acme, "p-web", open));
    await until(() => nerve.calls.length === 4, "the refresh");
    expect(Object.keys(nerve.calls[3]?.query ?? {})).not.toContain("cursor");
    nerve.calls[3]?.answer(json(200, refreshed));
    await until(() => refresh.settled, "the refresh");
    expect(store.getList("p-web", open)).toEqual(refreshed);
    nerve.calls[2]?.answer(json(200, pageOf([five], "c-3", 5)));
    await until(() => more.settled, "the page the refresh superseded");
    expect(more.value).toBeUndefined();
    expect(store.getList("p-web", open)).toEqual(refreshed);
  });

  it("a next page sent while a refresh is out adds nothing once the refresh ends elsewhere (P4 I1)", async () => {
    const { nerve, store } = await intakeTab();
    const path = listPath("p-web");
    await loadList(nerve, store, "p-web", open, pageOf([one, two], "c-1", 5));
    await answered(nerve, () => store.fetchMore("p-web", open), ["GET", path], pageOf([three], "c-2", 5), "page two");

    // the refresh is out as the caller scrolls on: that read names the cursor of the pages the refresh replaces
    const refresh = track(store.fetchList(acme, "p-web", open));
    await until(() => nerve.calls.length === 3, "the refresh");
    const more = track(store.fetchMore("p-web", open));
    await until(() => nerve.calls.length === 4, "the page after the held pages");
    expect(nerve.calls[3]?.query.cursor).toBe("c-2");

    const refreshed = pageOf([four, one], "c-9", 3);
    nerve.calls[2]?.answer(json(200, refreshed));
    await until(() => refresh.settled, "the refresh");
    nerve.calls[3]?.answer(json(200, pageOf([five], "c-3", 5)));
    await until(() => more.settled, "the page after the pages the refresh replaced");
    // the refreshed first page ends at c-9: the page after c-2 is not its next, so the list skips nothing
    expect(store.getList("p-web", open)).toEqual(refreshed);
  });

  it("writes a project's late answers into its own list alone: another project's rows and cursor stay", async () => {
    const { nerve, store } = await intakeTab();
    const theirs = pageOf([intakeIssueOf("api-one", { project_id: "p-api" })], "ca-1", 2);
    const ours = pageOf([one], "cw-1", 2);
    // p-web's first page is out as the caller moves to p-api, whose page nerve answers first
    const webList = track(store.fetchList(acme, "p-web", open));
    await until(() => nerve.calls.length === 1, "p-web's list");
    await loadList(nerve, store, "p-api", open, theirs);
    nerve.calls[0]?.answer(json(200, ours));
    await until(() => webList.settled, "p-web's late list");
    expect(store.getList("p-api", open)).toEqual(theirs);
    expect(store.getList("p-web", open)).toEqual(ours);

    // p-web's next page, out as the caller moves to p-api again: it lands in p-web's list alone
    const webMore = track(store.fetchMore("p-web", open));
    await until(() => nerve.calls.length === 3, "p-web's next page");
    expect(nerve.calls[2]).toMatchObject({ path: listPath("p-web"), query: { cursor: "cw-1" } });
    nerve.calls[2]?.answer(json(200, pageOf([two], null, 2)));
    await until(() => webMore.settled, "p-web's late page");
    expect(store.getList("p-api", open)).toEqual(theirs);
    expect(store.getList("p-web", open)).toEqual(pageOf([one, two], null, 2));
  });

  it.each<[string, (store: IIntakeStore) => Promise<unknown>]>([
    ["the first page", (store) => store.fetchList(acme, "p-web", open)],
    ["the next page", (store) => store.fetchMore("p-web", open)],
  ])(
    "hands nerve's refusal of %s to the caller, the list as it was; nothing once the session changed",
    async (_, read) => {
      const { nerve, api, store } = await intakeTab();
      const page = pageOf([one], "c-1", 2);
      await loadList(nerve, store, "p-web", open, page);
      const refused = track(read(store));
      await until(() => nerve.calls.length === 2, "the read");
      nerve.calls[1]?.answer(problem(403, "forbidden"));
      await until(() => refused.settled, "the refusal");
      expect(refused.error).toBeInstanceOf(ApiError);
      expect(refused.error).toMatchObject({ status: 403, problem: { code: "forbidden" } });
      expect(store.getList("p-web", open)).toEqual(page);

      // another session took the tab: the read is no failure, it gives nothing, and nothing changes
      FakeNerve.replaceSession(api);
      expect(await settle(read(store), "the cut read")).toEqual({ settled: true, value: undefined });
      expect(nerve.calls).toHaveLength(2);
      expect(store.getList("p-web", open)).toEqual(page);
    }
  );
});
