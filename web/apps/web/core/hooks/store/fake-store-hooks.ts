/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

// A stand-in for the store hooks that the hooks of a page's fetches read (the modules of core/hooks/store that have
// the hooks below), for their tests: a test file mocks each of those modules with this one, for instance
// vi.mock("@/hooks/store/use-workspace", () => import("@/hooks/store/fake-store-hooks")), says in `stores` what the
// stores hold, and reads there what they were asked to fetch. The hooks then run as plain functions, outside React,
// with fake-session-swr.ts. How the stores fetch is their own tests'.

import type { Profile, Project, Workspace, WorkspaceMember } from "@nerve/api-client";

/** What the stores hold, and the fetches they were asked for, in order; a test resets it before each case. */
export const stores: {
  workspaces: Workspace[] | undefined;
  /** The slug of the address's workspace (the router's), which the stores' reads of "the current workspace" follow. */
  address: string | undefined;
  /** Each workspace's memberships, by its slug, then by the member's account id. */
  members: Record<string, Record<string, WorkspaceMember>>;
  profile: Profile | undefined;
  /** The projects the project store gives (getProjectById). */
  projects: Project[];
  fetched: string[];
} = { workspaces: undefined, address: undefined, members: {}, profile: undefined, projects: [], fetched: [] };

/** The stores as a test starts: they hold nothing and were asked for nothing. */
export function emptyStores() {
  Object.assign(stores, {
    workspaces: undefined,
    address: undefined,
    members: {},
    profile: undefined,
    projects: [],
    fetched: [],
  });
}

/** A store's fetch, which says what it fetched; its Promise is what SWR gets. */
const fetching = (what: string) => Promise.resolve(stores.fetched.push(what));

/** A workspace as a fetch names it: its slug, and its id. */
const named = ({ id, slug }: Pick<Workspace, "id" | "slug">) => `${slug} (${id})`;

export function useWorkspace() {
  return {
    workspaces: stores.workspaces,
    currentWorkspace: stores.workspaces?.find((workspace) => workspace.slug === stores.address),
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
      getMemberships: (slug: string) => stores.members[slug],
      getWorkspaceMemberDetails: (userId: string) =>
        (stores.address === undefined ? undefined : stores.members[stores.address]?.[userId]) ?? null,
    },
    project: {
      fetchProjectMembers: (projectId: string) => fetching(`the members of ${projectId}`),
    },
  };
}

export function useProject() {
  return {
    getProjectById: (projectId: string) => stores.projects.find((project) => project.id === projectId),
    fetchProjects: (workspace: Pick<Workspace, "id" | "slug">) => fetching(`the projects of ${named(workspace)}`),
    fetchArchivedProjects: (workspace: Pick<Workspace, "id" | "slug">) =>
      fetching(`the archived projects of ${named(workspace)}`),
    fetchProject: (projectId: string) => fetching(`the project ${projectId}`),
  };
}

export function useProjectPreferences() {
  return {
    fetchNavigation: (projectId: string) => fetching(`the tab bar in ${projectId}`),
  };
}

export function useLabel() {
  return {
    fetchProjectLabels: (projectId: string) => fetching(`the labels of ${projectId}`),
  };
}

export function useProjectState() {
  return {
    fetchWorkspaceStates: (workspace: Pick<Workspace, "id" | "slug">) => fetching(`the states of ${named(workspace)}`),
    fetchProjectStates: (projectId: string) => fetching(`the states of ${projectId}`),
  };
}

export function useUserProfile() {
  return { data: stores.profile };
}
