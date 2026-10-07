/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { beforeEach, describe, expect, it, vi } from "vitest";
import { handed } from "@/lib/fake-session-swr";

// A page of a workspace fetches what its caller may read (M3 design 3.1, 7.1), with fake-session-swr.ts's stand-in for
// useSessionSWR and a stand-in for the store.

vi.mock("@/lib/use-session-swr", () => import("@/lib/fake-session-swr"));
const fetched = vi.hoisted((): { calls: string[] } => ({ calls: [] }));
vi.mock("@/hooks/store/use-workspace", () => ({
  useWorkspace: () => ({
    fetchWorkspaces: () => Promise.resolve(fetched.calls.push("the workspaces")),
  }),
}));

const { useWorkspaceFetch } = await import("./use-workspace-fetch");

beforeEach(() => {
  handed.length = 0;
  fetched.calls = [];
});

describe("useWorkspaceFetch", () => {
  it("fetches the caller's workspaces", async () => {
    useWorkspaceFetch();
    expect(handed.map(([fetch]) => fetch)).toEqual([["WORKSPACES"]]);
    await Promise.all(handed.map(([, fetcher]) => fetcher()));
    expect(fetched.calls).toEqual(["the workspaces"]);
  });
});
