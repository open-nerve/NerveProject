/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { beforeEach, describe, expect, it, vi } from "vitest";
import { emptyStores, stores } from "@/hooks/store/fake-store-hooks";
import { fetchHanded, handed } from "@/lib/fake-session-swr";
import { workspaceOf } from "@/store/workspace/fake-workspaces";

// A page of a workspace fetches what its caller may read (M3 design 3.1, 7.1), with fake-session-swr.ts's stand-in for
// useSessionSWR and fake-store-hooks.ts's for the stores: the caller's list has acme alone.

vi.mock("@/lib/use-session-swr", () => import("@/lib/fake-session-swr"));
vi.mock("@/hooks/store/use-workspace", () => import("@/hooks/store/fake-store-hooks"));
vi.mock("@/hooks/store/use-member", () => import("@/hooks/store/fake-store-hooks"));

const { useWorkspaceFetch } = await import("./use-workspace-fetch");

beforeEach(() => {
  handed.length = 0;
  emptyStores();
  stores.workspaces = [workspaceOf("acme")];
});

describe("useWorkspaceFetch", () => {
  it("fetches the caller's workspaces and, in one of his, its members and his settings", async () => {
    useWorkspaceFetch("acme");
    expect(handed.map(([fetch]) => fetch)).toEqual([
      ["WORKSPACES"],
      ["WORKSPACE_MEMBERS", "id-acme", "acme"],
      ["WORKSPACE_PREFERENCES", "id-acme", "acme"],
    ]);
    await fetchHanded();
    expect(stores.fetched).toEqual([
      "the workspaces",
      "the members of acme (id-acme)",
      "the settings in acme (id-acme)",
    ]);
  });

  it("fetches only the caller's workspaces where the address names none of his", () => {
    useWorkspaceFetch("elsewhere");
    expect(handed.map(([fetch]) => fetch)).toEqual([["WORKSPACES"], null, null]);
  });
});
