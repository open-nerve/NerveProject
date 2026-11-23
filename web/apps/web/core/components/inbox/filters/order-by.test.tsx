// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { press } from "@/components/workspace-notifications/sidebar/fake-header-menus";
import { listQuery } from "@/store/issue/intake/filters";
import { changeFilters, filterState, renderFilterPart } from "./fake-filter-state";
import { IntakeOrderBy } from "./order-by";

// The order of the list of the intake page's tab (P5a spec 2.2 C4, 2.7): a menu its button opens by the keyboard,
// Escape closing it and giving the focus back; its items the three fields, then the two directions. A field chosen
// keeps the direction, a direction chosen keeps the field: the two make one of nerve's six orders, the newest first
// until the user chooses, the one the list's query sends. Its button is named by the field and the direction.
// @nerve/i18n's text is its key, then the values it was given.

vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n-values"));

let host: HTMLDivElement, root: Root;
globalThis.IS_REACT_ACT_ENVIRONMENT = true;
beforeEach(() => {
  vi.useFakeTimers();
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

const render = () => renderFilterPart(root, <IntakeOrderBy />);
const menuButton = () => host.querySelector<HTMLButtonElement>("button[aria-label]")!;
const items = () => [...document.querySelectorAll<HTMLButtonElement>('[role="menuitem"]')];
const item = (label: string) => items().find((found) => found.textContent === label)!;
/** The order chosen by the item that says label, the menu opened by its button's Enter, the item by Enter. */
async function choose(label: string) {
  await press(menuButton(), "Enter");
  await press(item(label), "Enter");
}
const named = (field: string, direction: "asc" | "desc") =>
  `inbox_issue.a11y.order_by inbox_issue.order_by.${field} common.order_by.${direction}`;

describe("IntakeOrderBy", () => {
  it("opens on the newest first, its button named by the field and the direction", async () => {
    await render();
    expect(filterState.current.filters.order).toBe("-created_at");
    expect(menuButton().getAttribute("aria-label")).toBe(named("created_at", "desc"));
  });

  it("opens by its button's keys, the three fields then the two directions, Escape giving the focus back", async () => {
    await render();
    await press(menuButton(), "Enter");
    expect(items().map((found) => found.textContent)).toEqual([
      "inbox_issue.order_by.created_at",
      "inbox_issue.order_by.updated_at",
      "inbox_issue.order_by.id",
      "common.order_by.asc",
      "common.order_by.desc",
    ]);
    await press(document.querySelector<HTMLElement>('[role="menu"]')!, "Escape");
    expect(items()).toEqual([]);
    expect(document.activeElement).toBe(menuButton());
  });

  it("keeps the direction as a field is chosen and the field as a direction is: one of nerve's six (C4)", async () => {
    await render();
    await choose("inbox_issue.order_by.updated_at");
    expect(filterState.current.filters.order).toBe("-updated_at");
    await choose("common.order_by.asc");
    expect(filterState.current.filters.order).toBe("updated_at");
    await choose("inbox_issue.order_by.id");
    expect(filterState.current.filters.order).toBe("sequence_id");
    expect(menuButton().getAttribute("aria-label")).toBe(named("id", "asc"));
    await choose("common.order_by.desc");
    expect(filterState.current.filters.order).toBe("-sequence_id");
    await choose("inbox_issue.order_by.created_at");
    expect(listQuery(filterState.current.filters).order_by).toBe("-created_at");
  });

  it("keeps the filters as the order changes", async () => {
    await render();
    await changeFilters({ statuses: ["pending", "snoozed"], priorities: ["high"] });
    await choose("common.order_by.asc");
    expect(filterState.current.filters).toMatchObject({
      statuses: ["pending", "snoozed"],
      priorities: ["high"],
      order: "created_at",
    });
  });
});
