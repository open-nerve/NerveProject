/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { SessionChangedError } from "@/lib/auth/token-manager";
import { loadProject, loadProjects, projectOf } from "@/store/project/fake-projects";
import { loadWorkspaces, workspaceOf } from "@/store/workspace/fake-workspaces";
import { contentAssetTab } from "./fake-assets";

vi.mock("@/lib/auth/api-client", () => ({ tokenManager: {}, publicClient: {} }));
vi.mock("@/lib/store-context", () => ({ rootStore: {} }));
const close: (() => void)[] = [];
beforeEach(() => vi.useFakeTimers());
afterEach(() => {
  close.splice(0).forEach((dispose) => dispose());
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
  vi.useRealTimers();
});
async function fixture() {
  const tab = await contentAssetTab();
  close.push(() => tab.app.dispose());
  return tab;
}

it("permits a member's issue writes without imposing project cover administration", async () => {
  const tab = await fixture();
  await loadWorkspaces(tab.nerve, tab.app.workspaceRoot, [{ ...tab.workspace, role: 15 }]);
  await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [{ ...tab.project, member_role: 15 }]);
  await tab.readIssue();
  const view = tab.app.asset.openContent(tab.hosts.issue);
  expect(view.current).toBe(true);
  expect(view.canWrite("attachment")).toBe(true);
  expect(view.canWrite("description")).toBe(true);
});

it("a guest can comment on a loaded issue and edit their own comment but cannot edit the issue", async () => {
  const tab = await fixture();
  await loadWorkspaces(tab.nerve, tab.app.workspaceRoot, [{ ...tab.workspace, role: 5 }]);
  await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [{ ...tab.project, member_role: 5 }]);
  await tab.readIssue();
  await tab.readComments();
  const issue = tab.app.asset.openContent(tab.hosts.issue);
  expect(issue.current).toBe(true);
  expect(issue.canWrite("attachment")).toBe(false);
  expect(tab.app.asset.openContent(tab.hosts.new_comment).canWrite("attachment")).toBe(true);
  expect(tab.app.asset.openContent(tab.hosts.comment).canWrite("description")).toBe(true);
  expect(tab.app.asset.openContent(tab.hosts.draft).current).toBe(false);
});

it("administration does not allow adding files to another author's comment", async () => {
  const tab = await fixture();
  await tab.readComments([{ ...tab.comment, actor_id: "someone-else" }]);
  const view = tab.app.asset.openContent(tab.hosts.comment);
  expect(view.current).toBe(true);
  expect(view.canWrite("attachment")).toBe(false);
  expect(view.canWrite("description")).toBe(false);
});

it.each(["issue", "project"])("keeps an archived %s readable but without content writes", async (kind) => {
  const tab = await fixture();
  if (kind === "issue")
    await tab.readIssue({ ...tab.issue, archived_at: "2026-11-05T00:00:00Z", updated_at: "2026-11-05T00:00:00Z" });
  else
    await loadProject(tab.nerve, tab.app.projectRoot.project, {
      ...tab.project,
      archived_at: "2026-11-05T00:00:00Z",
      updated_at: "2026-11-05T00:00:00Z",
    });
  const view = tab.app.asset.openContent(tab.hosts.issue);
  expect(view.current).toBe(true);
  expect(view.canWrite("description")).toBe(false);
  expect(tab.app.asset.openContent(tab.hosts.new_comment).canWrite("attachment")).toBe(false);
});

it("private drafts do not require project membership, including null and archived projects", async () => {
  const tab = await fixture();
  await loadWorkspaces(tab.nerve, tab.app.workspaceRoot, [{ ...tab.workspace, role: 15 }]);
  await loadProject(tab.nerve, tab.app.projectRoot.project, {
    ...tab.project,
    member_role: null,
    archived_at: "2026-11-05T00:00:00Z",
  });
  await tab.readDrafts();
  expect(tab.app.asset.openContent(tab.hosts.draft).canWrite("attachment")).toBe(true);
  await tab.readDrafts([{ ...tab.draft, project_id: tab.project.id, updated_at: "2026-11-05T00:00:01Z" }]);
  expect(tab.app.asset.openContent({ ...tab.hosts.draft, projectID: tab.project.id }).canWrite("description")).toBe(
    true
  );
});

it("project visibility alone cannot authorize an old issue detail", async () => {
  const tab = await fixture();
  await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [{ ...tab.project, member_role: null }]);
  const view = tab.app.asset.openContent(tab.hosts.issue);
  expect(view.current).toBe(false);
  expect(view.signal.aborted).toBe(true);
  expect(() => tab.app.asset.createContentUpload(view, "attachment")).toThrow(SessionChangedError);
});

