// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { NotificationUnreadCount } from "@nerve/api-client";
import { countOf } from "@/store/notification/fake-notifications";
import { NotificationPageStateProvider, useNotificationPageState } from "../page-state";
import { NotificationTabs } from "./tabs";

// The notifications page's two tabs (P4 spec 2.6, R5, R16): the all tab and the mentions tab, tabs as a screen reader
// reads them, the all tab selected first; each named by its label and its unread count, as the one fetch of the
// route's workspace's counts gives it (useUnreadNotificationCount, a stand-in here): all for the all tab, mentions for
// the mentions tab, none shown at 0, 99+ past 99. Choosing one sets the page's tab, and its panel shows the list.
// @nerve/i18n's text is its key, then the values it was given.

const held = vi.hoisted(() => ({
  count: undefined as NotificationUnreadCount | undefined,
  state: undefined as unknown as ReturnType<typeof import("../page-state").useNotificationPageState>,
}));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n-values"));
vi.mock("@/hooks/use-unread-notification-count", () => ({ useUnreadNotificationCount: () => held.count }));

function Read() {
  held.state = useNotificationPageState();
  return <p>the list</p>;
}

let host: HTMLDivElement, root: Root;
globalThis.IS_REACT_ACT_ENVIRONMENT = true;
beforeEach(() => {
  held.count = undefined;
  host = document.createElement("div");
  document.body.append(host);
  root = createRoot(host);
});
afterEach(async () => {
  await act(async () => root.unmount());
  host.remove();
});

const render = () =>
  act(async () =>
    root.render(
      <NotificationPageStateProvider>
        <NotificationTabs>
          <Read />
        </NotificationTabs>
      </NotificationPageStateProvider>
    )
  );
const tabs = () => [...host.querySelectorAll<HTMLElement>('[role="tab"]')];
const names = () => tabs().map((tab) => tab.getAttribute("aria-label") ?? tab.textContent);
const selected = () => tabs().map((tab) => tab.getAttribute("aria-selected"));
/** The panel shown, the chosen tab's (the others are hidden placeholders). */
const panel = () => host.querySelector('[role="tabpanel"]:not([aria-hidden="true"])');
/** The counts the tabs show, as each tab shows it after its label (nothing at 0). */
const chips = () => tabs().map((tab) => tab.textContent?.replace(/^notification\.tabs\.(?:all|mentions)/, "") || null);

describe("NotificationTabs", () => {
  it("are tabs, the all tab selected, each named by its label and its tab's unread count", async () => {
    held.count = countOf(3, 2);
    await render();
    expect(host.querySelector('[role="tablist"]')).not.toBeNull();
    expect(names()).toEqual([
      "notification.a11y.unread_count notification.tabs.all 3",
      "notification.a11y.unread_count notification.tabs.mentions 2",
    ]);
    expect(selected()).toEqual(["true", "false"]);
    expect(chips()).toEqual(["3", "2"]);
    expect(panel()?.textContent).toBe("the list");
  });

  it("show no count for a tab with no unread notification, nor before nerve counts them, naming it by its label alone", async () => {
    held.count = countOf(0, 5);
    await render();
    expect(names()).toEqual(["notification.tabs.all", "notification.a11y.unread_count notification.tabs.mentions 5"]);
    expect(chips()).toEqual([null, "5"]);
    held.count = undefined;
    await render();
    expect(names()).toEqual(["notification.tabs.all", "notification.tabs.mentions"]);
    expect(chips()).toEqual([null, null]);
  });

  it("show 99+ for a count past 99", async () => {
    held.count = countOf(100, 99);
    await render();
    expect(chips()).toEqual(["99+", "99"]);
    expect(names()).toEqual([
      "notification.a11y.unread_count notification.tabs.all 99+",
      "notification.a11y.unread_count notification.tabs.mentions 99",
    ]);
  });

  it("set the page's tab as one is chosen, its panel showing the list", async () => {
    held.count = countOf(1, 1);
    await render();
    await act(async () => tabs()[1]?.click());
    expect(held.state.filters.tab).toBe("mentions");
    expect(selected()).toEqual(["false", "true"]);
    expect(panel()?.textContent).toBe("the list");
    await act(async () => tabs()[0]?.click());
    expect(held.state.filters.tab).toBe("all");
    expect(selected()).toEqual(["true", "false"]);
  });
});
