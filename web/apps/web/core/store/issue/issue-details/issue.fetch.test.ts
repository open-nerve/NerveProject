/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { configure, runInAction } from "mobx";
// MobX actions remain real; configurable descriptors allow call-through observations.
configure({ safeDescriptors: false });
import { AxiosHeaders, type AxiosResponse } from "axios";
import { APIService } from "@/services/api.service";
import { heldChange } from "@/lib/auth/fake-tab";
import type { TIssueReaction, TIssueLink, TIssueRelation } from "@nerve/types";
import { until, track } from "@/lib/auth/fake-time";
import { FakeNerve, answered, json, noContent, problem } from "@/lib/auth/fake-nerve";
import { SessionChangedError } from "@/lib/auth/token-manager";
import { issueOf, requestedIssue } from "@/store/issue/fake-issues";
import { workspaceOf, loadWorkspaces } from "@/store/workspace/fake-workspaces";
import { projectOf, loadProjects, stateOf } from "@/store/project/fake-projects";
// Public APIService replies include Axios metadata; entity data remains the complete real response fixture.
function reply<T>(data: T): AxiosResponse<T> {
  return { data, status: 200, statusText: "OK", headers: {}, config: { headers: new AxiosHeaders() } };
}
vi.stubGlobal("localStorage", { getItem: () => null, setItem: () => {}, removeItem: () => {} });
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: {}, publicClient: {} }));
const binding = vi.hoisted(() => ({ app: undefined as unknown }));
vi.mock("@/lib/store-context", () => ({
  get rootStore() {
    return binding.app;
  },
}));
const { RootStore } = await import("@/store/root.store");
let app: InstanceType<typeof RootStore>;
let nerve: FakeNerve;
const ws = workspaceOf("ws");
const project = projectOf("A", ws.id);
const a = stateOf(project, "A", "unstarted", 1000);
const b = stateOf(project, "B", "started", 2000);
beforeEach(async () => {
  vi.restoreAllMocks();
  vi.useFakeTimers();
  nerve = new FakeNerve();
  app = new RootStore(nerve.client());
  binding.app = app;
  runInAction(() => {
    app.issue.workspaceSlug = ws.slug;
    app.issue.projectId = project.id;
    app.router.setQuery({ workspaceSlug: ws.slug, projectId: project.id });
  });
  await loadWorkspaces(nerve, app.workspaceRoot, [ws]);
  await loadProjects(nerve, app.projectRoot.project, ws, [project]);
  await answered(
    nerve,
    () => app.state.fetchProjectStates(project.id),
    ["GET", `/api/v0/projects/${project.id}/states`],
    { data: [a, b] },
    "states"
  );
});
afterEach(() => {
  app.dispose();
  vi.useRealTimers();
});

