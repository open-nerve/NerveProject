/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { renderToStaticMarkup } from "react-dom/server";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { Project } from "@nerve/api-client";
import type { TPowerKCommandConfig, TPowerKContext } from "@/components/power-k/core/types";
import type { IProjectStore } from "@/store/project/project.store";
import { callerInWeb, its, type Caller } from "@/store/user/fake-permissions";
import type { IUserPermissionStore } from "@/store/user/permissions.store";
import { usePowerKCreationCommandsRecord } from "./command";

// The command palette's new view ("nv", P6a spec 2.2, 2.8, R4, R13): offered by viewsCreatableBy, so on a project's
// page to its admin and its member while its views are on, as nerve's view.create decides; not to its guest, whom the
// ways into its views are offered (viewEntryOn) but nerve refuses a new one, nor to one with no role in it, nor on a
// page of no project. The caller's role is the permission store's own (fake-permissions.ts), the project the projects'
// store's, both over what nerve lists in fake time.

const held = vi.hoisted(() => ({
  permissions: undefined as IUserPermissionStore | undefined,
  projects: undefined as IProjectStore | undefined,
}));
vi.mock("@/hooks/store/user", () => ({ useUser: () => ({ permission: held.permissions }) }));
vi.mock("@/hooks/store/use-project", () => ({ useProject: () => held.projects }));
vi.mock("@/hooks/store/use-workspace", () => ({ useWorkspace: () => ({ getWorkspaceBySlug: () => undefined }) }));
vi.mock("@/hooks/store/use-command-palette", () => ({ useCommandPalette: () => ({}) }));
vi.mock("@/hooks/store/use-instance", () => ({ useInstance: () => ({ config: {} }) }));

let createView: TPowerKCommandConfig;
function Consumer() {
  createView = usePowerKCreationCommandsRecord().create_view;
  return null;
}
beforeEach(() => vi.useFakeTimers());
afterEach(() => vi.useRealTimers());

/** The caller on web's page, or on acme's when inProject says not, web as fields say: the palette's context there. */
async function on(caller: Caller, fields: Partial<Project> = {}, inProject = true): Promise<TPowerKContext> {
  const { workspaceRole, projectRole, guestViewAll = false } = caller;
  const { permissions, projects, acme, web } = await callerInWeb(workspaceRole, projectRole, guestViewAll, {
    issue_views_view: true,
    ...fields,
  });
  Object.assign(held, { permissions, projects });
  renderToStaticMarkup(<Consumer />);
  return {
    params: inProject ? { workspaceSlug: acme.slug, projectId: web.id } : { workspaceSlug: acme.slug },
    activeCommand: null,
    activeContext: null,
    shouldShowContextBasedActions: true,
    setShouldShowContextBasedActions: () => {},
    navigate: () => {},
    closePalette: () => {},
    setActiveCommand: () => {},
    setActivePage: () => {},
  };
}
const offered = (context: TPowerKContext) => [createView.isVisible(context), createView.isEnabled(context)];

it.each([its.admin, its.member, its.memberAdmin])("offers a new view to $who, shown as enabled", async (caller) => {
  expect(offered(await on(caller))).toEqual([true, true]);
});

it.each([its.guest, its.guestSeesAll, its.outsider])("offers no new view to $who", async (caller) => {
  expect(offered(await on(caller))).toEqual([false, false]);
});

it("offers no new view while the project's views are off, to its admin", async () => {
  expect(offered(await on(its.admin, { issue_views_view: false }))).toEqual([false, false]);
});

it("offers no new view on a page of no project, to the admin of a project whose views are on", async () => {
  expect(offered(await on(its.admin, {}, false))).toEqual([false, false]);
});
