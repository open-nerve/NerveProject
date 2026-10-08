/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { projectTab } from "@/store/project/fake-projects";
import { loadWorkspaces, workspaceOf } from "@/store/workspace/fake-workspaces";

// The projects page's filters (v0 design 7.7: what a store keeps for each workspace is keyed by the workspace's id),
// against a fake nerve. The tab's address names acme; beta is the caller's other workspace.

const acme = workspaceOf("acme");
const beta = workspaceOf("beta");

beforeEach(() => {
  vi.useFakeTimers({ now: 1_000_000 });
});
afterEach(() => {
  vi.useRealTimers();
});

describe("ProjectFilterStore", () => {
  it("starts a workspace made again under a deleted one's slug from the default filters: they are kept by its id", async () => {
    const { nerve, workspaceRoot, projectRoot } = await projectTab({ workspace: acme }, { workspace: beta });
    const filters = projectRoot.projectFilter;
    filters.updateDisplayFilters("acme", { order_by: "sort_order", my_projects: true });
    filters.updateFilters("acme", { access: ["public"] });
    expect(filters.currentWorkspaceDisplayFilters).toEqual({ order_by: "sort_order", my_projects: true });
    expect(filters.currentWorkspaceFilters).toEqual({ access: ["public"] });

    // acme is deleted, and a workspace made since takes its slug: the caller's list now has it under another id
    await loadWorkspaces(nerve, workspaceRoot, [workspaceOf("acme", { id: "id-acme-2" }), beta]);
    expect(filters.currentWorkspaceDisplayFilters).toEqual({ order_by: "created_at" });
    expect(filters.currentWorkspaceFilters).toEqual({});
  });
});
