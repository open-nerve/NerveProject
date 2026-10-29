/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { beforeEach, afterEach, expect, it, vi } from "vitest";
import { configure } from "mobx";
import { issueStoreTab, issueDetailOf, requestedIssue } from "../fake-issues";
import { json, problem } from "@/lib/auth/fake-nerve";
import { loadProjects, projectOf } from "@/store/project/fake-projects";
import { until, track } from "@/lib/auth/fake-time";
import { SessionChangedError } from "@/lib/auth/token-manager";

configure({ safeDescriptors: false });
vi.stubGlobal("localStorage", { getItem: () => null, setItem: () => {}, removeItem: () => {} });
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: {}, publicClient: {} }));
vi.mock("@/lib/store-context", () => ({ rootStore: undefined }));
let tab: Awaited<ReturnType<typeof issueStoreTab>>;
beforeEach(async () => {
  vi.useFakeTimers();
  tab = await issueStoreTab();
});
afterEach(() => {
  tab.app.dispose();
  vi.restoreAllMocks();
  vi.useRealTimers();
});

function isolateAuxiliary() {
  const detail = tab.app.issue.issueDetail;
  const reactions = vi.spyOn(detail, "fetchReactions").mockResolvedValue([]);
  vi.spyOn(detail, "fetchLinks").mockResolvedValue([]);
  const activity = vi.spyOn(detail.activity, "fetchActivities").mockResolvedValue([]);
  vi.spyOn(detail.comment, "fetchComments").mockResolvedValue([]);
  vi.spyOn(detail.subIssues, "fetchSubIssues").mockResolvedValue({
    sub_issues: [],
    state_distribution: { backlog: [], unstarted: [], started: [], completed: [], cancelled: [] },
  });
  vi.spyOn(detail.relation, "fetchRelations").mockResolvedValue({
    blocking: [],
    blocked_by: [],
    duplicate: [],
    relates_to: [],
  });
  vi.spyOn(tab.app.state, "fetchProjectStates").mockResolvedValue([]);
  return { detail, reactions, activity };
}

it.each(["id", "project_id", "workspace_id"] as const)(
  "rejects a mismatched %s before canonical or auxiliary reads",
  async (field) => {
    const { detail, reactions } = isolateAuxiliary();
    const { call, result } = await requestedIssue(tab.nerve, () => detail.fetchIssue("ws", "A", "one"), [
      "GET",
      "/api/v0/issues/one",
    ]);
    call.answer(json(200, { ...issueDetailOf("one"), [field]: "wrong" }));
    await until(() => result.settled, "mismatched detail");
    expect(result.error).toBeInstanceOf(SessionChangedError);
    expect(tab.app.issue.issues.held("one")).toBeUndefined();
    expect(tab.app.issue.issues.held("wrong")).toBeUndefined();
    expect(reactions).not.toHaveBeenCalled();
  }
);

it.each(["a-1", "\u7814\u53d1-x-001"])(
  "the server parses the whole identifier %s before a cold project is proved",
  async (identifier) => {
    const { detail, reactions } = isolateAuxiliary();
    const project = projectOf("returned", tab.workspace.id, { id: "returned", identifier: "Server" });
    const { call, result } = await requestedIssue(tab.nerve, () => detail.fetchIssueWithIdentifier("ws", identifier), [
      "GET",
      `/api/v0/workspaces/ws/issues/${encodeURIComponent(identifier)}`,
    ]);
    call.answer(json(200, issueDetailOf("one", project.id)));
    await until(() => tab.nerve.to(`/api/v0/projects/${project.id}`).length === 1, "project proof");
    expect(tab.app.issue.issues.held("one")).toBeUndefined();
    expect(reactions).not.toHaveBeenCalled();
    tab.nerve.to(`/api/v0/projects/${project.id}`)[0]!.answer(json(200, project));
    await until(() => result.settled, "proved detail");
    expect(result.error).toBeUndefined();
    expect(result.value?.id).toBe("one");
    expect(tab.app.issue.issues.getIssueIdByIdentifier(identifier)).toBe("one");
    expect(reactions).toHaveBeenCalledExactlyOnceWith("ws", project.id, "one");
  }
);

