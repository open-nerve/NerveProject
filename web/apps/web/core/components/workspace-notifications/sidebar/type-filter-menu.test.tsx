// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import {
  changeHeadState,
  headState,
  menuButton,
  menuItem,
  menuItems,
  press,
  renderHeadPart,
} from "./fake-header-menus";
import { NotificationTypeFilterMenu } from "./type-filter-menu";

// The type filter of the notifications page's all tab (P4 spec 2.6, R10, R16, re-93 I3): a menu (PopoverMenu) whose
// button is an icon and its name for a screen reader, with no button inside it (W7); its items each a button that says
// whether it is pressed, one for each sender, reached by Tab (the menu has no arrow keys), toggled once by Enter and
// once by Space, never submitting the form around it (the property of the old menu-option-item.test.tsx); Escape closes
// it and gives the focus back to its button. Not there in the mentions tab, where nerve takes no type; the all tab's
// choice stays. @nerve/i18n's text is its key.

vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));

let host: HTMLDivElement, root: Root;
const submitted = vi.fn();
globalThis.IS_REACT_ACT_ENVIRONMENT = true;
beforeEach(() => {
  vi.useFakeTimers();
  vi.spyOn(Element.prototype, "getAnimations").mockReturnValue([]);
  submitted.mockClear();
  host = document.createElement("div");
  document.body.append(host);
  root = createRoot(host);
});
afterEach(async () => {
  await act(async () => root.unmount());
  host.remove();
  vi.restoreAllMocks();
  vi.useRealTimers();
});

const render = () => renderHeadPart(root, <NotificationTypeFilterMenu />, submitted);
const button = () => menuButton(host, "notification.options.filters");
const ASSIGNED = "notification.filter.assigned";
const CREATED = "notification.filter.created";
const SUBSCRIBED = "notification.filter.subscribed";

describe("NotificationTypeFilterMenu", () => {
  it("is a button named for a screen reader, with no button in it, opening to a pressed-or-not item for each sender", async () => {
    await render();
    expect(button()?.querySelector(".sr-only")?.textContent).toBe("notification.options.filters");
    expect(button()?.querySelector("svg")?.getAttribute("aria-hidden")).toBe("true");
    expect(host.querySelector("button button")).toBeNull();
    await press(button()!, "Enter");
    expect(button()?.getAttribute("aria-expanded")).toBe("true");
    expect(menuItems(host).map(({ label, pressed }) => [label, pressed])).toEqual([
      [ASSIGNED, "false"],
      [CREATED, "false"],
      [SUBSCRIBED, "false"],
    ]);
    expect(host.querySelector("button button")).toBeNull();
  });

  it("has items Tab reaches, each a plain button: no menu role, whose arrow keys it lacks", async () => {
    await render();
    await press(button()!, "Enter");
    for (const { item } of menuItems(host)) {
      expect(item.type).toBe("button");
      expect(item.tabIndex).toBe(0);
      expect(item.getAttribute("role")).toBeNull();
    }
  });

  it.each([
    ["Enter", "Enter"],
    ["Space", " "],
  ])("toggles a sender once by %s, submitting no form", async (_, key) => {
    await render();
    await press(button()!, " ");
    await press(menuItem(host, CREATED)!, key);
    expect(headState.current.filters.types).toEqual(["created"]);
    expect(menuItem(host, CREATED)?.getAttribute("aria-pressed")).toBe("true");
    await press(menuItem(host, ASSIGNED)!, key);
    expect(headState.current.filters.types).toEqual(["created", "assigned"]);
    await press(menuItem(host, CREATED)!, key);
    expect(headState.current.filters.types).toEqual(["assigned"]);
    expect(menuItem(host, CREATED)?.getAttribute("aria-pressed")).toBe("false");
    expect(submitted).not.toHaveBeenCalled();
  });

  it("closes on Escape, giving the focus back to its button", async () => {
    await render();
    await press(button()!, "Enter");
    await press(menuItem(host, SUBSCRIBED)!, "Escape");
    expect(button()?.getAttribute("aria-expanded")).toBe("false");
    expect(menuItems(host)).toEqual([]);
    expect(document.activeElement).toBe(button());
  });

  it("is not there in the mentions tab, and the all tab's choice is still there back in the all tab", async () => {
    await render();
    await press(button()!, "Enter");
    await press(menuItem(host, ASSIGNED)!, "Enter");
    await changeHeadState((state) => state.setTab("mentions"));
    expect(button()).toBeUndefined();
    expect(menuItems(host)).toEqual([]);
    await changeHeadState((state) => state.setTab("all"));
    await press(button()!, "Enter");
    expect(menuItem(host, ASSIGNED)?.getAttribute("aria-pressed")).toBe("true");
  });
});
