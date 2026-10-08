/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { beforeEach, describe, expect, it, vi } from "vitest";
import type { Workspace } from "@nerve/api-client";
import { emptyStores, stores } from "@/hooks/store/fake-store-hooks";
import { fetchHanded, handed } from "@/lib/fake-session-swr";
import { workspaceOf } from "@/store/workspace/fake-workspaces";

// The projects page's own fetch (M3 design 7.1), with fake-session-swr.ts's stand-in for useSessionSWR and
// fake-store-hooks.ts's for the stores: the caller's list has acme and beta, unless a test says otherwise. The
// projects that are not archived are use-workspace-fetch.test.ts's.

vi.mock("@/lib/use-session-swr", () => import("@/lib/fake-session-swr"));
vi.mock("@/hooks/store/use-workspace", () => import("@/hooks/store/fake-store-hooks"));
vi.mock("@/hooks/store/use-project", () => import("@/hooks/store/fake-store-hooks"));

const { useArchivedProjectsFetch } = await import("./use-archived-projects-fetch");

beforeEach(() => {
  handed.length = 0;
  emptyStores();
  stores.workspaces = [workspaceOf("acme"), workspaceOf("beta")];
});

describe("useArchivedProjectsFetch", () => {
  it("fetches the archived projects of the address's workspace, keyed by its id, from its slug's address", async () => {
    useArchivedProjectsFetch("beta");
    expect(handed.map(([fetch]) => fetch)).toEqual([["ARCHIVED_PROJECTS", "id-beta", "beta"]]);
    await fetchHanded();
    expect(stores.fetched).toEqual(["the archived projects of beta (id-beta)"]);
  });

  it.each<{ when: string; slug: string; workspaces: Workspace[] | undefined }>([
    { when: "the address names no workspace of his", slug: "elsewhere", workspaces: [workspaceOf("acme")] },
    { when: "his workspaces are not there yet", slug: "acme", workspaces: undefined },
  ])("fetches nothing when $when", ({ slug, workspaces }) => {
    stores.workspaces = workspaces;
    useArchivedProjectsFetch(slug);
    expect(handed.map(([fetch]) => fetch)).toEqual([null]);
  });
});
