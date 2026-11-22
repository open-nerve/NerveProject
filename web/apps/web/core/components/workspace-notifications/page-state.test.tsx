// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { DEFAULT_FILTERS } from "@/store/notification/filters";
import type { NotificationSelection } from "./page-state";
import { NotificationPageStateProvider, useNotificationPageState } from "./page-state";

// The notifications page's state (P4 spec 2.5, R3, R25): its tab, its filters and the notification selected, which
// the page's layout holds, not a store, so that it ends with the page (W6). The archived and the snoozed are never
// both shown; a change of tab keeps the all tab's types and the selection, which only the notification it names
// clears. The state is read as the page renders it.

const held: { state: ReturnType<typeof useNotificationPageState> | undefined } = { state: undefined };
function Read() {
  held.state = useNotificationPageState();
  return null;
}

let host: HTMLDivElement, root: Root;
globalThis.IS_REACT_ACT_ENVIRONMENT = true;
beforeEach(() => {
  held.state = undefined;
  host = document.createElement("div");
  document.body.append(host);
  root = createRoot(host);
});
afterEach(async () => {
  await act(async () => root.unmount());
  host.remove();
});

const mount = () =>
  act(async () =>
    root.render(
      <NotificationPageStateProvider>
        <Read />
      </NotificationPageStateProvider>
    )
  );
/** The page leaves: nothing of it is mounted. */
const leave = () => act(async () => root.render(null));
/** What the page does to its state, then the state it renders with. */
async function change(does: (state: ReturnType<typeof useNotificationPageState>) => void) {
  await act(async () => does(held.state!));
  return held.state!;
}
const one: NotificationSelection = { notificationId: "n-one", issueId: "i-web-1", projectId: "p-web" };
const two: NotificationSelection = { notificationId: "n-two", issueId: "i-web-2", projectId: "p-web" };

describe("the notifications page's state", () => {
  it("opens on the all tab, every sender, none of the toggles, nothing selected", async () => {
    await mount();
    expect(held.state?.filters).toEqual(DEFAULT_FILTERS);
    expect(held.state?.selected).toBeUndefined();
  });

  it("ends with the page: once it leaves and comes back, the filters are the default ones and nothing is selected (W6)", async () => {
    await mount();
    await change((state) => state.setTab("mentions"));
    await change((state) => state.toggleType("assigned"));
    await change((state) => state.toggleUnread());
    await change((state) => state.toggleArchived());
    await change((state) => state.select(one));
    await leave();
    await mount();
    expect(held.state?.filters).toEqual(DEFAULT_FILTERS);
    expect(held.state?.selected).toBeUndefined();
  });

  it("shows the archived or the snoozed, never both: turning one on turns the other off (R25)", async () => {
    await mount();
    let state = await change((s) => s.toggleArchived());
    expect([state.filters.archived, state.filters.snoozed]).toEqual([true, false]);
    state = await change((s) => s.toggleSnoozed());
    expect([state.filters.archived, state.filters.snoozed]).toEqual([false, true]);
    state = await change((s) => s.toggleArchived());
    expect([state.filters.archived, state.filters.snoozed]).toEqual([true, false]);
    state = await change((s) => s.toggleArchived());
    expect([state.filters.archived, state.filters.snoozed]).toEqual([false, false]);
  });

  it("keeps the all tab's types and the notification selected as the tab changes", async () => {
    await mount();
    await change((state) => state.toggleType("created"));
    await change((state) => state.toggleType("assigned"));
    await change((state) => state.select(one));
    let state = await change((s) => s.setTab("mentions"));
    expect(state.filters).toEqual({ ...DEFAULT_FILTERS, tab: "mentions", types: ["created", "assigned"] });
    expect(state.selected).toEqual(one);
    state = await change((s) => s.setTab("all"));
    expect(state.filters).toEqual({ ...DEFAULT_FILTERS, types: ["created", "assigned"] });
    expect(state.selected).toEqual(one);
  });

  it("turns a type on and off, and clears the types; turns the unread on and off", async () => {
    await mount();
    await change((state) => state.toggleType("assigned"));
    let state = await change((s) => s.toggleType("subscribed"));
    expect(state.filters.types).toEqual(["assigned", "subscribed"]);
    state = await change((s) => s.toggleType("assigned"));
    expect(state.filters.types).toEqual(["subscribed"]);
    state = await change((s) => s.clearTypes());
    expect(state.filters.types).toEqual([]);
    state = await change((s) => s.toggleUnread());
    expect(state.filters.unread).toBe(true);
    state = await change((s) => s.toggleUnread());
    expect(state.filters.unread).toBe(false);
  });

  it("selects a notification in place of another, and clears the selection only of the notification it names", async () => {
    await mount();
    await change((state) => state.select(one));
    let state = await change((s) => s.select(two));
    expect(state.selected).toEqual(two);
    state = await change((s) => s.clearSelection(one.notificationId));
    expect(state.selected).toEqual(two);
    state = await change((s) => s.clearSelection(two.notificationId));
    expect(state.selected).toBeUndefined();
  });
});
