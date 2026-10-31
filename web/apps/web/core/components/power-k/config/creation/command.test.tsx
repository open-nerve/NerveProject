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
it("an unknown route never opens a form using the router store's previous projects", () => {
  renderToStaticMarkup(<Consumer />);
  const item = commands.create_work_item;
  expect(item.isVisible(ctx("missing"))).toBe(false);
  if (item.type === "action") void item.action(ctx("missing"));
  expect(tab.app.commandPalette.isCreateIssueModalOpen).toBe(false);
});
