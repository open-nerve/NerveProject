/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { json, noContent, problem } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { SessionChangedError } from "@/lib/auth/token-manager";
import { issueOf, issueDetailOf, issueStoreTab, requestedIssue } from "../fake-issues";
import { loadProjects, stateOf } from "@/store/project/fake-projects";

vi.stubGlobal("localStorage", { getItem: () => null, setItem: () => {}, removeItem: () => {} });
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: {}, publicClient: {} }));
vi.mock("@/lib/store-context", () => ({ rootStore: undefined }));
let tab: Awaited<ReturnType<typeof issueStoreTab>>;
const parent = issueDetailOf("parent");
const child = { ...issueOf("child"), parent_id: parent.id };
beforeEach(async () => {
  vi.useFakeTimers();
  tab = await issueStoreTab();
  tab.app.issue.issues.acceptRead(parent, tab.app.issue.captureRead());
});
afterEach(() => {
  tab.app.dispose();
  vi.restoreAllMocks();
  vi.useRealTimers();
});
const store = () => tab.app.issue.issueDetail.subIssues;
const read = () => store().fetchSubIssues("ws", "A", "parent");
function directoriesAndParent() {
  tab.nerve.respond = (call) => {
    if (call.method !== "GET") return;
    if (call.path === "/api/v0/projects/A/states")
      call.answer(json(200, { data: [{ ...stateOf(tab.project, "Todo", "unstarted", 1000), id: "state" }] }));
    if (call.path === "/api/v0/issues/parent")
      call.answer(json(200, { ...parent, sub_issues_count: 8, updated_at: "2026-10-29T00:00:00Z" }));
  };
}
it.each(["fetch", "create"])(
  "%s uses native children, derives state groups and reads the authoritative count",
  async (kind) => {
    directoriesAndParent();
    const operation = kind === "fetch" ? read : () => store().createSubIssues("ws", "A", "parent", [child.id]);
    const sent = await requestedIssue<unknown>(tab.nerve, operation, [
      kind === "fetch" ? "GET" : "POST",
      "/api/v0/issues/parent/sub-issues",
    ]);
    if (kind === "create") expect(sent.call.body).toEqual({ sub_issue_ids: ["child"] });
    sent.call.answer(json(200, { data: [child] }));
    await sent.promise;
    expect(store().subIssuesByIssueId(parent.id)).toEqual([child.id]);
    expect(store().stateDistributionByIssueId(parent.id)).toEqual({
      backlog: [],
      unstarted: [child.id],
      started: [],
      completed: [],
      cancelled: [],
    });
    expect(tab.app.issue.issues.getIssueById(parent.id)?.sub_issues_count).toBe(8);
    expect(tab.app.issue.issues.getIssueById(child.id)).toEqual(child);
    expect(tab.nerve.calls.every((call) => call.path.startsWith("/api/v0/"))).toBe(true);
    expect(tab.nerve.calls.some((call) => /cycles|modules/.test(call.path))).toBe(false);
  }
);
it("keeps the last complete collection across an ordinary refusal", async () => {
  directoriesAndParent();
  const first = await requestedIssue(tab.nerve, read, ["GET", "/api/v0/issues/parent/sub-issues"]);
  first.call.answer(json(200, { data: [child] }));
  await first.promise;
  const next = await requestedIssue(tab.nerve, read, ["GET", "/api/v0/issues/parent/sub-issues"]);
  next.call.answer(problem(500, "unavailable"));
  await expect(next.promise).rejects.toMatchObject({ status: 500 });
  expect(store().subIssuesByIssueId(parent.id)).toEqual([child.id]);
});
it("deletedParentDiscardsLateSubReadWithoutAuthoritativeFollowup", async () => {
  const loading = await requestedIssue(tab.nerve, read, ["GET", "/api/v0/issues/parent/sub-issues"]);
  const deleting = await requestedIssue(
    tab.nerve,
    () => tab.app.issue.deleteIssue(parent.id, tab.app.issue.scope("ws", "A")),
    ["DELETE", "/api/v0/issues/parent"]
  );
  deleting.call.answer(noContent());
  await deleting.promise;
  loading.call.answer(json(200, { data: [child] }));
  await expect(loading.promise).rejects.toBeInstanceOf(SessionChangedError);
  expect(tab.nerve.calls).toHaveLength(2);
  expect(tab.app.issue.issues.held(child.id)).toBeUndefined();
  expect(store().subIssuesByIssueId(parent.id)).toBeUndefined();
});
it("deletedParentDiscardsSubReadWaitingForAuthoritativeParent", async () => {
  const reading = await requestedIssue(tab.nerve, read, ["GET", "/api/v0/issues/parent/sub-issues"]);
  reading.call.answer(json(200, { data: [child] }));
  await until(() => tab.nerve.to("/api/v0/projects/A/states").length > 0, "states");
  tab.nerve.to("/api/v0/projects/A/states")[0]!.answer(json(200, { data: [] }));
  await until(() => tab.nerve.to("/api/v0/issues/parent").length > 0, "parent verification");
  const deleting = await requestedIssue(
    tab.nerve,
    () => tab.app.issue.deleteIssue(parent.id, tab.app.issue.scope("ws", "A")),
    ["DELETE", "/api/v0/issues/parent"]
  );
  deleting.call.answer(noContent());
  await deleting.promise;
  tab.nerve.to("/api/v0/issues/parent")[0]!.answer(json(200, parent));
  await expect(reading.promise).rejects.toBeInstanceOf(SessionChangedError);
  expect(store().subIssuesByIssueId(parent.id)).toBeUndefined();
  expect(store().stateDistributionByIssueId(parent.id)).toBeUndefined();
  expect(tab.app.issue.issues.held(child.id)).toBeNull();
});
it.each(["unknown", "revoked"])("rejects %s scope before GET", async (kind) => {
  if (kind === "revoked") await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, []);
  const count = tab.nerve.calls.length;
  await expect(store().fetchSubIssues("ws", kind === "unknown" ? "unknown" : "A", parent.id)).rejects.toBeInstanceOf(
    SessionChangedError
  );
  expect(tab.nerve.calls).toHaveLength(count);
});
it("lets a legitimate guest send a request and see its refusal", async () => {
  await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [{ ...tab.project, member_role: 5 }]);
  const sent = await requestedIssue(tab.nerve, read, ["GET", "/api/v0/issues/parent/sub-issues"]);
  sent.call.answer(problem(403, "forbidden"));
  await expect(sent.promise).rejects.toMatchObject({ status: 403 });
});
it("discards a response across loss and recovery of its own project", async () => {
  const sent = await requestedIssue(tab.nerve, read, ["GET", "/api/v0/issues/parent/sub-issues"]);
  await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, []);
  await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [tab.project]);
  const count = tab.nerve.calls.length;
  sent.call.answer(json(200, { data: [child] }));
  await expect(sent.promise).rejects.toBeInstanceOf(SessionChangedError);
  expect(tab.nerve.calls).toHaveLength(count);
  expect(tab.app.issue.issues.held(child.id)).toBeUndefined();
});
it.each(["project_id", "workspace_id", "parent_id"])(
  "rejects a child with mismatched %s before any canonical write",
  async (key) => {
    const sent = await requestedIssue(tab.nerve, read, ["GET", "/api/v0/issues/parent/sub-issues"]);
    sent.call.answer(json(200, { data: [{ ...child, [key]: "wrong" }] }));
    await expect(sent.promise).rejects.toBeInstanceOf(SessionChangedError);
    expect(tab.app.issue.issues.held(child.id)).toBeUndefined();
  }
);
it("does not publish a partial resource when its required directory fails", async () => {
  const sent = await requestedIssue(tab.nerve, read, ["GET", "/api/v0/issues/parent/sub-issues"]);
  sent.call.answer(json(200, { data: [child] }));
  await until(() => tab.nerve.to("/api/v0/projects/A/states").length > 0, "states");
  tab.nerve.to("/api/v0/projects/A/states")[0]!.answer(problem(500, "unavailable"));
  await expect(sent.promise).rejects.toMatchObject({ status: 500 });
  expect(store().subIssuesByIssueId(parent.id)).toBeUndefined();
});
