/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { beforeEach, describe, expect, it, vi } from "vitest";
import { emptyStores, stores } from "@/hooks/store/fake-store-hooks";
import { handed } from "@/lib/fake-session-swr";
import { workspaceOf } from "@/store/workspace/fake-workspaces";

// A profile page fetches the members of a workspace its caller may read (M3 design 7.1), with fake-session-swr.ts's
// stand-in for useSessionSWR and fake-store-hooks.ts's for the stores: the caller's list has acme alone.

vi.mock("@/lib/use-session-swr", () => import("@/lib/fake-session-swr"));
vi.mock("@/hooks/store/use-workspace", () => import("@/hooks/store/fake-store-hooks"));
vi.mock("@/hooks/store/use-member", () => import("@/hooks/store/fake-store-hooks"));

const { useProfileMember } = await import("./use-profile-member");

beforeEach(() => {
  handed.length = 0;
  emptyStores();
  stores.workspaces = [workspaceOf("acme")];
});

describe("useProfileMember", () => {
  it("fetches the members of one of the caller's workspaces", async () => {
    useProfileMember("acme", "u-bob");
    expect(handed.map(([fetch]) => fetch)).toEqual([["WORKSPACE_MEMBERS", "acme"]]);
    await Promise.all(handed.map(([, fetcher]) => fetcher("acme")));
    expect(stores.fetched).toEqual(["the members of acme"]);
  });

  it("fetches nothing where the address names none of the caller's workspaces", () => {
    useProfileMember("elsewhere", "u-bob");
    expect(handed.map(([fetch]) => fetch)).toEqual([null]);
  });
});
