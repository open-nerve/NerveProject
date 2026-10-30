// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { FILTER_FIELD_TYPE, FILTER_NODE_TYPE, COMPARISON_OPERATOR } from "@nerve/types";
import type { TDateRangeFilterFieldConfig } from "@nerve/types";
import { FilterValueInput } from "../root";

vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@/hooks/store/user", () => ({ useUserProfile: () => ({ data: { start_of_the_week: 1 } }) }));
let host: HTMLDivElement, root: ReturnType<typeof createRoot>;
const changed = vi.fn();
const config: TDateRangeFilterFieldConfig<string> = { type: FILTER_FIELD_TYPE.DATE_RANGE };
beforeEach(() => {
  globalThis.IS_REACT_ACT_ENVIRONMENT = true;
  vi.useFakeTimers();
  vi.setSystemTime(new Date(2026, 9, 30, 12));
  vi.stubGlobal(
    "ResizeObserver",
    class {
      observe() {}
      unobserve() {}
      disconnect() {}
    }
  );
  vi.spyOn(Element.prototype, "getAnimations").mockReturnValue([]);
  host = document.createElement("div");
  document.body.append(host);
  root = createRoot(host);
  changed.mockClear();
});
afterEach(async () => {
  await act(async () => root.unmount());
  host.remove();
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
  vi.useRealTimers();
});
const render = (value: string[] = [], id = "first", field = config) =>
  act(async () =>
    root.render(
      <FilterValueInput
        condition={{
          id,
          type: FILTER_NODE_TYPE.CONDITION,
          property: "target_date",
          operator: COMPARISON_OPERATOR.RANGE,
          value,
        }}
        filterFieldConfig={field}
        onChange={changed}
      />
    )
  );
const button = () => host.querySelector<HTMLButtonElement>("button")!;
const day = (date: string) => document.querySelector<HTMLButtonElement>(`[data-day="${date}"] button`)!;
const clickDay = (date: string) => act(async () => day(date).click());
function expectError(key: string) {
  expect(button().getAttribute("aria-invalid")).toBe("true");
  const errorId = button().getAttribute("aria-describedby");
  expect(errorId).toBeTruthy();
  expect(document.getElementById(errorId!)?.textContent).toBe(key);
}

it("keeps the first day as a draft until a second local day completes the range", async () => {
  await render();
  await clickDay("2026-10-12");
  expect(changed).not.toHaveBeenCalled();
  expectError("common.filter_dates.incomplete");
  await render();
  await clickDay("2026-10-19");
  expect(changed.mock.calls).toEqual([[["2026-10-12", "2026-10-19"]]]);
  expect(button().getAttribute("aria-invalid")).not.toBe("true");
});

it("describes a reversed draft without clearing the saved condition, then accepts a valid endpoint", async () => {
  await render(["2026-10-01", "2026-10-31"]);
  await act(async () => button().click());
  await clickDay("2026-10-20");
  await clickDay("2026-10-15");
  expectError("common.filter_dates.reversed");
  expect(changed).not.toHaveBeenCalled();
  await clickDay("2026-10-25");
  expect(changed.mock.calls).toEqual([[["2026-10-20", "2026-10-25"]]]);
});

it.each([
  ["2026-10-01", ""],
  ["", "2026-10-31"],
])("preserves a loaded open range %s / %s until an explicit edit", async (from, to) => {
  await render([from, to]);
  expect(button().getAttribute("aria-invalid")).not.toBe("true");
  expect(changed).not.toHaveBeenCalled();
  await act(async () => button().click());
  await clickDay("2026-10-15");
  expectError("common.filter_dates.incomplete");
  expect(changed).not.toHaveBeenCalled();
  await clickDay("2026-10-20");
  expect(changed.mock.calls).toEqual([[["2026-10-15", "2026-10-20"]]]);
});

it("ends a condition's unfinished range when its identity changes and clears only explicitly", async () => {
  await render();
  await clickDay("2026-10-12");
  await render([], "second");
  await clickDay("2026-10-20");
  expect(changed).not.toHaveBeenCalled();
  await clickDay("2026-10-20");
  expect(changed.mock.calls).toEqual([[["2026-10-20", "2026-10-20"]]]);
  const clear = host.querySelector<HTMLButtonElement>('button[aria-label^="common.clear_date"]');
  expect(clear).not.toBeNull();
  await act(async () => clear!.click());
  expect(changed.mock.calls.at(-1)).toEqual([[]]);
});

it("does not revive a draft when an external confirmed value changes and later returns", async () => {
  await render();
  await clickDay("2026-10-12");
  await render(["2026-10-01", "2026-10-31"]);
  await render();
  await clickDay("2026-10-20");
  expect(changed).not.toHaveBeenCalled();
  await clickDay("2026-10-25");
  expect(changed.mock.calls).toEqual([[["2026-10-20", "2026-10-25"]]]);
});

it.each(["2024-02-29", "2026-12-31", "2027-01-01"])(
  "keeps saved %s on the local calendar day in either timezone",
  async (date) => {
    await render([date, date]);
    await act(async () => button().click());
    const selected = document.querySelector(`[data-day="${date}"]`);
    expect(selected?.getAttribute("aria-selected")).toBe("true");
    await clickDay(date);
    expect(changed).not.toHaveBeenCalled();
    await clickDay(date);
    expect(changed.mock.calls).toEqual([[[date, date]]]);
  }
);

it("describes an out-of-bounds saved range and disables unavailable calendar days", async () => {
  await render(["2026-10-01", "2026-10-31"], "bounded", {
    ...config,
    min: new Date(2026, 9, 10),
    max: new Date(2026, 9, 20),
  });
  expectError("common.filter_dates.out_of_bounds");
  await act(async () => button().click());
  expect(day("2026-10-09").disabled).toBe(true);
  expect(day("2026-10-21").disabled).toBe(true);
  expect(changed).not.toHaveBeenCalled();
});
