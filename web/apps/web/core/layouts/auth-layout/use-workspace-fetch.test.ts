/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { beforeEach, describe, expect, it, vi } from "vitest";
import type { Workspace } from "@nerve/api-client";
import { emptyStores, stores } from "@/hooks/store/fake-store-hooks";
import { fetchHanded, handed, response } from "@/lib/fake-session-swr";
import { workspaceOf } from "@/store/workspace/fake-workspaces";

// A page of a workspace fetches what its caller may read (M3 design 3.1, 7.1), and its wrapper shows what his
// workspaces decide (8.3), with fake-session-swr.ts's stand-in for useSessionSWR and fake-store-hooks.ts's for the
// stores: the caller's list has acme alone, unless a test says otherwise.

vi.mock("@/lib/use-session-swr", () => import("@/lib/fake-session-swr"));
vi.mock("@/hooks/store/use-workspace", () => import("@/hooks/store/fake-store-hooks"));
vi.mock("@/hooks/store/use-member", () => import("@/hooks/store/fake-store-hooks"));

const { useWorkspaceFetch } = await import("./use-workspace-fetch");

const acme = workspaceOf("acme");

beforeEach(() => {
  handed.length = 0;
  response.current = {};
  emptyStores();
  stores.workspaces = [acme];
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

  it.each<{ when: string; workspaces: Workspace[] | undefined; failed?: true; shows: object }>([
    {
      when: "nerve cannot give the list",
      workspaces: [acme],
      failed: true,
      shows: { kind: "unavailable", retry: expect.any(Function) },
    },
    { when: "the list is not there yet", workspaces: undefined, shows: { kind: "loading" } },
    {
      when: "the list has others, not the address's",
      workspaces: [workspaceOf("beta")],
      shows: { kind: "not-found", hasWorkspaces: true },
    },
    { when: "the list has none", workspaces: [], shows: { kind: "not-found", hasWorkspaces: false } },
    { when: "the list has the address's", workspaces: [acme], shows: { kind: "ready", workspace: acme } },
  ])("shows the wrapper what to render when $when", ({ workspaces, failed, shows }) => {
    stores.workspaces = workspaces;
    if (failed) response.current = { error: new Error("nerve cannot be reached") };
    expect(useWorkspaceFetch("acme")).toEqual(shows);
  });

  it("fetches the list again when the page's retry asks", () => {
    const mutate = vi.fn(() => Promise.resolve(undefined));
    response.current = { error: new Error("nerve cannot be reached"), mutate };
    const access = useWorkspaceFetch("acme");
    if (access.kind === "unavailable") access.retry();
    expect(mutate).toHaveBeenCalledOnce();
  });
});
