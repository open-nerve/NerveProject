// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { Notification, NotificationPage } from "@nerve/api-client";
import type { Call } from "@/lib/auth/fake-nerve";
import { json, problem } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { toasts } from "@/lib/fake-toast";
import { acme, notificationRequests, notificationTab, pageOf } from "@/store/notification/fake-notifications";
import { answer, cardOf, cards, changeState, listRead, noteOf, renderList } from "../fake-notification-list";
import { press } from "./fake-header-menus";

// Where the focus goes as a card that holds it leaves the list (P4 spec 2.6, R16, R25): the card's own archive or
// snooze, which the view no longer lists, or nerve's notification.not_found, gives it to the next card's main button,
// else the previous card's, else the list's region (its empty state, or "Load more"), never to the page's body. A read
// change leaves the card in its place (R8) and the focus with it; another card leaving moves it not, nor a card the
// focus left before it did. The custom
// dialog's snooze is snooze-modal.test.tsx's. The store and HTTP are real; SWR is, with the test's cache. @nerve/i18n's
// text is its key, then the values it was given.

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
// the card's custom snooze dialog reads the profile (DateDropdown); it is never opened here
vi.mock("@/hooks/store/user", () => ({ useUserProfile: () => ({ data: undefined }) }));

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
const NOW = "2026-11-20T12:00:00Z";
const [one, two, three] = [noteOf("One", 1), noteOf("Two", 2), noteOf("Three", 3)];
/** The card's main button, its first. */
const main = (issue: string) => cardOf(host, issue)?.querySelector("button");
const action = (issue: string, name: string) =>
  host.querySelector<HTMLButtonElement>(`button[aria-label="notification.card.actions.${name} ${issue}"]`)!;
/** The button, focused and pressed as a pointer presses it. */
const pressed = (button: HTMLElement) =>
  act(async () => {
    button.focus();
    button.click();
  });
/** The change of the notification nerve receives, answered as reply says once it is. */
async function changed(n: Notification, reply: Response) {
  const path = notificationRequests.at(n);
  await act(async () =>
    until(
      () =>
        nerve()
          .to(path)
          .some((call) => call.method === "PATCH"),
      "the change"
    )
  );
  const call: Call = nerve()
    .to(path)
    .find((found) => found.method === "PATCH")!;
  call.answer(reply);
  await act(async () => vi.advanceTimersByTimeAsync(50));
}

/** The list at acme's address shows the page nerve gives in the view change makes, if any. */
async function listed(page: NotificationPage, change?: Parameters<typeof changeState>[0]) {
  await renderList(root, cache);
  if (change === undefined) return answer(await listRead(nerve(), 0), page);
  await answer(await listRead(nerve(), 0), pageOf([]));
  await changeState(change);
  await answer(await listRead(nerve(), 1), page);
}

it("gives the focus to the next card's main button once the card's own archive takes it from the view", async () => {
  await listed(pageOf([one, two, three]));
  await pressed(action("WEB-2 Two", "archive"));
  await changed(two, json(200, { ...two, archived_at: NOW }));
  expect(cards(host)).toEqual(["WEB-1 One", "WEB-3 Three"]);
  expect(document.activeElement).toBe(main("WEB-3 Three"));
});

it("gives it to the previous card's when no card follows", async () => {
  await listed(pageOf([one, two, three]));
  await pressed(action("WEB-3 Three", "archive"));
  await changed(three, json(200, { ...three, archived_at: NOW }));
  expect(document.activeElement).toBe(main("WEB-2 Two"));
});

it("gives it to the list once no card is left: its empty state, or its Load more", async () => {
  await listed(pageOf([one]));
  await pressed(action("WEB-1 One", "archive"));
  await changed(one, json(200, { ...one, archived_at: NOW }));
  const region = document.activeElement as HTMLElement;
  // the list's own element, never the page's body
  expect(region).not.toBe(document.body);
  expect(host.contains(region)).toBe(true);
  expect(region.tabIndex).toBe(-1);
  expect(region.textContent).toContain("workspace_empty_state.inbox_sidebar_all.title");

  await act(async () => root.unmount());
  root = createRoot(host);
  cache = new Map();
  fixture.tab = await notificationTab();
  await listed(pageOf([two], "c-1"));
  await pressed(action("WEB-2 Two", "archive"));
  await changed(two, json(200, { ...two, archived_at: NOW }));
  const list = document.activeElement as HTMLElement;
  expect(list).not.toBe(document.body);
  expect(host.contains(list)).toBe(true);
  expect(list.tabIndex).toBe(-1);
  expect(list.querySelector("button")?.textContent).toBe("common.load_more");
});

it("gives it to the next card once a preset snooze, chosen by the keyboard, takes the card from the view", async () => {
  await listed(pageOf([one, two, three]));
  await press(action("WEB-1 One", "snooze"), "Enter");
  await press(document.querySelector<HTMLElement>('[role="menu"]')!, "Enter");
  await changed(one, json(200, { ...one, snoozed_till: "2026-11-21T12:00:00Z" }));
  expect(cards(host)).toEqual(["WEB-2 Two", "WEB-3 Three"]);
  expect(document.activeElement).toBe(main("WEB-2 Two"));
});

