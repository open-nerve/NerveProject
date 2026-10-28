/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */

import type { Issue, IssueDetail } from "@nerve/api-client";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { configure, runInAction } from "mobx";
// MobX actions remain real; configurable descriptors allow call-through observations.
configure({ safeDescriptors: false });
import type { TInboxIssue } from "@nerve/types";
import { EInboxIssueStatus, EInboxIssueSource } from "@nerve/types";
import { AxiosHeaders, type AxiosResponse } from "axios";
import { heldChange } from "@/lib/auth/fake-tab";
import { InboxIssueService } from "@/services/inbox";
import { FakeNerve, answered, json, problem } from "@/lib/auth/fake-nerve";
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

function envelope(issue: Issue | IssueDetail): TInboxIssue {
  return {
    id: "inbox-row",
    status: EInboxIssueStatus.PENDING,
    snoozed_till: null,
    duplicate_to: undefined,
    source: EInboxIssueSource.IN_APP,
    issue,
    created_by: "actor",
    duplicate_issue_detail: undefined,
  };
}
function local(html?: string) {
  const row = envelope({
    ...issueOf("R1", project.id, project.workspace_id),
    ...(html === undefined ? {} : { description_html: html }),
  });
  app.projectInbox.createOrUpdateInboxIssue([row], ws.slug, project.id);
  return app.projectInbox.getIssueInboxByIssueId(row.issue.id)!;
}
function activities() {
  vi.spyOn(app.issue.issueDetail, "fetchActivities").mockResolvedValue([]);
}
it.each(["<p>before</p>", "", undefined])(
  "keeps flags on HTTP only and restores HTML own-key state on swallowed refusal (%s)",
  async (html) => {
    const store = local(html);
    const before = { ...store.issue };
    const http = heldChange<AxiosResponse<TInboxIssue>>();
    const patch = vi.spyOn(InboxIssueService.prototype, "patch").mockReturnValue(http.sent);
    const request = { description_html: "<p>after</p>", skip_activity: "true" };
    const updating = store.updateIssue(request);
    expect(patch).toHaveBeenCalledExactlyOnceWith(
      `/api/workspaces/${ws.slug}/projects/${project.id}/inbox-issues/R1/`,
      { issue: request }
    );
    expect(store.issue.description_html).toBe(request.description_html);
    expect(store.issue).not.toHaveProperty("skip_activity");
    expect(app.issue.issues.getIssueById("R1")).toBeUndefined();
    http.refuse({ response: { data: { code: "refused" } } });
    await expect(updating).resolves.toBeUndefined();
    expect(store.issue).toEqual(before);
    expect(Object.hasOwn(store.issue, "description_html")).toBe(html !== undefined);
    expect(store.issue).not.toHaveProperty("skip_activity");
  }
);
it("retains original null fields on refusal and sends nothing without an issue id", async () => {
  const store = local();
  const http = heldChange<AxiosResponse<TInboxIssue>>();
  const patch = vi.spyOn(InboxIssueService.prototype, "patch").mockReturnValue(http.sent);
  const updating = store.updateIssue({ parent_id: "parent" });
  http.refuse({ response: { data: { code: "refused" } } });
  await updating;
  expect(store.issue.parent_id).toBeNull();
  runInAction(() => {
    store.issue = {};
  });
  await store.updateIssue({ description_html: "missing" });
  expect(patch).toHaveBeenCalledTimes(1);
});
it.each(["<p>new</p>", ""])(
  "inbox accept receives authoritative HTML rather than the partial editor (%s)",
  async (html) => {
    const store = local("<p>old editor</p>");
    const current = {
      ...issueOf("R1", project.id, project.workspace_id),
      description_html: "confirmed",
      updated_at: "2026-10-29T00:00:00Z",
    };
    app.issue.issues.acceptRead(current, app.issue.captureRead());
    const response = { ...issueOf("R1", project.id, project.workspace_id), name: "partial old" };
    const patch = vi
      .spyOn(InboxIssueService.prototype, "patch")
      .mockResolvedValue(reply({ ...envelope(response), status: EInboxIssueStatus.ACCEPTED }));
    const request = await requestedIssue(nerve, () => store.updateInboxIssueStatus(EInboxIssueStatus.ACCEPTED), [
      "GET",
      "/api/v0/issues/R1",
    ]);
    expect(store.status).toBe(EInboxIssueStatus.ACCEPTED);
    expect(app.issue.issues.getIssueById("R1")).toBeUndefined();
    const authoritative = { ...current, description_html: html, updated_at: "2026-10-29T00:00:01Z" };
    request.call.answer(json(200, authoritative));
    await request.promise;
    expect(app.issue.issues.getIssueDetailById("R1")).toEqual(authoritative);
    expect(patch).toHaveBeenCalledTimes(1);
  }
);
it("inbox authoritative refusal returns without reverting or resending the successful accept", async () => {
  const store = local("edited");
  const patch = vi
    .spyOn(InboxIssueService.prototype, "patch")
    .mockResolvedValue(
      reply({ ...envelope(issueOf("R1", project.id, project.workspace_id)), status: EInboxIssueStatus.ACCEPTED })
    );
  const request = await requestedIssue(nerve, () => store.updateInboxIssueStatus(EInboxIssueStatus.ACCEPTED), [
    "GET",
    "/api/v0/issues/R1",
  ]);
  const refusing = expect(request.promise).rejects.toMatchObject({ status: 500 });
  request.call.answer(problem(500, "internal"));
  await refusing;
  expect(store.status).toBe(EInboxIssueStatus.ACCEPTED);
  expect(app.issue.issues.getIssueById("R1")).toBeUndefined();
  expect(patch).toHaveBeenCalledTimes(1);
});
it("retrieve preserves its own inbox resources before accepted canonical is read", async () => {
  activities();
  vi.spyOn(app.issue.issueDetail, "fetchComments").mockResolvedValue([]);
  vi.spyOn(app.issue.issueDetail, "fetchReactions").mockResolvedValue([]);
  const row = envelope({ ...issueOf("R1", project.id, project.workspace_id), description_html: "<p>retrieved</p>" });
  const get = vi.spyOn(InboxIssueService.prototype, "get").mockResolvedValue(reply(row));
  await app.projectInbox.fetchInboxIssueById(ws.slug, project.id, "R1");
  expect(get).toHaveBeenCalledExactlyOnceWith(
    `/api/workspaces/${ws.slug}/projects/${project.id}/inbox-issues/R1/?expand=issue_inbox`
  );
  expect(app.projectInbox.getIssueInboxByIssueId("R1")?.issue.description_html).toBe("<p>retrieved</p>");
  expect(app.issue.issues.getIssueById("R1")).toBeUndefined();
});
it("status refusal is swallowed, restores status and never adds canonical", async () => {
  const store = local();
  const before = nerve.calls.length;
  vi.spyOn(InboxIssueService.prototype, "patch").mockRejectedValue({ response: { data: { code: "refused" } } });
  await expect(store.updateInboxIssueStatus(EInboxIssueStatus.ACCEPTED)).resolves.toBeUndefined();
  expect(store.status).toBe(EInboxIssueStatus.PENDING);
  expect(nerve.calls).toHaveLength(before);
  expect(app.issue.issues.getIssueById("R1")).toBeUndefined();
});
it("ordinary inbox cache has no invented HTML while empty details remain known", () => {
  expect(local().issue).not.toHaveProperty("description_html");
  expect(local("").issue.description_html).toBe("");
});

it("deletes an optimistic HTML own key after swallowed refusal when originally absent", async () => {
  const store = local();
  vi.spyOn(InboxIssueService.prototype, "patch").mockRejectedValue({ response: { data: { code: "refused" } } });
  await expect(store.updateIssue({ description_html: "added" })).resolves.toBeUndefined();
  expect(Object.hasOwn(store.issue, "description_html")).toBe(false);
});

it.each([false, true])("empty HTML edits stay actual empty strings with optional context (%s)", async (withContext) => {
  activities();
  const store = local("before");
  const request = { description_html: "", ...(withContext ? { skip_activity: "true" } : {}) };
  const patch = vi
    .spyOn(InboxIssueService.prototype, "patch")
    .mockResolvedValue(reply(envelope(issueOf("R1", project.id, project.workspace_id))));
  await store.updateIssue(request);
  expect(patch).toHaveBeenCalledExactlyOnceWith(`/api/workspaces/${ws.slug}/projects/${project.id}/inbox-issues/R1/`, {
    issue: request,
  });
  expect(store.issue.description_html).toBe("");
  expect(store.issue).not.toHaveProperty("skip_activity");
});
