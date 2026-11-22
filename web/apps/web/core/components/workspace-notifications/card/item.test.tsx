// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { problem } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { toasts } from "@/lib/fake-toast";
import { notificationRequests, notificationTab, pageOf } from "@/store/notification/fake-notifications";
import { answer, cardOf, cards, listRead, noteOf, pageState, renderList, settled } from "../fake-notification-list";

// A card of the notifications page's list (P4 spec 2.6, R16, R23): one button, which selects its notification and
// opens its preview at once (the selection is the preview's, root.tsx) and, for an unread one, queues marking it read,
// which the preview never waits for; a refusal of that says why, the preview open still, but for
// notification.not_found, after which the card is gone and its selection with it. The button says the work item, as
// it was when the notification was sent, and when: till when it is snoozed, while it is (snoozedNow), else how long
// ago; an unread one's dot says so to a screen reader. The store and HTTP are real; SWR is, with the test's cache.

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
vi.mock("@/hooks/store/use-workspace", () => ({ useWorkspace: () => fixture.tab.workspaceRoot }));
vi.mock("@/hooks/store/use-notifications", () => ({ useNotifications: () => fixture.tab.store }));

const NOW = new Date("2026-11-20T12:00:00Z");
let host: HTMLDivElement, root: Root, cache: Map<string, unknown>;
globalThis.IS_REACT_ACT_ENVIRONMENT = true;
beforeEach(async () => {
  vi.useFakeTimers({ now: NOW });
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
const two = noteOf("Two", 2, { read_at: "2026-11-19T09:00:00Z" });
const selectionOf = (n: typeof one) => ({ notificationId: n.id, issueId: n.issue_id, projectId: n.project_id });
/** The changes of the notification nerve received so far. */
const patches = (n: typeof one) =>
  nerve()
    .to(notificationRequests.at(n))
    .filter((call) => call.method === "PATCH");
const open = (issue: string) => act(async () => cardOf(host, issue)?.querySelector("button")?.click());

/** The list at acme's address shows the notifications of nerve's first page. */
async function listed(...notifications: (typeof one)[]) {
  await renderList(root, cache);
  await answer(await listRead(nerve(), 0), pageOf(notifications));
}

describe("NotificationItem", () => {
  it("is one button that says the work item, selecting the notification and opening its preview at once", async () => {
    await listed(one, two);
    expect(cardOf(host, "WEB-1 One")?.querySelectorAll("button")).toHaveLength(1);
    await open("WEB-1 One");
    expect(pageState.current.selected).toEqual(selectionOf(one));
    await open("WEB-2 Two");
    expect(pageState.current.selected).toEqual(selectionOf(two));
  });

  it("queues marking an unread one read, and opens it without waiting for nerve", async () => {
    await listed(one);
    await open("WEB-1 One");
    expect(pageState.current.selected).toEqual(selectionOf(one));
    await act(async () => until(() => patches(one).length === 1, "the change"));
    expect(patches(one)[0]?.body).toEqual({ read: true });
    // nerve has not answered: the preview is open all the same
    expect(pageState.current.selected).toEqual(selectionOf(one));
  });

  it("marks nothing for a read one", async () => {
    await listed(two);
    await open("WEB-2 Two");
    expect(pageState.current.selected).toEqual(selectionOf(two));
    await settled();
    expect(patches(two)).toEqual([]);
  });

  it("says why nerve refused to mark it read, its preview still open", async () => {
    await listed(one);
    await open("WEB-1 One");
    await act(async () => until(() => patches(one).length === 1, "the change"));
    patches(one)[0]?.answer(problem(503, "server_busy"));
    await settled();
    expect(toasts).toEqual([{ type: "error", title: "toast.error", message: "errors.server_busy" }]);
    expect(pageState.current.selected).toEqual(selectionOf(one));
    expect(cards(host)).toEqual(["WEB-1 One"]);
  });

  it("says nothing of nerve's refusal once the tab has moved to another account meanwhile (followInSession)", async () => {
    await listed(one);
    await open("WEB-1 One");
    await act(async () => until(() => patches(one).length === 1, "the change"));
    fixture.session.loginId = "two";
    patches(one)[0]?.answer(problem(503, "server_busy"));
    await settled();
    expect(toasts).toEqual([]);
  });

  it("is gone, and so is its selection, once nerve no longer gives the notification (notification.not_found)", async () => {
    await listed(one, two);
    await open("WEB-1 One");
    await act(async () => until(() => patches(one).length === 1, "the change"));
    patches(one)[0]?.answer(problem(404, "notification.not_found"));
    await settled();
    expect(toasts).toEqual([{ type: "error", title: "toast.error", message: "errors.notification_not_found" }]);
    expect(cards(host)).toEqual(["WEB-2 Two"]);
    expect(pageState.current.selected).toBeUndefined();
  });

  it("keeps another notification selected meanwhile when nerve no longer gives the one it marked", async () => {
    await listed(one, two);
    await open("WEB-1 One");
    await act(async () => until(() => patches(one).length === 1, "the change"));
    await open("WEB-2 Two");
    patches(one)[0]?.answer(problem(404, "notification.not_found"));
    await settled();
    expect(cards(host)).toEqual(["WEB-2 Two"]);
    expect(pageState.current.selected).toEqual(selectionOf(two));
  });

  it("says till when it is snoozed while it is, and how long ago otherwise, a snooze over included", async () => {
    const snoozed = noteOf("Later", 3, { snoozed_till: "2026-11-21T08:30:00Z" });
    const over = noteOf("Over", 4, { snoozed_till: "2026-11-20T11:00:00Z", created_at: "2026-11-18T12:00:00Z" });
    await listed(snoozed, over);
    expect(cardOf(host, "WEB-3 Later")?.textContent).toContain("notification.card.till");
    expect(cardOf(host, "WEB-4 Over")?.textContent).not.toContain("notification.card.till");
    expect(cardOf(host, "WEB-4 Over")?.textContent).toContain("2 days ago");
  });

  it("says to a screen reader that an unread one is unread, and nothing of a read one", async () => {
    await listed(one, two);
    expect(cardOf(host, "WEB-1 One")?.querySelector(".sr-only")?.textContent).toBe("notification.card.unread");
    expect(cardOf(host, "WEB-2 Two")?.querySelector(".sr-only")).toBeNull();
  });
});
