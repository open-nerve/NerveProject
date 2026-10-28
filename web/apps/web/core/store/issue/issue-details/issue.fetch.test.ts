/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { configure, runInAction } from "mobx";
// MobX actions remain real; configurable descriptors allow call-through observations.
configure({ safeDescriptors: false });
import { AxiosHeaders, type AxiosResponse } from "axios";
import { APIService } from "@/services/api.service";
import { heldChange } from "@/lib/auth/fake-tab";
import type { TIssueReaction, TIssueLink } from "@nerve/types";
import { until, track } from "@/lib/auth/fake-time";
import { FakeNerve, answered } from "@/lib/auth/fake-nerve";
import { issueOf } from "@/store/issue/fake-issues";
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
const ws = workspaceOf("ws");
const project = projectOf("A", ws.id);
const a = stateOf(project, "A", "unstarted", 1000);
const b = stateOf(project, "B", "started", 2000);
beforeEach(async () => {
  vi.restoreAllMocks();
  vi.useFakeTimers();
  const nerve = new FakeNerve();
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
function resources(parentId: string | null) {
  const detail = app.issue.issueDetail;
  const source = { ...issueOf("R1", project.id), parent_id: parentId, description_html: "<p>detail</p>" };
  const parent = issueOf("parent", project.id);
  // Only pre-existing, non-target detached side fetches are isolated.
  vi.spyOn(detail.activity, "fetchActivities").mockResolvedValue([]);
  vi.spyOn(detail.comment, "fetchComments").mockResolvedValue([]);
  vi.spyOn(detail.subIssues, "fetchSubIssues").mockResolvedValue({
    state_distribution: { backlog: [], unstarted: [], started: [], completed: [], cancelled: [] },
    sub_issues: [],
  });
  vi.spyOn(detail.relation, "fetchRelations").mockResolvedValue({
    blocking: [],
    blocked_by: [],
    duplicate: [],
    relates_to: [],
  });
  vi.spyOn(app.state, "fetchProjectStates").mockResolvedValue([]);
  const get = vi.spyOn(APIService.prototype, "get").mockImplementation(async (url) => {
    if (url.endsWith("/reactions/")) return reply([reaction]);
    if (url.endsWith("/issue-links/")) return reply([link]);
    if (url.endsWith("/issues/parent/")) return reply(parent);
    return reply(source);
  });
  const fetch = (identifier: boolean) =>
    identifier
      ? detail.fetchIssueWithIdentifier(ws.slug, project.identifier, "1")
      : detail.fetchIssue(ws.slug, project.id, source.id);
  return { detail, source, parent, get, fetch };
}
it.each([false, true].flatMap((identifier) => [null, "parent"].map((parentId) => ({ identifier, parentId }))))(
  "both retrieve paths fill real resource caches from independent responses ($identifier/$parentId)",
  async ({ identifier, parentId }) => {
    const { detail, source, parent, get, fetch } = resources(parentId);
    const add = vi.spyOn(app.issue.issues, "addIssue");
    await fetch(identifier);
    expect(app.issue.issues.getIssueById(source.id)).toEqual(source);
    expect(detail.reaction.getReactionById(reaction.id)).toEqual(reaction);
    expect(detail.reaction.getReactionsByIssueId(source.id)).toEqual({ thumbsup: [reaction.id] });
    expect(detail.link.getLinksByIssueId(source.id)).toEqual([link.id]);
    expect(detail.link.getLinkById(link.id)).toEqual(link);
    expect(add.mock.calls.map(([issues]) => issues.map((issue) => issue.id))).toEqual(
      parentId ? [[source.id], [parent.id]] : [[source.id]]
    );
    expect(get).toHaveBeenCalledTimes(parentId ? 4 : 3);
    expect(get.mock.calls[0]?.[1]).toEqual({ params: undefined });
    expect(get.mock.calls.filter(([url]) => url.includes("/issues/parent/"))).toHaveLength(parentId ? 1 : 0);
    if (parentId) expect(app.issue.issues.getIssueById(parent.id)).toEqual(parent);
    else expect(app.issue.issues.getIssueById(parent.id)).toBeUndefined();
    for (const field of ["parent", "issue_reactions", "issue_link", "issue_attachments"])
      expect(source).not.toHaveProperty(field);
  }
);
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
