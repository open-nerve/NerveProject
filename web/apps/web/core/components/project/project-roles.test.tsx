/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { renderToStaticMarkup } from "react-dom/server";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { ProjectMembersAdd, ProjectRole, WorkspaceRole } from "@nerve/api-client";
import { emptyShown, pick, shown, submitModalForm } from "@/lib/fake-controls";
import { membershipOf } from "@/store/member/workspace/fake-members";
import { projectMemberDetailsOf, projectOf } from "@/store/project/fake-projects";
import { AddProjectMembersModal } from "./add-project-members-modal";
import {
  PROJECT_ROLES,
  addableRoles,
  canRemove,
  roleChoices,
  type MembershipCaller,
  type ShownMembership,
} from "./project-roles";
import { AccountTypeColumn } from "./settings/member-columns";

// What the project's role selects send (M3 design 3.19): nerve's ProjectMemberNew and ProjectMemberUpdate take a role
// as a ProjectRole, a number, and refuse a string. The pages render on the server with stand-ins for the UI kit's
// controls (fake-controls.ts), which keep the props they were given: the test picks a member and a role as the
// selects would, and submits the modal's form. The caller is an admin of web; ann is a member of its workspace. Then
// which roles and removals the members page offers whom (M3 design 3.5).

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
vi.mock("@/hooks/store/use-project", () => ({ useProject: () => ({ getProjectById: () => web }) }));
vi.mock("react-router", () => ({ useNavigate: () => vi.fn() }));
vi.mock("@nerve/ui", () => import("@/lib/fake-controls"));
vi.mock("@nerve/propel/button", () => import("@/lib/fake-controls"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
// the workspace members' fakes build the account's store, which imports the tab's session; a role's change is
// followed in it
vi.mock("@/lib/auth/api-client", () => import("@/lib/auth/fake-tab"));

/** Each role a select offers: its name; the key the selects show it by (ROLE_DETAILS's, translated); its number. */
const roles: { label: string; key: string; role: ProjectRole }[] = [
  { label: "Guest", key: "role_details.guest.title", role: 5 },
  { label: "Member", key: "role_details.member.title", role: 15 },
  { label: "Admin", key: "role_details.admin.title", role: 20 },
];

beforeEach(() => {
  emptyShown();
  store.bulkAddMembersToProject.mockClear();
  store.updateMemberRole.mockClear();
});

describe("the project's role selects", () => {
  it.each(roles)("add ann as $label with that role's number", async ({ key, role }) => {
    renderToStaticMarkup(<AddProjectMembersModal isOpen onClose={vi.fn()} projectId={web.id} />);
    const [members] = shown.searchSelects;
    const [roleSelect] = shown.selects;
    if (!members || !roleSelect) throw new Error("the modal showed no member or no role select");
    members.onChange(ann.member.id);
    pick(roleSelect, key);
    await submitModalForm();
    expect(store.bulkAddMembersToProject.mock.calls).toEqual([
      [web.id, { members: [{ member_id: ann.member.id, role }] }],
    ]);
  });

  it.each(roles)("make ann $label with that role's number", ({ key, role }) => {
    renderToStaticMarkup(
      <AccountTypeColumn
        rowData={projectMemberDetailsOf(web, "ann")}
        choices={PROJECT_ROLES}
        projectId={web.id}
        workspaceSlug="acme"
      />
    );
    const [roleSelect] = shown.selects;
    if (!roleSelect) throw new Error("the column showed no role select");
    pick(roleSelect, key);
    expect(store.updateMemberRole.mock.calls).toEqual([[web.id, ann.member.id, role]]);
  });

  it.each(roles)("show ann $label by that role's name, where no change of it is offered", ({ key, role }) => {
    const markup = renderToStaticMarkup(
      <AccountTypeColumn
        rowData={projectMemberDetailsOf(web, "ann", role)}
        choices={[]}
        projectId={web.id}
        workspaceSlug="acme"
      />
    );
    expect([markup.includes(`<span>${key}</span>`), shown.selects]).toEqual([true, []]);
  });
});

/** A membership the page shows: its role, its member's role in the workspace, the caller's own or not. */
const shownAs = (role: ProjectRole, workspaceRole: WorkspaceRole, own = false): ShownMembership => ({
  own,
  role,
  workspaceRole,
});
const projectAdmin: MembershipCaller = { workspaceRole: 15, projectRole: 20 };
const workspaceAdmin: MembershipCaller = { workspaceRole: 20, projectRole: 20 };

describe("the roles a member is added with", () => {
  it.each<{ who: string; workspaceRole: WorkspaceRole | undefined; offered: ProjectRole[] }>([
    { who: "the workspace's admin: an admin's", workspaceRole: 20, offered: [20] },
    { who: "its member: any", workspaceRole: 15, offered: [5, 15, 20] },
    { who: "its guest: a guest's", workspaceRole: 5, offered: [5] },
    { who: "no one picked yet: any", workspaceRole: undefined, offered: [5, 15, 20] },
  ])("$who", ({ workspaceRole, offered }) => {
    expect(addableRoles(workspaceRole)).toEqual(offered);
  });
});

describe("what the members page offers", () => {
  it.each<{
    who: string;
    caller: MembershipCaller;
    membership: ShownMembership;
    choices: ProjectRole[];
    removable: boolean;
  }>([
    {
      who: "a project admin, for a member: the roles below his own, and the removal",
      caller: projectAdmin,
      membership: shownAs(15, 15),
      choices: [5, 15],
      removable: true,
    },
    {
      who: "a project admin, for another admin: no role, the removal",
      caller: projectAdmin,
      membership: shownAs(20, 15),
      choices: [],
      removable: true,
    },
    {
      who: "a project admin, for himself: neither",
      caller: projectAdmin,
      membership: shownAs(20, 15, true),
      choices: [],
      removable: false,
    },
    {
      who: "a project admin, for a guest of the workspace: a guest's role, and the removal",
      caller: projectAdmin,
      membership: shownAs(5, 5),
      choices: [5],
      removable: true,
    },
    {
      who: "a project admin, for a guest of the project who is the workspace's member: the roles below his own, and the removal",
      caller: projectAdmin,
      membership: shownAs(5, 15),
      choices: [5, 15],
      removable: true,
    },
    {
      who: "the workspace's admin, for another admin: every role, and the removal",
      caller: workspaceAdmin,
      membership: shownAs(20, 15),
      choices: [5, 15, 20],
      removable: true,
    },
    {
      who: "the workspace's admin, for a guest of the workspace: a guest's role alone, and the removal",
      caller: workspaceAdmin,
      membership: shownAs(5, 5),
      choices: [5],
      removable: true,
    },
    {
      who: "the workspace's admin, for himself: every role, no removal",
      caller: workspaceAdmin,
      membership: shownAs(20, 20, true),
      choices: [5, 15, 20],
      removable: false,
    },
    {
      who: "the workspace's admin who is the project's member, for its admin: every role, no removal",
      caller: { workspaceRole: 20, projectRole: 15 },
      membership: shownAs(20, 15),
      choices: [5, 15, 20],
      removable: false,
    },
    {
      who: "a project member, for a guest: neither",
      caller: { workspaceRole: 15, projectRole: 15 },
      membership: shownAs(5, 15),
      choices: [],
      removable: false,
    },
    {
      who: "the workspace's admin who is not the project's member: neither",
      caller: { workspaceRole: 20, projectRole: null },
      membership: shownAs(15, 15),
      choices: [],
      removable: false,
    },
  ])("$who", ({ caller, membership, choices, removable }) => {
    expect([roleChoices(caller, membership), canRemove(caller, membership)]).toEqual([choices, removable]);
  });
});
