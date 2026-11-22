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
import { answer, cardOf, cards, changeState, listRead, noteOf, renderList } from "../fake-notification-list";
import { press } from "../sidebar/fake-header-menus";

// The custom snooze dialog (P4 spec 2.7, R15), which a card's snooze menu opens and the card holds: named by its title;
// a day, today at the earliest, and one of its half hours, today's after now alone, each said in the page's language;
// either missing says so under it, the first refused taking the focus; it sends the moment of the day's half hour in
// the browser's zone, closing once nerve has snoozed it, saying so; nerve's must_be_future shows under the time, any
// other refusal in a toast, the dialog open. It cannot be closed while nerve snoozes; closed, the focus goes back to
// the row's snooze button. What follows nerve's answer is done only in the session it was sent in. The store and HTTP
// are real; SWR is, with the test's cache. @nerve/i18n's text is its key, then the values it was given; the page's
// language is English.

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
vi.mock("@nerve/i18n", async () => {
  const { useTranslation } = await import("@/lib/fake-i18n-values");
  return { useTranslation: () => ({ ...useTranslation(), language: "en" }) };
});
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@/hooks/store/use-workspace", () => ({ useWorkspace: () => fixture.tab.workspaceRoot }));
vi.mock("@/hooks/store/use-notifications", () => ({ useNotifications: () => fixture.tab.store }));
vi.mock("@/hooks/store/use-member", () => ({ useMember: () => ({ getUserDetails: () => undefined }) }));
vi.mock("@/hooks/use-profile-image", () => import("@/lib/fake-profile-image"));
vi.mock("@/hooks/store/user", () => ({ useUserProfile: () => ({ data: { start_of_the_week: 1 } }) }));