const reaction: TIssueReaction = {
  id: "reaction",
  actor: "actor",
  issue: "R1",
  reaction: "thumbsup",
  display_name: "Actor",
};
const link: TIssueLink = {
  id: "link",
  created_by_id: "actor",
  issue_id: "R1",
  title: "Reference",
  url: "https://example.test",
  metadata: {},
  created_at: new Date("2026-10-28"),
};
function resources(parentId: string | null, withRelations = false) {
  const detail = app.issue.issueDetail;
  const source = {
    ...issueOf("R1", project.id, project.workspace_id),
    parent_id: parentId,
    description_html: "<p>detail</p>",
  };
  const parent = issueOf("parent", project.id, project.workspace_id);
  // Only pre-existing, non-target detached side fetches are isolated.
  vi.spyOn(detail.activity, "fetchActivities").mockResolvedValue([]);
  vi.spyOn(detail.comment, "fetchComments").mockResolvedValue([]);
  vi.spyOn(detail.subIssues, "fetchSubIssues").mockResolvedValue({
    state_distribution: { backlog: [], unstarted: [], started: [], completed: [], cancelled: [] },
    sub_issues: [],
  });
  const related = issueOf("related", project.id, project.workspace_id);
  const relations: TIssueRelation = { blocking: [related], blocked_by: [], duplicate: [], relates_to: [] };
  const relationFetch = vi.spyOn(detail.relation, "fetchRelations");
  if (!withRelations) relationFetch.mockResolvedValue({ blocking: [], blocked_by: [], duplicate: [], relates_to: [] });
  vi.spyOn(app.state, "fetchProjectStates").mockResolvedValue([]);
  const get = vi.spyOn(APIService.prototype, "get").mockImplementation(async (url) => {
    if (url.endsWith("/issue-relation/")) return reply(relations);
    if (url.endsWith("/reactions/")) return reply([reaction]);
    if (url.endsWith("/issue-links/")) return reply([link]);
    if (url.endsWith("/issues/parent/")) return reply(parent);
    return reply(source);
  });
  const fetch = (identifier: boolean) =>
    identifier
      ? detail.fetchIssueWithIdentifier(ws.slug, project.identifier, "1")
      : detail.fetchIssue(ws.slug, project.id, source.id);
  return { detail, source, parent, get, fetch, related, relations, relationFetch };
}
it.each([false, true].flatMap((identifier) => [null, "parent"].map((parentId) => ({ identifier, parentId }))))(
  "both retrieve paths fill real resource caches from independent responses ($identifier/$parentId)",
  async ({ identifier, parentId }) => {
    const { detail, source, parent, get, fetch } = resources(parentId);
    const add = vi.spyOn(app.issue.issues, "acceptRead");
    await fetch(identifier);
    expect(app.issue.issues.getIssueById(source.id)).toEqual(source);
    expect(detail.reaction.getReactionById(reaction.id)).toEqual(reaction);
    expect(detail.reaction.getReactionsByIssueId(source.id)).toEqual({ thumbsup: [reaction.id] });
    expect(detail.link.getLinksByIssueId(source.id)).toEqual([link.id]);
    expect(detail.link.getLinkById(link.id)).toEqual(link);
    expect(add.mock.calls.map(([issue]) => [issue.id])).toEqual(parentId ? [[source.id], [parent.id]] : [[source.id]]);
    if (parentId) {
      expect(add.mock.calls[1]![1].serial).toBeGreaterThan(add.mock.calls[0]![1].serial);
      expect(add.mock.calls[1]![1].owner).toBe(add.mock.calls[0]![1].owner);
    }
    expect(get).toHaveBeenCalledTimes(parentId ? 4 : 3);
    expect(get.mock.calls[0]?.[1]).toEqual({ params: undefined });
    expect(get.mock.calls.filter(([url]) => url.includes("/issues/parent/"))).toHaveLength(parentId ? 1 : 0);
    if (parentId) expect(app.issue.issues.getIssueById(parent.id)).toEqual(parent);
    else expect(app.issue.issues.getIssueById(parent.id)).toBeUndefined();
    for (const field of ["parent", "issue_reactions", "issue_link", "issue_attachments"])
      expect(source).not.toHaveProperty(field);
  }
);

it.each([false, true])("legacyDetailConsumersCannotBypassCanonicalVersion (%s)", async (identifier) => {
  const { source, parent, related, fetch, detail } = resources("parent", true);
  const updated_at = "2026-10-29T00:00:00.000000009Z";
  for (const row of [source, parent, related])
    app.issue.issues.acceptRead(
      { ...row, name: "new", updated_at, description_html: "new HTML" },
      app.issue.captureRead()
    );
  await fetch(identifier);
  await until(
    () => detail.relation.getRelationsByIssueId(source.id)?.blocking?.includes(related.id) === true,
    "legacy related row"
  );
  for (const row of [source, parent, related]) {
    expect(app.issue.issues.getIssueById(row.id)?.name).toBe("new");
    expect(app.issue.issues.getIssueDetailById(row.id)?.description_html).toBe("new HTML");
  }
});

it.each([200, 500])("detailRetryIsBoundedAndRefusalReturns (%s)", async (status) => {
  const { source, fetch } = resources(null);
  app.issue.issues.acceptRead(
    { ...issueOf(source.id, project.id, project.workspace_id), updated_at: "2026-10-30T00:00:00Z" },
    app.issue.captureRead()
  );
  const request = await requestedIssue(nerve, () => fetch(false), ["GET", "/api/v0/issues/R1"]);
  const outcome = status === 500 ? expect(request.promise).rejects.toMatchObject({ status: 500 }) : request.promise;
  request.call.answer(status === 500 ? problem(500, "internal") : json(200, source));
  await outcome;
  expect(app.issue.issues.getIssueDetailById(source.id)).toBeUndefined();
  expect(nerve.calls.filter((call) => call.path === "/api/v0/issues/R1")).toHaveLength(1);
});

