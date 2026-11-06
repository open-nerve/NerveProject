/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { json, problem } from "@/lib/auth/fake-nerve";
import { track, until } from "@/lib/auth/fake-time";
import { loadProjects } from "@/store/project/fake-projects";
import { contentAssetTab } from "@/store/asset/fake-assets";
import { IssueDetailInputState } from "./detail-input-state";
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: {}, publicClient: {} }));
vi.mock("@/lib/store-context", () => ({ rootStore: {} }));
const close: (() => void)[] = [];
beforeEach(() => vi.useFakeTimers());
afterEach(() => {
  close.splice(0).forEach((done) => done());
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
  vi.useRealTimers();
});
async function fixture() {
  const tab = await contentAssetTab();
  const inputs = new IssueDetailInputState(tab.app.issue, tab.workspace.id, tab.project.id, tab.issue.id);
  inputs.activate();
  close.push(() => {
    inputs.deactivate();
    tab.app.dispose();
  });
  return { ...tab, inputs };
}
const snapshot = {
  value: "Unsent title",
  confirmed: "Server title",
  revision: 3,
  attempted: 2,
  dirty: true,
  failure: undefined,
};
it("keeps only local input and confirmed receipts through pending and a temporary owner read refusal", async () => {
  const tab = await fixture();
  const slot = tab.inputs.slot("title");
  slot.store(snapshot);
  tab.inputs.confirm({ kind: "uploaded", id: "uploaded-file" }, "files");
  const read = track(tab.app.issue.issues.refreshAuthoritative(tab.issue.id, { afterWrite: true }));
  expect(tab.app.issue.issues.getIssueDetailById(tab.issue.id)).toBeUndefined();
  expect(tab.inputs.current).toBe(true);
  expect(slot.read()).toEqual(snapshot);
  await until(() => tab.nerve.to(`/api/v0/issues/${tab.issue.id}`).length === 1, "owner pending");
  tab.nerve.to(`/api/v0/issues/${tab.issue.id}`)[0]!.answer(problem(503, "server_busy"));
  await until(() => read.settled, "owner refused");
  expect(read.error).toMatchObject({ status: 503 });
  expect(slot.read()).toEqual(snapshot);
  expect(tab.inputs.receipts).toEqual([{ kind: "uploaded", id: "uploaded-file" }]);
  await tab.readIssue();
  expect(slot.read()).toEqual(snapshot);
  // The focus the remount lost returns once, to the control that sent the command.
  expect(tab.inputs.takeFocus("other files")).toBeUndefined();
  expect(tab.inputs.takeFocus("files")).toBe("uploaded");
  expect(tab.inputs.takeFocus("files")).toBeUndefined();
  // The control has its focus back; the notice stays with the shown body and ends when the body is next withdrawn.
  expect(tab.inputs.receipts).toEqual([{ kind: "uploaded", id: "uploaded-file" }]);
  const later = track(tab.app.issue.issues.refreshAuthoritative(tab.issue.id, { afterWrite: true }));
  expect(tab.app.issue.issues.getIssueDetailById(tab.issue.id)).toBeUndefined();
  expect(tab.inputs.receipts).toEqual([]);
  const reads = () => tab.nerve.to(`/api/v0/issues/${tab.issue.id}`);
  await until(() => reads().length === 3, "later owner read");
  reads()[2]!.answer(json(200, tab.issue));
  await until(() => later.settled, "later owner read settled");
});
it("a new command ends the previous result's notice and its pending focus return", async () => {
  const tab = await fixture();
  tab.inputs.confirm({ kind: "uploaded", id: "uploaded-file" }, "files");
  tab.inputs.beginCommand();
  expect(tab.inputs.receipts).toEqual([]);
  expect(tab.inputs.takeFocus("files")).toBeUndefined();
  tab.inputs.confirm({ kind: "deleted", id: "deleted-file" }, "files");
  expect(tab.inputs.receipts).toEqual([{ kind: "deleted", id: "deleted-file" }]);
});
it.each([401, 403, 404])(
  "a final owner %i retires local input permanently, even when the same ID returns",
  async (status) => {
    const tab = await fixture();
    const slot = tab.inputs.slot("title");
    slot.store(snapshot);
    const read = track(tab.app.issue.issues.refreshAuthoritative(tab.issue.id));
    await until(() => tab.nerve.calls.length === 1, "owner read");
    tab.nerve.calls[0]!.answer(problem(status, status === 404 ? "issue.not_found" : "forbidden"));
    await until(() => read.settled, "owner refusal");
    expect(tab.inputs.current).toBe(false);
    expect(slot.read()).toBeUndefined();
    expect(tab.inputs.receipts).toEqual([]);
    await tab.readIssue();
    slot.store(snapshot);
    expect(slot.read()).toBeUndefined();
    expect(tab.inputs.current).toBe(false);
  }
);
it("permission ABA cannot revive an earlier host or its local edits", async () => {
  const tab = await fixture();
  const slot = tab.inputs.slot("title");
  slot.store(snapshot);
  await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, []);
  await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [tab.project]);
  await tab.readIssue();
  expect(tab.inputs.current).toBe(false);
  expect(slot.read()).toBeUndefined();
  tab.inputs.confirm({ kind: "deleted", id: "old" }, "files");
  expect(tab.inputs.receipts).toEqual([]);
  expect(tab.inputs.takeFocus("files")).toBeUndefined();
  const fresh = new IssueDetailInputState(tab.app.issue, tab.workspace.id, tab.project.id, tab.issue.id);
  fresh.activate();
  close.push(() => fresh.deactivate());
  expect(fresh.current).toBe(true);
  expect(fresh.slot("title").read()).toBeUndefined();
});
it("a confirmed issue deletion clears input without borrowing another issue's row", async () => {
  const tab = await fixture();
  const slot = tab.inputs.slot("title");
  slot.store(snapshot);
  tab.app.issue.issues.confirmChange({
    id: tab.issue.id,
    beforeSnapshot: { kind: "known", issue: tab.issue },
    after: null,
  });
  expect(tab.inputs.current).toBe(false);
  expect(slot.read()).toBeUndefined();
});
it("host instances and form keys do not share local text, receipts or selected IDs", async () => {
  const tab = await fixture();
  const peek = new IssueDetailInputState(tab.app.issue, tab.workspace.id, tab.project.id, tab.issue.id);
  peek.activate();
  close.push(() => peek.deactivate());
  tab.inputs.slot("title").store(snapshot);
  tab.inputs.slot("description").store({ ...snapshot, value: "<p>Body</p>" });
  tab.inputs
    .form("new-comment")
    .store({ html: "<p>Comment</p>", baseHTML: "", attachmentIDs: ["asset-a"], revision: 1 });
  expect(peek.slot("title").read()).toBeUndefined();
  expect(tab.inputs.slot("description").read()?.value).toBe("<p>Body</p>");
  expect(tab.inputs.form("edit:comment-1").read()).toBeUndefined();
  const form = tab.inputs.form("new-comment").read()!;
  form.attachmentIDs.push("must-not-leak");
  expect(tab.inputs.form("new-comment").read()?.attachmentIDs).toEqual(["asset-a"]);
  tab.inputs.deactivate();
  expect(tab.inputs.slot("title").read()).toBeUndefined();
  expect(peek.current).toBe(true);
});
it("StrictMode reactivation starts empty and old slots cannot write into the new activation", async () => {
  const tab = await fixture();
  const old = tab.inputs.slot("title");
  old.store(snapshot);
  tab.inputs.deactivate();
  tab.inputs.activate();
  old.store(snapshot);
  expect(old.read()).toBeUndefined();
  expect(tab.inputs.slot("title").read()).toBeUndefined();
  expect(tab.inputs.current).toBe(true);
});
it("a known write refusal is local input, not a reason to discard it", async () => {
  const tab = await fixture();
  const error = { status: 422 };
  const slot = tab.inputs.slot("description");
  slot.store({ ...snapshot, failure: { error }, attempted: 3 });
  expect(slot.read()?.failure?.error).toBe(error);
  expect(tab.inputs.current).toBe(true);
});