it("a cold directory refusal remains observable without publishing any issue", async () => {
  const { detail, reactions } = isolateAuxiliary();
  const { call, result } = await requestedIssue(tab.nerve, () => detail.fetchIssueWithIdentifier("ws", "cold-1"), [
    "GET",
    "/api/v0/workspaces/ws/issues/cold-1",
  ]);
  call.answer(json(200, issueDetailOf("one", "cold")));
  await until(() => tab.nerve.to("/api/v0/projects/cold").length === 1, "directory request");
  tab.nerve.to("/api/v0/projects/cold")[0]!.answer(problem(403, "forbidden"));
  await until(() => result.settled, "directory refusal");
  expect(result.error).toMatchObject({ status: 403 });
  expect(tab.app.issue.issues.held("one")).toBeUndefined();
  expect(reactions).not.toHaveBeenCalled();
});

it("identifier responses from another workspace start no project or resource requests", async () => {
  const { detail, reactions } = isolateAuxiliary();
  const { call, result } = await requestedIssue(tab.nerve, () => detail.fetchIssueWithIdentifier("ws", "A-1"), [
    "GET",
    "/api/v0/workspaces/ws/issues/A-1",
  ]);
  call.answer(json(200, { ...issueDetailOf("one"), workspace_id: "other" }));
  await until(() => result.settled, "wrong workspace");
  expect(result.error).toBeInstanceOf(SessionChangedError);
  expect(tab.nerve.calls).toHaveLength(1);
  expect(reactions).not.toHaveBeenCalled();
  expect(tab.app.issue.issues.held("one")).toBeUndefined();
});

it("an older read of the same entry cannot publish after the newer one", async () => {
  const { detail } = isolateAuxiliary();
  const first = await requestedIssue(tab.nerve, () => detail.fetchIssue("ws", "A", "one"), [
    "GET",
    "/api/v0/issues/one",
  ]);
  const second = await requestedIssue(tab.nerve, () => detail.fetchIssue("ws", "A", "one"), [
    "GET",
    "/api/v0/issues/one",
  ]);
  second.call.answer(json(200, { ...issueDetailOf("one"), name: "new read" }));
  await until(() => second.result.settled, "new detail");
  first.call.answer(json(200, { ...issueDetailOf("one"), name: "old read" }));
  await until(() => first.result.settled, "old detail");
  expect(tab.app.issue.issues.getIssueById("one")?.name).toBe("new read");
  expect(first.result.error).toBeInstanceOf(SessionChangedError);
});

it("auxiliary activity and directory refusals are awaited with the confirmed core retained", async () => {
  const { detail, activity } = isolateAuxiliary();
  const error = new Error("activity refused");
  activity.mockRejectedValue(error);
  const { call, result } = await requestedIssue(tab.nerve, () => detail.fetchIssue("ws", "A", "one"), [
    "GET",
    "/api/v0/issues/one",
  ]);
  call.answer(json(200, issueDetailOf("one")));
  await until(() => result.settled, "owned activity refusal");
  expect(result.error).toBe(error);
  expect(tab.app.issue.issues.getIssueDetailById("one")).toEqual(issueDetailOf("one"));
});

it("a directory response after permission ABA cannot revive identifier detail", async () => {
  const { detail, reactions } = isolateAuxiliary();
  const reading = track(detail.fetchIssueWithIdentifier("ws", "A-1"));
  await until(() => tab.nerve.calls.length === 1, "identifier starts");
  await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, []);
  await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [tab.project]);
  tab.nerve.calls[0]!.answer(json(200, issueDetailOf("one")));
  await until(() => reading.settled, "invalidated identifier");
  expect(reading.error).toBeInstanceOf(SessionChangedError);
  expect(reactions).not.toHaveBeenCalled();
  expect(tab.app.issue.issues.held("one")).toBeUndefined();
});
