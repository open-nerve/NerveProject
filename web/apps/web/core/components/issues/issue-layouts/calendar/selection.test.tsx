// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { tab, destinationDate, mount, tile, writes, cards } from "./fake-calendar";
import { act } from "react";
import { expect, it, vi } from "vitest";
import { issueOf, issueDetailOf } from "@/store/issue/fake-issues";
import { json, problem } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { toasts } from "@/lib/fake-toast";
import { searchSelection, selectionButton } from "@/components/core/modals/fake-issue-selection";

const one = { ...issueOf("one"), priority: "urgent" as const };
const two = { ...issueOf("two"), priority: "urgent" as const, sequence_id: 2 };
async function open(day = destinationDate) {
  const trigger = [...tile(day).element.querySelectorAll("button")].find(
    (node) => node.textContent === "issue.add.label"
  )!;
  vi.spyOn(trigger, "getBoundingClientRect").mockReturnValue(new DOMRect(1, 1, 150, 30));
  await act(async () => trigger.click());
  await act(async () =>
    [...document.querySelectorAll<HTMLButtonElement>('[role="menuitem"]')]
      .find((node) => node.textContent === "issue.add.existing")!
      .click()
  );
  await until(() => tab.searches().length > 0, "calendar search");
}
const send = () => act(async () => selectionButton("issue.select.add_selected").click());
const confirm = (id: string) =>
  json(200, {
    ...issueDetailOf(id),
    priority: "urgent",
    target_date: destinationDate,
    updated_at: "2027-01-01T01:00:00Z",
  });
it("the real calendar selector filters dates, keeps none with typed search, and retries only the failed target_date write", async () => {
  await mount();
  await open();
  expect(tab.searches()[0]!.query).toEqual({ target_date: "none", project_id: "A" });
  await act(async () =>
    tab.searches()[0]!.answer(
      json(200, {
        data: [
          one,
          two,
          { ...issueOf("too-late"), start_date: "2027-01-02" },
          { ...issueOf("already-dated"), target_date: "2027-01-01" },
        ],
      })
    )
  );
  expect([...document.querySelectorAll('[role="option"]')].map((node) => node.textContent)).toEqual([
    "A-1one",
    "A-2two",
  ]);
  await searchSelection("one");
  await act(async () => vi.advanceTimersByTimeAsync(500));
  expect(tab.searches()[1]!.query).toEqual({ target_date: "none", project_id: "A", q: "one" });
  await act(async () => tab.searches()[1]!.answer(json(200, { data: [one] })));
  await act(async () => selectionButton("issue.select.select_all").click());
  await searchSelection("");
  await until(() => tab.searches().length === 3, "cleared search");
  await act(async () => tab.searches()[2]!.answer(json(200, { data: [one, two] })));
  await act(async () => selectionButton("issue.select.select_all").click());
  await send();
  expect(writes()).toHaveLength(2);
  expect(writes().map((call) => call.body)).toEqual([
    { target_date: destinationDate },
    { target_date: destinationDate },
  ]);
  await act(async () => writes()[0]!.answer(confirm("one")));
  expect(selectionButton("common.adding").disabled).toBe(true);
  await act(async () => writes()[1]!.answer(problem(422, "validation_failed")));
  expect(document.querySelector('[aria-label="common.remove one"]')).toBeNull();
  expect(document.querySelector('[aria-label="common.remove two"]')).not.toBeNull();
  expect(cards(destinationDate)).toContain("issue-one");
  await send();
  expect(writes()).toHaveLength(3);
  expect(writes()[2]!.path).toBe("/api/v0/issues/two");
  await act(async () => writes()[2]!.answer(confirm("two")));
  await act(async () => vi.advanceTimersByTimeAsync(300));
  expect(document.querySelector('[role="dialog"]')).toBeNull();
  expect(cards(destinationDate)).toEqual(["issue-two", "issue-one"]);
  expect(toasts).toEqual([expect.objectContaining({ message: "errors.validation_failed" })]);
});
it("closing a pending day selector and opening another day leaves the new selector alive after the old answer", async () => {
  await mount();
  await open();
  await act(async () => tab.searches()[0]!.answer(json(200, { data: [one] })));
  await act(async () => selectionButton("issue.select.select_all").click());
  await send();
  await act(async () => selectionButton("common.cancel").click());
  await act(async () => vi.advanceTimersByTimeAsync(300));
  await open("2027-01-02");
  await until(() => tab.searches().length === 2, "new day search");
  await act(async () => tab.searches()[1]!.answer(json(200, { data: [two] })));
  await act(async () => writes()[0]!.answer(problem(403, "forbidden")));
  expect(document.querySelector('[role="dialog"]')).not.toBeNull();
  expect(toasts).toHaveLength(0);
  await act(async () => selectionButton("issue.select.select_all").click());
  await send();
  expect(writes()[1]!.body).toEqual({ target_date: "2027-01-02" });
});
