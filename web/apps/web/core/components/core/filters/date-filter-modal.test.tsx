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
// Exercise the form's validation boundary with a reversed selection that the calendar normally disables.
vi.mock("@nerve/propel/calendar", () => ({
  Calendar: ({ onSelect }: { onSelect: (date: Date) => void }) => (
    <button type="button" onClick={() => onSelect(new Date(2026, 8, 1))}>
      September 1
    </button>
  ),
}));
let host: HTMLDivElement, root: ReturnType<typeof createRoot>;
const selected = vi.fn(),
  closed = vi.fn();
beforeEach(async () => {
  globalThis.IS_REACT_ACT_ENVIRONMENT = true;
  vi.useFakeTimers();
  vi.spyOn(Element.prototype, "getAnimations").mockReturnValue([]);
  host = document.createElement("div");
  document.body.append(host);
  root = createRoot(host);
  selected.mockClear();
  closed.mockClear();
  await act(async () =>
    root.render(<DateFilterModal title="Start date" handleClose={closed} isOpen onSelect={selected} />)
  );
});
afterEach(async () => {
  await act(async () => root.unmount());
  host.remove();
  vi.restoreAllMocks();
  vi.useRealTimers();
});
const apply = () =>
  [...document.querySelectorAll<HTMLButtonElement>("button")].find((item) => item.textContent === "date_filter.apply")!;
it("compares the actual second date and refuses a reversed range", async () => {
  await act(async () =>
    [...document.querySelectorAll<HTMLButtonElement>("button")]
      .filter((item) => item.textContent === "September 1")[1]
      .click()
  );
  expect(apply().disabled).toBe(true);
  await act(async () => apply().click());
  expect(selected).not.toHaveBeenCalled();
  expect(closed).not.toHaveBeenCalled();
});
it("keeps the legacy semicolon callback for a valid range", async () => {
  await act(async () => apply().click());
  expect(selected).toHaveBeenCalledTimes(1);
  expect(selected.mock.calls[0][0]).toEqual([
    expect.stringMatching(/^\d{4}-\d{2}-\d{2};after$/),
    expect.stringMatching(/^\d{4}-\d{2}-\d{2};before$/),
  ]);
  expect(closed).toHaveBeenCalledTimes(1);
});
