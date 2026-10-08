/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { ProjectRole, WorkspaceRole } from "@nerve/api-client";
import { EUserPermissions, EUserPermissionsLevel } from "@nerve/constants";
import { FakeNerve, noContent } from "@/lib/auth/fake-nerve";
import { inTurn } from "@/store/fake-queue";
import { fakeRoot } from "@/store/fake-root";
import { ProjectRootStore } from "@/store/project";
import { loadProjects, projectOf } from "@/store/project/fake-projects";
import { RouterStore } from "@/store/router.store";
import { UserPermissionStore } from "@/store/user/permissions.store";
import { WorkspaceRootStore } from "@/store/workspace";
import { loadWorkspaces, workspaceOf } from "@/store/workspace/fake-workspaces";

// The caller's role in a workspace is the one nerve lists with the workspace (Workspace.role, M3 design 7.2, 7.3),
// his role in a project the one nerve gives with the project (Project.member_role), and the page's permissions follow
// them as nerve's do (9.2's columns). On the web the caller who was never a member, the one removed and the one whose
// workspace was deleted are alike: nerve does not list the workspace.

const { ADMIN, MEMBER, GUEST } = EUserPermissions;
const WORKSPACE = EUserPermissionsLevel.WORKSPACE;
const PROJECT = EUserPermissionsLevel.PROJECT;

/**
 * The checks the pages make: an admin's (settings, invitations), a member's (projects), anyone's (the workspace); and
 * a guest's alone, a set that skips the higher roles: a check names the roles it allows, not a lowest one.
 */
type Check = "admin" | "member" | "anyone" | "guest alone";
const CHECKS: Record<Check, EUserPermissions[]> = {
  admin: [ADMIN],
  member: [ADMIN, MEMBER],
  anyone: [ADMIN, MEMBER, GUEST],
  "guest alone": [GUEST],
};
const CHECKED: Check[] = ["admin", "member", "anyone", "guest alone"];

const IDENTITIES: { who: string; role: WorkspaceRole | undefined; allowed: Check[] }[] = [
  { who: "an admin", role: 20, allowed: ["admin", "member", "anyone"] },
  { who: "a member", role: 15, allowed: ["member", "anyone"] },
  { who: "a guest", role: 5, allowed: ["anyone", "guest alone"] },
  { who: "no member", role: undefined, allowed: [] },
];

/**
 * 9.2's project columns but X (the workspace's, above): the caller's role in the workspace, and the project as nerve
 * gives it to him, with his member_role (null: he sees it, no member), or not at all (undefined: nerve lists it not).
 * Then two in which the workspace's admin has no role either: a project the store does not hold (not fetched, or not
 * his to see; WM-私, WG- and P-前 are the other roles), and a project of another of his workspaces (elsewhere: nerve
 * gives it in ws-0's list, with his role there).
 */
const IN_PROJECTS: {
  who: string;
  role: WorkspaceRole;
  memberRole?: ProjectRole | null;
  elsewhere?: boolean;
  allowed: Check[];
}[] = [
  { who: "PA, its admin", role: 15, memberRole: 20, allowed: ["admin", "member", "anyone"] },
  { who: "PM, its member", role: 15, memberRole: 15, allowed: ["member", "anyone"] },
  { who: "PG, its guest", role: 5, memberRole: 5, allowed: ["anyone", "guest alone"] },
  {
    who: "PM+WA, its member and the workspace's admin",
    role: 20,
    memberRole: 15,
    allowed: ["admin", "member", "anyone"],
  },
  { who: "WA-, the workspace's admin", role: 20, memberRole: null, allowed: [] },
  { who: "WM-公, a member of the workspace, of a public project", role: 15, memberRole: null, allowed: [] },
  { who: "WM-私, a member of the workspace, of a private project", role: 15, allowed: [] },
  { who: "WG-, a guest of the workspace", role: 5, allowed: [] },
  { who: "P-前, once a member of a private project", role: 15, allowed: [] },
  { who: "WA, of a project the store does not hold", role: 20, allowed: [] },
  {
    who: "WA, of his project in another workspace, its admin there",
    role: 20,
    memberRole: 20,
    elsewhere: true,
    allowed: [],
  },
];

/** The caller's workspaces as nerve lists them: one in which he has each role. */
async function setUp() {
  const nerve = new FakeNerve();
  const router = new RouterStore();
  const workspaceRoot = new WorkspaceRootStore(fakeRoot({ router }), nerve.client());
  const permissions = new UserPermissionStore(fakeRoot({ router, workspaceRoot }));
  const listed = IDENTITIES.flatMap(({ role }) => (role ? [workspaceOf(`ws-${role}`, { role })] : []));
  await loadWorkspaces(nerve, workspaceRoot, listed);
  return { router, permissions };
}

/** The workspace in which the caller has role: one the list does not have when he has none. */
const slugOf = (role: WorkspaceRole | undefined) => (role ? `ws-${role}` : "ws-elsewhere");

/**
 * For each identity of IN_PROJECTS, a workspace of the caller's (ws-0 …) and its project as nerve gives it (p-p0 …),
 * in ws-0's list for one of another workspace.
 */
