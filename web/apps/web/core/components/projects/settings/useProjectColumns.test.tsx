/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { isValidElement, type ComponentProps, type ReactNode } from "react";
import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it, vi } from "vitest";
import type { ProjectRole, WorkspaceRole } from "@nerve/api-client";
import type { AccountTypeColumn, NameColumn } from "@/components/project/settings/member-columns";
import type { IProjectMemberDetails } from "@/store/member/project/project-member.store";
import { membershipOf } from "@/store/member/workspace/fake-members";
import { projectMemberOf, projectOf } from "@/store/project/fake-projects";
import { useProjectColumns } from "./useProjectColumns";

// What the members page's columns give each row (M3 design 3.5): the caller's role in the workspace and in the
// project, and the row's member's role in the workspace, reach the rules (project-roles.ts) that decide the roles the
// row offers and whether it offers the removal or, on the caller's own row, the leaving. The hook runs in a component
// rendered on the server, with stand-ins for the stores: the caller, me, has the case's roles; acme's members are gus,
// its guest, and max and me, its members. The test reads the props each column gives its cell, unrendered.

const web = projectOf("WEB", "w-acme");
const members = [membershipOf("gus", { role: 5 }), membershipOf("max"), membershipOf("me")];
/** The caller's roles, in acme and in web, as the case gives them. */
const caller = vi.hoisted((): { workspaceRole: WorkspaceRole; projectRole: ProjectRole } => ({
  workspaceRole: 15,
  projectRole: 20,
}));
vi.mock("@/hooks/store/user", () => ({
  useUser: () => ({ data: { id: "u-me" } }),
  useUserPermissions: () => ({
    getWorkspaceRoleByWorkspaceSlug: (slug: string) => (slug === "acme" ? caller.workspaceRole : undefined),
  }),
}));
vi.mock("@/hooks/store/use-project", () => ({
  useProject: () => ({
    getProjectById: (projectId: string) =>
      projectId === web.id ? { ...web, member_role: caller.projectRole } : undefined,
  }),
}));
vi.mock("@/hooks/store/use-member", () => ({
  useMember: () => ({
    project: { filters: { getFilters: () => undefined, updateFilters: vi.fn() } },
    workspace: {
      getWorkspaceMemberDetails: (userId: string) => members.find((member) => member.member.id === userId) ?? null,
    },
  }),
}));
vi.mock("@/components/project/settings/member-columns", () => ({
  NameColumn: () => null,
  AccountTypeColumn: () => null,
}));
vi.mock("@/components/project/member-header-column", () => ({ MemberHeaderColumn: () => null }));
// the workspace members' fakes build the account's store, which imports the tab's session
vi.mock("@/lib/auth/api-client", () => import("@/lib/auth/fake-tab"));

/** The columns of web's members page, as the hook gives them in a render. */
const rendered: { columns: ReturnType<typeof useProjectColumns>["columns"] } = { columns: [] };
function Columns() {
  rendered.columns = useProjectColumns({ projectId: web.id, workspaceSlug: "acme" }).columns;
  return null;
}

/** The props of a column's cell, a component's element. */
function propsOf<P>(cell: ReactNode): P {
  if (!isValidElement<P>(cell)) throw new Error("the column gave no cell");
  return cell.props;
}

/** What the row of rowData offers the caller: whether it is his own, whether he may remove it, the roles he may give. */
function offered(rowData: IProjectMemberDetails) {
  renderToStaticMarkup(<Columns />);
  const cell = (key: string) => rendered.columns.find((column) => column.key === key)?.tdRender(rowData);
  const name = propsOf<ComponentProps<typeof NameColumn>>(cell("Full Name"));
  const role = propsOf<ComponentProps<typeof AccountTypeColumn>>(cell("Account Type"));
  return { own: name.own, removable: name.removable, choices: role.choices };
}

/** name's row of web's members page, of role. */
const rowOf = (name: string, role: ProjectRole): IProjectMemberDetails => ({
  ...projectMemberOf(web, name, role),
  member: membershipOf(name).member,
});

describe("useProjectColumns", () => {
  it.each<{
    who: string;
    workspaceRole: WorkspaceRole;
    projectRole: ProjectRole;
    row: IProjectMemberDetails;
    offers: ReturnType<typeof offered>;
  }>([
    {
      who: "a project admin, for gus, a guest of the workspace: a guest's role alone, and the removal",
      workspaceRole: 15,
      projectRole: 20,
      row: rowOf("gus", 5),
      offers: { own: false, removable: true, choices: [5] },
    },
    {
      who: "the workspace's admin who is the project's member, for max, its admin: every role, no removal",
      workspaceRole: 20,
      projectRole: 15,
      row: rowOf("max", 20),
      offers: { own: false, removable: false, choices: [5, 15, 20] },
    },
    {
      who: "a project admin, for his own row: his leaving, no role",
      workspaceRole: 15,
      projectRole: 20,
      row: rowOf("me", 20),
      offers: { own: true, removable: false, choices: [] },
    },
  ])("$who", ({ workspaceRole, projectRole, row, offers }) => {
    Object.assign(caller, { workspaceRole, projectRole });
    expect(offered(row)).toEqual(offers);
  });
});