it("gives it to the next card once nerve no longer gives the notification (notification.not_found)", async () => {
  await listed(pageOf([one, two]));
  await pressed(main("WEB-1 One")!);
  await changed(one, problem(404, "notification.not_found"));
  expect(cards(host)).toEqual(["WEB-2 Two"]);
  expect(document.activeElement).toBe(main("WEB-2 Two"));
});

it("keeps it where it is as a read change leaves the card in its place (R8)", async () => {
  await listed(pageOf([one, two]), (state) => state.toggleUnread());
  const mark = action("WEB-1 One", "mark_read");
  await pressed(mark);
  await changed(one, json(200, { ...one, read_at: NOW }));
  expect(cards(host)).toEqual(["WEB-1 One", "WEB-2 Two"]);
  expect(document.activeElement).toBe(mark);
  expect(mark.getAttribute("aria-label")).toBe("notification.card.actions.mark_unread WEB-1 One");
});

it("leaves it where it went once the card no longer held it as it left", async () => {
  await renderList(root, cache, <button type="button">elsewhere</button>);
  await answer(await listRead(nerve(), 0), pageOf([one, two, three]));
  await pressed(action("WEB-2 Two", "archive"));
  const elsewhere = [...host.querySelectorAll("button")].find((button) => button.textContent === "elsewhere")!;
  await act(async () => elsewhere.focus());
  await changed(two, json(200, { ...two, archived_at: NOW }));
  expect(cards(host)).toEqual(["WEB-1 One", "WEB-3 Three"]);
  expect(document.activeElement).toBe(elsewhere);
});

it("knows the card holds it still when a blur's listener gives it back into the card before the list hears the blur", async () => {
  await listed(pageOf([one, two, three]));
  const archive = action("WEB-2 Two", "archive");
  await act(async () => archive.focus());
  // as the dialog's keepFocusIn does in a browser, whose microtask runs between the blur's listeners (a browser's own
  // event): the card's main button has the focus before the list's listener hears the archive button lose it
  archive.addEventListener("focusout", () => main("WEB-2 Two")?.focus(), { once: true });
  await act(async () => archive.blur());
  expect(document.activeElement).toBe(main("WEB-2 Two"));
  const archiving = fixture.tab.store.update(acme, two.id, { archived: true });
  await changed(two, json(200, { ...two, archived_at: NOW }));
  await archiving;
  expect(cards(host)).toEqual(["WEB-1 One", "WEB-3 Three"]);
  expect(document.activeElement).toBe(main("WEB-3 Three"));
});

/**
 * A browser's blur of a control that holds the focus as it is disabled (Chromium, Firefox): to nothing, the document's
 * body having it. jsdom blurs no disabled control, so the control is let go of for the blur.
 */
const disabledBlur = (button: HTMLButtonElement) =>
  act(async () => {
    expect(button.disabled).toBe(true);
    button.disabled = false;
    button.blur();
    button.disabled = true;
  });

it("gives it to the next card once the card leaves though its pressed archive, disabled while out, lost it to nothing", async () => {
  await listed(pageOf([one, two, three]));
  const archive = action("WEB-2 Two", "archive");
  await pressed(archive);
  await disabledBlur(archive);
  await changed(two, json(200, { ...two, archived_at: NOW }));
  expect(cards(host)).toEqual(["WEB-1 One", "WEB-3 Three"]);
  expect(document.activeElement).toBe(main("WEB-3 Three"));
});

it("gives it back to the pressed control once it is enabled again, the card in its place, having lost it to nothing", async () => {
  await listed(pageOf([one, two]), (state) => state.toggleUnread());
  const mark = action("WEB-1 One", "mark_read");
  await pressed(mark);
  await disabledBlur(mark);
  expect(document.activeElement).toBe(document.body);
  await changed(one, json(200, { ...one, read_at: NOW }));
  expect(document.activeElement).toBe(mark);
  expect(mark.disabled).toBe(false);
});

it("gives it back to the snooze menu's button once nerve refuses its preset, the card in its place, the menu gone (O4)", async () => {
  await listed(pageOf([one, two]));
  const snooze = action("WEB-1 One", "snooze");
  await press(snooze, "Enter");
  await press(document.querySelector<HTMLElement>('[role="menu"]')!, "Enter");
  // the menu went with the focus, its button disabled while the snooze is out: the focus has nowhere to go back to
  expect(snooze.disabled).toBe(true);
  expect(document.activeElement).toBe(document.body);
  await changed(one, problem(503, "server_busy"));
  expect(cards(host)).toEqual(["WEB-1 One", "WEB-2 Two"]);
  expect(snooze.disabled).toBe(false);
  expect(document.activeElement).toBe(snooze);
});

