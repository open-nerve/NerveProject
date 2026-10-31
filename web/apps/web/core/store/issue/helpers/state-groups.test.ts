/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { expect, it } from "vitest";
import { projectOf, stateOf } from "@/store/project/fake-projects";
import { issueOf } from "../fake-issues";
import { getFilteredWorkItems, getGroupedWorkItemIds } from "./base-issues-utils";

const project = projectOf("P", "workspace");
const state = stateOf(project, "Done", "completed", 1000);
const row = { ...issueOf("R1"), state_id: state.id, state__group: "backlog" as const };
const catalog = { [state.id]: state };
it("groups by the state catalog and gives unknown states no group", () => {
  expect(getGroupedWorkItemIds([row], "state_detail.group", "-created_at", catalog)).toEqual({ completed: ["R1"] });
  expect(getGroupedWorkItemIds([row], "state_detail.group", "-created_at", {})).toEqual({ none: ["R1"] });
});
it("filters state groups using catalog facts instead of entity fallback", () => {
  expect(getFilteredWorkItems([row], { state_group: ["completed"] }, catalog)).toEqual([row]);
  expect(getFilteredWorkItems([row], { state_group: ["backlog"] }, {})).toEqual([]);
});
