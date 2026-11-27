/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { beforeEach, describe, expect, it, vi } from "vitest";
import { emptyStores, stores } from "@/hooks/store/fake-store-hooks";
import { fetchHanded, handed } from "@/lib/fake-session-swr";
import { workspaceOf } from "@/store/workspace/fake-workspaces";

// The read of the workspace's label catalog (M7/P6b spec 2.2, R17), with fake-session-swr.ts's stand-in for
// useSessionSWR and fake-store-hooks.ts's for the stores: the caller's list has acme and beta, unless a test says
// otherwise. Which page reads it, behind which gate, is each page's test.

vi.mock("@/lib/use-session-swr", () => import("@/lib/fake-session-swr"));
vi.mock("@/hooks/store/use-workspace", () => import("@/hooks/store/fake-store-hooks"));
vi.mock("@/hooks/store/use-label", () => import("@/hooks/store/fake-store-hooks"));

const { useWorkspaceLabelCatalog } = await import("./use-workspace-label-catalog");

beforeEach(() => {
  handed.length = 0;
  emptyStores();
  stores.workspaces = [workspaceOf("acme"), workspaceOf("beta")];
});

describe("useWorkspaceLabelCatalog", () => {
  it("reads the catalog of the route's workspace, keyed by its id and its slug, into the label store", async () => {
    useWorkspaceLabelCatalog("beta", true);
    expect(handed.map(([fetch]) => fetch)).toEqual([["WORKSPACE_LABELS", "id-beta", "beta"]]);
    await fetchHanded();
    expect(stores.fetched).toEqual(["the labels of beta (id-beta)"]);
  });

  it.each([
    { when: "its gate is closed", slug: "acme", enabled: false },
    { when: "the caller's list has not the route's workspace", slug: "elsewhere", enabled: true },
  ])("reads nothing when $when", async ({ slug, enabled }) => {
    useWorkspaceLabelCatalog(slug, enabled);
    expect(handed.map(([fetch]) => fetch)).toEqual([null]);
    await fetchHanded();
    expect(stores.fetched).toEqual([]);
  });
});
