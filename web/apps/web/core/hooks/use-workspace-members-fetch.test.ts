/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { beforeEach, describe, expect, it, vi } from "vitest";
import { emptyStores, stores } from "@/hooks/store/fake-store-hooks";
import { fetchHanded, handed } from "@/lib/fake-session-swr";
import { workspaceOf } from "@/store/workspace/fake-workspaces";

// The one fetch of a workspace's members (M3 design 7.1), with fake-session-swr.ts's stand-in for useSessionSWR and
// fake-store-hooks.ts's for the stores. Which workspace each page gives it is its own hook's test.

vi.mock("@/lib/use-session-swr", () => import("@/lib/fake-session-swr"));
vi.mock("@/hooks/store/use-member", () => import("@/hooks/store/fake-store-hooks"));

const { useWorkspaceMembersFetch } = await import("./use-workspace-members-fetch");

beforeEach(() => {
  handed.length = 0;
  emptyStores();
});

describe("useWorkspaceMembersFetch", () => {
  it("fetches the members of a workspace of the caller's, keyed by its id, from its slug's address", async () => {
    useWorkspaceMembersFetch(workspaceOf("acme"));
    expect(handed.map(([fetch]) => fetch)).toEqual([["WORKSPACE_MEMBERS", "id-acme", "acme"]]);
    await fetchHanded();
    expect(stores.fetched).toEqual(["the members of acme (id-acme)"]);
  });

  it("fetches nothing without a workspace of the caller's", () => {
    useWorkspaceMembersFetch(null);
    expect(handed.map(([fetch]) => fetch)).toEqual([null]);
  });
});
