/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { renderToStaticMarkup } from "react-dom/server";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { ProjectMembersAdd, ProjectRole } from "@nerve/api-client";
import { emptyShown, pick, shown, submitModalForm } from "@/lib/fake-controls";
import { membershipOf } from "@/store/member/workspace/fake-members";
import { projectMemberOf, projectOf } from "@/store/project/fake-projects";
import { AddProjectMembersModal } from "./add-project-members-modal";
import { AccountTypeColumn } from "./settings/member-columns";

// What the project's role selects send (M3 design 3.19): nerve's ProjectMemberNew and ProjectMemberUpdate take a role
// as a ProjectRole, a number, and refuse a string. The pages render on the server with stand-ins for the UI kit's
// controls (fake-controls.ts), which keep the props they were given: the test picks a member and a role as the
// selects would, and submits the modal's form. The caller is an admin of web; ann is a member of its workspace.

const web = projectOf("WEB", "w-acme");
const ann = membershipOf("ann");
const store = vi.hoisted(() => ({
  bulkAddMembersToProject: vi.fn((_projectId: string, _data: ProjectMembersAdd) => Promise.resolve([])),
  updateMemberRole: vi.fn((_projectId: string, _userId: string, _role: ProjectRole) => Promise.resolve()),
}));
vi.mock("@/hooks/store/use-member", () => ({
  useMember: () => ({
    project: {
      getProjectMemberDetails: () => null,
      bulkAddMembersToProject: store.bulkAddMembersToProject,
      updateMemberRole: store.updateMemberRole,
    },
    workspace: {
      workspaceMemberIds: [ann.member.id],
      getWorkspaceMemberDetails: (userId: string) => (userId === ann.member.id ? ann : null),
    },
  }),
}));
vi.mock("@/hooks/store/user", () => ({
  useUser: () => ({ data: { id: "u-me" } }),
  useUserPermissions: () => ({ getProjectRoleByWorkspaceSlugAndProjectId: () => 20 }),
}));
vi.mock("@nerve/ui", () => import("@/lib/fake-controls"));
vi.mock("@nerve/propel/button", () => import("@/lib/fake-controls"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
// the workspace members' fakes build the account's store, which imports the tab's session
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: {}, publicClient: {} }));

/** Each role a select offers, by its label, and the number nerve takes. */
const roles: { label: string; role: ProjectRole }[] = [
  { label: "Guest", role: 5 },
  { label: "Member", role: 15 },
  { label: "Admin", role: 20 },
];

beforeEach(() => {
  emptyShown();
  store.bulkAddMembersToProject.mockClear();
  store.updateMemberRole.mockClear();
});

describe("the project's role selects", () => {
  it.each(roles)("add ann as $label with that role's number", async ({ label, role }) => {
    renderToStaticMarkup(<AddProjectMembersModal isOpen onClose={vi.fn()} projectId={web.id} workspaceSlug="acme" />);
    const [members] = shown.searchSelects;
    const [roleSelect] = shown.selects;
    if (!members || !roleSelect) throw new Error("the modal showed no member or no role select");
    members.onChange(ann.member.id);
    pick(roleSelect, label);
    await submitModalForm();
    expect(store.bulkAddMembersToProject.mock.calls).toEqual([
      [web.id, { members: [{ member_id: ann.member.id, role }] }],
    ]);
  });

  it.each(roles)("make ann $label with that role's number", ({ label, role }) => {
    renderToStaticMarkup(
      <AccountTypeColumn
        rowData={{ ...projectMemberOf(web, "ann"), member: ann.member }}
        currentProjectRole={20}
        projectId={web.id}
        workspaceSlug="acme"
      />
    );
    const [roleSelect] = shown.selects;
    if (!roleSelect) throw new Error("the column showed no role select");
    pick(roleSelect, label);
    expect(store.updateMemberRole.mock.calls).toEqual([[web.id, ann.member.id, role]]);
  });
});
