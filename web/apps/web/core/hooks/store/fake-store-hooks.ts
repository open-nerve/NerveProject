/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

// A stand-in for the store hooks that the hooks of a page's fetches read (use-workspace.ts, use-member.ts and user's
// useUserProfile), for their tests: a test file mocks each of those modules with this one, for instance
// vi.mock("@/hooks/store/use-workspace", () => import("@/hooks/store/fake-store-hooks")), says in `stores` what the
// stores hold, and reads there what they were asked to fetch. The hooks then run as plain functions, outside React,
// with fake-session-swr.ts. How the stores fetch is their own tests'.

import type { Profile, Workspace, WorkspaceMember } from "@nerve/api-client";

/** What the stores hold, and the fetches they were asked for, in order; a test resets it before each case. */
export const stores: {
  workspaces: Workspace[] | undefined;
  /** The memberships of the address's workspace, by account id. */
  members: Record<string, WorkspaceMember>;
  profile: Profile | undefined;
  fetched: string[];
} = { workspaces: undefined, members: {}, profile: undefined, fetched: [] };

/** The stores as a test starts: they hold nothing and were asked for nothing. */
export function emptyStores() {
  Object.assign(stores, { workspaces: undefined, members: {}, profile: undefined, fetched: [] });
}

/** A store's fetch, which says what it fetched; its Promise is what SWR gets. */
const fetching = (what: string) => Promise.resolve(stores.fetched.push(what));

/** A workspace as a fetch names it: its slug, and its id. */
const named = ({ id, slug }: Pick<Workspace, "id" | "slug">) => `${slug} (${id})`;

export function useWorkspace() {
  return {
    workspaces: stores.workspaces,
    getWorkspaceBySlug: (slug: string) => stores.workspaces?.find((workspace) => workspace.slug === slug) ?? null,
    fetchWorkspaces: () => fetching("the workspaces"),
    preferences: {
      fetchPreferences: (workspace: Pick<Workspace, "id" | "slug">) => fetching(`the settings in ${named(workspace)}`),
    },
  };
}

export function useMember() {
  return {
    workspace: {
      fetchWorkspaceMembers: (workspace: Pick<Workspace, "id" | "slug">) =>
        fetching(`the members of ${named(workspace)}`),
      fetchWorkspaceMemberInvitations: (workspace: Pick<Workspace, "id" | "slug">) =>
        fetching(`the invitations of ${named(workspace)}`),
      getWorkspaceMemberDetails: (userId: string) => stores.members[userId] ?? null,
    },
  };
}

export function useUserProfile() {
  return { data: stores.profile };
}
