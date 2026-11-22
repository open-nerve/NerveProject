// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { problem } from "@/lib/auth/fake-nerve";
import { toasts } from "@/lib/fake-toast";
import { notificationTab, pageOf } from "@/store/notification/fake-notifications";
import { answer, cards, changeState, listRead, noteOf, renderList, settled } from "../fake-notification-list";

// The notifications page's list (P4 spec 2.4, 2.6, R7, R21): the first page of the route's workspace's notifications
// its filters select, by the session's SWR keyed by them (useNotificationList), so that a change of filters reads the
// list they name; a wait while nerve has not answered, why nerve could not read it with a retry (never a wait that does
// not end), the tab's own empty state; "Load more", a button, while nerve gives a cursor, out of use while it reads,
// over a page of no rows too (re-93 m6). The store and HTTP are real; SWR is, with the test's cache. @nerve/i18n's
// text is its key, then the values it was given; the empty state says its title.

const fixture = vi.hoisted(() => ({
  session: { loginId: "one" },
  tab: undefined as unknown as Awaited<
    ReturnType<typeof import("@/store/notification/fake-notifications").notificationTab>
  >,
}));
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: fixture.session }, publicClient: {} }));
vi.mock("@/lib/auth/use-session", () => ({
  useSession: () => ({ status: "signed-in", loginId: fixture.session.loginId }),
}));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n-values"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@nerve/propel/empty-state", () => ({
  EmptyStateCompact: (props: { title?: string }) => <p data-empty-state>{props.title}</p>,
}));
vi.mock("@/hooks/store/use-workspace", () => ({ useWorkspace: () => fixture.tab.workspaceRoot }));
vi.mock("@/hooks/store/use-notifications", () => ({ useNotifications: () => fixture.tab.store }));

let host: HTMLDivElement, root: Root, cache: Map<string, unknown>;
globalThis.IS_REACT_ACT_ENVIRONMENT = true;
beforeEach(async () => {
  vi.useFakeTimers({ now: new Date("2026-11-20T12:00:00Z") });
  toasts.length = 0;
  fixture.session.loginId = "one";
  fixture.tab = await notificationTab();
  cache = new Map();
  host = document.createElement("div");
  document.body.append(host);
  root = createRoot(host);
});
afterEach(async () => {
  await act(async () => root.unmount());
  host.remove();
  vi.useRealTimers();
});

const nerve = () => fixture.tab.nerve;
const one = noteOf("One", 1);
const two = noteOf("Two", 2);
const placeholder = () => host.querySelector('[role="status"]');
const failure = () => host.querySelector('[role="alert"]');
const emptyState = () => host.querySelector("[data-empty-state]")?.textContent;
const button = (name: string) => [...host.querySelectorAll("button")].find((found) => found.textContent === name);
const loadMore = () => button("common.load_more") ?? button("common.loading");

describe("NotificationList", () => {
  it("reads the first page of the list its filters select, and the list they name as they change", async () => {
    await renderList(root, cache);
    const first = await listRead(nerve(), 0);
    expect(first.query).toEqual({ tab: "all", archived: "false", snoozed: "false", limit: "100" });
    await answer(first, pageOf([one]));
    expect(cards(host)).toEqual(["WEB-1 One"]);

    await changeState((state) => state.toggleUnread());
    const unread = await listRead(nerve(), 1);
    expect(unread.query).toEqual({ tab: "all", read: "false", archived: "false", snoozed: "false", limit: "100" });
    await answer(unread, pageOf([two]));
    expect(cards(host)).toEqual(["WEB-2 Two"]);

    await changeState((state) => state.setTab("mentions"));
    const mentions = await listRead(nerve(), 2);
    expect(mentions.query).toEqual({
      tab: "mentions",
      read: "false",
      archived: "false",
      snoozed: "false",
      limit: "100",
    });
  });

  it("waits for nerve's first page, saying it is loading", async () => {
    await renderList(root, cache);
    const first = await listRead(nerve(), 0);
    expect(placeholder()?.textContent).toContain("common.loading");
    expect(failure()).toBeNull();
    await answer(first, pageOf([one]));
    expect(placeholder()).toBeNull();
  });

  it("says why nerve could not read the list, waiting no more, and reads it again on retry", async () => {
    await renderList(root, cache);
    const first = await listRead(nerve(), 0);
    first.answer(problem(503, "server_busy"));
    await settled();
    expect(failure()?.textContent).toContain("errors.server_busy");
    expect(placeholder()).toBeNull();
    await act(async () => button("common.retry")?.click());
    const again = await listRead(nerve(), 1);
    await answer(again, pageOf([one]));
    expect(failure()).toBeNull();
    expect(cards(host)).toEqual(["WEB-1 One"]);
  });

  it("says each tab's own empty state", async () => {
    await renderList(root, cache);
    await answer(await listRead(nerve(), 0), pageOf([]));
    expect(emptyState()).toBe("workspace_empty_state.inbox_sidebar_all.title");
    await changeState((state) => state.setTab("mentions"));
    await answer(await listRead(nerve(), 1), pageOf([]));
    expect(emptyState()).toBe("workspace_empty_state.inbox_sidebar_mentions.title");
    expect(loadMore()).toBeUndefined();
  });

  it("offers load more, not the empty state, over a page of no rows that has a cursor (re-93 m6)", async () => {
    await renderList(root, cache);
    await answer(await listRead(nerve(), 0), pageOf([], "c-1"));
    expect(emptyState()).toBeUndefined();
    expect(loadMore()?.textContent).toBe("common.load_more");
  });

  it("offers load more, a button, only while nerve gives a cursor; out of use while it reads the next page by it", async () => {
    await renderList(root, cache);
    await answer(await listRead(nerve(), 0), pageOf([one], "c-1"));
    expect(loadMore()?.tagName).toBe("BUTTON");
    expect(loadMore()?.getAttribute("type")).toBe("button");
    await act(async () => loadMore()?.click());
    const next = await listRead(nerve(), 1);
    expect(next.query).toMatchObject({ tab: "all", cursor: "c-1", limit: "100" });
    expect(loadMore()?.textContent).toBe("common.loading");
    expect(loadMore()?.hasAttribute("disabled")).toBe(true);
    await answer(next, pageOf([two]));
    expect(cards(host)).toEqual(["WEB-1 One", "WEB-2 Two"]);
    expect(loadMore()).toBeUndefined();
  });

  it("says why nerve could not read the next page, and offers load more again", async () => {
    await renderList(root, cache);
    await answer(await listRead(nerve(), 0), pageOf([one], "c-1"));
    await act(async () => loadMore()?.click());
    (await listRead(nerve(), 1)).answer(problem(503, "server_busy"));
    await settled();
    expect(toasts).toEqual([{ type: "error", title: "toast.error", message: "errors.server_busy" }]);
    expect(loadMore()?.textContent).toBe("common.load_more");
    expect(loadMore()?.hasAttribute("disabled")).toBe(false);
    expect(cards(host)).toEqual(["WEB-1 One"]);
  });

  it("says nothing of nerve's refusal of the next page once the tab has moved to another account meanwhile", async () => {
    await renderList(root, cache);
    await answer(await listRead(nerve(), 0), pageOf([one], "c-1"));
    await act(async () => loadMore()?.click());
    const next = await listRead(nerve(), 1);
    fixture.session.loginId = "two";
    next.answer(problem(503, "server_busy"));
    await settled();
    expect(toasts).toEqual([]);
  });
});
