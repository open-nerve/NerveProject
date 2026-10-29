/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { json, problem } from "@/lib/auth/fake-nerve";
import { track, until } from "@/lib/auth/fake-time";
import { SessionChangedError } from "@/lib/auth/token-manager";
import { loadProjects, projectOf } from "@/store/project/fake-projects";
import { loadWorkspaces } from "@/store/workspace/fake-workspaces";
import { descriptionVersionOf, issueDetailOf, issueStoreTab, requestedIssue } from "../fake-issues";
vi.stubGlobal("localStorage", { getItem: () => null, setItem: () => {}, removeItem: () => {} });
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: {}, publicClient: {} }));
vi.mock("@/lib/store-context", () => ({ rootStore: undefined }));
let tab: Awaited<ReturnType<typeof issueStoreTab>>;
const store = () => tab.app.issue.issueDetail.versions;
const route = "/api/v0/issues/x/description-versions";
const row = descriptionVersionOf("z");
const list = (next = false) => store().fetchVersions("ws", "A", "x", next);
const detail = (id = "z", issue = "x") => store().fetchVersion("ws", "A", issue, id);
beforeEach(async () => {
  vi.useFakeTimers();
  tab = await issueStoreTab();
});
afterEach(() => {
  tab.app.dispose();
  vi.useRealTimers();
});
async function loaded() {
  const sent = await requestedIssue(tab.nerve, list, ["GET", route]);
  sent.call.answer(json(200, { data: [row], next_cursor: "opaque+/=" }));
  await sent.promise;
}
it("loads pages on demand in last_saved_at / id descending order with nanosecond precision", async () => {
  await loaded();
  const sent = await requestedIssue(tab.nerve, () => list(true), ["GET", route]);
  expect(sent.call.queryEntries).toEqual([["cursor", "opaque+/="]]);
  const newer = { ...row, id: "a", last_saved_at: "2026-10-30T00:01:00.123456790Z" };
  const same = { ...row, id: "y" };
  sent.call.answer(json(200, { data: [same, newer, row], next_cursor: null }));
  await sent.promise;
  expect(store().getVersions("x")).toEqual([newer, row, same]);
  expect(store().hasMore("x")).toBe(false);
  const before = tab.nerve.calls.length;
  await list(true);
  expect(tab.nerve.calls).toHaveLength(before);
});
it("a refused next page keeps the loaded page and cursor for a read-only retry", async () => {
  await loaded();
  const sent = await requestedIssue(tab.nerve, () => list(true), ["GET", route]);
  sent.call.answer(problem(500, "unavailable"));
  await expect(sent.promise).rejects.toMatchObject({ status: 500 });
  expect(store().getVersions("x")).toEqual([row]);
  expect(store().getError("x")).toMatchObject({ status: 500 });
  const again = await requestedIssue(tab.nerve, () => list(true), ["GET", route]);
  expect(again.call.queryEntries).toEqual([["cursor", "opaque+/="]]);
  again.call.answer(json(200, { data: [], next_cursor: null }));
  await again.promise;
  expect(store().getError("x")).toBeUndefined();
});
it.each([5, 15, 20] as const)("only effective project members/admins may read versions (role %s)", async (role) => {
  await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [{ ...tab.project, member_role: role }]);
  tab.nerve.calls.length = 0;
  if (role === 5) {
    const first = track(list());
    const second = track(detail());
    await until(() => (first.settled && second.settled) || tab.nerve.calls.length > 0, "refused guest");
    expect(tab.nerve.calls).toHaveLength(0);
    expect(first.error).toBeInstanceOf(SessionChangedError);
    expect(second.error).toBeInstanceOf(SessionChangedError);
  } else {
    await loaded();
    const sent = await requestedIssue(tab.nerve, detail, ["GET", "/api/v0/issue-description-versions/z"]);
    sent.call.answer(json(200, { ...row, description_html: "" }));
    expect(await sent.promise).toMatchObject({ description_html: "" });
  }
});
it("workspace admin has effective project admin access", async () => {
  await loadWorkspaces(tab.nerve, tab.app.workspaceRoot, [{ ...tab.workspace, role: 20 }]);
  await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [{ ...tab.project, member_role: 5 }]);
  await loaded();
  expect(store().getVersions("x")).toEqual([row]);
});
it.each(["permission", "retire"])("hides prior versions and rejects late detail on %s", async (boundary) => {
  await loaded();
  const sent = await requestedIssue(tab.nerve, detail, ["GET", "/api/v0/issue-description-versions/z"]);
  if (boundary === "retire") tab.app.issue.retire();
  else {
    await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [{ ...tab.project, member_role: 5 }]);
    expect(store().getVersions("x")).toBeUndefined();
    await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [tab.project]);
  }
  sent.call.answer(json(200, { ...row, description_html: "old secret" }));
  await expect(sent.promise).rejects.toBeInstanceOf(SessionChangedError);
  expect(store().getVersions("x")).toBeUndefined();
  expect(store().getVersion("x", "z")).toBeUndefined();
});
it("independent version reads keep their own identity and empty HTML", async () => {
  const a = await requestedIssue(tab.nerve, detail, ["GET", "/api/v0/issue-description-versions/z"]);
  const b = await requestedIssue(tab.nerve, () => detail("a"), ["GET", "/api/v0/issue-description-versions/a"]);
  b.call.answer(json(200, { ...descriptionVersionOf("a"), description_html: "" }));
  await b.promise;
  a.call.answer(json(200, { ...row, description_html: "old" }));
  await a.promise;
  expect(store().getVersion("x", "a")?.description_html).toBe("");
  expect(store().getVersion("other", "a")).toBeUndefined();
  expect(store().getVersion("x", "z")?.description_html).toBe("old");
});
it.each(["list", "detail"])("rejects %s answers belonging to another issue", async (kind) => {
  const sent = await requestedIssue(tab.nerve, async () => (kind === "list" ? list() : detail()), [
    "GET",
    kind === "list" ? route : "/api/v0/issue-description-versions/z",
  ]);
  sent.call.answer(
    json(
      200,
      kind === "list"
        ? { data: [descriptionVersionOf("z", "wrong")], next_cursor: null }
        : { ...descriptionVersionOf("z", "wrong"), description_html: "secret" }
    )
  );
  await expect(sent.promise).rejects.toBeInstanceOf(SessionChangedError);
  expect(store().getVersions("x")).toBeUndefined();
  expect(store().getVersion("x", "z")).toBeUndefined();
});
it.each([403, 422, 200])("restore is one ordinary confirmed PATCH; status %s", async (status) => {
  const core = await requestedIssue(tab.nerve, () => tab.app.issue.issueDetail.issue.fetchIssueCore("ws", "A", "x"), [
    "GET",
    "/api/v0/issues/x",
  ]);
  core.call.answer(json(200, issueDetailOf("x")));
  await core.promise;
  const sent = await requestedIssue(tab.nerve, () => store().restore("ws", "A", "x", "<p>chosen</p>"), [
    "PATCH",
    "/api/v0/issues/x",
  ]);
  expect(sent.call.body).toEqual({ description_html: "<p>chosen</p>" });
  expect(tab.app.issue.issues.getIssueDetailById("x")?.description_html).toBe("<p>Server detail</p>");
  sent.call.answer(
    status === 200
      ? json(200, { ...issueDetailOf("x"), description_html: "<p>sanitized</p>" })
      : problem(status, "original.code")
  );
  if (status === 200) expect(await sent.promise).toBe("<p>sanitized</p>");
  else await expect(sent.promise).rejects.toMatchObject({ status });
  expect(tab.app.issue.issues.getIssueDetailById("x")?.description_html).toBe(
    status === 200 ? "<p>sanitized</p>" : "<p>Server detail</p>"
  );
  expect(tab.nerve.calls.map((call) => call.method)).toEqual(["GET", "PATCH"]);
});

