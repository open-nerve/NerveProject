/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { readFileSync } from "node:fs";
import type { Issue, IssueListOrder, IssuePriority } from "@nerve/api-client";
import { expect, test } from "vitest";
import { issueOf } from "@/store/issue/fake-issues";
import { userOf } from "@/store/member/fake-users";
import { labelOf, projectOf, stateOf } from "@/store/project/fake-projects";
import {
  compareCodePoints,
  compareIssueTime,
  compareRows,
  issueNameKey,
  issueSortKey,
  type IssueCatalog,
} from "./issue-order";
import type { Row, SortKey } from "./paged-groups";

type Scenario = {
  order: IssueListOrder;
  rows: { id: string; names?: string[]; key: SortKey }[];
  ascending_ids: string[];
  descending_ids?: string[];
};
const raw = readFileSync(new URL("../../../../../api/testdata/issue-order.json", import.meta.url), "utf8");
// Node 24 supplies the original numeric token before JSON's number rounding.
const fixture: { name_keys: { name: string; key: string }[]; scenarios: Scenario[] } = JSON.parse(
  raw,
  (key: string, value: unknown, context?: { source?: string }) =>
    key === "key" && typeof value === "number" && !Number.isSafeInteger(value) && Number.isInteger(value)
      ? BigInt(context?.source ?? "")
      : value
);
const project = projectOf("A", "ws");
const emptyCatalog: IssueCatalog = { states: new Map(), labels: new Map(), members: new Map() };

function fixtureRow(row: Scenario["rows"][number], order: IssueListOrder): Row {
  const issue = issueOf(row.id);
  const catalog: IssueCatalog = { states: new Map(), labels: new Map(), members: new Map() };
  const names = row.names ?? [];
  if (order === "sub_issues_count" || order === "link_count") {
    if (typeof row.key !== "number" && typeof row.key !== "bigint") throw new Error("count fixture");
    return { id: row.id, key: BigInt(row.key), updatedAt: issue.updated_at };
  }
  if (order === "state_name") {
    const states = names.map((name) => stateOf(project, name, "started", 1));
    issue.state_id = states[0]?.id ?? "missing";
    catalog.states = new Map(states.map((state) => [state.id, state]));
  } else if (order === "label_name") {
    const labels = names.map((name) => labelOf(project, name, 1));
    issue.label_ids = labels.map((label) => label.id);
    catalog.labels = new Map(labels.map((label) => [label.id, label]));
  } else if (order === "assignee_name") {
    const members = names.map(userOf);
    issue.assignee_ids = members.map((member) => member.id);
    catalog.members = new Map(members.map((member) => [member.id, member]));
  } else if (order === "priority") {
    const priorities: IssuePriority[] = ["none", "low", "medium", "high", "urgent"];
    if (typeof row.key !== "number" || !priorities[row.key]) throw new Error("priority fixture");
    issue.priority = priorities[row.key]!;
  } else if (order === "sort_order") {
    if (typeof row.key !== "number") throw new Error("sort fixture");
    issue.sort_order = row.key;
  } else if (order === "start_date" || order === "target_date") {
    if (typeof row.key !== "string" && row.key !== null) throw new Error("date fixture");
    issue[order] = row.key;
  } else if (order === "created_at" || order === "updated_at") {
    if (typeof row.key !== "string") throw new Error("time fixture");
    issue[order] = row.key;
  } else throw new Error(`Unexpected fixture order: ${order}`);
  const key = issueSortKey(issue, order, catalog);
  expect(key).toBe(row.key);
  return { id: issue.id, key, updatedAt: issue.updated_at };
}

test("ordersAllSharedFixtureScenarios", () => {
  for (const { name, key } of fixture.name_keys) expect(issueNameKey(name)).toBe(key);
  expect(fixture.scenarios).toHaveLength(11);
  for (const scenario of fixture.scenarios) {
    const rows = scenario.rows.map((row) => fixtureRow(row, scenario.order));
    expect(
      rows.toSorted((a, b) => compareRows(a, b, scenario.order)).map((row) => row.id),
      scenario.order
    ).toEqual(scenario.ascending_ids);
    if (scenario.descending_ids) {
      const descending = `-${scenario.order}` as IssueListOrder;
      expect(
        rows.toSorted((a, b) => compareRows(a, b, descending)).map((row) => row.id),
        descending
      ).toEqual(scenario.descending_ids);
    }
  }
});

test("preservesRawFixtureIntegerTokens", () => {
  const counts = fixture.scenarios.find((scenario) => scenario.order === "sub_issues_count");
  expect(counts?.rows.map((row) => row.key)).toEqual([
    9007199254740992n,
    9007199254740992n,
    9007199254740993n,
    9223372036854775807n,
  ]);
});

test("comparesUnicodeCodePointsWithoutLocaleOrUTF16Order", () => {
  expect(compareCodePoints("～", "😀")).toBeLessThan(0);
  expect(compareCodePoints("z", "Ä")).toBeLessThan(0);
  expect(compareCodePoints("a", "aa")).toBeLessThan(0);
  expect(compareCodePoints("same", "same")).toBe(0);
});

