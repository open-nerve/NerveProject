/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

// The head of the notifications page's sidebar, its buttons, menus and applied filters, in their tests (P4 spec 2.5,
// 2.6): rendered at acme's address under the page's state (read as it renders), inside a form, each opened and pressed
// by the keys a browser gives a button. A test file uses fake timers (the menus' transitions) and stubs
// Element.getAnimations, which jsdom lacks.

import { act } from "react";
import type { ReactNode } from "react";
import type { Root } from "react-dom/client";
import { MemoryRouter, Route, Routes } from "react-router";
import { vi } from "vitest";
import { NotificationPageStateProvider, useNotificationPageState } from "../page-state";

/** The page's state as the page last rendered it. */
export const headState = { current: undefined as unknown as ReturnType<typeof useNotificationPageState> };

function Read() {
  headState.current = useNotificationPageState();
  return null;
}

/** Renders a part of the page's head at acme's notifications page, under the page's state, in a form whose submit is given. */
export const renderHeadPart = (root: Root, part: ReactNode, submitted: () => void = () => {}) =>
  act(async () =>
    root.render(
      <MemoryRouter initialEntries={["/acme/notifications"]}>
        <Routes>
          <Route
            path="/:workspaceSlug/notifications"
            element={
              <NotificationPageStateProvider>
                <Read />
                <form
                  onSubmit={(event) => {
                    event.preventDefault();
                    submitted();
                  }}
                >
                  {part}
                </form>
              </NotificationPageStateProvider>
            }
          />
        </Routes>
      </MemoryRouter>
    )
  );

/** The page changes its state as does says, and renders what follows. */
export const changeHeadState = (does: (state: ReturnType<typeof useNotificationPageState>) => void) =>
  act(async () => does(headState.current));

/** Time passes for the menus' transitions to end. */
const tick = () => act(async () => vi.advanceTimersByTimeAsync(200));

/**
 * The key pressed on the element, focused, as a browser gives it: keydown and keyup, and for Enter and Space the
 * element's own activation, a click, which a native button alone has and which jsdom does not make (once, unless a
 * handler prevented the key's default).
 */
export async function press(element: HTMLElement, key: string) {
  await act(async () => {
    element.focus();
    const down = new KeyboardEvent("keydown", { key, bubbles: true, cancelable: true });
    const up = new KeyboardEvent("keyup", { key, bubbles: true, cancelable: true });
    element.dispatchEvent(down);
    element.dispatchEvent(up);
    if (
      !down.defaultPrevented &&
      !up.defaultPrevented &&
      ["Enter", " "].includes(key) &&
      element instanceof HTMLButtonElement
    )
      element.click();
  });
  await tick();
}

/** The menu's button: the one that says whether its menu is open, named by name. */
export const menuButton = (host: HTMLElement, name: string) =>
  [...host.querySelectorAll<HTMLButtonElement>("button[aria-expanded]")].find((button) =>
    button.textContent?.includes(name)
  );

/** The open menu's items: each a button that says whether it is pressed, by its text. */
export const menuItems = (host: HTMLElement) =>
  [...host.querySelectorAll<HTMLButtonElement>("button[aria-pressed]")].map((item) => ({
    item,
    label: item.textContent,
    pressed: item.getAttribute("aria-pressed"),
  }));

/** The open menu's item that says label. */
export const menuItem = (host: HTMLElement, label: string) =>
  menuItems(host).find((found) => found.label === label)?.item;
