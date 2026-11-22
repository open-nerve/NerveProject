// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { Tooltip } from "@makeplane/propel/components/tooltip";
import { AppSidebarItem } from "./sidebar-item";

// An item of the app's bars as the trigger of a tooltip (the top navigation's inbox, T15's ruling): the real Propel
// tooltip, whose Base UI trigger merges its ref and its handlers into the element it renders, so the link the item
// renders takes them, and the tooltip opens as the link has the focus or the pointer; the link keeps its own address
// and name. A wrapper's props are the link's alone: the button variant, a menu's, takes none.

let host: HTMLDivElement, root: Root;
globalThis.IS_REACT_ACT_ENVIRONMENT = true;
beforeEach(() => {
  vi.useFakeTimers();
  vi.stubGlobal(
    "ResizeObserver",
    class {
      observe() {}
      unobserve() {}
      disconnect() {}
    }
  );
  host = document.createElement("div");
  document.body.append(host);
  root = createRoot(host);
});
afterEach(async () => {
  await act(async () => root.unmount());
  document.body.replaceChildren();
  vi.unstubAllGlobals();
  vi.useRealTimers();
});

const render = () =>
  act(async () =>
    root.render(
      <MemoryRouter>
        <Tooltip label="Inbox" side="bottom">
          <AppSidebarItem
            variant="link"
            item={{ href: "/acme/notifications", label: "Inbox, 5 unread", icon: <span data-icon /> }}
          />
        </Tooltip>
      </MemoryRouter>
    )
  );
const link = () => host.querySelector("a")!;
const tooltip = () => document.querySelector('[role="tooltip"]');
const later = () => act(async () => vi.advanceTimersByTimeAsync(1_000));

describe("AppSidebarItem, a link, as a tooltip's trigger", () => {
  it("opens the tooltip as the link has the keyboard's focus", async () => {
    await render();
    expect(tooltip()).toBeNull();
    await act(async () => {
      document.dispatchEvent(new KeyboardEvent("keydown", { key: "Tab", bubbles: true }));
      link().focus();
    });
    await later();
    expect(tooltip()?.textContent).toBe("Inbox");
    expect(link().getAttribute("href")).toBe("/acme/notifications");
    expect(link().getAttribute("aria-label")).toBe("Inbox, 5 unread");
  });

  it("opens the tooltip as the pointer rests on the link", async () => {
    await render();
    // a mouse's, as a browser gives it: the pointer's events, then the mouse's; Base UI opens once it rests (mousemove)
    await act(async () => {
      link().dispatchEvent(new MouseEvent("pointerover", { bubbles: true }));
      link().dispatchEvent(new MouseEvent("pointerenter"));
      link().dispatchEvent(new MouseEvent("mouseover", { bubbles: true }));
      link().dispatchEvent(new MouseEvent("mouseenter"));
      link().dispatchEvent(new MouseEvent("mousemove", { bubbles: true }));
    });
    await later();
    expect(tooltip()?.textContent).toBe("Inbox");
  });
});

describe("AppSidebarItem's props", () => {
  it("take a wrapper's on the link alone: the button, a menu's, has no place for them (re-review m2)", () => {
    const linkItem = <AppSidebarItem variant="link" item={{ href: "/acme/notifications" }} onFocus={() => {}} />;
    // @ts-expect-error a button item takes no wrapper's handler, which it would drop
    const buttonItem = <AppSidebarItem variant="button" item={{ label: "Help" }} onFocus={() => {}} />;
    expect([linkItem.props.variant, buttonItem.props.variant]).toEqual(["link", "button"]);
  });
});
