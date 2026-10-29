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
    workspaceDraftIssues: { marker: "draft" },
  },
}));
vi.mock("react", () => ({ useContext: () => store }));
vi.mock("@/lib/store-context", () => ({ StoreContext: {} }));

describe("issue store selection", () => {
  it.each(["GLOBAL", "CYCLE", "MODULE", "PROJECT_VIEW", "unknown"])(
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
});
