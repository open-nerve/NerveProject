/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { beforeEach, describe, expect, it, vi } from "vitest";
import { handed } from "@/lib/fake-session-swr";

// A page of a workspace fetches what its caller may read (M3 design 3.1, 7.1), with fake-session-swr.ts's stand-in for
// useSessionSWR and stand-ins for the stores: the caller's list has acme alone.

vi.mock("@/lib/use-session-swr", () => import("@/lib/fake-session-swr"));
const fetched = vi.hoisted((): { calls: string[] } => ({ calls: [] }));
vi.mock("@/hooks/store/use-workspace", () => ({
  useWorkspace: () => ({
    fetchWorkspaces: () => Promise.resolve(fetched.calls.push("the workspaces")),
    getWorkspaceBySlug: (slug: string) => (slug === "acme" ? { slug } : null),
    preferences: {
      fetchPreferences: (slug: string) => Promise.resolve(fetched.calls.push(`the settings in ${slug}`)),
    },
  }),
}));
vi.mock("@/hooks/store/use-member", () => ({
  useMember: () => ({
    workspace: {
      fetchWorkspaceMembers: (slug: string) => Promise.resolve(fetched.calls.push(`the members of ${slug}`)),
    },
  }),
}));

const { useWorkspaceFetch } = await import("./use-workspace-fetch");

beforeEach(() => {
  handed.length = 0;
  fetched.calls = [];
});

describe("useWorkspaceFetch", () => {
  it("fetches the caller's workspaces and, in one of his, its members and his settings", async () => {
    useWorkspaceFetch("acme");
    expect(handed.map(([fetch]) => fetch)).toEqual([
      ["WORKSPACES"],
      ["WORKSPACE_MEMBERS", "acme"],
      ["WORKSPACE_PREFERENCES", "acme"],
    ]);
    await Promise.all(handed.map(([, fetcher]) => fetcher("acme")));
    expect(fetched.calls).toEqual(["the workspaces", "the members of acme", "the settings in acme"]);
  });

  it("fetches only the caller's workspaces where the address names none of his", () => {
    useWorkspaceFetch("elsewhere");
    expect(handed.map(([fetch]) => fetch)).toEqual([["WORKSPACES"], null, null]);
  });
});
