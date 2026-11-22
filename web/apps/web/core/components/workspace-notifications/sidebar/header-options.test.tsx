// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { Call } from "@/lib/auth/fake-nerve";
import { json, problem } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { response as listResponse } from "@/lib/fake-session-swr";
import { toasts } from "@/lib/fake-toast";
import { acme, countOf, notificationRequests, notificationTab, pageOf } from "@/store/notification/fake-notifications";
import { changeHeadState, renderHeadPart } from "./fake-header-menus";
import { NotificationHeaderOptions } from "./header-options";

// The head's buttons of the notifications page (P4 spec 2.6, R7, R9): mark all read marks the unread ones the page's
// filters select, its body made of them (markAllBody: the all tab's senders only, and only when some are chosen),
// out of use while nerve marks; says how many nerve marked or why it refused; either way the list's first page and
// the count are read again (nerve marks in batches). Refresh reads the first page and the count again, turning
// meanwhile. What follows nerve's answer is done only in the session it was sent in (followInSession). The store and
// HTTP are real; the list's read is fake-session-swr.ts's, which reads nothing and takes Refresh's answer as it comes
// (refresh.test.tsx renders the head over the list with SWR). @nerve/i18n's text is its key, then the values it was
// given.

const fixture = vi.hoisted(() => ({
  session: { loginId: "one" },
  tab: undefined as unknown as Awaited<
    ReturnType<typeof import("@/store/notification/fake-notifications").notificationTab>
  >,
}));
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: fixture.session }, publicClient: {} }));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n-values"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@/hooks/store/use-workspace", () => ({ useWorkspace: () => fixture.tab.workspaceRoot }));
vi.mock("@/hooks/store/use-notifications", () => ({ useNotifications: () => fixture.tab.store }));
vi.mock("@/lib/use-session-swr", () => import("@/lib/fake-session-swr"));

