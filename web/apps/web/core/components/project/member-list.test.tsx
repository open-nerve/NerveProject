/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { renderToStaticMarkup } from "react-dom/server";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { ProjectRole, WorkspaceRole } from "@nerve/api-client";
import { FakeNerve } from "@/lib/auth/fake-nerve";
import { emptyShown, shown } from "@/lib/fake-controls";
import { fakeRoot } from "@/store/fake-root";
import { ProjectRootStore } from "@/store/project";
import { loadProjects, projectOf } from "@/store/project/fake-projects";
import { RouterStore } from "@/store/router.store";
import { UserPermissionStore, type IUserPermissionStore } from "@/store/user/permissions.store";
import { WorkspaceRootStore } from "@/store/workspace";
import { loadWorkspaces, workspaceOf } from "@/store/workspace/fake-workspaces";
import { ProjectMemberList } from "./member-list";

// Whom the project's members page offers "Add member" (M3 design 3.5, 9.2): those nerve lets add members to the
// project (project_member.add: its admins, and its members who are the workspace's admins). The page renders on the
// server with stand-ins for its button (fake-controls.ts, which keeps the props it was given) and for the list's other
// parts; the caller's permissions are the store's own, over acme and its project web as nerve lists them to him, the
// address's.

const caller = vi.hoisted(() => {
  const held: { permissions?: IUserPermissionStore } = {};
  return held;
});
vi.mock("@/hooks/store/user", () => ({ useUserPermissions: () => caller.permissions }));
vi.mock("@/hooks/store/use-member", () => ({
  useMember: () => ({
    project: {
      projectMemberIds: null,
      getFilteredProjectMemberDetails: () => null,
      filters: { getFilters: () => ({}) },
    },
  }),
}));
vi.mock("@/components/ui/loader/settings/members", () => ({ MembersSettingsLoader: () => null }));
vi.mock("./dropdowns/filters/member-list", () => ({ MemberListFiltersDropdown: () => null }));
vi.mock("./member-list-item", () => ({ ProjectMemberListItem: () => null }));
vi.mock("./add-project-members-modal", () => ({ AddProjectMembersModal: () => null }));
vi.mock("@nerve/propel/button", () => import("@/lib/fake-controls"));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));

/**
 * Signs the caller in as nerve lists acme and web to him, his role in each (null: he sees web, no member of it), opens
 * web's members page and gives what its buttons say.
 */
async function buttonsFor(workspaceRole: WorkspaceRole, projectRole: ProjectRole | null) {
  const nerve = new FakeNerve();
  const api = nerve.client();
  const router = new RouterStore();
  const workspaceRoot = new WorkspaceRootStore(fakeRoot({ router }), api);
  const projectRoot = new ProjectRootStore(fakeRoot({ router, workspaceRoot }), api);
  const acme = workspaceOf("acme", { role: workspaceRole });
  await loadWorkspaces(nerve, workspaceRoot, [acme]);
  const web = projectOf("WEB", acme.id, { member_role: projectRole });
  await loadProjects(nerve, projectRoot.project, acme, [web]);
  router.setQuery({ workspaceSlug: acme.slug, projectId: web.id });
  caller.permissions = new UserPermissionStore(fakeRoot({ router, workspaceRoot, projectRoot }));
  emptyShown();
  renderToStaticMarkup(<ProjectMemberList projectId={web.id} workspaceSlug={acme.slug} />);
  return shown.buttons.map((button) => button.children);
}

beforeEach(() => {
  vi.useFakeTimers({ now: 1_000_000 });
});
afterEach(() => {
  vi.useRealTimers();
});

describe("the project's members page", () => {
  it.each<{ who: string; workspaceRole: WorkspaceRole; projectRole: ProjectRole | null; offered: string[] }>([
    { who: "its admin", workspaceRole: 15, projectRole: 20, offered: ["add_member"] },
    { who: "its member", workspaceRole: 15, projectRole: 15, offered: [] },
    { who: "its guest", workspaceRole: 5, projectRole: 5, offered: [] },
    { who: "its member who is the workspace's admin", workspaceRole: 20, projectRole: 15, offered: ["add_member"] },
    { who: "the workspace's admin who is not its member", workspaceRole: 20, projectRole: null, offered: [] },
  ])('shows $who "Add member" when nerve lets him add members', async ({ workspaceRole, projectRole, offered }) => {
    expect(await buttonsFor(workspaceRole, projectRole)).toEqual(offered);
  });
});
