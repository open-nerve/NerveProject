/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it, vi } from "vitest";
import type { ProjectRole, WorkspaceRole } from "@nerve/api-client";
import { emptyShown, shown } from "@/lib/fake-controls";
import { callerInWeb, its, type Caller } from "@/store/user/fake-permissions";
import type { IUserPermissionStore } from "@/store/user/permissions.store";
import { ProjectMemberList } from "./member-list";

// Whom the project's members page offers "Add member" (M3 design 3.5, 9.2): those nerve lets add members to the
// project (project_member.add: its admins, and its members who are the workspace's admins). The page renders on the
// server with stand-ins for its button (fake-controls.ts, which keeps the props it was given) and for the list's other
// parts; the caller's permissions are the store's own, over acme and its project web as nerve lists them to him, at
// web's address (fake-permissions.ts).

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
  const { permissions, acme, web } = await callerInWeb(workspaceRole, projectRole);
  caller.permissions = permissions;
  emptyShown();
  renderToStaticMarkup(<ProjectMemberList projectId={web.id} workspaceSlug={acme.slug} />);
  return shown.buttons.map((button) => button.children);
}

describe("the project's members page", () => {
  it.each<Caller & { offered: string[] }>([
    { ...its.admin, offered: ["add_member"] },
    { ...its.member, offered: [] },
    { ...its.guest, offered: [] },
    { ...its.memberAdmin, offered: ["add_member"] },
    { ...its.outsider, offered: [] },
  ])('shows $who "Add member" when nerve lets him add members', async ({ workspaceRole, projectRole, offered }) => {
    expect(await buttonsFor(workspaceRole, projectRole)).toEqual(offered);
  });
});
