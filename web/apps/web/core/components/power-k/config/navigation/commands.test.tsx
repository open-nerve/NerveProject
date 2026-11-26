/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { renderToStaticMarkup } from "react-dom/server";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { Project } from "@nerve/api-client";
import type { TPowerKCommandConfig, TPowerKContext } from "@/components/power-k/core/types";
import { loadProject, projectOf } from "@/store/project/fake-projects";
import type { IProjectStore } from "@/store/project/project.store";
import { its, permissionsOver, type Caller } from "@/store/user/fake-permissions";
import type { IUserPermissionStore } from "@/store/user/permissions.store";
import { workspaceOf } from "@/store/workspace/fake-workspaces";
import { usePowerKNavigationCommands } from "./root";

// The command palette's navigation commands (P7c spec 2.1, R3), written down before they were split by level so that
// the split is seen to move them unchanged: their order, each one's shape and where it goes, and on which pages it is
// offered. The caller's role is the permission store's own (fake-permissions.ts), the project the projects' store's,
// both over what nerve lists and reads in fake time.

const held = vi.hoisted(() => ({
  permissions: undefined as IUserPermissionStore | undefined,
  projects: undefined as IProjectStore | undefined,
}));
vi.mock("@/hooks/store/user", () => ({ useUser: () => ({ data: { id: "me" }, permission: held.permissions }) }));
vi.mock("@/hooks/store/use-project", () => ({ useProject: () => held.projects }));

let commands: TPowerKCommandConfig[];
function Consumer() {
  commands = usePowerKNavigationCommands();
  return null;
}
const navigate = vi.fn();
const ctx = (params: TPowerKContext["params"]): TPowerKContext => ({
  params,
  activeCommand: null,
  activeContext: null,
  shouldShowContextBasedActions: true,
  setShouldShowContextBasedActions: () => {},
  navigate,
  closePalette: () => {},
  setActiveCommand: () => {},
  setActivePage: () => {},
});
beforeEach(() => {
  vi.useFakeTimers();
  navigate.mockReset();
});
afterEach(() => vi.useRealTimers());

/**
 * The pages the palette is opened on: acme's (its member, no project in the address), and web's, the caller and web
 * being as each says. Web's four switches are on but where off says.
 */
const switchesOn: Partial<Project> = { cycle_view: true, module_view: true, issue_views_view: true, intake_view: true };
const pages = {
  workspace: { caller: its.member, project: switchesOn, inProject: false },
  admin: { caller: its.admin, project: switchesOn, inProject: true },
  guest: { caller: its.guest, project: switchesOn, inProject: true },
  off: {
    caller: its.admin,
    project: { cycle_view: false, module_view: false, issue_views_view: false, intake_view: false },
    inProject: true,
  },
  archived: { caller: its.admin, project: { ...switchesOn, archived_at: "2026-10-31T00:00:00Z" }, inProject: true },
} satisfies Record<string, { caller: Caller; project: Partial<Project>; inProject: boolean }>;
type Page = keyof typeof pages;

/**
 * The caller on the page: the palette's context there, the commands read. An archived web is not among the projects
 * nerve lists unarchived (the store keeps it out of that list), and is held by its page's own read of it.
 */
async function on(page: Page) {
  const { caller, project, inProject } = pages[page];
  const acme = workspaceOf("acme", { role: caller.workspaceRole });
  const web = projectOf("WEB", acme.id, { member_role: caller.projectRole, ...project });
  const { nerve, router, projects, permissions } = await permissionsOver([acme], [web]);
  if (web.archived_at) await loadProject(nerve, projects, web);
  expect(projects.getProjectById(web.id)).toEqual(web);
  const params = inProject ? { workspaceSlug: acme.slug, projectId: web.id } : { workspaceSlug: acme.slug };
  router.setQuery(params);
  Object.assign(held, { permissions, projects });
  renderToStaticMarkup(<Consumer />);
  return ctx(params);
}

/** What a page-changing command is handed when one picks on its page. */
const picked: Record<string, unknown> = {
  "open-workspace": { slug: "beta" },
  "open-project": { id: "p-api" },
  "open-workspace-setting": "members",
  "open-project-cycle": { id: "c-1" },
  "open-project-module": { id: "m-1" },
  "open-project-view": { id: "v-1" },
  "open-project-setting": "labels",
};

/** Where the command goes from web's page, at once: its action, or its pick of what picked says. */
function routeOf(command: TPowerKCommandConfig, context: TPowerKContext) {
  navigate.mockReset();
  if (command.type === "action") void command.action(context);
  else void command.onSelect(picked[command.page], context);
  return navigate.mock.calls.map(([to]) => to);
}

// Each command in the palette's order: id, type, key sequence, page, title (power_k.navigation_actions.<id>), where it
// goes from web's page as its admin, and whether it is offered (shown and enabled) on each page — workspace, admin,
// guest, off, archived. The four ways into web's cycles and modules follow the planning entry rule (P7c spec 2.2, R2):
// none on the archived web's page either.
const T = true,
  F = false;
