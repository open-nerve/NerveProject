/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { configure, runInAction } from "mobx";
// MobX actions remain real; configurable descriptors allow call-through observations.
configure({ safeDescriptors: false });
import { ALL_ISSUES } from "@nerve/constants";
import { AxiosHeaders, type AxiosResponse } from "axios";
import { heldChange } from "@/lib/auth/fake-tab";
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
const { ProjectIssues } = await import("@/store/issue/project/issue.store");
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

function initial(html?: string) {
  const store = app.issue.projectIssues;
  if (!(store instanceof ProjectIssues)) throw new Error("expected the real project store");
  const before = {
    ...issueOf("R1", project.id, project.workspace_id),
    state_id: a.id,
    ...(html === undefined ? {} : { description_html: html }),
  };
  app.issue.issues.addIssue([before]);
  runInAction(() => {
    app.issue.projectIssuesFilter.filters[project.id] = {
      richFilters: {},
      displayFilters: { layout: "list", group_by: null, sub_group_by: null, order_by: "sort_order", sub_issue: true },
      displayProperties: undefined,
      kanbanFilters: undefined,
    };
    store.groupedIssueIds = { [ALL_ISSUES]: [before.id] };
    store.groupedIssueCount = { [ALL_ISSUES]: 1 };
  });
  return { store, before };
}
it.each(["<p>before</p>", "", undefined])(
  "isolates request context and restores original HTML/absence on HTTP refusal (%s)",
  async (html) => {
    const { store, before } = initial(html);
    const http = heldChange<AxiosResponse<typeof before>>();
    const patch = vi.spyOn(store.issueService, "patch").mockReturnValue(http.sent);
    const groups = vi.spyOn(store, "updateIssueList");
    const stats = vi.spyOn(store, "updateParentStats");
    const request = { description_html: "<p>after</p>", skip_activity: "true" };
    const updating = store.updateIssue(ws.slug, project.id, before.id, request);
    const error = { code: "refused" };
    const refused = expect(updating).rejects.toBe(error);
    expect(patch).toHaveBeenCalledExactlyOnceWith(
      `/api/workspaces/${ws.slug}/projects/${project.id}/issues/${before.id}/`,
      request
    );
    const current = app.issue.issues.getIssueById(before.id)!;
    expect(app.issue.issues.getIssueDetailById(before.id)?.description_html).toBe(request.description_html);
    expect(current).not.toHaveProperty("skip_activity");
    for (const call of [...groups.mock.calls, ...stats.mock.calls])
      for (const value of call) if (value) expect(value).not.toHaveProperty("skip_activity");
    expect(store.getIssueIds()).toEqual([before.id]);
    expect(store.groupedIssueCount[ALL_ISSUES]).toBe(1);
    http.refuse({ response: { data: error } });
    await refused;
    expect(app.issue.issues.getIssueById(before.id)).toEqual(before);
    expect(Object.hasOwn(app.issue.issues.getIssueById(before.id)!, "description_html")).toBe(html !== undefined);
  }
);
it("sends the unchanged string flag once on success and keeps empty HTML", async () => {
  const { store, before } = initial("before");
  const patch = vi.spyOn(store.issueService, "patch").mockResolvedValue(reply(before));
  await store.updateIssue(ws.slug, project.id, before.id, { description_html: "", skip_activity: "true" });
  expect(patch).toHaveBeenCalledTimes(1);
  expect(app.issue.issues.getIssueDetailById(before.id)?.description_html).toBe("");
  expect(app.issue.issues.getIssueById(before.id)).not.toHaveProperty("skip_activity");
});
it("local-only updates never send context or create missing entities", async () => {
  const { store, before } = initial();
  const patch = vi.spyOn(store.issueService, "patch");
  await store.issueUpdate(ws.slug, project.id, before.id, { description_html: "", skip_activity: "true" }, false);
  await store.issueUpdate(ws.slug, project.id, "absent", { name: "missing", skip_activity: "true" }, false);
  expect(patch).not.toHaveBeenCalled();
  expect(app.issue.issues.getIssueById(before.id)).not.toHaveProperty("skip_activity");
  expect(app.issue.issues.getIssueById("absent")).toBeUndefined();
  expect(store.getIssueIds()).toEqual([before.id]);
});
it("moves actual state groups/counts while pending and restores IDs, order, counts and entity after refusal", async () => {
  const { store, before } = initial("<p>before</p>");
  const r1 = { ...before, sort_order: 2000 };
  const r2 = { ...issueOf("R2", project.id, project.workspace_id), state_id: a.id, sort_order: 3000, sequence_id: 2 };
  const r3 = { ...issueOf("R3", project.id, project.workspace_id), state_id: b.id, sort_order: 1000, sequence_id: 3 };
  app.issue.issues.addIssue([r1, r2, r3]);
  runInAction(() => {
    app.issue.projectIssuesFilter.filters[project.id]!.displayFilters!.group_by = "state";
    store.groupedIssueIds = { [a.id]: [r1.id, r2.id], [b.id]: [r3.id] };
    store.groupedIssueCount = { [ALL_ISSUES]: 3, [a.id]: 2, [b.id]: 1 };
  });
  expect([store.groupBy, store.issueGroupKey, store.orderBy]).toEqual(["state", "state_id", "sort_order"]);
  const http = heldChange<AxiosResponse<typeof r1>>();
  const patch = vi.spyOn(store.issueService, "patch").mockReturnValue(http.sent);
  const request = { state_id: b.id, description_html: "<p>moved</p>" };
  const updating = store.updateIssue(ws.slug, project.id, r1.id, request);
  const error = { code: "move-refused" };
  const refused = expect(updating).rejects.toBe(error);
  expect(patch).toHaveBeenCalledExactlyOnceWith(
    `/api/workspaces/${ws.slug}/projects/${project.id}/issues/${r1.id}/`,
    request
  );
  expect(app.issue.issues.getIssueById(r1.id)).toMatchObject(request);
  expect(store.getIssueIds(a.id)).toEqual([r2.id]);
  expect(store.getIssueIds(b.id)).toEqual([r3.id, r1.id]);
  expect(store.groupedIssueCount).toMatchObject({ [ALL_ISSUES]: 3, [a.id]: 1, [b.id]: 2 });
  http.refuse({ response: { data: error } });
  await refused;
  expect(app.issue.issues.getIssueById(r1.id)).toEqual(r1);
  expect(store.getIssueIds(a.id)).toEqual([r1.id, r2.id]);
  expect(store.getIssueIds(b.id)).toEqual([r3.id]);
  expect(store.groupedIssueCount).toMatchObject({ [ALL_ISSUES]: 3, [a.id]: 2, [b.id]: 1 });
});

it("removes an optimistic HTML own key after refusal when it was originally absent", async () => {
  const { store, before } = initial();
  const error = { code: "refused" };
  vi.spyOn(store.issueService, "patch").mockRejectedValue({ response: { data: error } });
  await expect(store.updateIssue(ws.slug, project.id, before.id, { description_html: "added" })).rejects.toBe(error);
  expect(Object.hasOwn(app.issue.issues.getIssueById(before.id)!, "description_html")).toBe(false);
});

it("preserves empty HTML and null parent input without request context", async () => {
  const { store, before } = initial("before");
  const request = { description_html: "", parent_id: null };
  const patch = vi.spyOn(store.issueService, "patch").mockResolvedValue(reply(before));
  await store.updateIssue(ws.slug, project.id, before.id, request);
  expect(patch).toHaveBeenCalledExactlyOnceWith(
    `/api/workspaces/${ws.slug}/projects/${project.id}/issues/${before.id}/`,
    request
  );
  expect(app.issue.issues.getIssueById(before.id)).toMatchObject(request);
});
