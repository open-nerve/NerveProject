// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// A cycle's page's work items (P5b spec 2.5, 2.6, R5): a cycle of another project than the page's is not the page's,
// whose work items stay read-only, joining nothing, as while the page does not have the cycle (P6b spec 5, review re-71
// M2). The rest of the cycle's list state is pinned by its page's tests (header.test.tsx, cycle-create.test.tsx).
import { expect, it } from "vitest";
import { cycleOf, projectOf } from "@/store/project/fake-projects";
import { cycleListState } from "./use-cycle-list-state";

const project = projectOf("A", "w", { id: "A" });

it("keeps the work items read-only, joining nothing, while the cycle held is another project's than the page's", () => {
  const other = projectOf("B", "w", { id: "B" });
  expect(cycleListState(cycleOf(project, "Now", { status: "current" }), other)).toEqual({
    readOnly: true,
    joinable: false,
  });
});

it("keeps the work items read-only, joining nothing, until the page has the cycle", () => {
  expect(cycleListState(undefined, project)).toEqual({ readOnly: true, joinable: false });
});

it("lets a new work item join a cycle under way of the page's project while its cycles are on", () => {
  expect(cycleListState(cycleOf(project, "Now", { status: "current" }), project)).toEqual({
    readOnly: false,
    joinable: true,
  });
});
