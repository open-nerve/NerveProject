// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { DateFilterModal } from "./date-filter-modal";

vi.hoisted(() => {
  vi.useFakeTimers();
  vi.setSystemTime(new Date(2026, 9, 30, 12));
});
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
let host: HTMLDivElement, root: ReturnType<typeof createRoot>;
const selected = vi.fn(),
  closed = vi.fn();
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
  selected.mockClear();
  closed.mockClear();
});
afterEach(async () => {
  await act(async () => root.unmount());
  host.remove();
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
  vi.useRealTimers();
});
const render = () =>
  act(async () => root.render(<DateFilterModal title="Start date" handleClose={closed} isOpen onSelect={selected} />));
const button = (text: string) =>
  [...document.querySelectorAll<HTMLButtonElement>("button")].find((node) => node.textContent === text)!;
const day = (date: string) => document.querySelector<HTMLButtonElement>(`[data-day="${date}"] button`)!;
async function mode(value: "before" | "after") {
  await act(async () => {
    button("date_filter.range").click();
    await vi.advanceTimersByTimeAsync(20);
  });
  const option = [...document.querySelectorAll<HTMLElement>('[role="option"]')].find(
    (node) => node.textContent === `date_filter.${value}`
  )!;
  expect(option).toBeDefined();
  await act(async () => {
    option.click();
    await vi.advanceTimersByTimeAsync(20);
  });
}
async function december() {
  const months = document.querySelector<HTMLSelectElement>("select.rdp-months_dropdown")!;
  expect(months).not.toBeNull();
  await act(async () => {
    months.value = "11";
    months.dispatchEvent(new Event("change", { bubbles: true }));
  });
}
it.each(["before", "after"] as const)("single %s can select beyond the hidden range end", async (value) => {
  await render();
  await mode(value);
  await december();
  expect(day("2026-12-15")).not.toBeNull();
  expect(day("2026-12-15").disabled).toBe(false);
  await act(async () => day("2026-12-15").click());
  await act(async () => button("date_filter.apply").click());
  expect(selected).toHaveBeenCalledExactlyOnceWith([`2026-12-15;${value}`]);
  expect(closed).toHaveBeenCalledOnce();
});
it("range mode still prevents a start later than its visible end", async () => {
  await render();
  await december();
  expect(day("2026-12-15").disabled).toBe(true);
  await act(async () => day("2026-12-15").click());
  await act(async () => button("date_filter.apply").click());
  expect(selected).toHaveBeenCalledExactlyOnceWith(["2026-10-30;after", "2026-11-30;before"]);
});
it("a newly opened form uses its opening day after the application has stayed open overnight", async () => {
  vi.setSystemTime(new Date(2026, 10, 2, 12));
  await render();
  await act(async () => button("date_filter.apply").click());
  expect(selected).toHaveBeenCalledExactlyOnceWith(["2026-11-02;after", "2026-12-02;before"]);
});