it("gives it back to the snooze menu's button once nerve refuses its preset chosen by the pointer, which focused the item (O4)", async () => {
  await listed(pageOf([one, two]));
  const snooze = action("WEB-1 One", "snooze");
  // the pointer opens the menu: its button takes no focus, the menu does
  await act(async () => {
    snooze.dispatchEvent(new MouseEvent("pointerdown", { bubbles: true, cancelable: true }));
    snooze.dispatchEvent(new MouseEvent("mousedown", { bubbles: true, cancelable: true }));
    snooze.click();
  });
  await act(async () => vi.advanceTimersByTimeAsync(200));
  expect(document.activeElement?.getAttribute("role")).toBe("menu");
  const day = [...document.querySelectorAll<HTMLButtonElement>('[role="menuitem"]')].find(
    (item) => item.textContent === "notification.snooze.1_day"
  )!;
  // as a browser's press of a button does, the item's button has the focus before its click chooses it
  await act(async () => {
    day.focus();
    day.click();
  });
  await act(async () => vi.advanceTimersByTimeAsync(200));
  expect(snooze.disabled).toBe(true);
  expect(document.activeElement).toBe(document.body);
  await changed(one, problem(503, "server_busy"));
  expect(cards(host)).toEqual(["WEB-1 One", "WEB-2 Two"]);
  expect(document.activeElement).toBe(snooze);
});

it("leaves it where it went meanwhile, the pressed control having lost it to nothing", async () => {
  await renderList(root, cache, <button type="button">elsewhere</button>);
  await answer(await listRead(nerve(), 0), pageOf([one, two, three]));
  const archive = action("WEB-2 Two", "archive");
  await pressed(archive);
  await disabledBlur(archive);
  const elsewhere = [...host.querySelectorAll("button")].find((button) => button.textContent === "elsewhere")!;
  await act(async () => elsewhere.focus());
  await changed(two, json(200, { ...two, archived_at: NOW }));
  expect(cards(host)).toEqual(["WEB-1 One", "WEB-3 Three"]);
  expect(document.activeElement).toBe(elsewhere);
});

it("leaves it where it went meanwhile as the pressed control, having lost it to nothing, is enabled again in its card", async () => {
  await renderList(root, cache, <button type="button">elsewhere</button>);
  await answer(await listRead(nerve(), 0), pageOf([one, two]));
  const mark = action("WEB-1 One", "mark_read");
  await pressed(mark);
  await disabledBlur(mark);
  const elsewhere = [...host.querySelectorAll("button")].find((button) => button.textContent === "elsewhere")!;
  await act(async () => elsewhere.focus());
  await changed(one, json(200, { ...one, read_at: NOW }));
  expect(cards(host)).toEqual(["WEB-1 One", "WEB-2 Two"]);
  expect(mark.disabled).toBe(false);
  expect(document.activeElement).toBe(elsewhere);
});

/** The watches of elements' disabled the page makes from now on: each with its element, and whether it was ended. */
function watchesMade() {
  const made: { element: Node; ended: boolean }[] = [];
  const { observe, disconnect } = MutationObserver.prototype;
  const of = new WeakMap<MutationObserver, (typeof made)[number]>();
  vi.spyOn(MutationObserver.prototype, "observe").mockImplementation(
    function (this: MutationObserver, element, options) {
      const watch = { element, ended: false };
      made.push(watch);
      of.set(this, watch);
      observe.call(this, element, options);
    }
  );
  vi.spyOn(MutationObserver.prototype, "disconnect").mockImplementation(function (this: MutationObserver) {
    const watch = of.get(this);
    if (watch) watch.ended = true;
    disconnect.call(this);
  });
  return made;
}

it("ends the watch of a card's control once the card leaves, the focus gone elsewhere before (re-review m1)", async () => {
  await renderList(root, cache, <button type="button">elsewhere</button>);
  await answer(await listRead(nerve(), 0), pageOf([one, two]));
  const watches = watchesMade();
  const archive = action("WEB-1 One", "archive");
  await act(async () => archive.focus());
  const elsewhere = [...host.querySelectorAll("button")].find((button) => button.textContent === "elsewhere")!;
  await act(async () => elsewhere.focus());
  const archiving = fixture.tab.store.update(acme, one.id, { archived: true });
  await changed(one, json(200, { ...one, archived_at: NOW }));
  await archiving;
  expect(cards(host)).toEqual(["WEB-2 Two"]);
  expect(document.activeElement).toBe(elsewhere);
  // nothing watches the control the card took away: no detached element is kept
  expect(watches.filter((watch) => watch.element === archive && !watch.ended)).toEqual([]);
});

it("keeps it where it is as another card leaves the list", async () => {
  await listed(pageOf([one, two, three]));
  await act(async () => main("WEB-1 One")?.focus());
  const archiving = fixture.tab.store.update(acme, two.id, { archived: true });
  await changed(two, json(200, { ...two, archived_at: NOW }));
  await archiving;
  expect(cards(host)).toEqual(["WEB-1 One", "WEB-3 Three"]);
  expect(document.activeElement).toBe(main("WEB-1 One"));
});
