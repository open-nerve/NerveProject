/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { beforeEach, describe, expect, it, vi } from "vitest";
import { EUserWorkspaceRoles } from "@nerve/types";
import { emptyStores, stores } from "@/hooks/store/fake-store-hooks";
import { fetchHanded, handed } from "@/lib/fake-session-swr";
import { workspaceOf } from "@/store/workspace/fake-workspaces";

// The members settings fetch what the caller may read (M3 design 7.1, 9.5), with fake-session-swr.ts's stand-in for
// useSessionSWR and fake-store-hooks.ts's for the stores: the caller's role in acme is the one his workspaces list gives.

vi.mock("@/lib/use-session-swr", () => import("@/lib/fake-session-swr"));
vi.mock("@/hooks/store/use-workspace", () => import("@/hooks/store/fake-store-hooks"));
vi.mock("@/hooks/store/use-member", () => import("@/hooks/store/fake-store-hooks"));

const { useMembersSettingsFetch } = await import("./use-members-settings-fetch");

beforeEach(() => {
  handed.length = 0;
  emptyStores();
});

describe("useMembersSettingsFetch", () => {
  it("fetches an admin the members and the invitations", async () => {
    stores.workspaces = [workspaceOf("acme", { role: EUserWorkspaceRoles.ADMIN })];
    useMembersSettingsFetch("acme");
    expect(handed.map(([fetch]) => fetch)).toEqual([
      ["WORKSPACE_MEMBERS", "id-acme", "acme"],
      ["WORKSPACE_INVITATIONS", "id-acme", "acme"],
    ]);
    await fetchHanded();
    expect(stores.fetched).toEqual(["the members of acme (id-acme)", "the invitations of acme (id-acme)"]);
  });

  it.each([
    { who: "a member", role: EUserWorkspaceRoles.MEMBER },
    { who: "a guest", role: EUserWorkspaceRoles.GUEST },
  ])("fetches $who the members alone", ({ role }) => {
    stores.workspaces = [workspaceOf("acme", { role })];
    useMembersSettingsFetch("acme");
    expect(handed.map(([fetch]) => fetch)).toEqual([["WORKSPACE_MEMBERS", "id-acme", "acme"], null]);
  });

  it("fetches nothing for a caller whose list does not have the workspace", () => {
    stores.workspaces = [workspaceOf("beta", { role: EUserWorkspaceRoles.ADMIN })];
    useMembersSettingsFetch("acme");
    expect(handed.map(([fetch]) => fetch)).toEqual([null, null]);
  });
});
