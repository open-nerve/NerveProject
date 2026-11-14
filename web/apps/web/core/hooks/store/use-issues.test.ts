/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { describe, expect, it, vi } from "vitest";
import { EIssuesStoreType } from "@nerve/types";
import { useIssues } from "./use-issues";

const store = vi.hoisted(() => ({
  issue: {
    issues: { issuesMap: {} },
    projectIssues: { marker: "project" },
    projectIssuesFilter: {},
    profileIssues: { marker: "profile" },
    profileIssuesFilter: {},
    archivedIssues: { marker: "archived" },
    archivedIssuesFilter: {},
    cycleIssues: { marker: "cycle" },
    cycleIssuesFilter: { marker: "cycle preferences" },
    moduleIssues: { marker: "module" },
    moduleIssuesFilter: { marker: "module preferences" },
    workspaceDraftIssues: { marker: "draft" },
  },
}));
vi.mock("react", () => ({ useContext: () => store }));
vi.mock("@/lib/store-context", () => ({ StoreContext: {} }));

describe("issue store selection", () => {
  it.each(["GLOBAL", "PROJECT_VIEW", "unknown"])(
    "rejects the retired %s list instead of selecting project state",
    (kind) => {
      expect(() => useIssues(kind as EIssuesStoreType)).toThrow("Unsupported issue store");
    }
  );
  it("keeps default, project, profile, archived and workspace draft selection", () => {
    expect(useIssues().issues).toEqual(store.issue.projectIssues);
    expect(useIssues(EIssuesStoreType.DEFAULT).issues).toEqual(store.issue.projectIssues);
    expect(useIssues(EIssuesStoreType.PROJECT).issues).toEqual(store.issue.projectIssues);
    expect(useIssues(EIssuesStoreType.PROFILE).issues).toEqual(store.issue.profileIssues);
    expect(useIssues(EIssuesStoreType.ARCHIVED).issues).toEqual(store.issue.archivedIssues);
    expect(useIssues(EIssuesStoreType.WORKSPACE_DRAFT).issues).toEqual(store.issue.workspaceDraftIssues);
    expect(useIssues(EIssuesStoreType.WORKSPACE_DRAFT)).not.toHaveProperty("issuesFilter");
  });
  it("selects the cycle page's own list and preferences, not the project page's", () => {
    const cycle = useIssues(EIssuesStoreType.CYCLE);
    expect(cycle.issues).toEqual(store.issue.cycleIssues);
    expect(cycle.issuesFilter).toEqual(store.issue.cycleIssuesFilter);
  });
  it("selects the module page's own list and preferences, not the project page's or the cycle page's", () => {
    const module = useIssues(EIssuesStoreType.MODULE);
    expect(module.issues).toEqual(store.issue.moduleIssues);
    expect(module.issuesFilter).toEqual(store.issue.moduleIssuesFilter);
  });
});