async function inProjects() {
  const nerve = new FakeNerve();
  const api = nerve.client();
  const router = new RouterStore();
  const workspaceRoot = new WorkspaceRootStore(fakeRoot({ router }), api);
  const projectRoot = new ProjectRootStore(fakeRoot({ router, workspaceRoot }), api);
  const permissions = new UserPermissionStore(fakeRoot({ router, workspaceRoot, projectRoot }));
  const workspaces = IN_PROJECTS.map(({ role }, i) => workspaceOf(`ws-${i}`, { role }));
  await loadWorkspaces(nerve, workspaceRoot, workspaces);
  // each workspace's list, one after another: the projects of the rows that are of it (a row's own, unless elsewhere)
  const projects = IN_PROJECTS.flatMap(({ memberRole, elsewhere }, i) =>
    memberRole === undefined ? [] : [projectOf(`P${i}`, `id-ws-${elsewhere ? 0 : i}`, { member_role: memberRole })]
  );
  const listedIn = (workspaceId: string) => projects.filter((held) => held.workspace_id === workspaceId);
  await workspaces.reduce<Promise<unknown>>(
    (before, workspace) =>
      before.then(() => loadProjects(nerve, projectRoot.project, workspace, listedIn(workspace.id))),
    Promise.resolve()
  );
  return { nerve, router, workspaceRoot, permissions };
}

beforeEach(() => {
  vi.useFakeTimers({ now: 1_000_000 });
});
afterEach(() => {
  vi.useRealTimers();
});

describe("UserPermissionStore, in a workspace", () => {
  it.each(IDENTITIES)("gives $who the role nerve lists", async ({ role }) => {
    const { permissions } = await setUp();
    expect(permissions.getWorkspaceRoleByWorkspaceSlug(slugOf(role))).toBe(role);
  });

  it.each(IDENTITIES)("allows $who what nerve allows, in the workspace named or in the address's", async (identity) => {
    const { router, permissions } = await setUp();
    const slug = slugOf(identity.role);
    for (const check of CHECKED) {
      const roles = CHECKS[check];
      const allowed = identity.allowed.includes(check);
      expect(permissions.allowPermissions(roles, WORKSPACE, slug), `${check}, named`).toBe(allowed);
      router.setQuery({ workspaceSlug: slug });
      expect(permissions.allowPermissions(roles, WORKSPACE), `${check}, the address's`).toBe(allowed);
      router.setQuery({});
    }
  });

  it("allows nothing before the list is fetched", () => {
    const router = new RouterStore();
    const workspaceRoot = new WorkspaceRootStore(fakeRoot({ router }), new FakeNerve().client());
    const permissions = new UserPermissionStore(fakeRoot({ router, workspaceRoot }));
    expect(permissions.getWorkspaceRoleByWorkspaceSlug("ws-20")).toBeUndefined();
    expect(permissions.allowPermissions(CHECKS.anyone, WORKSPACE, "ws-20")).toBe(false);
  });
});

describe("UserPermissionStore, in a project", () => {
  it.each(IN_PROJECTS)(
    "allows $who what nerve allows in it, in the project named or in the address's",
    async (identity) => {
      const { router, permissions } = await inProjects();
      const i = IN_PROJECTS.indexOf(identity);
      const [slug, projectId, allowed] = [`ws-${i}`, `p-p${i}`, identity.allowed];
      for (const check of CHECKED) {
        const roles = CHECKS[check];
        const allows = allowed.includes(check);
        expect(permissions.allowPermissions(roles, PROJECT, slug, projectId), `${check}, named`).toBe(allows);
        router.setQuery({ workspaceSlug: slug, projectId });
        expect(permissions.allowPermissions(roles, PROJECT), `${check}, the address's`).toBe(allows);
        router.setQuery({});
      }
    }
  );

  it("gives no role in a project through another workspace than its own", async () => {
    const { permissions } = await inProjects();
    expect(permissions.getProjectRoleByWorkspaceSlugAndProjectId("ws-0", "p-p0")).toBe(ADMIN);
    expect(permissions.getProjectRoleByWorkspaceSlugAndProjectId("ws-3", "p-p0")).toBeUndefined();
    expect(permissions.getProjectRoleByWorkspaceSlugAndProjectId("ws-elsewhere", "p-p0")).toBeUndefined();
  });

  it("gives no role in the projects of a workspace the caller left, nor of one made again under its slug", async () => {
    const { nerve, workspaceRoot, permissions } = await inProjects();
    const k = nerve.calls.length;
    const left = workspaceRoot.leaveWorkspace(workspaceOf("ws-0"));
    await inTurn(nerve, k, ["POST", "/api/v0/workspaces/ws-0/leave"], noContent());
    await left;
    expect(permissions.getProjectRoleByWorkspaceSlugAndProjectId("ws-0", "p-p0")).toBeUndefined();

    // another tab of his deleted ws-1 and made it again: the projects of the old one are not his in the new one
    const again = workspaceOf("ws-1", { id: "id-ws-1-again", role: 20 });
    await loadWorkspaces(nerve, workspaceRoot, [again]);
    expect(permissions.getProjectRoleByWorkspaceSlugAndProjectId("ws-1", "p-p1")).toBeUndefined();
  });
});
