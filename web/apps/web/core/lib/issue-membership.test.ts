/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { expect, it } from "vitest";
import { listContext, listIssue } from "@/store/issue/fake-issues";
import { issueMembership, issueParts } from "./issue-membership";

const M1 = "019a0000-0000-7000-8000-00000000000a";
const M2 = "019a0000-0000-7000-8000-00000000000b";

// Modules group as labels do (M6 P3b spec 2.1): a work item is in the group
// of each of its modules once, and one without modules in none.
it("places a work item in each of its modules' groups, and one without modules in none", () => {
  const both = listIssue("both", 1, { module_ids: [M1, M2, M1] });
  const neither = listIssue("neither", 2);
  const grouped = listContext({ group_by: "module_id" });
  expect(issueParts(both, grouped)).toEqual([{ group: M1 }, { group: M2 }]);
  expect(issueParts(neither, grouped)).toEqual([{ group: "none" }]);
  expect(issueMembership(both, grouped, { kind: "group", group: M2 })).toBe(true);
  expect(issueMembership(both, grouped, { kind: "group", group: "none" })).toBe(false);
  expect(issueMembership(neither, grouped, { kind: "group", group: "none" })).toBe(true);
  const lanes = listContext({ group_by: "cycle_id", sub_group_by: "module_id" });
  expect(issueParts(both, lanes)).toEqual([
    { group: "none", subgroup: M1 },
    { group: "none", subgroup: M2 },
  ]);
  expect(issueMembership(neither, lanes, { kind: "subgroup", subgroup: "none" })).toBe(true);
});

const C1 = "019a0000-0000-7000-8000-0000000000c1";
const C2 = "019a0000-0000-7000-8000-0000000000c2";

// A cycle page's list is the project list with cycle_id (M6 P5b spec 2.1, R1): a work item belongs to it only while
// its cycle is one of the chosen alternatives, none standing for no cycle, as nerve answers.
it.each<[string[], (string | null)[], (string | null)[]]>([
  [[C1], [C1], [C2, null]],
  [["none"], [null], [C1, C2]],
  [[C1, "none"], [C1, null], [C2]],
])("a cycle_id query %j holds work items in %j and not in %j", (cycles, inside, outside) => {
  const context = listContext({ cycle_id: cycles });
  for (const cycle of inside) {
    const issue = listIssue("x", 1, { cycle_id: cycle });
    expect(issueMembership(issue, context)).toBe(true);
    expect(issueParts(issue, context)).toEqual([{ group: "" }]);
  }
  for (const cycle of outside) {
    const issue = listIssue("x", 1, { cycle_id: cycle });
    expect(issueMembership(issue, context)).toBe(false);
    expect(issueParts(issue, context)).toEqual([]);
  }
});

// Modules filter as labels do (M6 P5b R2): the chosen ids meet the work item's modules, none standing for no module.
it.each<[string[], string[][], string[][]]>([
  [[M1], [[M1], [M2, M1]], [[M2], []]],
  [["none"], [[]], [[M1], [M1, M2]]],
  [[M1, "none"], [[M1], []], [[M2]]],
])("a module_id query %j holds work items in modules %j and not in %j", (modules, inside, outside) => {
  const context = listContext({ module_id: modules });
  for (const module_ids of inside) expect(issueMembership(listIssue("x", 1, { module_ids }), context)).toBe(true);
  for (const module_ids of outside) {
    const issue = listIssue("x", 1, { module_ids });
    expect(issueMembership(issue, context)).toBe(false);
    expect(issueParts(issue, context)).toEqual([]);
  }
});
