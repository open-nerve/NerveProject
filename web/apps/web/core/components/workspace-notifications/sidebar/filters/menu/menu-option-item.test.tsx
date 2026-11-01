// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { ENotificationFilterType } from "@nerve/constants";
import { NotificationFilterOptionItem } from "./menu-option-item";
const state = vi.hoisted(() => ({ filters: { type: {} }, updateFilters: vi.fn() }));
vi.mock("@/hooks/store/notifications", () => ({ useWorkspaceNotifications: () => state }));
globalThis.IS_REACT_ACT_ENVIRONMENT = true;
let host: HTMLDivElement, root: ReturnType<typeof createRoot>;
beforeEach(() => {
  state.filters.type = {};
  state.updateFilters.mockClear();
  host = document.createElement("div");
  document.body.append(host);
  root = createRoot(host);
});
afterEach(async () => {
  await act(async () => root.unmount());
  host.remove();
});
it.each(["Enter", " "])("toggles a notification filter by %s once without submitting its parent form", async (key) => {
  const submitted = vi.fn();
  await act(async () =>
    root.render(
      <form onSubmit={submitted}>
        <NotificationFilterOptionItem label="Assigned" value={ENotificationFilterType.ASSIGNED} />
      </form>
    )
  );
  const option = host.querySelector<HTMLButtonElement>("button")!;
  expect(option).not.toBeNull();
  option.focus();
  expect(document.activeElement).toBe(option);
  await act(async () => {
    const down = new KeyboardEvent("keydown", { key, bubbles: true, cancelable: true });
    const up = new KeyboardEvent("keyup", { key, bubbles: true, cancelable: true });
    option.dispatchEvent(down);
    option.dispatchEvent(up);
    // jsdom does not synthesize a browser's native activation; only a real button has that default action.
    if (!down.defaultPrevented && !up.defaultPrevented && option instanceof HTMLButtonElement) option.click();
  });
  expect(state.updateFilters).toHaveBeenCalledExactlyOnceWith("type", { [ENotificationFilterType.ASSIGNED]: true });
  expect(submitted).not.toHaveBeenCalled();
});
