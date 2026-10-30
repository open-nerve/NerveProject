// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { tab, container, first, a, b, c, mount, payload, send, visible } from "./fake-kanban";
import { act } from "react";
import { expect, it, vi } from "vitest";
import { json, problem } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { issueDetailOf, requestedIssue } from "@/store/issue/fake-issues";
import { getDestinationFromDropPayload, getSourceFromDropPayload, handleGroupDragDrop } from "../utils";

it.each([
  [false, "b", ["a", "first", "b", "c"], 2500],
  [true, "b", ["a", "b", "first", "c"], 3500],
  [false, undefined, ["a", "b", "c", "first"], 69535],
] as const)(
  "real card produces below=%s target=%s, then moves once after confirmation",
  async (below, target, expected, order) => {
    await mount();
    const data = payload("first", "todo", target, "done", below);
    expect(getSourceFromDropPayload(data as never)).toMatchObject({ groupId: "todo", id: "first" });
    expect(getDestinationFromDropPayload(data as never)).toMatchObject({
      groupId: "done",
      id: target,
      canAddIssueBelow: below,
    });
    const call = await send("first", "todo", target, below);
    expect(call.body).toEqual({ state_id: "done", sort_order: order });
    expect(visible("todo")).toEqual(["first"]);
    expect(visible("done")).toEqual(["a", "b", "c"]);
    expect(tab.app.issue.projectIssues.getGroupIssueCount("todo")).toBe(1);
    expect(tab.app.issue.projectIssues.getGroupIssueCount("done")).toBe(3);
    await act(async () => vi.advanceTimersByTimeAsync(250));
    expect(container.querySelector(".highlight")).toBeNull();
    await act(async () =>
      call.answer(
        json(200, {
          ...issueDetailOf("first"),
          updated_at: "2026-10-29T00:00:00Z",
          state_id: "done",
          sort_order: order,
        })
      )
    );
    expect(visible("todo")).toEqual([]);
    expect(visible("done")).toEqual(expected);
    expect(tab.app.issue.projectIssues.getGroupIssueCount("todo")).toBe(0);
    expect(tab.app.issue.projectIssues.getGroupIssueCount("done")).toBe(4);
    expect(tab.nerve.calls.filter((entry) => entry.method === "PATCH")).toHaveLength(1);
  }
);
it.each([false, true])("same-column removes source before resolving neighbors, below=%s", async (below) => {
  await mount();
  const call = await send("a", "done", "b", below);
  expect(call.body).toEqual({ sort_order: below ? 3500 : -62535 });
  await act(async () =>
    call.answer(
      json(200, {
        ...issueDetailOf("a"),
        updated_at: "2026-10-29T00:00:00Z",
        state_id: "done",
        sort_order: below ? 3500 : -62535,
      })
    )
  );
  expect(visible("done")).toEqual(below ? ["b", "a", "c"] : ["a", "b", "c"]);
});
it("empty destination uses the native initial sort order", async () => {
  await mount([]);
  const call = await send("first", "todo", undefined);
  expect(call.body).toEqual({ state_id: "done", sort_order: 65535 });
  await act(async () =>
    call.answer(
      json(200, { ...issueDetailOf("first"), updated_at: "2026-10-29T00:00:00Z", state_id: "done", sort_order: 65535 })
    )
  );
  expect(visible("done")).toEqual(["first"]);
});
it.each([403, 422])("refused %s drop leaves cards, canonical and counts unchanged", async (status) => {
  await mount();
  const call = await send("first", "todo", "b", true);
  await act(async () => call.answer(problem(status, status === 403 ? "forbidden" : "validation_failed")));
  expect(visible("todo")).toEqual(["first"]);
  expect(visible("done")).toEqual(["a", "b", "c"]);
  expect(tab.app.issue.issues.getIssueById("first")).toMatchObject({ state_id: "todo", sort_order: 1000 });
  expect(tab.app.issue.projectIssues.getGroupIssueCount("done")).toBe(3);
});
it("an initially missing drop target cannot silently become the column tail", async () => {
  const update = vi.fn();
  await handleGroupDragDrop(
    { id: "first", columnId: "todo", groupId: "todo" },
    { id: "gone", columnId: "done", groupId: "done" },
    () => first,
    () => ["a", "b"],
    update,
    "state",
    undefined
  );
  expect(update).not.toHaveBeenCalled();
});

it("an older read cannot reset a confirmed tail; refused refresh retries only the read", async () => {
  await mount();
  const binding = tab.app.issue.projectIssues.currentList!;
  const stale = await requestedIssue(tab.nerve, () => binding.fetchFirst(), ["GET", "/api/v0/projects/A/issues"]);
  const call = await send("first", "todo", undefined);
  await act(async () =>
    call.answer(
      json(200, { ...issueDetailOf("first"), updated_at: "2026-10-29T00:00:00Z", state_id: "done", sort_order: 69535 })
    )
  );
  await act(async () => {
    stale.call.answer(
      json(200, {
        total: 4,
        groups: [
          { key: "todo", total: 1, data: [first], next_cursor: null },
          { key: "done", total: 3, data: [a, b, c], next_cursor: null },
        ],
      })
    );
    await stale.promise;
  });
  expect(visible("todo")).toEqual([]);
  expect(visible("done")).toEqual(["a", "b", "c", "first"]);
  expect(tab.app.issue.projectIssues.getGroupIssueCount("done")).toBe(4);
  const reload = await requestedIssue(tab.nerve, () => binding.fetchFirst(), ["GET", "/api/v0/projects/A/issues"]);
  await act(async () => {
    reload.call.answer(problem(503, "server_busy"));
    await expect(reload.promise).rejects.toBeDefined();
  });
  expect(visible("done")).toEqual(["a", "b", "c", "first"]);
  const retry = [...container.querySelectorAll("button")].find((button) => button.textContent === "common.retry")!;
  expect(retry).toBeDefined();
  const count = tab.nerve.calls.length;
  await act(async () => {
    retry.click();
    await until(() => tab.nerve.calls.length > count, "read retry");
  });
  expect(tab.nerve.calls.at(-1)!.method).toBe("GET");
  expect(tab.nerve.calls.filter((entry) => entry.method === "PATCH")).toHaveLength(1);
  await act(async () =>
    tab.nerve.calls.at(-1)!.answer(
      json(200, {
        total: 4,
        groups: [
          { key: "todo", total: 0, data: [], next_cursor: null },
          {
            key: "done",
            total: 4,
            data: [a, b, c, { ...first, state_id: "done", sort_order: 69535, updated_at: "2026-10-29T00:00:00Z" }],
            next_cursor: null,
          },
        ],
      })
    )
  );
  expect(visible("done")).toEqual(["a", "b", "c", "first"]);
});