let host: HTMLDivElement, root: Root;
globalThis.IS_REACT_ACT_ENVIRONMENT = true;
beforeEach(async () => {
  vi.useFakeTimers({ now: new Date("2026-11-20T12:00:00Z") });
  vi.spyOn(Element.prototype, "getAnimations").mockReturnValue([]);
  toasts.length = 0;
  fixture.session.loginId = "one";
  fixture.tab = await notificationTab();
  listResponse.current = { mutate: (answer: Promise<unknown>) => answer };
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

const nerve = () => fixture.tab.nerve;
const render = () => renderHeadPart(root, <NotificationHeaderOptions />);
const button = (name: string) => host.querySelector<HTMLButtonElement>(`button[aria-label="${name}"]`);
const markAll = () => button("notification.options.mark_all_as_read");
const refresh = () => button("notification.options.refresh");
const settled = () => act(async () => vi.advanceTimersByTimeAsync(50));
/** The request to path nerve received after the `before` ones it received so far. */
async function sent(path: string, before = 0): Promise<Call> {
  await act(async () => until(() => nerve().to(path).length > before, path));
  return nerve().to(path)[before]!;
}
const MARK_ALL = notificationRequests.markAllRead(acme);
const LIST = notificationRequests.list(acme);
const COUNT = notificationRequests.unreadCount(acme);

describe("NotificationHeaderOptions: mark all read", () => {
  it("marks the unread ones the page's filters select, says how many, and reads the first page and the count again", async () => {
    await render();
    await changeHeadState((state) => {
      state.toggleType("created");
      state.toggleType("assigned");
      state.toggleArchived();
    });
    await act(async () => markAll()?.click());
    const post = await sent(MARK_ALL);
    expect(post.body).toEqual({ tab: "all", type: ["assigned", "created"], archived: true, snoozed: false });
    expect(nerve().to(LIST)).toEqual([]);
    post.answer(json(200, { marked: 3 }));
    await settled();
    expect(toasts).toEqual([{ type: "success", title: "toast.success", message: "notification.toasts.all_read 3" }]);
    expect((await sent(LIST)).query).toEqual({
      tab: "all",
      type: "assigned,created",
      archived: "true",
      snoozed: "false",
      limit: "100",
    });
    await sent(COUNT);
  });

  it("sends no sender in the mentions tab, the all tab's choice aside, nor in the all tab with none chosen", async () => {
    await render();
    await changeHeadState((state) => {
      state.toggleType("subscribed");
      state.setTab("mentions");
    });
    await act(async () => markAll()?.click());
    const mentions = await sent(MARK_ALL);
    expect(mentions.body).toEqual({ tab: "mentions", archived: false, snoozed: false });
    mentions.answer(json(200, { marked: 0 }));
    await settled();
    await changeHeadState((state) => {
      state.setTab("all");
      state.clearTypes();
    });
    await act(async () => markAll()?.click());
    expect((await sent(MARK_ALL, 1)).body).toEqual({ tab: "all", archived: false, snoozed: false });
  });

  it("is out of use while nerve marks, so that a second press sends nothing more", async () => {
    await render();
    await act(async () => markAll()?.click());
    const post = await sent(MARK_ALL);
    expect(markAll()?.disabled).toBe(true);
    await act(async () => markAll()?.click());
    post.answer(json(200, { marked: 1 }));
    await settled();
    await settled();
    expect(nerve().to(MARK_ALL)).toHaveLength(1);
    expect(markAll()?.disabled).toBe(false);
  });

  it("says why nerve refused, and reads the first page and the count again all the same", async () => {
    await render();
    await act(async () => markAll()?.click());
    (await sent(MARK_ALL)).answer(problem(503, "server_busy"));
    await settled();
    expect(toasts).toEqual([{ type: "error", title: "toast.error", message: "errors.server_busy" }]);
    await sent(LIST);
    await sent(COUNT);
    expect(markAll()?.disabled).toBe(false);
  });

  it.each([
    ["marked", () => json(200, { marked: 2 })],
    ["refused", () => problem(503, "server_busy")],
  ])("says nothing once the tab has moved to another account before nerve %s", async (_, response) => {
    await render();
    await act(async () => markAll()?.click());
    const post = await sent(MARK_ALL);
    fixture.session.loginId = "two";
    post.answer(response());
    await settled();
    expect(toasts).toEqual([]);
  });
});

describe("NotificationHeaderOptions: refresh", () => {
  it("reads the list's first page the filters select and the count again, out of use and turning meanwhile", async () => {
    await render();
    await changeHeadState((state) => state.toggleUnread());
    await act(async () => refresh()?.click());
    const list = await sent(LIST);
    expect(list.query).toEqual({ tab: "all", read: "false", archived: "false", snoozed: "false", limit: "100" });
    const count = await sent(COUNT);
    expect(refresh()?.disabled).toBe(true);
    expect(refresh()?.querySelector("svg")?.getAttribute("class")).toContain("animate-spin");
    list.answer(json(200, pageOf([])));
    count.answer(json(200, countOf(0, 0)));
    await settled();
    expect(refresh()?.disabled).toBe(false);
    expect(refresh()?.querySelector("svg")?.getAttribute("class")).not.toContain("animate-spin");
    expect(toasts).toEqual([]);
  });

  it("says why nerve could not read them, unless the tab has moved to another account meanwhile", async () => {
    await render();
    await act(async () => refresh()?.click());
    (await sent(LIST)).answer(problem(503, "server_busy"));
    (await sent(COUNT)).answer(json(200, countOf(0, 0)));
    await settled();
    expect(toasts).toEqual([{ type: "error", title: "toast.error", message: "errors.server_busy" }]);
    toasts.length = 0;
    await act(async () => refresh()?.click());
    const list = await sent(LIST, 1);
    fixture.session.loginId = "two";
    list.answer(problem(503, "server_busy"));
    await settled();
    expect(toasts).toEqual([]);
  });
});
