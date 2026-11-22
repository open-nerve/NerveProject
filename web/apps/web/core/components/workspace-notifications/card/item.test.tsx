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

// A card of the notifications page's list (P4 spec 2.6, R16, R23): one main button, which selects its notification and
// opens its preview at once (the selection is the preview's, root.tsx) and, for an unread one, queues marking it read,
// which the preview never waits for; a refusal of that says why, the preview open still, but for
// notification.not_found, after which the card is gone and its selection with it. The button says who made the change
// and what (sentence.tsx), a comment's excerpt as plain text, the work item, as it was when the notification was sent,
// and when: till when it is snoozed, while it is (snoozedNow), else how long ago; an unread one's dot says so to a
// screen reader. The store and HTTP are real; SWR is, with the test's cache; the members are a stand-in's (Ann).

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
vi.mock("@/hooks/store/use-member", () => ({
  useMember: () => ({ getUserDetails: (id: string) => (id === "u-ann" ? { id, display_name: "ann" } : undefined) }),
}));
vi.mock("@/hooks/use-profile-image", () => import("@/lib/fake-profile-image"));
// the card's custom snooze dialog reads the profile (DateDropdown); it is never opened here
vi.mock("@/hooks/store/user", () => ({ useUserProfile: () => ({ data: undefined }) }));

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
  it("is one main button that says the work item, selecting the notification and opening its preview at once", async () => {
    await listed(one, two);
    const main = cardOf(host, "WEB-1 One")?.querySelector("button");
    expect(main?.textContent).toContain("WEB-1 One");
    expect(main?.querySelector("button, a")).toBeNull();
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

describe("NotificationItem's words", () => {
  const changed = (name: string, sequence: number, change: Partial<typeof one.data.change>) =>
    noteOf(name, sequence, { data: { ...noteOf(name, sequence).data, change: { ...one.data.change, ...change } } });

  it("says who made the change and what, before the work item", async () => {
    await listed(one);
    expect(cardOf(host, "WEB-1 One")?.textContent).toMatch(/ann notification\.card\.renamed Sign-in.*WEB-1 One/);
  });

  it("shows a comment's excerpt as plain text, in two lines at most, nothing of it rendered as markup", async () => {
    const commented = changed("Talk", 3, {
      field: "comment",
      verb: "created",
      comment_id: "c-1",
      excerpt: "Ship it <img src=x> <b>now</b> @Bob",
      old_value: null,
      new_value: null,
    });
    await listed(commented);
    const card = cardOf(host, "WEB-3 Talk");
    const excerpt = [...(card?.querySelectorAll(".line-clamp-2") ?? [])].find((shown) =>
      shown.textContent?.startsWith("Ship it")
    );
    expect(excerpt?.textContent).toBe("Ship it <img src=x> <b>now</b> @Bob");
    expect(card?.querySelector("img, b")).toBeNull();
    expect(card?.textContent).toContain("ann notification.card.commented");
  });

  it("says a mention in a comment, with the comment's excerpt", async () => {
    const mentioned = changed("Ping", 4, {
      field: "comment",
      verb: "mentioned",
      comment_id: "c-2",
      excerpt: "@ann have a look",
      old_value: null,
      new_value: null,
    });
    await listed({ ...mentioned, sender: "mentioned" });
    const card = cardOf(host, "WEB-4 Ping");
    expect(card?.textContent).toContain("ann notification.card.mentioned_comment");
    expect(card?.textContent).toContain("@ann have a look");
  });

  it("links to no attachment, nor to anything else: its main button holds no other control", async () => {
    const attached = changed("File", 5, { field: "attachment", verb: "created", old_value: null, new_value: null });
    await listed(attached);
    const card = cardOf(host, "WEB-5 File");
    expect(card?.textContent).toContain("ann notification.card.attachment_added");
    expect(card?.querySelector("a")).toBeNull();
    expect(card?.querySelector("button")?.querySelector("button, a")).toBeNull();
  });
});
