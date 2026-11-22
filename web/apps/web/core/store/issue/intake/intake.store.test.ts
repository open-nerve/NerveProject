/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { IntakeCounts, Workspace } from "@nerve/api-client";
import { ApiError } from "@/lib/api-error";
import { FakeNerve, answered, json, problem } from "@/lib/auth/fake-nerve";
import { track, until } from "@/lib/auth/fake-time";
import { countsOf } from "@/store/issue/intake/fake-intake";
import type { IIntakeStore } from "@/store/issue/intake/intake.store";
import { loadWorkspaces, workspaceOf } from "@/store/workspace/fake-workspaces";

vi.mock("@/lib/auth/api-client", () => ({ tokenManager: {}, publicClient: {} }));
vi.mock("@/lib/store-context", () => ({ rootStore: {} }));

const { RootStore } = await import("@/store/root.store");

// The pending work items of the caller's intakes in each of his workspaces (P5a spec 2.3, R7), as the intake store of
// a session's issue root holds them, against a fake nerve that answers each request when the test says. The tab's
// caller has acme and beta; the store keeps each workspace's counts by its id, and its readers name the workspace by the
// address's slug, as the caller's workspaces list it now (v0 design 7.7).

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