it.each(["issue", "comment", "draft"] as const)(
  "an unread $kind host has no capability and cannot later revive",
  async (kind) => {
    const tab = await fixture();
    const host =
      kind === "issue"
        ? { ...tab.hosts.issue, issueID: "missing" }
        : kind === "comment"
          ? { ...tab.hosts.comment, commentID: "missing" }
          : { ...tab.hosts.draft, draftID: "missing" };
    const view = tab.app.asset.openContent(host);
    expect(view.current).toBe(false);
    expect(() => tab.app.asset.createContentUpload(view, "attachment")).toThrow(SessionChangedError);
    expect(tab.nerve.calls).toHaveLength(0);
    if (kind === "issue") await tab.readIssue({ ...tab.issue, id: "missing" });
    if (kind === "comment") await tab.readComments([{ ...tab.comment, id: "missing" }]);
    if (kind === "draft") await tab.readDrafts([{ ...tab.draft, id: "missing" }]);
    expect(view.current).toBe(false);
    expect(tab.app.asset.openContent(host).current).toBe(true);
  }
);

it("rejects caller-supplied parent mismatches without extra owner reads", async () => {
  const tab = await fixture();
  for (const host of [
    { ...tab.hosts.issue, projectID: "other-project" },
    { ...tab.hosts.issue, workspaceID: "other-workspace" },
    { ...tab.hosts.comment, issueID: "other-issue" },
    { ...tab.hosts.draft, projectID: "A" },
  ])
    expect(tab.app.asset.openContent(host).current).toBe(false);
  expect(tab.nerve.calls).toHaveLength(0);
});

it("retires a comment view on confirmed owner absence, even when the same ID returns", async () => {
  const tab = await fixture();
  const view = tab.app.asset.openContent(tab.hosts.comment);
  expect(view.current).toBe(true);
  await tab.readComments([]);
  expect(view.signal.aborted).toBe(true);
  await tab.readComments();
  expect(view.current).toBe(false);
  expect(tab.app.asset.openContent(tab.hosts.comment).current).toBe(true);
});

it("retires draft A-to-B-to-A views without requiring a project role", async () => {
  const tab = await fixture();
  const view = tab.app.asset.openContent(tab.hosts.draft);
  await tab.readDrafts([{ ...tab.draft, project_id: "A", updated_at: "2026-11-05T00:00:01Z" }]);
  expect(view.signal.aborted).toBe(true);
  await tab.readDrafts([{ ...tab.draft, updated_at: "2026-11-05T00:00:02Z" }]);
  expect(view.current).toBe(false);
  expect(tab.app.asset.openContent(tab.hosts.draft).current).toBe(true);
});

it("respects the draft owner's conservative collection lifetime while an unrelated issue project remains independent", async () => {
  const tab = await fixture();
  const peer = projectOf("peer", tab.workspace.id);
  await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [tab.project, peer]);
  await tab.readIssue();
  await tab.readDrafts();
  const issueView = tab.app.asset.openContent(tab.hosts.issue);
  const draftView = tab.app.asset.openContent(tab.hosts.draft);
  await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [tab.project]);
  expect(issueView.current).toBe(true);
  expect(draftView.current).toBe(false);
  await tab.readDrafts();
  expect(draftView.current).toBe(false);
  expect(tab.app.asset.openContent(tab.hosts.draft).current).toBe(true);
});

it("ignores another workspace's membership change", async () => {
  const tab = await fixture();
  const other = workspaceOf("other");
  await loadWorkspaces(tab.nerve, tab.app.workspaceRoot, [tab.workspace, other]);
  const views = Object.values(tab.hosts).map((host) => tab.app.asset.openContent(host));
  await loadWorkspaces(tab.nerve, tab.app.workspaceRoot, [tab.workspace, { ...other, role: 5 }]);
  expect(views.map((view) => view.current)).toEqual([true, true, true, true]);
});

it("does not lend a different root's view to the current upload service", async () => {
  const first = await fixture();
  const view = first.app.asset.openContent(first.hosts.issue);
  const second = await fixture();
  expect(() => second.app.asset.createContentUpload(view, "attachment")).toThrow(SessionChangedError);
  expect(second.nerve.calls).toHaveLength(0);
});

it("freezes the host identity and retires every view on root disposal", async () => {
  const tab = await fixture();
  const input = { ...tab.hosts.issue };
  const view = tab.app.asset.openContent(input);
  input.issueID = "another";
  expect(view.host).toMatchObject({ issueID: "issue-1" });
  tab.app.dispose();
  expect(view.current).toBe(false);
  expect(view.signal.aborted).toBe(true);
});
