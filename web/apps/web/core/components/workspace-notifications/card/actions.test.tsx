// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { Call } from "@/lib/auth/fake-nerve";
import { json, problem } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { toasts } from "@/lib/fake-toast";
import { notificationRequests, notificationTab, pageOf } from "@/store/notification/fake-notifications";
import type { Notification } from "@nerve/api-client";
import { answer, cardOf, cards, changeState, listRead, noteOf, pageState, renderList } from "../fake-notification-list";
import { press } from "../sidebar/fake-header-menus";

// A card's actions (P4 spec 2.6, R16, R25): beside its one main button, each named by its row and reached by the
// keyboard, shown as the card is hovered or holds the focus: mark read or unread; archive or unarchive, unless the
// snoozed are listed; the snooze menu, unless the archived are (viewOffers: the page never makes a notification both
// archived and snoozed, which nerve lists in no view). Each sends its change through the store, one at a time, out of
// use meanwhile; says it once nerve has made it, or why not, never both; for notification.not_found the card is gone
// and its selection with it. What follows nerve's answer is done only in the session it was sent in. The store and
// HTTP are real; SWR is, with the test's cache. @nerve/i18n's text is its key, then the values it was given.

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
vi.mock("@/hooks/store/use-member", () => ({ useMember: () => ({ getUserDetails: () => undefined }) }));
vi.mock("@/hooks/use-profile-image", () => import("@/lib/fake-profile-image"));

let host: HTMLDivElement, root: Root, cache: Map<string, unknown>;
globalThis.IS_REACT_ACT_ENVIRONMENT = true;
beforeEach(async () => {
  vi.useFakeTimers({ now: new Date("2026-11-20T12:00:00Z") });
  vi.spyOn(Element.prototype, "getAnimations").mockReturnValue([]);
  // Headless UI's menu watches its button's place as it closes, which jsdom cannot measure
  vi.stubGlobal(
    "ResizeObserver",
    class {
      observe() {}
      unobserve() {}
      disconnect() {}
    }
  );
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
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
  vi.useRealTimers();
});

const nerve = () => fixture.tab.nerve;
const settled = () => act(async () => vi.advanceTimersByTimeAsync(50));
const one = noteOf("One", 1);
const read = noteOf("Two", 2, { read_at: "2026-11-19T09:00:00Z" });
const snoozed = noteOf("Later", 3, { snoozed_till: "2026-11-21T08:30:00Z" });
const archived = noteOf("Old", 4, { archived_at: "2026-11-19T09:00:00Z" });
/** The card's actions: its buttons but the first, its main one. */
const actionsOf = (issue: string) => [...(cardOf(host, issue)?.querySelectorAll("button") ?? [])].slice(1);
const names = (issue: string) => actionsOf(issue).map((button) => button.getAttribute("aria-label"));
const action = (issue: string, name: string) =>
  actionsOf(issue).find((button) => button.getAttribute("aria-label") === `notification.card.actions.${name} ${issue}`);
/** The changes of the notification nerve received so far. */
const patches = (n: Notification) =>
  nerve()
    .to(notificationRequests.at(n))
    .filter((call) => call.method === "PATCH");
async function patched(n: Notification, before = 0): Promise<Call> {
  await act(async () => until(() => patches(n).length > before, "the change"));
  return patches(n)[before]!;
}
const click = (button: HTMLButtonElement | undefined) => act(async () => button?.click());
const success = (key: string) => ({ type: "success", title: "toast.success", message: `notification.toasts.${key}` });

/** The list at acme's address shows the notifications nerve gives in the view change makes, if any. */
async function listed(notifications: Notification[], change?: Parameters<typeof changeState>[0]) {
  await renderList(root, cache);
  if (change === undefined) return answer(await listRead(nerve(), 0), pageOf(notifications));
  await answer(await listRead(nerve(), 0), pageOf([]));
  await changeState(change);
  await answer(await listRead(nerve(), 1), pageOf(notifications));
}