it("legacyLinkCountComesFromAuthoritativeVersionAndRefusalDoesNotMutate", async () => {
  const { source, detail } = resources(null);
  app.issue.issues.acceptRead({ ...source, link_count: 4 }, app.issue.captureRead());
  vi.spyOn(APIService.prototype, "post").mockResolvedValue(reply(link));
  const creating = await requestedIssue(
    nerve,
    () => detail.link.createLink(ws.slug, project.id, source.id, { url: link.url }),
    ["GET", "/api/v0/issues/R1"]
  );
  expect(app.issue.issues.getIssueById(source.id)).toBeUndefined();
  const authoritative = { ...source, link_count: 11, updated_at: "2026-10-29T00:00:00Z" };
  creating.call.answer(json(200, authoritative));
  await creating.promise;
  expect(app.issue.issues.getIssueDetailById(source.id)).toEqual(authoritative);
  vi.spyOn(APIService.prototype, "delete").mockResolvedValue(reply(undefined));
  const removing = await requestedIssue(nerve, () => detail.link.removeLink(ws.slug, project.id, source.id, link.id), [
    "GET",
    "/api/v0/issues/R1",
  ]);
  const refused = expect(removing.promise).rejects.toMatchObject({ status: 403 });
  removing.call.answer(problem(403, "forbidden"));
  await refused;
  expect(app.issue.issues.getIssueById(source.id)).toBeUndefined();
  app.issue.issues.acceptRead(authoritative, app.issue.captureRead());
  expect(app.issue.issues.getIssueDetailById(source.id)).toEqual(authoritative);
});
it.each([false, true])(
  "fetch owns pending auxiliary loads instead of returning before they finish (%s)",
  async (identifier) => {
    const { get, fetch } = resources(null);
    const http = heldChange<AxiosResponse<TIssueReaction[]>>();
    const fallback = get.getMockImplementation()!;
    get.mockImplementation((url, ...args) => (url.endsWith("/reactions/") ? http.sent : fallback(url, ...args)));
    const loading = fetch(identifier);
    const observed = track(loading);
    await until(() => get.mock.calls.some(([url]) => url.endsWith("/reactions/")), "resource GET sent");
    await vi.advanceTimersByTimeAsync(0);
    const settledBeforeRelease = observed.settled;
    http.answer(reply([reaction]));
    await loading;
    expect(settledBeforeRelease).toBe(false);
  }
);
it.each([false, true].flatMap((identifier) => ["retire", "delete"].map((boundary) => ({ identifier, boundary }))))(
  "auxiliaryCompletionCannotFollowInvalidatedDetail ($identifier/$boundary)",
  async ({ identifier, boundary }) => {
    const { detail, source, get, fetch } = resources("parent");
    const held = heldChange<AxiosResponse<TIssueReaction[]>>();
    const fallback = get.getMockImplementation()!;
    get.mockImplementation((url, ...args) => (url.endsWith("/reactions/") ? held.sent : fallback(url, ...args)));
    // Observe the real generated state transport, not just a detached method stub.
    const states = vi.mocked(app.state.fetchProjectStates);
    states.mockRestore();
    const identifiers = vi.spyOn(app.issue.issues, "addIssueIdentifier");
    const loading = track(fetch(identifier));
    await until(() => get.mock.calls.some(([url]) => url.endsWith("/reactions/")), "held auxiliary read");
    if (boundary === "retire") app.dispose();
    else {
      const deleting = await requestedIssue(
        nerve,
        () => app.issue.deleteIssue(source.id, app.issue.scope(ws.slug, project.id)),
        ["DELETE", "/api/v0/issues/R1"]
      );
      deleting.call.answer(noContent());
      await deleting.promise;
    }
    const requestsBeforeRelease = nerve.calls.length;
    const identifiersBeforeRelease = identifiers.mock.calls.length;
    held.answer(reply([reaction]));
    await until(() => loading.settled, "invalidated detail settles");
    await vi.advanceTimersByTimeAsync(1);
    const lateCalls = nerve.calls.slice(requestsBeforeRelease);
    // Settle an unexpected real state request on the red run without hiding the assertion.
    for (const call of lateCalls) call.answer(json(200, { data: [a, b] }));
    await vi.advanceTimersByTimeAsync(1);
    expect(lateCalls.map((call) => call.path)).toEqual([]);
    expect(identifiers.mock.calls).toHaveLength(identifiersBeforeRelease);
    expect(detail.activity.fetchActivities).not.toHaveBeenCalled();
    expect(detail.comment.fetchComments).not.toHaveBeenCalled();
    expect(detail.subIssues.fetchSubIssues).not.toHaveBeenCalled();
    expect(detail.relation.fetchRelations).not.toHaveBeenCalled();
    expect(loading.error).toBeInstanceOf(SessionChangedError);
    expect(loading.value).toBeUndefined();
  }
);
it.each(
  [false, true].flatMap((identifier) =>
    ["reactions/", "issue-links/", "issues/parent/"].map((resource) => ({ identifier, resource }))
  )
)(
  "auxiliary HTTP refusal belongs to fetch, with existing extraction and no detached rejection ($identifier/$resource)",
  async ({ identifier, resource }) => {
    const { get, fetch } = resources("parent");
    const fallback = get.getMockImplementation()!;
    const domainError = { code: "resource-refused" };
    get.mockImplementation((url, ...args) =>
      url.endsWith(resource) ? Promise.reject({ response: { data: domainError } }) : fallback(url, ...args)
    );
    const result = track(fetch(identifier));
    await until(() => result.settled, "detail fetch settles");
    expect(result.error).toEqual(resource === "issue-links/" ? { data: domainError } : domainError);
  }
);

