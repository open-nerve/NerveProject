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
it("retrieve then accept uses response required fields and retains the one known HTML field", async () => {
  activities();
  vi.spyOn(app.issue.issueDetail, "fetchComments").mockResolvedValue([]);
  vi.spyOn(app.issue.issueDetail, "fetchReactions").mockResolvedValue([]);
  const row = envelope({ ...issueOf("R1", project.id, project.workspace_id), description_html: "<p>retrieved</p>" });
  const get = vi.spyOn(InboxIssueService.prototype, "get").mockResolvedValue(reply(row));
  await app.projectInbox.fetchInboxIssueById(ws.slug, project.id, "R1");
  expect(get).toHaveBeenCalledExactlyOnceWith(
    `/api/workspaces/${ws.slug}/projects/${project.id}/inbox-issues/R1/?expand=issue_inbox`
  );
  const store = app.projectInbox.getIssueInboxByIssueId("R1")!;
  expect(store.issue.description_html).toBe("<p>retrieved</p>");
  expect(app.issue.issues.getIssueById("R1")).toBeUndefined();
  const response = {
    ...issueOf("R1", project.id, project.workspace_id),
    name: "server",
    sequence_id: 9,
    sort_order: 99,
    priority: "high" as const,
    created_by_id: "response-actor",
  };
  const add = vi.spyOn(app.issue.issues, "addIssue");
  const patch = vi
    .spyOn(InboxIssueService.prototype, "patch")
    .mockResolvedValue(reply({ ...envelope(response), status: EInboxIssueStatus.ACCEPTED }));
  await store.updateInboxIssueStatus(EInboxIssueStatus.ACCEPTED);
  expect(patch).toHaveBeenCalledExactlyOnceWith(`/api/workspaces/${ws.slug}/projects/${project.id}/inbox-issues/R1/`, {
    status: EInboxIssueStatus.ACCEPTED,
  });
  expect(add).toHaveBeenCalledExactlyOnceWith([{ ...response, description_html: "<p>retrieved</p>" }]);
  expect(app.issue.issues.getIssueById("R1")).toEqual({ ...response, description_html: "<p>retrieved</p>" });
});
it.each(["<p>edited</p>", ""])("successful edit then accept retains actual string HTML (%s)", async (html) => {
  activities();
  const store = local();
  const response = issueOf("R1", project.id, project.workspace_id);
  const patch = vi.spyOn(InboxIssueService.prototype, "patch").mockResolvedValue(reply(envelope(response)));
  await store.updateIssue({ description_html: html, skip_activity: "true" });
  expect(store.issue).not.toHaveProperty("skip_activity");
  await store.updateInboxIssueStatus(EInboxIssueStatus.ACCEPTED);
  expect(patch.mock.calls.map((call) => call[1])).toEqual([
    { issue: { description_html: html, skip_activity: "true" } },
    { status: EInboxIssueStatus.ACCEPTED },
  ]);
  expect(app.issue.issues.getIssueById("R1")).toEqual({ ...response, description_html: html });
});
it.each([
  { localHtml: undefined, responseHtml: undefined },
  { localHtml: "old", responseHtml: "" },
])(
  "accept respects absent HTML and response empty detail ($localHtml/$responseHtml)",
  async ({ localHtml, responseHtml }) => {
    const store = local(localHtml);
    const response = {
      ...issueOf("R1", project.id, project.workspace_id),
      ...(responseHtml === undefined ? {} : { description_html: responseHtml }),
    };
    const add = vi.spyOn(app.issue.issues, "addIssue");
    vi.spyOn(InboxIssueService.prototype, "patch").mockResolvedValue(reply(envelope(response)));
    await store.updateInboxIssueStatus(EInboxIssueStatus.ACCEPTED);
    expect(add).toHaveBeenCalledExactlyOnceWith([response]);
    expect(Object.hasOwn(app.issue.issues.getIssueById("R1")!, "description_html")).toBe(responseHtml !== undefined);
  }
);
it("status refusal is swallowed, restores status and never adds canonical", async () => {
  const store = local();
  const add = vi.spyOn(app.issue.issues, "addIssue");
  vi.spyOn(InboxIssueService.prototype, "patch").mockRejectedValue({ response: { data: { code: "refused" } } });
  await expect(store.updateInboxIssueStatus(EInboxIssueStatus.ACCEPTED)).resolves.toBeUndefined();
  expect(store.status).toBe(EInboxIssueStatus.PENDING);
  expect(add).not.toHaveBeenCalled();
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
