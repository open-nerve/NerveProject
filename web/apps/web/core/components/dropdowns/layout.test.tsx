// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { EIssueLayoutTypes } from "@nerve/types";
import { LayoutDropDown } from "./layout";

const language = vi.hoisted(() => ({ current: "en" }));
vi.mock("@nerve/i18n", () => ({
  useTranslation: () => {
    const locale = language.current;
    return { t: (key: string) => `${locale}:${key}` };
  },
}));
globalThis.IS_REACT_ACT_ENVIRONMENT = true;
let host: HTMLDivElement, root: ReturnType<typeof createRoot>;
const changed = vi.fn(),
  submitted = vi.fn();
beforeEach(() => {
  language.current = "en";
  vi.useFakeTimers();
  vi.stubGlobal(
    "ResizeObserver",
    class {
      observe() {}
      unobserve() {}
      disconnect() {}
    }
  );
  vi.spyOn(Element.prototype, "getAnimations").mockReturnValue([]);
  changed.mockClear();
  submitted.mockClear();
  host = document.createElement("div");
  document.body.append(host);
  root = createRoot(host);
});
afterEach(async () => {
  await act(async () => root.unmount());
  host.remove();
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
  vi.useRealTimers();
});
async function render() {
  await act(async () =>
    root.render(
      <form
        onSubmit={(event) => {
          event.preventDefault();
          submitted();
        }}
      >
        <LayoutDropDown
          value={EIssueLayoutTypes.LIST}
          onChange={changed}
          disabledLayouts={[EIssueLayoutTypes.CALENDAR]}
        />
        <button type="button">After</button>
      </form>
    )
  );
}
async function press(element: Element, key: string) {
  await act(async () => {
    const down = new KeyboardEvent("keydown", { key, bubbles: true, cancelable: true });
    element.dispatchEvent(down);
    const up = new KeyboardEvent("keyup", { key, bubbles: true, cancelable: true });
    element.dispatchEvent(up);
    if (
      !down.defaultPrevented &&
      !up.defaultPrevented &&
      ["Enter", " "].includes(key) &&
      element instanceof HTMLButtonElement
    )
      element.click();
  });
  await act(async () => vi.advanceTimersByTimeAsync(50));
}
it("keeps translated labels current in the same mounted layout selector", async () => {
  await render();
  const trigger = host.querySelector<HTMLButtonElement>("button")!;
  expect(trigger.textContent).toContain("en:");
  language.current = "zh-CN";
  await render();
  expect(trigger.textContent).toContain("zh-CN:");
  await act(async () => trigger.click());
  expect(document.querySelector('[role="listbox"]')?.textContent).toContain("zh-CN:");
  expect(document.querySelector('[role="listbox"]')?.textContent).not.toContain("en:");
});
it("selects a layout by keyboard once, closes and restores focus without submitting", async () => {
  await render();
  const trigger = host.querySelector<HTMLButtonElement>("button")!;
  trigger.focus();
  await press(trigger, "Enter");
  const list = document.querySelector<HTMLElement>('[role="listbox"]');
  expect(list).not.toBeNull();
  expect(document.activeElement).toBe(list);
  expect(list!.textContent).not.toContain("calendar");
  await press(list!, "End");
  await press(list!, "Enter");
  expect(changed).toHaveBeenCalledExactlyOnceWith(EIssueLayoutTypes.SPREADSHEET);
  expect(document.querySelector('[role="listbox"]')).toBeNull();
  expect(document.activeElement).toBe(trigger);
  expect(submitted).not.toHaveBeenCalled();
  await press(trigger, " ");
  await press(document.querySelector('[role="listbox"]')!, "Escape");
  expect(document.activeElement).toBe(trigger);
  expect(changed).toHaveBeenCalledTimes(1);
});