it("a queued restoration cannot cross a permission ABA even after the new scope reads the issue", async () => {
  const load = async () => {
    const sent = await requestedIssue(tab.nerve, () => tab.app.issue.issueDetail.issue.fetchIssueCore("ws", "A", "x"), [
      "GET",
      "/api/v0/issues/x",
    ]);
    sent.call.answer(json(200, issueDetailOf("x")));
    await sent.promise;
  };
  await load();
  let release!: () => void;
  let started = false;
  const gate = tab.app.issue.enqueueIssueChanges(["x"], async () => {
    started = true;
    await new Promise<void>((yes) => {
      release = yes;
    });
  });
  await until(() => started, "queue reserved");
  const result = track(store().restore("ws", "A", "x", "old choice"));
  await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [{ ...tab.project, member_role: 5 }]);
  await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [tab.project]);
  await load();
  release();
  await gate;
  await until(
    () => result.settled || tab.nerve.calls.some((call) => call.method === "PATCH"),
    "restoration disposition"
  );
  expect(tab.nerve.calls.filter((call) => call.method === "PATCH")).toHaveLength(0);
  expect(result.error).toBeInstanceOf(SessionChangedError);
});
it("a confirmed old restoration cannot return HTML to a recovered permission scope", async () => {
  const core = await requestedIssue(tab.nerve, () => tab.app.issue.issueDetail.issue.fetchIssueCore("ws", "A", "x"), [
    "GET",
    "/api/v0/issues/x",
  ]);
  core.call.answer(json(200, issueDetailOf("x")));
  await core.promise;
  const sent = await requestedIssue(tab.nerve, () => store().restore("ws", "A", "x", "old choice"), [
    "PATCH",
    "/api/v0/issues/x",
  ]);
  await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, []);
  await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [tab.project]);
  sent.call.answer(json(200, { ...issueDetailOf("x"), description_html: "old confirmed" }));
  await expect(sent.promise).rejects.toBeInstanceOf(SessionChangedError);
});

it("does not restore an issue through another readable project's scope", async () => {
  const core = await requestedIssue(tab.nerve, () => tab.app.issue.issueDetail.issue.fetchIssueCore("ws", "A", "x"), [
    "GET",
    "/api/v0/issues/x",
  ]);
  core.call.answer(json(200, issueDetailOf("x")));
  await core.promise;
  await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [
    tab.project,
    projectOf("B", tab.workspace.id, { id: "B" }),
  ]);
  const result = track(store().restore("ws", "B", "x", "wrong scope"));
  await until(() => result.settled || tab.nerve.calls.some((call) => call.method === "PATCH"), "mismatched scope");
  expect(tab.nerve.calls.filter((call) => call.method === "PATCH")).toHaveLength(0);
  expect(result.error).toBeInstanceOf(SessionChangedError);
});