let host: HTMLDivElement, root: Root, cache: Map<string, unknown>;
globalThis.IS_REACT_ACT_ENVIRONMENT = true;
beforeEach(async () => {
  // 20 November 2026, 15:10 in the browser's zone
  vi.useFakeTimers({ now: new Date(2026, 10, 20, 15, 10) });
  vi.spyOn(Element.prototype, "getAnimations").mockReturnValue([]);
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
const settled = () => act(async () => vi.advanceTimersByTimeAsync(400));
const one = noteOf("One", 1);
/** One snoozed till 21 November, 09:00: the snoozed view lists it, and lists it still snoozed till another time. */
const later = noteOf("One", 1, { snoozed_till: new Date(2026, 10, 21, 9, 0).toISOString() });
const snoozeButton = () =>
  host.querySelector<HTMLButtonElement>('button[aria-label="notification.card.actions.snooze WEB-1 One"]')!;
const dialog = () => document.querySelector<HTMLElement>('[role="dialog"]');
const inDialog = (selector: string) => dialog()?.querySelector<HTMLElement>(selector) ?? null;
const dateButton = () => inDialog('button[aria-label="notification.snooze_modal.date"]') as HTMLButtonElement;
const timeButton = () => inDialog('button[aria-haspopup="listbox"]') as HTMLButtonElement;
const buttonSaying = (text: string) =>
  [...(dialog()?.querySelectorAll("button") ?? [])].find((button) => button.textContent === text);
const day = (date: string) => document.querySelector<HTMLButtonElement>(`[data-day="${date}"] button`)!;
const options = () =>
  [...document.querySelectorAll<HTMLElement>('[role="option"]')].map((option) =>
    option.textContent?.replace(/\s/g, " ")
  );
const option = (label: string) =>
  [...document.querySelectorAll<HTMLElement>('[role="option"]')].find(
    (found) => found.textContent?.replace(/\s/g, " ") === label
  );
const describedBy = (control: HTMLElement) =>
  document.getElementById(control.getAttribute("aria-describedby") ?? "")?.textContent;
/** Escape, on what has the focus, as a browser gives it. */
const escape = () =>
  act(async () =>
    (document.activeElement ?? document.body).dispatchEvent(
      new KeyboardEvent("keydown", { key: "Escape", bubbles: true, cancelable: true })
    )
  );
/** A click on the backdrop, outside the dialog, as a pointer makes it. */
async function clickOutside() {
  const backdrop = document.querySelector<HTMLElement>('[class*="bg-backdrop"]');
  expect(backdrop).not.toBeNull();
  await act(async () => {
    for (const type of ["pointerdown", "mousedown", "pointerup", "mouseup"])
      backdrop!.dispatchEvent(new MouseEvent(type, { bubbles: true, cancelable: true }));
    backdrop!.click();
  });
}
async function patched(): Promise<Call> {
  const path = notificationRequests.at(one);
  await act(async () =>
    until(
      () =>
        nerve()
          .to(path)
          .some((call) => call.method === "PATCH"),
      "the change"
    )
  );
  return nerve()
    .to(path)
    .find((call) => call.method === "PATCH")!;
}

/**
 * The list shows one (later, in the snoozed view); its snooze menu's last item, custom, chosen by the keyboard, opens
 * the dialog.
 */
async function opened(view: "default" | "snoozed" = "default") {
  await renderList(root, cache);
  if (view === "default") await answer(await listRead(nerve(), 0), pageOf([one]));
  else {
    await answer(await listRead(nerve(), 0), pageOf([]));
    await changeState((state) => state.toggleSnoozed());
    await answer(await listRead(nerve(), 1), pageOf([later]));
  }
  await press(snoozeButton(), "Enter");
  const items = [...document.querySelectorAll('[role="menuitem"]')].map((item) => item.textContent);
  expect(items.at(-1)).toBe("notification.snooze.custom");
  await press(document.querySelector<HTMLElement>('[role="menu"]')!, "End");
  await press(document.querySelector<HTMLElement>('[role="menu"]')!, "Enter");
  await settled();
  expect(dialog()).not.toBeNull();
}

/** The day picked in the dialog's calendar, and the half hour of label in its list of times. */
async function picked(date: string, label?: string) {
  await act(async () => dateButton().click());
  await act(async () => day(date).click());
  await settled();
  if (label === undefined) return;
  await press(timeButton(), "Enter");
  await act(async () => option(label)?.click());
  await settled();
}

/** The submit button pressed by the pointer, which gives it the focus first. */
const submit = () =>
  act(async () => {
    const button = buttonSaying("notification.snooze_modal.submit");
    button?.focus();
    button?.click();
  });

describe("NotificationSnoozeModal", () => {
  it("opens from the snooze menu's last item, named by its title", async () => {
    await opened();
    const title = document.getElementById(dialog()!.getAttribute("aria-labelledby") ?? "");
    expect(title?.textContent).toBe("notification.snooze_modal.title");
  });

  it("says each field it needs under it, the day taking the focus first, and sends nothing", async () => {
    await opened();
    await submit();
    await settled();
    for (const control of [dateButton(), timeButton()]) {
      expect(control.getAttribute("aria-invalid")).toBe("true");
      expect(describedBy(control)).toBe("common.errors.required");
    }
    expect(document.activeElement).toBe(dateButton());
    expect(nerve().to(notificationRequests.at(one))).toEqual([]);
  });

  it("offers today's half hours after now alone, any other day's all of them, each in the page's language", async () => {
    await opened();
    await picked("2026-11-20");
    await press(timeButton(), "Enter");
    expect(options()).toHaveLength(17);
    expect(options()[0]).toBe("3:30 PM");
    await escape();
    await settled();
    expect(dialog()).not.toBeNull();
    await picked("2026-11-21");
    await press(timeButton(), "Enter");
    expect(options()).toHaveLength(48);
    expect(options().slice(0, 2)).toEqual(["12:00 AM", "12:30 AM"]);
    expect(options()[24]).toBe("12:00 PM");
  });

  it("no longer holds a half hour the day then picked does not offer", async () => {
    await opened();
    await picked("2026-11-21", "12:30 AM");
    expect(timeButton().textContent?.replace(/\s/g, " ")).toBe("12:30 AM");
    await picked("2026-11-20");
    expect(timeButton().textContent).toBe("notification.snooze_modal.time");
    await picked("2026-11-21", "11:30 PM");
    await picked("2026-11-20");
    expect(timeButton().textContent?.replace(/\s/g, " ")).toBe("11:30 PM");
  });

  it("says no half hour is left today when none is", async () => {
    vi.setSystemTime(new Date(2026, 10, 20, 23, 45));
    await opened();
    await picked("2026-11-20");
    await press(timeButton(), "Enter");
    expect(options()).toEqual([]);
    expect(document.body.textContent).toContain("notification.snooze_modal.no_time");
  });

  it("snoozes it till the day's half hour in the browser's zone, leaving the default view", async () => {
    await opened();
    await picked("2026-11-21", "12:30 AM");
    await submit();
    const snoozing = await patched();
    expect(snoozing.body).toEqual({ snoozed_till: new Date(2026, 10, 21, 0, 30).toISOString() });
    await answer(snoozing, { ...one, snoozed_till: new Date(2026, 10, 21, 0, 30).toISOString() });
    await settled();
    expect(toasts).toEqual([{ type: "success", title: "toast.success", message: "notification.toasts.snoozed" }]);
    expect(cards(host)).toEqual([]);
  });

  it("closes once nerve has snoozed it, saying so, the focus back on the row's snooze button", async () => {
    await opened("snoozed");
    await picked("2026-11-22", "12:30 AM");
    await submit();
    const till = new Date(2026, 10, 22, 0, 30).toISOString();
    const snoozing = await patched();
    expect(snoozing.body).toEqual({ snoozed_till: till });
    await answer(snoozing, { ...later, snoozed_till: till });
    await settled();
    expect(dialog()).toBeNull();
    expect(toasts).toEqual([{ type: "success", title: "toast.success", message: "notification.toasts.snoozed" }]);
    // still listed, snoozed till the new time
    expect(cardOf(host, "WEB-1 One")).toBeDefined();
    expect(document.activeElement).toBe(snoozeButton());
  });

  it("shows nerve's must_be_future under the time, which takes the focus, the dialog open", async () => {
    await opened();
    await picked("2026-11-20", "3:30 PM");
    await submit();
    (await patched()).answer(
      json(422, {
        status: 422,
        code: "validation_failed",
        title: "",
        detail: "",
        errors: [{ field: "snoozed_till", code: "must_be_future" }],
      })
    );
    await settled();
    expect(timeButton().getAttribute("aria-invalid")).toBe("true");
    expect(describedBy(timeButton())).toBe("errors.field.must_be_future");
    expect(document.activeElement).toBe(timeButton());
    expect(toasts).toEqual([]);
    expect(dialog()).not.toBeNull();
  });

  it("says any other refusal in a toast, the dialog open", async () => {
    await opened();
    await picked("2026-11-21", "12:30 AM");
    await submit();
    (await patched()).answer(problem(503, "server_busy"));
    await settled();
    expect(toasts).toEqual([{ type: "error", title: "toast.error", message: "errors.server_busy" }]);
    expect(dialog()).not.toBeNull();
  });

  it("cannot be closed while nerve snoozes, by Escape, a click outside or Cancel", async () => {
    await opened();
    await picked("2026-11-21", "12:30 AM");
    await submit();
    const snoozing = await patched();
    await escape();
    await clickOutside();
    await act(async () => buttonSaying("common.cancel")?.click());
    await settled();
    expect(dialog()).not.toBeNull();
    expect(buttonSaying("common.cancel")?.disabled).toBe(true);
    await answer(snoozing, { ...one, snoozed_till: new Date(2026, 10, 21, 0, 30).toISOString() });
    await settled();
    expect(dialog()).toBeNull();
  });

  it.each([
    ["Escape", escape],
    ["Cancel", () => act(async () => buttonSaying("common.cancel")?.click())],
  ])("closes by %s, the focus back on the row's snooze button", async (_, close) => {
    await opened();
    await close();
    await settled();
    expect(dialog()).toBeNull();
    expect(document.activeElement).toBe(snoozeButton());
  });

  it.each([
    ["snoozed it", () => json(200, { ...later, snoozed_till: new Date(2026, 10, 22, 0, 30).toISOString() })],
    ["refused", () => problem(503, "server_busy")],
  ])("neither says nor closes anything once the tab has moved to another account before nerve %s", async (_, reply) => {
    await opened("snoozed");
    await picked("2026-11-22", "12:30 AM");
    await submit();
    const snoozing = await patched();
    fixture.session.loginId = "two";
    snoozing.answer(reply());
    await settled();
    expect(toasts).toEqual([]);
    expect(dialog()).not.toBeNull();
  });
});