test("comparesSameMillisecondVersionsExactly", () => {
  expect(compareIssueTime("2026-10-17T10:00:00.123456Z", "2026-10-17T10:00:00.123457Z")).toBeLessThan(0);
  expect(compareIssueTime("2026-10-17T10:00:00.1Z", "2026-10-17T10:00:00.100000000Z")).toBe(0);
  expect(compareIssueTime("2026-10-17T18:00:00.123456789+08:00", "2026-10-17T10:00:00.123456789Z")).toBe(0);
  expect(compareIssueTime("2026-10-17T09:00:00-01:00", "2026-10-17T10:00:00Z")).toBe(0);
});

test.each(["invalid", "2026-02-30T00:00:00Z", "2026-10-17T10:00:00.1234567890Z", "2026-10-17T24:00:00Z", "2026-10-17"])(
  "rejectsInvalidVersion %s",
  (time) => {
    expect(() => compareIssueTime(time, issueOf("a").updated_at)).toThrow();
  }
);

test("safeIssueCountProducesExactSortKey", () => {
  const issue: Issue = { ...issueOf("a"), sub_issues_count: Number.MAX_SAFE_INTEGER, link_count: 2 };
  expect(issueSortKey(issue, "sub_issues_count", emptyCatalog)).toBe(9007199254740991n);
  expect(issueSortKey(issue, "-link_count", emptyCatalog)).toBe(2n);
});

test.each([Number.MAX_SAFE_INTEGER + 1, -1, 0.5, Infinity, NaN])("unsafeIssueCountIsRejected %s", (count) => {
  expect(() => issueSortKey({ ...issueOf("a"), sub_issues_count: count }, "sub_issues_count", emptyCatalog)).toThrow();
});

test("nullKeysStayLastAndEqualIDsFollowDirection", () => {
  const a: Row = { id: "a", key: null, updatedAt: issueOf("a").updated_at };
  const b: Row = { ...a, id: "b" };
  expect(compareRows(a, b, "-start_date")).toBeGreaterThan(0);
  expect(compareRows(a, { ...b, key: "2026-01-01" }, "-start_date")).toBeGreaterThan(0);
});

test("ordersRealIssuePrioritiesByServerRanksInBothDirections", () => {
  // Row priority order (M4 §3.16) differs from the priority group display order.
  const samples: { id: string; priority: IssuePriority; rank: number }[] = [
    { id: "00000000-0000-0000-0000-000000000008", priority: "none", rank: 0 },
    { id: "00000000-0000-0000-0000-000000000007", priority: "low", rank: 1 },
    { id: "00000000-0000-0000-0000-000000000006", priority: "medium", rank: 2 },
    { id: "00000000-0000-0000-0000-000000000005", priority: "high", rank: 3 },
    { id: "00000000-0000-0000-0000-000000000004", priority: "urgent", rank: 4 },
    { id: "00000000-0000-0000-0000-000000000003", priority: "none", rank: 0 },
    { id: "00000000-0000-0000-0000-000000000002", priority: "urgent", rank: 4 },
  ];
  const issues: Issue[] = samples.map(({ id, priority }) => ({ ...issueOf(id), priority }));
  const rows = issues.map(
    (issue): Row => ({ id: issue.id, key: issueSortKey(issue, "priority", emptyCatalog), updatedAt: issue.updated_at })
  );
  expect.soft(rows.map((row) => row.key)).toEqual(samples.map((sample) => sample.rank));
  expect
    .soft(rows.toSorted((a, b) => compareRows(a, b, "priority")).map((row) => row.id))
    .toEqual([
      "00000000-0000-0000-0000-000000000003",
      "00000000-0000-0000-0000-000000000008",
      "00000000-0000-0000-0000-000000000007",
      "00000000-0000-0000-0000-000000000006",
      "00000000-0000-0000-0000-000000000005",
      "00000000-0000-0000-0000-000000000002",
      "00000000-0000-0000-0000-000000000004",
    ]);
  expect
    .soft(rows.toSorted((a, b) => compareRows(a, b, "-priority")).map((row) => row.id))
    .toEqual([
      "00000000-0000-0000-0000-000000000004",
      "00000000-0000-0000-0000-000000000002",
      "00000000-0000-0000-0000-000000000005",
      "00000000-0000-0000-0000-000000000006",
      "00000000-0000-0000-0000-000000000007",
      "00000000-0000-0000-0000-000000000008",
      "00000000-0000-0000-0000-000000000003",
    ]);
});

test("sharedPrioritySamplesRepresentNoneNoneUrgentIssues", () => {
  const scenario = fixture.scenarios.find((item) => item.order === "priority")!;
  // These are the actual priorities assigned to the shared samples by the server test.
  const priorities: IssuePriority[] = ["none", "none", "urgent"];
  expect.soft(scenario.rows).toHaveLength(priorities.length);
  const rows = scenario.rows.map((sample, index): Row => {
    const issue: Issue = { ...issueOf(sample.id), priority: priorities[index]! };
    return { id: issue.id, key: issueSortKey(issue, "priority", emptyCatalog), updatedAt: issue.updated_at };
  });
  expect.soft(rows.map((row) => row.key)).toEqual(scenario.rows.map((sample) => sample.key));
  expect
    .soft(rows.toSorted((a, b) => compareRows(a, b, "priority")).map((row) => row.id))
    .toEqual(scenario.ascending_ids);
  expect
    .soft(rows.toSorted((a, b) => compareRows(a, b, "-priority")).map((row) => row.id))
    .toEqual(scenario.descending_ids);
});