it.each([false, true].flatMap((identifier) => ["initial", "next"].map((page) => ({ identifier, page }))))(
  "pure list arrival preserves relations loaded by the real detail resource ($identifier/$page)",
  async ({ identifier, page }) => {
    const { detail, source, related, relations, relationFetch, get, fetch } = resources(null, true);
    const listRow = { ...issueOf(source.id, project.id, project.workspace_id), name: "list update" };
    const response = { total: 1, data: [listRow], next_cursor: null };
    const projectIssues = app.issue.projectIssues;
    if (page === "next") {
      await answered(
        nerve,
        () => projectIssues.fetchIssues(ws.slug, project.id, "init-loader", { limit: 100 }),
        ["GET", `/api/v0/projects/${project.id}/issues`],
        { ...response, next_cursor: "next" },
        "first list page"
      );
    }
    await fetch(identifier);
    await until(
      () => detail.relation.getRelationsByIssueId(source.id)?.blocking?.includes(related.id) === true,
      "relations loaded"
    );
    expect(relationFetch).toHaveBeenCalledExactlyOnceWith(ws.slug, project.id, source.id);
    expect(get.mock.calls.filter(([url]) => url.endsWith("/issue-relation/"))).toHaveLength(1);
    await answered(
      nerve,
      () =>
        page === "initial"
          ? projectIssues.fetchIssues(ws.slug, project.id, "init-loader", { limit: 100 })
          : projectIssues.fetchNextIssues(ws.slug, project.id),
      ["GET", `/api/v0/projects/${project.id}/issues`],
      response,
      "list page"
    );
    expect(detail.relation.getRelationsByIssueId(source.id)).toEqual({
      blocking: [related.id],
      blocked_by: [],
      duplicate: [],
      relates_to: [],
    });
    expect(app.issue.issues.getIssueById(related.id)).toEqual(relations.blocking[0]);
    expect(app.issue.issues.getIssueById(source.id)?.name).toBe("list update");
    expect(app.issue.issues.getIssueDetailById(source.id)?.description_html).toBe(source.description_html);
    expect(get).toHaveBeenCalledTimes(4);
    expect(nerve.calls.filter((call) => call.path === `/api/v0/projects/${project.id}/issues`)).toHaveLength(
      page === "initial" ? 1 : 2
    );
  }
);
