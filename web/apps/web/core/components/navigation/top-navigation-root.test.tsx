// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { MemoryRouter, Route, Routes } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { NotificationUnreadCount } from "@nerve/api-client";
import { json } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { handed } from "@/lib/fake-session-swr";
import { acme, beta, countOf, notificationRequests, notificationTab } from "@/store/notification/fake-notifications";
import { TopNavigationRoot } from "./top-navigation-root";

// The top navigation's inbox link on every page of a workspace (P4 spec 2.4, 2.6, R5): it fetches the caller's unread
// counts in the route's workspace (useUnreadNotificationCount), with fake-session-swr.ts's stand-in for useSessionSWR,
// into a notification store of a tab whose router store still names acme, the workspace the caller was on; it shows
// the dot while either tab has an unread notification, and names the link, its count in its name. The menus beside it
// are not this test's; @nerve/i18n's text is its key, then the values it was given; a tooltip shows its label beside
// what it is the tooltip of (fake-controls.ts).

const held = vi.hoisted(() => ({ tab: undefined as unknown as Awaited<ReturnType<typeof notificationTab>> }));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n-values"));
vi.mock("@/lib/use-session-swr", () => import("@/lib/fake-session-swr"));
vi.mock("@/hooks/store/use-workspace", () => ({ useWorkspace: () => held.tab.workspaceRoot }));
vi.mock("@/hooks/store/use-notifications", () => ({ useNotifications: () => held.tab.store }));
vi.mock("@makeplane/propel/components/tooltip", () => import("@/lib/fake-controls"));
vi.mock("@/components/navigation", () => ({ TopNavPowerK: () => null }));
vi.mock("@/components/workspace/sidebar/help-section/root", () => ({ HelpMenuRoot: () => null }));
vi.mock("@/components/workspace/sidebar/user-menu-root", () => ({ UserMenuRoot: () => null }));
vi.mock("@/components/workspace/sidebar/workspace-menu-root", () => ({ WorkspaceMenuRoot: () => null }));

let host: HTMLDivElement, root: Root;
globalThis.IS_REACT_ACT_ENVIRONMENT = true;
beforeEach(async () => {
  vi.useFakeTimers({ now: 1_000_000 });
  handed.length = 0;
  held.tab = await notificationTab();
  host = document.createElement("div");
  document.body.append(host);
  root = createRoot(host);
});
afterEach(async () => {
  await act(async () => root.unmount());
  host.remove();
  vi.useRealTimers();
});

/** The top navigation on the page at path, a workspace's (/:workspaceSlug/…) or none's ("/"). */
const render = (path: string) =>
  act(async () =>
    root.render(
      <MemoryRouter initialEntries={[path]}>
        <Routes>
          <Route path="/:workspaceSlug/*" element={<TopNavigationRoot />} />
          <Route path="/" element={<TopNavigationRoot />} />
        </Routes>
      </MemoryRouter>
    )
  );
const link = () => host.querySelector("a");
const dot = () => link()?.querySelector(".bg-danger-primary") ?? null;
const tooltip = () => host.querySelector("[data-tooltip]")?.textContent;
/** The fetches the page handed useSessionSWR, as it rendered. */
const keys = () => handed.map(([fetch]) => fetch);

/** SWR fetches the last fetch handed over, and nerve answers it with count; the page shows what the store then has. */
async function fetched(count: NotificationUnreadCount, workspace = acme) {
  const { nerve } = held.tab;
  const at = nerve.calls.length;
  const [fetch, fetcher] = handed.at(-1)!;
  let done = false;
  await act(async () => {
    if (fetch) void fetcher(...(fetch.slice(1) as string[])).finally(() => (done = true));
    await until(() => nerve.calls.length === at + 1, "the count's request");
  });
  expect(nerve.calls[at]).toMatchObject({ method: "GET", path: notificationRequests.unreadCount(workspace) });
  await act(async () => {
    nerve.calls[at]?.answer(json(200, count));
    await until(() => done, "the count");
  });
}

describe("TopNavigationRoot, the inbox link", () => {
  it("fetches the route's workspace's unread counts, keyed by its id, from its slug, not the router store's workspace", async () => {
    await render("/beta/projects");
    expect(keys().at(-1)).toEqual(["NOTIFICATION_UNREAD_COUNT", beta.id, beta.slug]);
    expect(link()?.getAttribute("href")).toBe("/beta/notifications");
    await fetched(countOf(0, 4), beta);
    expect(dot()).not.toBeNull();
    expect(link()?.getAttribute("aria-label")).toBe("notification.a11y.unread_count notification.label 4");
  });

  it.each([
    ["only the all tab", countOf(3, 0)],
    ["only the mentions tab", countOf(0, 2)],
    ["both tabs", countOf(1, 1)],
  ])("shows the dot while %s has unread notifications", async (_, count) => {
    await render("/acme");
    expect(dot()).toBeNull();
    await fetched(count);
    expect(dot()).not.toBeNull();
  });

  it("shows no dot while neither tab has an unread notification, and names its link the inbox alone", async () => {
    await render("/acme");
    expect(link()?.getAttribute("aria-label")).toBe("notification.label");
    await fetched(countOf(0, 0));
    expect(dot()).toBeNull();
    expect(link()?.getAttribute("aria-label")).toBe("notification.label");
  });

  it("names its link the inbox, with both tabs' unread notifications counted, and its tooltip in the page's language", async () => {
    await render("/acme");
    expect(tooltip()).toBe("notification.label");
    await fetched(countOf(3, 2));
    expect(link()?.getAttribute("aria-label")).toBe("notification.a11y.unread_count notification.label 5");
    expect(tooltip()).toBe("notification.label");
  });

  it.each([
    ["a workspace not the caller's", "/gamma"],
    ["no workspace", "/"],
  ])("fetches nothing on the page of %s, and shows no dot", async (_, path) => {
    await render(path);
    expect(keys()).not.toHaveLength(0);
    expect(keys().every((fetch) => fetch === null)).toBe(true);
    await act(async () => vi.advanceTimersByTimeAsync(1_000));
    expect(held.tab.nerve.calls).toEqual([]);
    expect(dot()).toBeNull();
    expect(link()?.getAttribute("aria-label")).toBe("notification.label");
  });
});
