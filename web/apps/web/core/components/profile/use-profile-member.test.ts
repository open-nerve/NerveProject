/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { beforeEach, describe, expect, it, vi } from "vitest";
import type { WorkspaceMember } from "@nerve/api-client";
import { emptyStores, stores } from "@/hooks/store/fake-store-hooks";
import { handed, response } from "@/lib/fake-session-swr";
import { workspaceOf } from "@/store/workspace/fake-workspaces";

// Which member of the workspace a profile page is about, with fake-session-swr.ts's stand-in for useSessionSWR and
// fake-store-hooks.ts's for the stores. The members' fetch it shares is use-workspace-members-fetch.test.ts's.

vi.mock("@/lib/use-session-swr", () => import("@/lib/fake-session-swr"));
vi.mock("@/hooks/store/use-member", () => import("@/hooks/store/fake-store-hooks"));
// The builders' module builds the account's store, which imports the tab's session; nothing here reads it.
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: {}, publicClient: {} }));

const { useProfileMember } = await import("./use-profile-member");
const { membershipOf } = await import("@/store/member/workspace/fake-members");

const acme = workspaceOf("acme");
const bob = membershipOf("bob");

beforeEach(() => {
  handed.length = 0;
  response.current = {};
  emptyStores();
});

describe("useProfileMember", () => {
  it("fetches the members of the page's workspace", () => {
    useProfileMember(acme, "u-bob");
    expect(handed.map(([fetch]) => fetch)).toEqual([["WORKSPACE_MEMBERS", "id-acme", "acme"]]);
  });

  it.each<{ when: string; fetched: object; members: Record<string, WorkspaceMember>; shows: object }>([
    { when: "the members are not there yet", fetched: {}, members: {}, shows: { kind: "loading" } },
    {
      when: "they failed to load",
      fetched: { error: new Error("nerve cannot be reached") },
      members: {},
      shows: { kind: "load-failed" },
    },
    {
      when: "he is one of them",
      fetched: { data: { "u-bob": bob } },
      members: { "u-bob": bob },
      shows: { kind: "member", member: bob.member, joinedAt: bob.created_at },
    },
    {
      when: "his membership ended",
      fetched: { data: { "u-bob": { ...bob, is_active: false } } },
      members: { "u-bob": { ...bob, is_active: false } },
      shows: { kind: "not-a-member" },
    },
    { when: "he is none of them", fetched: { data: {} }, members: {}, shows: { kind: "not-a-member" } },
  ])("says what the page shows of a user when $when", ({ fetched, members, shows }) => {
    response.current = fetched;
    stores.members = members;
    expect(useProfileMember(acme, "u-bob")).toEqual({ member: undefined, ...shows });
  });
});
