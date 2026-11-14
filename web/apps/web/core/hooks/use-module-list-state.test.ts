// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// A module's page's work items (P6b spec 2.7, R9): read-only while the page does not have the module and once it is
// archived, whatever its status; joinable by a new work item while not read-only and the project's modules are on. A
// module of another project than the page's is not the page's (spec 5, review re-71 M2).
import { expect, it } from "vitest";
import type { ModuleStatus } from "@nerve/api-client";
import { moduleOf, projectOf } from "@/store/project/fake-projects";
import { moduleListState } from "./use-module-list-state";

const project = projectOf("A", "w", { id: "A" });
const LATER = "2026-10-29T00:00:00Z";

it.each<ModuleStatus>(["backlog", "planned", "in-progress", "paused", "completed", "cancelled"])(
  "takes work items into a %s module, its status no matter",
  (status) => {
    expect(moduleListState(moduleOf(project, "Now", { status }), project)).toEqual({ readOnly: false, joinable: true });
  }
);

it.each<ModuleStatus>(["in-progress", "completed"])("keeps an archived %s module's work items read-only", (status) => {
  expect(moduleListState(moduleOf(project, "Old", { status, archived_at: LATER }), project)).toEqual({
    readOnly: true,
    joinable: false,
  });
});

it("keeps the work items read-only until the page has the module", () => {
  expect(moduleListState(undefined, project)).toEqual({ readOnly: true, joinable: false });
});

it("keeps the work items read-only, joining nothing, while the module held is another project's than the page's", () => {
  const other = projectOf("B", "w", { id: "B" });
  expect(moduleListState(moduleOf(project, "Now"), other)).toEqual({ readOnly: true, joinable: false });
});

it("lets no new work item join while the project's modules are off, or the page has no project", () => {
  const item = moduleOf(project, "Now");
  expect(moduleListState(item, { ...project, module_view: false })).toEqual({ readOnly: false, joinable: false });
  expect(moduleListState(item, undefined)).toEqual({ readOnly: false, joinable: false });
});
