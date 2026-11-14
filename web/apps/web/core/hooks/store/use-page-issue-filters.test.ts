/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// A module's page reads its own preferences by the route's module (P6b spec 2.6): neither the router store's module nor
// the cycle page's or the project page's preferences.
import { beforeEach, expect, it, vi } from "vitest";
import { EIssuesStoreType } from "@nerve/types";
import { usePageIssueFilters } from "./use-page-issue-filters";

const fixture = vi.hoisted(() => ({
  params: {} as Record<string, string>,
  store: {
    router: { projectId: "A", moduleId: "router-module", cycleId: "router-module" },
    issue: {
      projectIssuesFilter: { issueFilters: { marker: "project" } },
      cycleIssuesFilter: { getIssueFilters: (id: string) => ({ marker: `cycle ${id}` }) },
      moduleIssuesFilter: { getIssueFilters: (id: string) => ({ marker: `module ${id}` }) },
    },
  },
}));
vi.mock("react", () => ({ useContext: () => fixture.store }));
vi.mock("react-router", () => ({ useParams: () => fixture.params }));
vi.mock("@/lib/store-context", () => ({ StoreContext: {} }));
beforeEach(() => {
  fixture.params = { workspaceSlug: "ws", projectId: "A", moduleId: "M1" };
});

it("reads the route module's own preferences on its page", () => {
  expect(usePageIssueFilters(EIssuesStoreType.MODULE)).toEqual({ marker: "module M1" });
  fixture.params.moduleId = "M2";
  expect(usePageIssueFilters(EIssuesStoreType.MODULE)).toEqual({ marker: "module M2" });
});

it("reads none before the route names a module", () => {
  fixture.params = { workspaceSlug: "ws", projectId: "A" };
  expect(usePageIssueFilters(EIssuesStoreType.MODULE)).toBeUndefined();
});
