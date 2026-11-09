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
