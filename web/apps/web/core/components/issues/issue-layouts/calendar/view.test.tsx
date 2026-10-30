// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import {
  autoScrollCount,
  tab,
  sourceDate,
  destinationDate,
  first,
  lists,
  writes,
  page,
  mount,
  tile,
  cards,
  drop,
} from "./fake-calendar";
import { act } from "react";
import { expect, it, vi } from "vitest";
import { json, problem } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { toasts } from "@/lib/fake-toast";
import { loadProjects } from "@/store/project/fake-projects";
import { issueDetailOf } from "@/store/issue/fake-issues";
import { HIGHLIGHT_CLASS } from "../utils";
it.each(["Enter", " "])("the calendar card menu is keyboard reachable by %s and does not open peek", async (key) => {
  await mount();
  const button = tile(sourceDate).element.querySelector<HTMLButtonElement>('[aria-haspopup="menu"]')!;
  expect(button).not.toBeNull();
  await act(async () => {
    button.focus();
    button.dispatchEvent(new KeyboardEvent("keydown", { key, bubbles: true, cancelable: true }));
    await vi.advanceTimersByTimeAsync(50);
  });
  expect(button.getAttribute("aria-expanded")).toBe("true");
  expect(tab.app.issue.issueDetail.peekIssue).toBeUndefined();
  await act(async () => {
    document
      .querySelector('[role="menu"]:not([inert] *)')!
      .dispatchEvent(new KeyboardEvent("keydown", { key: "Escape", bubbles: true, cancelable: true }));
    await vi.advanceTimersByTimeAsync(50);
  });
  expect(button.getAttribute("aria-expanded")).toBe("false");
  expect(document.activeElement).toBe(button);
});
it.each(["month", "week"] as const)(
  "real %s dates keep local days across the year boundary and intersect saved filters",
  async (layout) => {
    await mount(layout);
    expect(lists()[0]!.query).toMatchObject({
      group_by: "target_date",
      priority: "urgent",
      target_date: layout === "month" ? "2026-12-15..2027-01-02" : "2026-12-27..2027-01-02",
      limit: layout === "month" ? "4" : "30",
    });
    expect(tile(sourceDate).element.querySelector('button[aria-label="2026-12-31"]')?.textContent?.trim()).toBe("31");
    expect(cards(sourceDate)).toEqual(["issue-first"]);
    expect(cards(destinationDate)).toEqual([]);
  }
);
it("the mobile selected-day section is mounted once", async () => {
  await mount();
  expect([...document.querySelectorAll("p")].filter((node) => node.textContent === "31 December, 2026")).toHaveLength(
    1
  );
});
it("date drag has no premature highlight or move, confirms only target_date and does not repeat a same-day drop", async () => {
  await mount();
  await drop();
  await until(() => writes().length === 1, "date PATCH");
  expect(writes()[0]!.body).toEqual({ target_date: destinationDate });
  await act(async () => vi.advanceTimersByTimeAsync(250));
  expect(document.querySelector(`.${HIGHLIGHT_CLASS}`)).toBeNull();
  expect(cards(sourceDate)).toEqual(["issue-first"]);
  expect(cards(destinationDate)).toEqual([]);
  await act(async () =>
    writes()[0]!.answer(
      json(200, {
        ...issueDetailOf("first"),
        ...first,
        target_date: destinationDate,
        updated_at: "2027-01-01T01:00:00Z",
      })
    )
  );
  expect(cards(sourceDate)).toEqual([]);
  expect(cards(destinationDate)).toEqual(["issue-first"]);
  expect(tab.app.issue.projectIssues.getGroupIssueCount(sourceDate)).toBe(0);
  expect(tab.app.issue.projectIssues.getGroupIssueCount(destinationDate)).toBe(1);
  await act(async () => vi.advanceTimersByTimeAsync(250));
  expect(tile(destinationDate).element.querySelector(`.${HIGHLIGHT_CLASS}`)).not.toBeNull();
  await drop(destinationDate);
  expect(writes()).toHaveLength(1);
});
it.each([403, 422])(
  "a %s refusal retains the original day and shows its localized reason without highlight",
  async (status) => {
    await mount();
    await drop();
    await until(() => writes().length === 1, "date PATCH");
    await act(async () => writes()[0]!.answer(problem(status, status === 403 ? "forbidden" : "validation_failed")));
    await act(async () => vi.advanceTimersByTimeAsync(250));
    expect(cards(sourceDate)).toEqual(["issue-first"]);
    expect(cards(destinationDate)).toEqual([]);
    expect(toasts).toEqual([
      expect.objectContaining({ message: status === 403 ? "errors.forbidden" : "errors.validation_failed" }),
    ]);
    expect(document.querySelector(`.${HIGHLIGHT_CLASS}`)).toBeNull();
  }
);
it("a date earlier than the start date is refused locally and does not patch", async () => {
  await mount();
  await drop("2026-12-29");
  expect(writes()).toHaveLength(0);
  expect(toasts).toEqual([expect.objectContaining({ message: "issue.calendar.due_before_start" })]);
});
it.each([false, true])("changing the visible month ends late drop feedback, success=%s", async (success) => {
  await mount();
  await drop();
  await until(() => writes().length === 1, "date PATCH");
  await act(async () =>
    document.querySelector<HTMLButtonElement>('button[aria-label="issue.calendar.next_period"]')!.click()
  );
  await until(() => lists().length === 2, "January");
  await act(async () =>
    writes()[0]!.answer(
      success
        ? json(200, {
            ...issueDetailOf("first"),
            ...first,
            target_date: destinationDate,
            updated_at: "2027-01-01T01:00:00Z",
          })
        : problem(403, "forbidden")
    )
  );
  await act(async () => lists()[1]!.answer(json(200, { total: 0, groups: [] })));
  await act(async () => vi.advanceTimersByTimeAsync(250));
  expect(toasts).toHaveLength(0);
  expect(document.querySelector(`.${HIGHLIGHT_CLASS}`)).toBeNull();
});
it.each([false, true])(
  "a superseded month read cannot replace the new month or report its error, failure=%s",
  async (failed) => {
    await mount("month", false);
    await act(async () =>
      document.querySelector<HTMLButtonElement>('button[aria-label="issue.calendar.next_period"]')!.click()
    );
    await until(() => lists().length === 2, "January read");
    expect(lists()[1]!.query.target_date).toBe("2026-12-27..2027-02-06");
    await act(async () => lists()[1]!.answer(json(200, { total: 0, groups: [] })));
    await act(async () => lists()[0]!.answer(failed ? problem(503, "internal_error") : json(200, page())));
    expect(cards(sourceDate)).toEqual([]);
    expect(toasts).toHaveLength(0);
    expect(tab.app.issue.projectIssues.currentList?.scope.query.target_date).toEqual(["2026-12-27..2027-02-06"]);
  }
);

it("calendar auto-scroll registers when the loaded viewport mounts and ends with its host", async () => {
  await mount();
  expect(autoScrollCount()).toBe(1);
  await tab.render(null);
  expect(autoScrollCount()).toBe(0);
});

it("the calendar guest has no creation or add-existing entry and cannot submit a date drag", async () => {
  await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [
    { ...tab.project, member_role: 5, guest_view_all_features: true },
  ]);
  tab.nerve.calls.length = 0;
  await mount();
  expect(
    [...tile(destinationDate).element.querySelectorAll("button")].some((node) => node.textContent === "issue.add.label")
  ).toBe(false);
  await drop();
  expect(writes()).toHaveLength(0);
});
