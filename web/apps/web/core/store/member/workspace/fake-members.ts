/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

// The workspace members' store, for the tests of its members and of its invitations, against a fake nerve. A test
// file that uses it mocks @/lib/auth/api-client: the account's store, which it builds, imports the tab's session.

import type { MemberUser, WorkspaceInvitation, WorkspaceMember } from "@nerve/api-client";
import { FakeNerve } from "@/lib/auth/fake-nerve";
import { fakeRoot } from "@/store/fake-root";
import { userOf } from "@/store/member/fake-users";
import { WorkspaceMemberStore } from "@/store/member/workspace/workspace-member.store";
import { RouterStore } from "@/store/router.store";
import { UserStore } from "@/store/user";
import { WorkspaceRootStore } from "@/store/workspace";
import { loadWorkspaces, workspaceOf } from "@/store/workspace/fake-workspaces";

/**
 * A membership of the workspace slug names (acme unless it says) as nerve lists it: the name names the member, and
 * with the slug the membership; an active member's, unless fields say not.
 */
export function membershipOf(name: string, fields: Partial<WorkspaceMember> = {}, slug = "acme"): WorkspaceMember {
  return {
    id: `m-${slug}-${name}`,
    workspace_id: `id-${slug}`,
    role: 15,
    is_active: true,
    created_at: "2026-10-01T09:00:00Z",
    member: userOf(name),
    ...fields,
  };
}

/**
 * An invitation of the workspace slug names (acme unless it says) as nerve lists it to an admin: the name names the
 * address and the invitation; pending, unless fields say not.
 */
export function invitationOf(
  name: string,
  fields: Partial<WorkspaceInvitation> = {},
  slug = "acme"
): WorkspaceInvitation {
  return {
    id: `i-${name}`,
    workspace_id: `id-${slug}`,
    email: `${name}@example.com`,
    role: 15,
    accepted: false,
    responded_at: null,
    created_at: "2026-10-02T09:00:00Z",
    created_by_id: "u-ann",
    token: `nrv_inv_${name}`,
    ...fields,
  };
}

/**
 * The store, the users the stores share and the store's client, of a tab whose address names acme, and whose
 * caller's list nerve gave as acme and globex (workspaceOf), the requests of which are then forgotten.
 */
export async function memberStore() {
  const nerve = new FakeNerve();
  const api = nerve.client();
  const router = new RouterStore();
  router.setQuery({ workspaceSlug: "acme" });
  const user = new UserStore(fakeRoot({ router }), api);
  const workspaceRoot = new WorkspaceRootStore(fakeRoot({ router }), api);
  await loadWorkspaces(nerve, workspaceRoot, [workspaceOf("acme"), workspaceOf("globex")]);
  nerve.calls.length = 0;
  const users: Record<string, MemberUser> = {};
  const store = new WorkspaceMemberStore({ memberMap: users }, fakeRoot({ router, user, workspaceRoot }), api);
  return { nerve, api, user, users, store };
}
