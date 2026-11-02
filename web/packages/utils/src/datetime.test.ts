/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { expect, it } from "vitest";
import { getDate, renderFormattedPayloadDate, parseDateFilter } from "./datetime";

it.each(["0001-01-01", "0099-12-31", "0100-01-01", "2024-02-29", "9999-12-31"])(
  "keeps the calendar year and local day of the API date %s",
  (text) => {
    const date = getDate(text)!;
    const [year, month, day] = text.split("-").map(Number);
    expect([date.getFullYear(), date.getMonth() + 1, date.getDate(), date.getHours()]).toEqual([year, month, day, 0]);
    expect(renderFormattedPayloadDate(date)).toBe(text);
  }
);

it("parses an absolute sub-item filter as a local calendar day", () => {
  const { date, type } = parseDateFilter("2026-10-08;after");
  expect([date.getFullYear(), date.getMonth() + 1, date.getDate(), date.getHours()]).toEqual([2026, 10, 8, 0]);
  expect(type).toBe("after");
});

it("preserves an existing Date instance supplied by the calendar", () => {
  const date = new Date(2026, 9, 8, 12, 30);
  expect(getDate(date)).toBe(date);
});
