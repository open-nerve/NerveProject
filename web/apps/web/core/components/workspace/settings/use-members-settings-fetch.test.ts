/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { beforeEach, describe, expect, it, vi } from "vitest";
import { EUserWorkspaceRoles } from "@nerve/types";
import { handed } from "@/lib/fake-session-swr";

// The members settings fetch what the caller may read (M3 design 7.1, 9.5), with fake-session-swr.ts's stand-in for
// useSessionSWR and stand-ins for the stores: the caller's role in acme is the one his workspaces list gives.

vi.mock("@/lib/use-session-swr", () => import("@/lib/fake-session-swr"));
const state = vi.hoisted((): { calls: string[]; role: number | undefined } => ({ calls: [], role: undefined }));
vi.mock("@/hooks/store/use-workspace", () => ({
  useWorkspace: () => ({
    getWorkspaceBySlug: (slug: string) =>
      slug === "acme" && state.role !== undefined ? { slug, role: state.role } : null,
  }),
}));
vi.mock("@/hooks/store/use-member", () => ({
  useMember: () => ({
    workspace: {
      fetchWorkspaceMembers: (slug: string) => Promise.resolve(state.calls.push(`members of ${slug}`)),
      fetchWorkspaceMemberInvitations: (slug: string) => Promise.resolve(state.calls.push(`invitations of ${slug}`)),
    },
  }),
}));

const { useMembersSettingsFetch } = await import("./use-members-settings-fetch");

beforeEach(() => {
  handed.length = 0;
  state.calls = [];
  state.role = undefined;
});

describe("useMembersSettingsFetch", () => {
  it("fetches an admin the members and the invitations", async () => {
    state.role = EUserWorkspaceRoles.ADMIN;
    useMembersSettingsFetch("acme");
    expect(handed.map(([fetch]) => fetch)).toEqual([
      ["WORKSPACE_MEMBERS", "acme"],
      ["WORKSPACE_INVITATIONS", "acme"],
    ]);
    await Promise.all(handed.map(([, fetcher]) => fetcher("acme")));
    expect(state.calls).toEqual(["members of acme", "invitations of acme"]);
  });

  it.each([
    { who: "a member", role: EUserWorkspaceRoles.MEMBER },
    { who: "a guest", role: EUserWorkspaceRoles.GUEST },
  ])("fetches $who the members alone", ({ role }) => {
    state.role = role;
    useMembersSettingsFetch("acme");
    expect(handed.map(([fetch]) => fetch)).toEqual([["WORKSPACE_MEMBERS", "acme"], null]);
  });

  it("fetches nothing for a caller whose list does not have the workspace", () => {
    useMembersSettingsFetch("acme");
    expect(handed.map(([fetch]) => fetch)).toEqual([null, null]);
  });
});
