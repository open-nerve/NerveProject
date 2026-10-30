// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { tab, container, setGrouping, renderBoard } from "./fake-kanban";
import { act } from "react";
import { expect, it } from "vitest";
import { answered, json, problem } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { listIssue } from "@/store/issue/fake-issues";
import { labelOf } from "@/store/project/fake-projects";

it("swimlane uses unique subgroup total, independent cell cursors and one canonical entity", async () => {
  setGrouping("labels", "priority");
  await answered(
    tab.nerve,
    () => tab.app.label.fetchProjectLabels("A"),
    ["GET", "/api/v0/projects/A/labels"],
    { data: [labelOf(tab.project, "Label A", 1, { id: "L1" }), labelOf(tab.project, "Label B", 2, { id: "L2" })] },
    "labels"
  );
  const shared = listIssue("shared", 1000, { label_ids: ["L1", "L2"], priority: "high" });
  await answered(
    tab.nerve,
    () =>
      tab.app.issue.projectIssues.fetchIssues("ws", "A", "init-loader", {
        query: { order_by: "sort_order", group_by: "label_id", sub_group_by: "priority" },
        limit: 1,
      }),
    ["GET", "/api/v0/projects/A/issues"],
    {
      total: 4,
      groups: ["L1", "L2"].map((key) => ({
        key,
        total: 3,
        subgroups: [{ key: "high", total: 3, data: [shared], next_cursor: `next:${key}` }],
      })),
      subgroup_totals: [{ key: "high", total: 4 }],
    },
    "swimlanes"
  );
  await renderBoard();
  const lane = [...container.querySelectorAll("button")].find((button) => button.textContent === "High4");
  expect(lane, "accessible lane header reports unique count, not 3+3 cells").toBeDefined();
  expect(container.querySelectorAll('a[id^="issue_shared_"]')).toHaveLength(2);
  expect(Object.keys(tab.app.issue.issues.issuesMap)).toEqual(["shared"]);
  const cell = (id: string) => container.querySelector(`[id="${id}__high"]`)!;
  const more = (id: string) =>
    [...cell(id).querySelectorAll("button")].find((el) => el.textContent === "common.load_more ↓")!;
  tab.nerve.calls.length = 0;
  await act(async () => {
    more("L1").click();
    more("L1").click();
    more("L2").click();
    await until(() => tab.nerve.calls.length >= 2, "two cells");
  });
  expect(tab.nerve.calls.map((entry) => entry.query)).toEqual([
    expect.objectContaining({ group_key: "L1", sub_group_key: "high", cursor: "next:L1" }),
    expect.objectContaining({ group_key: "L2", sub_group_key: "high", cursor: "next:L2" }),
  ]);
  expect(more("L1").disabled).toBe(true);
  expect(more("L2").disabled).toBe(true);
  await act(async () => {
    tab.nerve.calls[0]!.answer(problem(503, "server_busy"));
    tab.nerve.calls[1]!.answer(
      json(200, {
        total: 3,
        data: [shared, listIssue("second", 2000, { label_ids: ["L2"], priority: "high" })],
        next_cursor: null,
      })
    );
  });
  expect(more("L1").disabled).toBe(false);
  expect(more("L2")).toBeUndefined();
  await act(async () => {
    more("L1").click();
    await until(() => tab.nerve.calls.length === 3, "cell retry");
  });
  expect(tab.nerve.calls[2]!.query).toEqual(tab.nerve.calls[0]!.query);
  await act(async () => tab.nerve.calls[2]!.answer(json(200, { total: 3, data: [shared], next_cursor: null })));
  expect(more("L1")).toBeUndefined();
  expect(lane?.textContent).toBe("High4");
  expect(tab.app.issue.issues.getIssueById("shared")!.label_ids).toEqual(["L1", "L2"]);
});
