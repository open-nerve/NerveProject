/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

// The caller's permissions as the store gives them (UserPermissionStore, M3 design 7.3), for the tests of the store and
// of the pages that check them: over his workspaces and their projects as nerve lists them to him, each with his role
// in it.

import { vi } from "vitest";
import type { Project, ProjectRole, Workspace, WorkspaceRole } from "@nerve/api-client";
import { FakeNerve } from "@/lib/auth/fake-nerve";
import { fakeRoot, withoutImageObservation } from "@/store/fake-root";
import { ProjectRootStore } from "@/store/project";
import { loadProjects, projectOf } from "@/store/project/fake-projects";
import { RouterStore } from "@/store/router.store";
import { UserPermissionStore } from "@/store/user/permissions.store";
import { WorkspaceRootStore } from "@/store/workspace";
import { loadWorkspaces, workspaceOf } from "@/store/workspace/fake-workspaces";

/**
 * The permission store over workspaces, as nerve lists them, and their projects, each workspace's list one after
 * another; with nerve, the address and the workspaces' and the projects' stores, for a test that goes on with them.
 * Nerve answers the lists in fake time (fake-time.ts): the test runs under vitest's fake timers.
 */
export async function permissionsOver(workspaces: Workspace[], projects: Project[]) {
  const nerve = new FakeNerve();
  const api = nerve.client();
  const router = new RouterStore();
  const workspaceRoot = new WorkspaceRootStore(
    fakeRoot({ observeProfileRead: withoutImageObservation, router }),
    api,
    () => () => true
  );
  const projectRoot = new ProjectRootStore(
    fakeRoot({ observeProfileRead: withoutImageObservation, router, workspaceRoot }),
    api
  );
  const permissions = new UserPermissionStore(
    fakeRoot({ observeProfileRead: withoutImageObservation, router, workspaceRoot, projectRoot })
  );
  await loadWorkspaces(nerve, workspaceRoot, workspaces);
  const listedIn = (workspaceId: string) => projects.filter((held) => held.workspace_id === workspaceId);
  await workspaces.reduce<Promise<unknown>>(
    (before, workspace) =>
      before.then(() => loadProjects(nerve, projectRoot.project, workspace, listedIn(workspace.id))),
    Promise.resolve()
  );
  return { nerve, router, workspaceRoot, projects: projectRoot.project, permissions };
}

/**
 * A caller of acme, of role workspaceRole in it, and of role projectRole in its project web (null: he sees web, no
 * member of it), web showing its guests every work item when guestViewAll says so (guest_view_all_features): his
 * permissions, at web's address; the address (the router store) and the projects' store, for a test that moves the
 * address elsewhere; and the two. Nerve answers the lists in fake time, and a test that runs in real time is back in
 * it once they are loaded.
 */
export async function callerInWeb(workspaceRole: WorkspaceRole, projectRole: ProjectRole | null, guestViewAll = false) {
  const inRealTime = !vi.isFakeTimers();
  if (inRealTime) vi.useFakeTimers();
  try {
    const acme = workspaceOf("acme", { role: workspaceRole });
    const web = projectOf("WEB", acme.id, { member_role: projectRole, guest_view_all_features: guestViewAll });
    const { router, projects, permissions } = await permissionsOver([acme], [web]);
    router.setQuery({ workspaceSlug: acme.slug, projectId: web.id });
    return { permissions, router, projects, acme, web };
  } finally {
    if (inRealTime) vi.useRealTimers();
  }
}

/**
 * A caller of acme and web, as callerInWeb takes him: who he is, for a test's title, his role in each, and whether web
 * shows its guests every work item (none: it does not).
 */
export type Caller = {
  who: string;
  workspaceRole: WorkspaceRole;
  projectRole: ProjectRole | null;
  guestViewAll?: boolean;
};

/** The callers the pages' tests ask about, each by his roles in acme and in web. */
export const its = {
  admin: { who: "its admin", workspaceRole: 15, projectRole: 20 },
  member: { who: "its member", workspaceRole: 15, projectRole: 15 },
  guest: { who: "its guest", workspaceRole: 5, projectRole: 5 },
  memberAdmin: { who: "its member who is the workspace's admin", workspaceRole: 20, projectRole: 15 },
  guestAdmin: { who: "its guest who is the workspace's admin", workspaceRole: 20, projectRole: 5 },
  guestAdminSeesAll: {
    who: "its guest who is the workspace's admin, web showing its guests every work item",
    workspaceRole: 20,
    projectRole: 5,
    guestViewAll: true,
  },
  guestSeesAll: {
    who: "its guest who is the workspace's member, web showing its guests every work item",
    workspaceRole: 15,
    projectRole: 5,
    guestViewAll: true,
  },
  outsider: { who: "the workspace's admin who is not its member", workspaceRole: 20, projectRole: null },
} satisfies Record<string, Caller>;
