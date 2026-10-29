/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import type { IssueRelation } from "@nerve/api-client";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { json, noContent, problem } from "@/lib/auth/fake-nerve";
import { until, track } from "@/lib/auth/fake-time";
import { SessionChangedError } from "@/lib/auth/token-manager";
import { issueOf, issueDetailOf, issueStoreTab, requestedIssue } from "../fake-issues";
import { loadProjects, projectOf } from "@/store/project/fake-projects";

vi.stubGlobal("localStorage", { getItem: () => null, setItem: () => {}, removeItem: () => {} });
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: {}, publicClient: {} }));
vi.mock("@/lib/store-context", () => ({ rootStore: undefined }));
let tab: Awaited<ReturnType<typeof issueStoreTab>>;
const external = projectOf("B", "id-ws", { id: "B" });
const row: IssueRelation = {
  id: "relation-93",
  relation_type: "blocking",
  issue: issueOf("target-24", "B"),
  created_at: "2026-10-29T00:00:00Z",
};
const store = () => tab.app.issue.issueDetail.relation;
const read = () => store().fetchRelations("ws", "A", "source");
beforeEach(async () => {
  vi.useFakeTimers();
  tab = await issueStoreTab();
  tab.app.issue.issues.acceptRead(issueDetailOf("source"), tab.app.issue.captureRead());
  await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [tab.project, external]);
  tab.nerve.calls.length = 0;
});
afterEach(() => {
  tab.app.dispose();
  vi.restoreAllMocks();
  vi.useRealTimers();
});
function directories() {
  tab.nerve.respond = (call) => {
    if (call.method === "GET" && /^\/api\/v0\/projects\/B\/(states|members|labels)$/.test(call.path))
      call.answer(json(200, { data: [] }));
  };
}
async function loaded() {
  directories();
  const sent = await requestedIssue(tab.nerve, read, ["GET", "/api/v0/issues/source/relations"]);
  sent.call.answer(json(200, { data: [row] }));
  await sent.promise;
}
it("holds relation resource IDs while all target fields come from canonical", async () => {
  await loaded();
  expect(store().getRelationsByIssueId("source")?.blocking).toEqual([row.issue.id]);
  expect(store().getRelationRows("source")).toEqual([
    { relationId: row.id, type: row.relation_type, targetId: row.issue.id, createdAt: row.created_at },
  ]);
  const root = tab.app.issue;
  root.issues.acceptRead({ ...row.issue, name: "New title", updated_at: "2026-10-30T00:00:00Z" }, root.captureRead());
  expect(root.issueDetail.issue.getIssueById(store().getRelationRows("source")![0]!.targetId)?.name).toBe("New title");
  expect(tab.nerve.calls.some((call) => /cycles|modules/.test(call.path))).toBe(false);
});
it("deletes by relation ID only after confirmation and replays over an older GET", async () => {
  await loaded();
  const old = await requestedIssue(tab.nerve, read, ["GET", "/api/v0/issues/source/relations"]);
  const removal = await requestedIssue(
    tab.nerve,
    () => store().removeRelation("ws", "A", "source", "blocking", row.issue.id),
    ["DELETE", "/api/v0/issue-relations/relation-93"]
  );
  expect(store().getRelationsByIssueId("source")?.blocking).toEqual([row.issue.id]);
  removal.call.answer(noContent());
  // Endpoint details are authoritative followups, never another DELETE.
  await until(() => tab.nerve.to("/api/v0/issues/source").length > 0, "source detail");
  for (const [id, project] of [
    ["source", "A"],
    [row.issue.id, "B"],
  ]) {
    expect(tab.nerve.to(`/api/v0/issues/${id}`)).toHaveLength(1);
    tab.nerve.to(`/api/v0/issues/${id}`)[0]!.answer(json(200, issueDetailOf(id!, project)));
  }
  await removal.promise;
  old.call.answer(json(200, { data: [row] }));
  await old.promise;
  expect(store().getRelationsByIssueId("source")?.blocking).toEqual([]);
  expect(tab.nerve.calls.filter((call) => call.method === "DELETE")).toHaveLength(1);
});
it("a rejected delete retains the resource and starts no followup", async () => {
  await loaded();
  const at = tab.nerve.calls.length;
  const sent = await requestedIssue(
    tab.nerve,
    () => store().removeRelation("ws", "A", "source", "blocking", row.issue.id),
    ["DELETE", "/api/v0/issue-relations/relation-93"]
  );
  sent.call.answer(problem(403, "forbidden"));
  await expect(sent.promise).rejects.toMatchObject({ status: 403 });
  expect(store().getRelationsByIssueId("source")?.blocking).toEqual([row.issue.id]);
  expect(tab.nerve.calls).toHaveLength(at + 1);
});
it.each([403, 500])("retains a confirmed relation on post-write %s and retry sends only reads", async (status) => {
  await loaded();
  const sent = await requestedIssue(
    tab.nerve,
    () => store().createRelation("ws", "A", "source", "blocking", [row.issue.id]),
    ["POST", "/api/v0/issues/source/relations"]
  );
  expect(sent.call.body).toEqual({ relation_type: "blocking", issue_ids: [row.issue.id] });
  sent.call.answer(json(200, { data: [row, row] }));
  await until(() => tab.nerve.to("/api/v0/issues/source").length > 0, "post-write details");
  for (const id of ["source", row.issue.id])
    tab.nerve.to(`/api/v0/issues/${id}`)[0]!.answer(problem(status, "unavailable"));
  await sent.promise;
  expect(store().getRelationRows("source")).toHaveLength(1);
  expect(store().getError("source")).toMatchObject({ status });
  const retry = store().retry("ws", "A", "source");
  await until(() => tab.nerve.to("/api/v0/issues/source").length === 2, "retry details");
  tab.nerve.to("/api/v0/issues/source")[1]!.answer(json(200, issueDetailOf("source")));
  tab.nerve.to(`/api/v0/issues/${row.issue.id}`)[1]!.answer(json(200, issueDetailOf(row.issue.id, "B")));
  await retry;
  expect(store().getError("source")).toBeUndefined();
  expect(tab.nerve.calls.filter((call) => call.method === "POST")).toHaveLength(1);
});
it.each(["during", "held"])(
  "revokes a cross-project %s resource immediately and never revives it on ABA",
  async (when) => {
    if (when === "held") await loaded();
    const sent = await requestedIssue(tab.nerve, read, ["GET", "/api/v0/issues/source/relations"]);
    await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [tab.project]);
    expect(store().getRelationsByIssueId("source")).toBeUndefined();
    await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [tab.project, external]);
    const at = tab.nerve.calls.length;
    sent.call.answer(json(200, { data: [row] }));
    await expect(sent.promise).rejects.toBeInstanceOf(SessionChangedError);
    expect(store().getRelationsByIssueId("source")).toBeUndefined();
    expect(tab.nerve.calls).toHaveLength(at);
  }
);
it("retrying a confirmed create never replays rows removed by a later authoritative read", async () => {
  await loaded();
  const sent = await requestedIssue(
    tab.nerve,
    () => store().createRelation("ws", "A", "source", "blocking", [row.issue.id]),
    ["POST", "/api/v0/issues/source/relations"]
  );
  sent.call.answer(json(200, { data: [row] }));
  await until(() => tab.nerve.to("/api/v0/issues/source").length > 0, "endpoint reads");
  for (const id of ["source", row.issue.id])
    tab.nerve.to(`/api/v0/issues/${id}`)[0]!.answer(problem(500, "unavailable"));
  await sent.promise;
  const fresh = await requestedIssue(tab.nerve, read, ["GET", "/api/v0/issues/source/relations"]);
  fresh.call.answer(json(200, { data: [] }));
  await fresh.promise;
  const retry = store().retry("ws", "A", "source");
  await until(() => tab.nerve.to("/api/v0/issues/source").length === 2, "retry endpoints");
  tab.nerve.to("/api/v0/issues/source")[1]!.answer(json(200, issueDetailOf("source")));
  tab.nerve.to(`/api/v0/issues/${row.issue.id}`)[1]!.answer(json(200, issueDetailOf(row.issue.id, "B")));
  await retry;
  expect(store().getRelationRows("source")).toEqual([]);
  expect(tab.nerve.calls.filter((call) => call.method === "POST")).toHaveLength(1);
});
it("requires directory permission before accepting a cold target", async () => {
  await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [tab.project]);
  const sent = await requestedIssue(tab.nerve, read, ["GET", "/api/v0/issues/source/relations"]);
  sent.call.answer(json(200, { data: [row] }));
  await until(() => tab.nerve.to("/api/v0/projects/B").length > 0, "cold directory");
  tab.nerve.to("/api/v0/projects/B")[0]!.answer(problem(403, "forbidden"));
  await expect(sent.promise).rejects.toMatchObject({ status: 403 });
  expect(tab.app.issue.issues.held(row.issue.id)).toBeUndefined();
});
it("discards a relation response across loss and recovery of its own project", async () => {
  const sent = await requestedIssue(tab.nerve, read, ["GET", "/api/v0/issues/source/relations"]);
  await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [external]);
  await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [tab.project, external]);
  const at = tab.nerve.calls.length;
  sent.call.answer(json(200, { data: [row] }));
  await expect(sent.promise).rejects.toBeInstanceOf(SessionChangedError);
  expect(tab.nerve.calls).toHaveLength(at);
  expect(tab.app.issue.issues.held(row.issue.id)).toBeUndefined();
});
it.each(["unknown", "revoked", "guest"])("checks %s base scope", async (kind) => {
  if (kind !== "unknown")
    await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [
      { ...tab.project, member_role: kind === "guest" ? 5 : null },
      external,
    ]);
  const operation = () => store().fetchRelations("ws", kind === "unknown" ? "unknown" : "A", "source");
  if (kind !== "guest") {
    const at = tab.nerve.calls.length;
    await expect(operation()).rejects.toBeInstanceOf(SessionChangedError);
    expect(tab.nerve.calls).toHaveLength(at);
  } else {
    const sent = await requestedIssue(tab.nerve, operation, ["GET", "/api/v0/issues/source/relations"]);
    sent.call.answer(problem(403, "forbidden"));
    await expect(sent.promise).rejects.toMatchObject({ status: 403 });
  }
});
it.each(["mismatch", "deleted"])("rejects a known %s source before reading its relations", async (kind) => {
  const root = tab.app.issue;
  let id = "source";
  if (kind === "mismatch") {
    id = "wrong-source";
    root.issues.acceptRead(issueOf(id, "B"), root.captureRead());
  } else {
    const removal = await requestedIssue(tab.nerve, () => root.deleteIssue(id, root.scope("ws", "A")), [
      "DELETE",
      "/api/v0/issues/source",
    ]);
    removal.call.answer(noContent());
    await removal.promise;
  }
  const at = tab.nerve.calls.length;
  const loading = track(store().fetchRelations("ws", "A", id));
  await vi.advanceTimersByTimeAsync(20);
  expect(tab.nerve.calls).toHaveLength(at);
  expect(loading.error).toBeInstanceOf(SessionChangedError);
});
