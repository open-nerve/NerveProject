// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { Call } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { toasts } from "@/lib/fake-toast";
import { notificationRequests, notificationTab, pageOf } from "@/store/notification/fake-notifications";
import type { Notification } from "@nerve/api-client";
import { answer, cards, changeState, listRead, noteOf, renderList } from "../fake-notification-list";
import { press } from "../sidebar/fake-header-menus";

// A card's snooze menu (P4 spec 2.6, R15, R16): a menu (CustomMenu) the keyboard opens from its button, moves in by
// the arrows and chooses in by Enter, Escape closing it and giving the focus back to its button. Its presets snooze
// the notification for 1, 3, 5, 7 or 14 days from the moment one is chosen (presetMoment); a snoozed one's menu first
// ends its snooze, which is snoozed_till null (the contract's T1: a field left out changes nothing); custom, last,
// opens the dialog of a time of one's own (snooze-modal.test.tsx). The store and HTTP are real; SWR is, with the
// test's cache.

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
const one = noteOf("One", 1);
const snoozed = noteOf("Later", 3, { snoozed_till: "2026-11-21T08:30:00Z" });
const menuButton = (issue: string) =>
  host.querySelector<HTMLButtonElement>(`button[aria-label="notification.card.actions.snooze ${issue}"]`)!;
const menu = () => document.querySelector<HTMLElement>('[role="menu"]');
const items = () => [...document.querySelectorAll<HTMLButtonElement>('[role="menuitem"]')];
const PRESETS = ["1_day", "3_days", "5_days", "1_week", "2_weeks"].map((preset) => `notification.snooze.${preset}`);
async function patched(n: Notification): Promise<Call> {
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
  return nerve()
    .to(path)
    .find((call) => call.method === "PATCH")!;
}

describe("NotificationSnoozeMenu", () => {
  it("opens by its button's keys, its items the presets, moves by the arrows, and gives the focus back on Escape", async () => {
    await renderList(root, cache);
    await answer(await listRead(nerve(), 0), pageOf([one]));
    await press(menuButton("WEB-1 One"), "Enter");
    expect(items().map((item) => item.textContent)).toEqual([...PRESETS, "notification.snooze.custom"]);
    expect(menu()?.getAttribute("aria-activedescendant")).toBe(items()[0]?.id);
    await press(menu()!, "ArrowDown");
    expect(menu()?.getAttribute("aria-activedescendant")).toBe(items()[1]?.id);
    await press(menu()!, "Escape");
    expect(menu()).toBeNull();
    expect(document.activeElement).toBe(menuButton("WEB-1 One"));
  });

  it("snoozes it for the preset chosen by Enter, from the moment it is chosen, saying so; it leaves the view", async () => {
    await renderList(root, cache);
    await answer(await listRead(nerve(), 0), pageOf([one]));
    await press(menuButton("WEB-1 One"), "Enter");
    await press(menu()!, "ArrowDown");
    // the moment Enter chooses 3 days, hours after the list was read
    vi.setSystemTime(new Date("2026-11-20T15:30:00Z"));
    await press(menu()!, "Enter");
    const snoozing = await patched(one);
    expect(snoozing.body).toEqual({ snoozed_till: "2026-11-23T15:30:00.000Z" });
    await answer(snoozing, { ...one, snoozed_till: "2026-11-23T15:30:00.000Z" });
    expect(toasts).toEqual([{ type: "success", title: "toast.success", message: "notification.toasts.snoozed" }]);
    expect(cards(host)).toEqual([]);
  });

  it("first ends a snoozed one's snooze, sending snoozed_till null (T1), saying so; it leaves the snoozed view", async () => {
    await renderList(root, cache);
    await answer(await listRead(nerve(), 0), pageOf([]));
    await changeState((state) => state.toggleSnoozed());
    await answer(await listRead(nerve(), 1), pageOf([snoozed]));
    await press(menuButton("WEB-3 Later"), "Enter");
    expect(items().map((item) => item.textContent)).toEqual([
      "notification.options.mark_unsnooze",
      ...PRESETS,
      "notification.snooze.custom",
    ]);
    await act(async () => items()[0]?.click());
    const waking = await patched(snoozed);
    expect(waking.body).toEqual({ snoozed_till: null });
    await answer(waking, { ...snoozed, snoozed_till: null });
    expect(toasts).toEqual([{ type: "success", title: "toast.success", message: "notification.toasts.unsnoozed" }]);
    expect(cards(host)).toEqual([]);
  });
});
