/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { WorkspaceRole } from "@nerve/api-client";
import { EUserPermissions, EUserPermissionsLevel } from "@nerve/constants";
import { FakeNerve } from "@/lib/auth/fake-nerve";
import { fakeRoot } from "@/store/fake-root";
import { RouterStore } from "@/store/router.store";
import { UserPermissionStore } from "@/store/user/permissions.store";
import { WorkspaceRootStore } from "@/store/workspace";
import { loadWorkspaces, workspaceOf } from "@/store/workspace/fake-workspaces";

// The caller's role in a workspace is the one nerve lists with the workspace (Workspace.role, M3 design 7.2, 7.3),
// and the page's permissions follow it as nerve's do (9.2's workspace columns). On the web the caller who was never
// a member, the one removed and the one whose workspace was deleted are alike: nerve does not list the workspace.

const { ADMIN, MEMBER, GUEST } = EUserPermissions;
const WORKSPACE = EUserPermissionsLevel.WORKSPACE;

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

/** The caller's workspaces as nerve lists them: one in which he has each role. */
async function setUp() {
  const nerve = new FakeNerve();
  const router = new RouterStore();
  const workspaceRoot = new WorkspaceRootStore(fakeRoot({ router }), nerve.client());
  const permissions = new UserPermissionStore(fakeRoot({ router, workspaceRoot }), nerve.client());
  const listed = IDENTITIES.flatMap(({ role }) => (role ? [workspaceOf(`ws-${role}`, { role })] : []));
  await loadWorkspaces(nerve, workspaceRoot, listed);
  return { router, permissions };
}

/** The workspace in which the caller has role: one the list does not have when he has none. */
const slugOf = (role: WorkspaceRole | undefined) => (role ? `ws-${role}` : "ws-elsewhere");

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
    const permissions = new UserPermissionStore(fakeRoot({ router, workspaceRoot }), new FakeNerve().client());
    expect(permissions.getWorkspaceRoleByWorkspaceSlug("ws-20")).toBeUndefined();
    expect(permissions.allowPermissions(CHECKS.anyone, WORKSPACE, "ws-20")).toBe(false);
  });
});
