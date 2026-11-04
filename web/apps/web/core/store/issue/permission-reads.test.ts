/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { json } from "@/lib/auth/fake-nerve";
import { track, until } from "@/lib/auth/fake-time";
import { loadProject, loadProjects, projectOf } from "@/store/project/fake-projects";
import { loadWorkspaces, workspaceOf } from "@/store/workspace/fake-workspaces";
import { issueStoreTab } from "./fake-issues";

vi.mock("@/lib/auth/api-client", () => ({ tokenManager: {}, publicClient: {} }));
vi.mock("@/lib/store-context", () => ({ rootStore: {} }));
const dispose: (() => void)[] = [];
beforeEach(() => vi.useFakeTimers());
afterEach(() => {
  dispose.splice(0).forEach((close) => close());
  vi.useRealTimers();
});
async function fixture() {
  const tab = await issueStoreTab();
  dispose.push(() => tab.app.dispose());
  return tab;
}

it.each([null, 15, 5] as const)(
  "PC1 visible project role %s loses its old read forever after absence and recovery",
  async (role) => {
    const { app, nerve, workspace, project } = await fixture();
    const visible = { ...project, member_role: role };
    await loadProjects(nerve, app.projectRoot.project, workspace, [visible]);
    await loadProject(nerve, app.projectRoot.project, visible);
    const token = app.issue.captureRead();
    let invalidations = 0;
    app.issue.registerResource({
      affects: (ws, id) => ws === workspace.id && id === project.id,
      invalidate: () => {
        ++invalidations;
      },
      dispose() {},
    });
    expect(app.issue.isReadCurrent(token, [project.id], workspace.id)).toBe(true);
    const delayed = track(app.projectRoot.project.fetchProject(project.id));
    const at = nerve.calls.length;
    await until(() => nerve.calls.length > at, "old project detail");
    const old = nerve.calls[at]!;
    await loadProjects(nerve, app.projectRoot.project, workspace, []);
    expect(app.projectRoot.project.getProjectById(project.id)).toBeUndefined();
    expect.soft(app.issue.isReadCurrent(token, [project.id], workspace.id)).toBe(false);
    await loadProjects(nerve, app.projectRoot.project, workspace, [visible]);
    old.answer(json(200, { ...visible, name: "old detail" }));
    await until(() => delayed.settled, "old detail after recovery");
    expect(app.projectRoot.project.getProjectById(project.id)?.name).toBe(project.name);
    expect.soft(app.issue.isReadCurrent(token, [project.id], workspace.id)).toBe(false);
    expect.soft(invalidations).toBe(1);
    expect(app.issue.isReadCurrent(app.issue.captureRead(), [project.id], workspace.id)).toBe(true);
    expect(app.user.permission.getProjectRoleByWorkspaceSlugAndProjectId(workspace.slug, project.id)).toBe(
      role ?? undefined
    );
    if (role === null) expect(() => app.issue.assertProjectReadable(workspace.slug, project.id)).toThrow();
  }
);

it("PC1 unchanged public visibility and unrelated project/workspace changes keep the scoped token valid", async () => {
  const { app, nerve, workspace, project } = await fixture();
  const visible = { ...project, member_role: null };
  const peer = projectOf("peer", workspace.id);
  const elsewhere = workspaceOf("else");
  const remote = projectOf("remote", elsewhere.id);
  await loadWorkspaces(nerve, app.workspaceRoot, [workspace, elsewhere]);
  await loadProjects(nerve, app.projectRoot.project, workspace, [visible, peer]);
  await loadProjects(nerve, app.projectRoot.project, elsewhere, [remote]);
  const token = app.issue.captureRead();
  await loadProjects(nerve, app.projectRoot.project, workspace, [{ ...visible, name: "renamed" }, peer]);
  await loadProjects(nerve, app.projectRoot.project, workspace, [visible]);
  await loadProjects(nerve, app.projectRoot.project, elsewhere, []);
  expect(app.issue.isReadCurrent(token, [project.id], workspace.id)).toBe(true);
});

it("PC1 retains guest restriction changes and workspace retirement", async () => {
  const { app, nerve, workspace, project } = await fixture();
  const guest = { ...project, member_role: 5 as const, guest_view_all_features: true };
  await loadProjects(nerve, app.projectRoot.project, workspace, [guest]);
  const before = app.issue.captureRead();
  await loadProjects(nerve, app.projectRoot.project, workspace, [{ ...guest, guest_view_all_features: false }]);
  expect(app.issue.isReadCurrent(before, [project.id], workspace.id)).toBe(false);
  const narrowed = app.issue.captureRead();
  await loadWorkspaces(nerve, app.workspaceRoot, []);
  await loadWorkspaces(nerve, app.workspaceRoot, [workspace]);
  expect(app.issue.isReadCurrent(narrowed, [project.id], workspace.id)).toBe(false);
  const fresh = app.issue.captureRead();
  app.dispose();
  expect(app.issue.isReadCurrent(fresh, [project.id], workspace.id)).toBe(false);
});
