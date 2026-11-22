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
import { NotificationViewMenu } from "./view-menu";

// The view menu of the notifications page (P4 spec 2.5, 2.6, R16, R25, re-93 I3): a menu (PopoverMenu) whose button is
// an icon and its name for a screen reader, with no button inside it; its items show the unread alone, the archived
// alone, the snoozed alone, each a button that says whether it is pressed, reached by Tab, toggled once by Enter and
// once by Space, never submitting the form around it; the archived and the snoozed exclude each other, as nerve takes
// both as equalities and the page never asks for both. Escape closes it and gives the focus back to its button. In
// both tabs. @nerve/i18n's text is its key.

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

const render = () => renderHeadPart(root, <NotificationViewMenu />, submitted);
const button = () => menuButton(host, "common.display");
const UNREAD = "notification.options.show_unread";
const ARCHIVED = "notification.options.show_archived";
const SNOOZED = "notification.options.show_snoozed";
/** The toggles of the page's filters. */
const toggles = () => {
  const { unread, archived, snoozed } = headState.current.filters;
  return { unread, archived, snoozed };
};

describe("NotificationViewMenu", () => {
  it("is a button named for a screen reader, with no button in it, opening to the three views, none pressed", async () => {
    await render();
    expect(button()?.querySelector(".sr-only")?.textContent).toBe("common.display");
    expect(button()?.querySelector("svg")?.getAttribute("aria-hidden")).toBe("true");
    await press(button()!, "Enter");
    expect(menuItems(host).map(({ label, pressed }) => [label, pressed])).toEqual([
      [UNREAD, "false"],
      [ARCHIVED, "false"],
      [SNOOZED, "false"],
    ]);
    expect(host.querySelector("button button")).toBeNull();
    for (const { item } of menuItems(host)) {
      expect(item.type).toBe("button");
      expect(item.tabIndex).toBe(0);
      expect(item.getAttribute("role")).toBeNull();
    }
  });

  it.each([
    ["Enter", "Enter"],
    ["Space", " "],
  ])("toggles a view once by %s, the archived and the snoozed never both, submitting no form", async (_, key) => {
    await render();
    await press(button()!, "Enter");
    await press(menuItem(host, UNREAD)!, key);
    expect(toggles()).toEqual({ unread: true, archived: false, snoozed: false });
    await press(menuItem(host, ARCHIVED)!, key);
    expect(toggles()).toEqual({ unread: true, archived: true, snoozed: false });
    await press(menuItem(host, SNOOZED)!, key);
    expect(toggles()).toEqual({ unread: true, archived: false, snoozed: true });
    expect(menuItems(host).map(({ pressed }) => pressed)).toEqual(["true", "false", "true"]);
    await press(menuItem(host, SNOOZED)!, key);
    await press(menuItem(host, UNREAD)!, key);
    expect(toggles()).toEqual({ unread: false, archived: false, snoozed: false });
    expect(submitted).not.toHaveBeenCalled();
  });

  it("closes on Escape, giving the focus back to its button", async () => {
    await render();
    await press(button()!, " ");
    await press(menuItem(host, ARCHIVED)!, "Escape");
    expect(button()?.getAttribute("aria-expanded")).toBe("false");
    expect(document.activeElement).toBe(button());
  });

  it("is there in the mentions tab too", async () => {
    await render();
    await changeHeadState((state) => state.setTab("mentions"));
    await press(button()!, "Enter");
    await press(menuItem(host, ARCHIVED)!, "Enter");
    expect(toggles()).toEqual({ unread: false, archived: true, snoozed: false });
  });
});
