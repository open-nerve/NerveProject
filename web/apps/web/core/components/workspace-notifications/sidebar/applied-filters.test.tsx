// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { NotificationAppliedFilters } from "./applied-filters";
import { changeHeadState, headState, renderHeadPart } from "./fake-header-menus";

// The senders the notifications page's all tab is filtered by (P4 spec 2.6, R10): a pill for each, in the order of the
// type filter, a button named as removing it; and "clear all". Nothing while none is chosen, nor in the mentions tab,
// whose list takes no type; the all tab's choice stays for when it is back. @nerve/i18n's text is its key, then the
// values it was given.

vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n-values"));

let host: HTMLDivElement, root: Root;
const submitted = vi.fn();
globalThis.IS_REACT_ACT_ENVIRONMENT = true;
beforeEach(() => {
  submitted.mockClear();
  host = document.createElement("div");
  document.body.append(host);
  root = createRoot(host);
});
afterEach(async () => {
  await act(async () => root.unmount());
  host.remove();
});

const render = () => renderHeadPart(root, <NotificationAppliedFilters />, submitted);
/** The buttons shown, each by its name (its aria-label, else its text). */
const buttons = () =>
  [...host.querySelectorAll("button")].map((button) => button.getAttribute("aria-label") ?? button.textContent);
const button = (name: string) =>
  [...host.querySelectorAll("button")].find(
    (found) => (found.getAttribute("aria-label") ?? found.textContent) === name
  );
const REMOVE_ASSIGNED = "common.remove_item notification.filter.assigned";
const REMOVE_CREATED = "common.remove_item notification.filter.created";
const REMOVE_SUBSCRIBED = "common.remove_item notification.filter.subscribed";

describe("NotificationAppliedFilters", () => {
  it("shows nothing while no sender is chosen", async () => {
    await render();
    expect(host.textContent).toBe("");
  });

  it("shows a pill for each sender chosen, in the type filter's order, and clear all", async () => {
    await render();
    await changeHeadState((state) => {
      state.toggleType("subscribed");
      state.toggleType("assigned");
    });
    expect(buttons()).toEqual([REMOVE_ASSIGNED, REMOVE_SUBSCRIBED, "common.clear_all"]);
    expect(button(REMOVE_ASSIGNED)?.textContent).toBe("notification.filter.assigned");
    expect(button(REMOVE_ASSIGNED)?.getAttribute("type")).toBe("button");
  });

  it("removes a sender as its pill is pressed, and every one by clear all, submitting no form", async () => {
    await render();
    await changeHeadState((state) => {
      state.toggleType("assigned");
      state.toggleType("created");
      state.toggleType("subscribed");
    });
    await act(async () => button(REMOVE_CREATED)?.click());
    expect(headState.current.filters.types).toEqual(["assigned", "subscribed"]);
    expect(buttons()).toEqual([REMOVE_ASSIGNED, REMOVE_SUBSCRIBED, "common.clear_all"]);
    await act(async () => button("common.clear_all")?.click());
    expect(headState.current.filters.types).toEqual([]);
    expect(host.textContent).toBe("");
    expect(submitted).not.toHaveBeenCalled();
  });

  it("shows nothing in the mentions tab, and the all tab's choice again back in the all tab", async () => {
    await render();
    await changeHeadState((state) => state.toggleType("created"));
    await changeHeadState((state) => state.setTab("mentions"));
    expect(host.textContent).toBe("");
    await changeHeadState((state) => state.setTab("all"));
    expect(buttons()).toEqual([REMOVE_CREATED, "common.clear_all"]);
  });
});
