/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { renderToStaticMarkup } from "react-dom/server";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { ProjectRole, WorkspaceRole } from "@nerve/api-client";
import type { TPowerKContext } from "@/components/power-k/core/types";
import { issueStoreTab } from "@/store/issue/fake-issues";
import { loadWorkspaces, workspaceOf } from "@/store/workspace/fake-workspaces";
import { loadProjects, projectOf } from "@/store/project/fake-projects";
import { usePowerKCreationCommandsRecord } from "./command";
vi.stubGlobal("localStorage", { getItem: () => null, setItem: () => {}, removeItem: () => {} });
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "creation" } }, publicClient: {} }));
vi.mock("@/lib/store-context", () => ({ rootStore: {} }));
vi.mock("@/hooks/store/use-command-palette", () => ({ useCommandPalette: () => tab.app.commandPalette }));
vi.mock("@/hooks/store/use-project", () => ({ useProject: () => tab.app.projectRoot.project }));
vi.mock("@/hooks/store/use-workspace", () => ({ useWorkspace: () => tab.app.workspaceRoot }));
vi.mock("@/hooks/store/user", () => ({ useUser: () => tab.app.user }));
vi.mock("@/hooks/store/use-instance", () => ({
  useInstance: () => ({ config: { workspace_creation_enabled: true } }),
}));
let tab: Awaited<ReturnType<typeof issueStoreTab>>, commands: ReturnType<typeof usePowerKCreationCommandsRecord>;
function Consumer() {
  commands = usePowerKCreationCommandsRecord();
  return null;
}
const ctx = (slug: string): TPowerKContext => ({
  params: { workspaceSlug: slug },
  activeCommand: null,
  activeContext: null,
  shouldShowContextBasedActions: true,
  setShouldShowContextBasedActions: () => {},
  navigate: () => {},
  closePalette: () => {},
  setActiveCommand: () => {},
  setActivePage: () => {},
});
beforeEach(async () => {
  vi.useFakeTimers();
  tab = await issueStoreTab();
});
afterEach(() => {
  tab.app.dispose();
  vi.useRealTimers();
});
it.each([
  [5, 5, false, false],
  [15, 15, false, true],
  [20, 5, false, true],
  [15, 15, true, false],
  [20, null, false, false],
] as const)(
  "creationUsesRouteWorkspaceBeforeRouterEffect: workspace %s/project %s/archived %s",
  async (wr: WorkspaceRole, pr: ProjectRole | null, archived, allowed) => {
    const other = workspaceOf("other", { role: wr });
    const project = projectOf("OTHER", other.id, {
      member_role: pr,
      archived_at: archived ? "2026-10-31T00:00:00Z" : null,
    });
    await loadWorkspaces(tab.nerve, tab.app.workspaceRoot, [tab.workspace, other]);
    await loadProjects(tab.nerve, tab.app.projectRoot.project, other, [project]);
    renderToStaticMarkup(<Consumer />);
    expect(tab.app.router.workspaceSlug).toBe("ws");
    const context = ctx("other"),
      item = commands.create_work_item;
    expect(item.isVisible(context)).toBe(allowed);
    expect(item.isEnabled(context)).toBe(allowed);
    if (item.type !== "action") throw new Error("creation must be an action");
    void item.action(context);
    expect(tab.app.commandPalette.isCreateIssueModalOpen).toBe(allowed);
    if (allowed) expect(tab.app.commandPalette.createWorkItemAllowedProjectIds).toEqual([project.id]);
    expect(commands.create_project.isEnabled(context)).toBe(wr === 20 || wr === 15);
  }
);
// nerve's cycle.write (M6 design 3.7): the project's admins and members, and the workspace's admin, but not one nerve
// shows only his own work items (a guest of a project not showing its guests every work item), whatever his workspace
// role
it.each([
  [15, 15, false, true],
  [5, 5, false, false],
  [20, 5, false, false],
  [20, 5, true, true],
] as const)(
  "offers a new cycle as nerve lets one write the project's cycles: workspace %s/project %s/guests see all %s",
  async (wr: WorkspaceRole, pr: ProjectRole, guestViewAll, allowed) => {
    const other = workspaceOf("other", { role: wr });
    const project = projectOf("OTHER", other.id, { member_role: pr, guest_view_all_features: guestViewAll });
    await loadWorkspaces(tab.nerve, tab.app.workspaceRoot, [tab.workspace, other]);
    await loadProjects(tab.nerve, tab.app.projectRoot.project, other, [project]);
    renderToStaticMarkup(<Consumer />);
    const context: TPowerKContext = { ...ctx("other"), params: { workspaceSlug: "other", projectId: project.id } };
    expect(commands.create_cycle.isVisible(context)).toBe(allowed);
    expect(commands.create_cycle.isEnabled(context)).toBe(allowed);
  }
);
// nerve refuses every write of a project's cycles while they are turned off (planning.cycles_disabled), its admin's too
it.each([
  [20, 20],
  [15, 15],
] as const)(
  "offers no new cycle while the project's cycles are turned off: workspace %s/project %s",
  async (wr: WorkspaceRole, pr: ProjectRole) => {
    const other = workspaceOf("other", { role: wr });
    const project = projectOf("OTHER", other.id, { member_role: pr, cycle_view: false });
    await loadWorkspaces(tab.nerve, tab.app.workspaceRoot, [tab.workspace, other]);
    await loadProjects(tab.nerve, tab.app.projectRoot.project, other, [project]);
    renderToStaticMarkup(<Consumer />);
    const context: TPowerKContext = { ...ctx("other"), params: { workspaceSlug: "other", projectId: project.id } };
    expect(commands.create_cycle.isVisible(context)).toBe(false);
    expect(commands.create_cycle.isEnabled(context)).toBe(false);
  }
);
// nerve's module.write (P6a spec 2.3, R5), as its cycle.write: the project's admins and members, and the workspace's
// admin, but not one nerve shows only his own work items there, whatever his workspace role
it.each([
  [15, 15, false, true],
  [5, 5, false, false],
  [20, 5, false, false],
  [20, 5, true, true],
] as const)(
  "offers a new module as nerve lets one write the project's modules: workspace %s/project %s/guests see all %s",
  async (wr: WorkspaceRole, pr: ProjectRole, guestViewAll, allowed) => {
    const other = workspaceOf("other", { role: wr });
    const project = projectOf("OTHER", other.id, { member_role: pr, guest_view_all_features: guestViewAll });
    await loadWorkspaces(tab.nerve, tab.app.workspaceRoot, [tab.workspace, other]);
    await loadProjects(tab.nerve, tab.app.projectRoot.project, other, [project]);
    renderToStaticMarkup(<Consumer />);
    const context: TPowerKContext = { ...ctx("other"), params: { workspaceSlug: "other", projectId: project.id } };
    expect(commands.create_module.isVisible(context)).toBe(allowed);
    expect(commands.create_module.isEnabled(context)).toBe(allowed);
  }
);
// nerve refuses every write of a project's modules while they are turned off (planning.modules_disabled), its admin's too
it.each([
  [20, 20],
  [15, 15],
] as const)(
  "offers no new module while the project's modules are turned off: workspace %s/project %s",
  async (wr: WorkspaceRole, pr: ProjectRole) => {
    const other = workspaceOf("other", { role: wr });
    const project = projectOf("OTHER", other.id, { member_role: pr, module_view: false });
    await loadWorkspaces(tab.nerve, tab.app.workspaceRoot, [tab.workspace, other]);
    await loadProjects(tab.nerve, tab.app.projectRoot.project, other, [project]);
    renderToStaticMarkup(<Consumer />);
    const context: TPowerKContext = { ...ctx("other"), params: { workspaceSlug: "other", projectId: project.id } };
    expect(commands.create_module.isVisible(context)).toBe(false);
    expect(commands.create_module.isEnabled(context)).toBe(false);
  }
);
it("an unknown route never opens a form using the router store's previous projects", () => {
  renderToStaticMarkup(<Consumer />);
  const item = commands.create_work_item;
  expect(item.isVisible(ctx("missing"))).toBe(false);
  if (item.type === "action") void item.action(ctx("missing"));
  expect(tab.app.commandPalette.isCreateIssueModalOpen).toBe(false);
});