type Row = [string, "action" | "change-page", string | undefined, string | undefined, string, boolean[]];
const table: Row[] = [
  ["open_project_cycle", "change-page", "oc", "open-project-cycle", "/acme/projects/p-web/cycles/c-1", [F, T, F, F, F]],
  [
    "open_project_module",
    "change-page",
    "om",
    "open-project-module",
    "/acme/projects/p-web/modules/m-1",
    [F, T, F, F, F],
  ],
  ["open_project_view", "change-page", "ov", "open-project-view", "/acme/projects/p-web/views/v-1", [F, T, T, F, T]],
  [
    "open_project_setting",
    "change-page",
    "os",
    "open-project-setting",
    "/acme/settings/projects/p-web/labels",
    [F, T, T, T, T],
  ],
  ["open_project", "change-page", "op", "open-project", "/acme/projects/p-api/issues", [T, T, T, T, T]],
  ["open_workspace_setting", "change-page", "os", "open-workspace-setting", "/acme/members", [T, F, F, F, F]],
  ["open_workspace", "change-page", "ow", "open-workspace", "/beta", [T, T, T, T, T]],
  ["nav_home", "action", "gh", undefined, "/acme", [T, T, T, T, T]],
  ["nav_inbox", "action", "gx", undefined, "/acme/notifications", [T, T, T, T, T]],
  ["nav_your_work", "action", "gy", undefined, "/acme/profile/me", [T, T, F, T, T]],
  ["nav_project_work_items", "action", "gi", undefined, "/acme/projects/p-web/issues", [F, T, T, T, T]],
  ["nav_project_cycles", "action", "gc", undefined, "/acme/projects/p-web/cycles", [F, T, F, F, F]],
  ["nav_project_modules", "action", "gm", undefined, "/acme/projects/p-web/modules", [F, T, F, F, F]],
  ["nav_project_views", "action", "gv", undefined, "/acme/projects/p-web/views", [F, T, T, F, T]],
  ["nav_project_intake", "action", "gk", undefined, "/acme/projects/p-web/intake", [F, T, T, F, T]],
  ["nav_project_settings", "action", "gs", undefined, "/acme/settings/projects/p-web", [F, T, T, T, T]],
  ["nav_project_archives", "action", "gr", undefined, "/acme/projects/p-web/archives/issues", [F, T, F, T, T]],
  ["nav_all_workspace_work_items", "action", undefined, undefined, "/acme/workspace-views/all-issues", [T, T, T, T, T]],
  [
    "nav_assigned_workspace_work_items",
    "action",
    undefined,
    undefined,
    "/acme/workspace-views/assigned",
    [T, T, T, T, T],
  ],
  [
    "nav_created_workspace_work_items",
    "action",
    undefined,
    undefined,
    "/acme/workspace-views/created",
    [T, T, T, T, T],
  ],
  [
    "nav_subscribed_workspace_work_items",
    "action",
    undefined,
    undefined,
    "/acme/workspace-views/subscribed",
    [T, T, T, T, T],
  ],
  ["nav_workspace_settings", "action", "gs", undefined, "/acme/settings", [T, F, F, F, F]],
  ["nav_workspace_drafts", "action", "gj", undefined, "/acme/drafts", [T, T, F, T, T]],
  ["nav_workspace_archives", "action", "gr", undefined, "/acme/projects/archives", [T, F, F, F, F]],
  ["nav_projects_list", "action", "gp", undefined, "/acme/projects", [T, T, T, T, T]],
  ["nav_account_settings", "action", undefined, undefined, "/settings/profile/general", [T, T, T, T, T]],
];
const order: Page[] = ["workspace", "admin", "guest", "off", "archived"];

it("lists the navigation commands in the palette's order, each with its shape and where it goes", async () => {
  const context = await on("admin");
  const described = commands.map((command) => ({
    id: command.id,
    type: command.type,
    group: command.group,
    keySequence: command.keySequence,
    page: command.type === "change-page" ? command.page : undefined,
    title: command.i18n_title,
    closeOnSelect: command.closeOnSelect,
    shortcut: command.shortcut ?? command.modifierShortcut,
    route: routeOf(command, context),
  }));
  expect(described).toEqual(
    table.map(([id, type, keySequence, page, route]) => ({
      id,
      type,
      group: "navigation",
      keySequence,
      page,
      title: `power_k.navigation_actions.${id}`,
      closeOnSelect: true,
      shortcut: undefined,
      route: [route],
    }))
  );
});

it.each(order)("offers on the %s page the commands the table says, shown as enabled", async (page) => {
  const context = await on(page);
  const index = order.indexOf(page);
  expect(commands.map((command) => [command.id, command.isVisible(context), command.isEnabled(context)])).toEqual(
    table.map(([id, , , , , offered]) => [id, offered[index], offered[index]])
  );
});

// The two ways into web's views by the views entry rule (P6a spec 2.2, R14; M6-M7-07), as the sidebar and the tab bar
// offer them: to its member and its guest, and to none who has no role in it (the workspace's admin who is not its
// member). The table has neither a member's page nor one with no role.
/** The caller on web's page, web's four switches on: the palette's context there, the commands read. */
async function inWeb(caller: Caller) {
  const acme = workspaceOf("acme", { role: caller.workspaceRole });
  const web = projectOf("WEB", acme.id, { member_role: caller.projectRole, ...switchesOn });
  const { router, projects, permissions } = await permissionsOver([acme], [web]);
  const params = { workspaceSlug: acme.slug, projectId: web.id };
  router.setQuery(params);
  Object.assign(held, { permissions, projects });
  renderToStaticMarkup(<Consumer />);
  return ctx(params);
}
const viewWays = ["open_project_view", "nav_project_views"];

it.each([
  { caller: its.member, offered: T },
  { caller: its.guest, offered: T },
  { caller: its.outsider, offered: F },
])("offers the two ways into web's views to $caller.who as the views entry rule says: $offered", async (cell) => {
  const context = await inWeb(cell.caller);
  const ways = commands.filter((command) => viewWays.includes(command.id));
  expect(ways.map((command) => [command.id, command.isVisible(context), command.isEnabled(context)])).toEqual(
    viewWays.map((id) => [id, cell.offered, cell.offered])
  );
});