describe("NotificationActions", () => {
  it("names each action by its row, each a button beside the main one that the keyboard reaches", async () => {
    await listed([one, read]);
    expect(names("WEB-1 One")).toEqual([
      "notification.card.actions.mark_read WEB-1 One",
      "notification.card.actions.archive WEB-1 One",
      "notification.card.actions.snooze WEB-1 One",
    ]);
    expect(names("WEB-2 Two")[0]).toBe("notification.card.actions.mark_unread WEB-2 Two");
    const card = cardOf(host, "WEB-1 One")!;
    expect(card.querySelector("button")?.querySelector("button")).toBeNull();
    for (const button of actionsOf("WEB-1 One")) {
      expect(button.getAttribute("type")).toBe("button");
      expect(button.tabIndex).toBe(0);
      // shown as the card is hovered and as it holds the focus, nothing hiding it until then (Tab reaches it)
      const classes: string[] = [];
      for (let at: Element | null = button; at && at !== card; at = at.parentElement) classes.push(...at.classList);
      expect(classes).not.toContain("hidden");
      expect(classes).toEqual(expect.arrayContaining(["group-hover:opacity-100", "group-focus-within:opacity-100"]));
    }
  });

  it("offers no archive in the snoozed view, the snooze menu first ending the snooze (R25)", async () => {
    await listed([snoozed], (state) => state.toggleSnoozed());
    expect(names("WEB-3 Later")).toEqual([
      "notification.card.actions.mark_read WEB-3 Later",
      "notification.card.actions.snooze WEB-3 Later",
    ]);
    await press(action("WEB-3 Later", "snooze")!, "Enter");
    const items = [...document.querySelectorAll('[role="menuitem"]')].map((item) => item.textContent);
    expect(items[0]).toBe("notification.options.mark_unsnooze");
  });

  it("offers no snooze menu in the archived view, and unarchive (R25)", async () => {
    await listed([archived], (state) => state.toggleArchived());
    expect(names("WEB-4 Old")).toEqual([
      "notification.card.actions.mark_read WEB-4 Old",
      "notification.card.actions.unarchive WEB-4 Old",
    ]);
  });

  it("marks it read and unread, saying so, the card in its place", async () => {
    await listed([one], (state) => state.toggleUnread());
    await click(action("WEB-1 One", "mark_read"));
    const marked = await patched(one);
    expect(marked.body).toEqual({ read: true });
    await answer(marked, { ...one, read_at: "2026-11-20T12:00:00Z" });
    expect(toasts).toEqual([success("read")]);
    expect(cards(host)).toEqual(["WEB-1 One"]);
    await click(action("WEB-1 One", "mark_unread"));
    const unmarked = await patched(one, 1);
    expect(unmarked.body).toEqual({ read: false });
    await answer(unmarked, one);
    expect(toasts).toEqual([success("read"), success("unread")]);
  });

  it("archives it in the default view and unarchives it in the archived one, each leaving the view, saying so", async () => {
    await listed([one]);
    await click(action("WEB-1 One", "archive"));
    const archiving = await patched(one);
    expect(archiving.body).toEqual({ archived: true });
    await answer(archiving, { ...one, archived_at: "2026-11-20T12:00:00Z" });
    expect(toasts).toEqual([success("archived")]);
    expect(cards(host)).toEqual([]);

    await changeState((state) => state.toggleArchived());
    await answer(await listRead(nerve(), 1), pageOf([archived]));
    await click(action("WEB-4 Old", "unarchive"));
    const restoring = await patched(archived);
    expect(restoring.body).toEqual({ archived: false });
    await answer(restoring, { ...archived, archived_at: null });
    expect(toasts).toEqual([success("archived"), success("unarchived")]);
    expect(cards(host)).toEqual([]);
  });

  it("says why nerve refused, and never that it made the change", async () => {
    await listed([one]);
    await click(action("WEB-1 One", "archive"));
    (await patched(one)).answer(problem(503, "server_busy"));
    await settled();
    expect(toasts).toEqual([{ type: "error", title: "toast.error", message: "errors.server_busy" }]);
    expect(cards(host)).toEqual(["WEB-1 One"]);
  });

  it("is gone, and so is its selection, once nerve no longer gives the notification (notification.not_found)", async () => {
    await listed([read, one]);
    await act(async () => cardOf(host, "WEB-2 Two")?.querySelector("button")?.click());
    expect(pageState.current.selected?.notificationId).toBe(read.id);
    await click(action("WEB-2 Two", "archive"));
    (await patched(read)).answer(problem(404, "notification.not_found"));
    await settled();
    expect(toasts).toEqual([{ type: "error", title: "toast.error", message: "errors.notification_not_found" }]);
    expect(cards(host)).toEqual(["WEB-1 One"]);
    expect(pageState.current.selected).toBeUndefined();
  });

  it("is out of use while nerve makes a change, so that a second press sends nothing more", async () => {
    await listed([one]);
    await click(action("WEB-1 One", "mark_read"));
    const marked = await patched(one);
    expect(actionsOf("WEB-1 One").map((button) => button.disabled)).toEqual([true, true, true]);
    await click(action("WEB-1 One", "mark_read"));
    await click(action("WEB-1 One", "archive"));
    await answer(marked, { ...one, read_at: "2026-11-20T12:00:00Z" });
    await settled();
    expect(patches(one)).toHaveLength(1);
    expect(actionsOf("WEB-1 One").map((button) => button.disabled)).toEqual([false, false, false]);
  });

  it.each([
    ["made it", () => json(200, { ...one, read_at: "2026-11-20T12:00:00Z" })],
    ["refused", () => problem(503, "server_busy")],
  ])("says nothing once the tab has moved to another account before nerve %s (followInSession)", async (_, reply) => {
    await listed([one]);
    await click(action("WEB-1 One", "mark_read"));
    const marked = await patched(one);
    fixture.session.loginId = "two";
    marked.answer(reply());
    await settled();
    expect(toasts).toEqual([]);
  });
});
