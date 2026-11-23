// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { press } from "@/components/workspace-notifications/sidebar/fake-header-menus";
import { defaultFilters, presetRange } from "@/store/issue/intake/filters";
import { IntakeAppliedFilters } from "./applied";
import { changeFilters, filterState, renderFilterPart } from "./fake-filter-state";

// The filters applied to the list of the intake page's tab (P5a spec 2.7, R9; the property applied-filters.test.tsx:50
// pinned until T6b): nothing while they are the tab's first ones, an order of the user's own aside; else each filter,
// a group named by it, each of its values by its name with a native button named as removing that value, which
// removes it alone by the pointer, Enter or Space and never submits a form around it; the last status has none (a list
// names a status always). "Clear all" goes back to the tab's first filters, the order kept. Members and labels are the
// route project's, by their names; one the page no longer knows says so. @nerve/i18n's text is its key, then the
// values it was given.

vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n-values"));
vi.mock("@/hooks/store/use-member", () => ({
  useMember: () => ({
    getUserDetails: (id: string) => ({ "u-ann": { display_name: "Ann" }, "u-bob": { display_name: "Bob" } })[id],
  }),
}));
vi.mock("@/hooks/store/use-label", () => ({
  useLabel: () => ({
    getProjectLabels: (projectId: string) =>
      projectId === "p-web" ? [{ id: "l-bug", name: "Bug", color: "#f00" }] : undefined,
  }),
}));

let host: HTMLDivElement, root: Root;
const submitted = vi.fn();
const now = new Date("2026-11-23T20:00:00Z");
globalThis.IS_REACT_ACT_ENVIRONMENT = true;
beforeEach(() => {
  vi.useFakeTimers({ now });
  vi.stubEnv("TZ", "America/Los_Angeles");
  submitted.mockClear();
  host = document.createElement("div");
  document.body.append(host);
  root = createRoot(host);
});
afterEach(async () => {
  await act(async () => root.unmount());
  host.remove();
  vi.restoreAllMocks();
  vi.unstubAllEnvs();
  vi.useRealTimers();
});

const render = (search?: string) =>
  renderFilterPart(root, <IntakeAppliedFilters projectId="p-web" />, submitted, search);
/** Each filter applied: its group's name, and each value's name with its button's name (none for one that stays). */
const shown = () =>
  [...host.querySelectorAll<HTMLElement>('[role="group"]')].map((group) => [
    document.getElementById(group.getAttribute("aria-labelledby") ?? "")?.textContent,
    [...group.querySelectorAll(":scope > span:not([id])")].map((chip) => [
      chip.querySelector("span")?.textContent,
      chip.querySelector("button")?.getAttribute("aria-label") ?? null,
    ]),
  ]);
const button = (name: string, within: ParentNode = host) =>
  [...within.querySelectorAll("button")].find(
    (found) => (found.getAttribute("aria-label") ?? found.textContent) === name
  );
const group = (title: string) =>
  [...host.querySelectorAll<HTMLElement>('[role="group"]')].find(
    (found) => document.getElementById(found.getAttribute("aria-labelledby") ?? "")?.textContent === title
  )!;
const removing = (name: string) => `common.remove_item ${name}`;
const status = (name: string) => `inbox_issue.status.${name}.title`;

describe("IntakeAppliedFilters", () => {
  it("shows nothing while the filters are the tab's first ones, nor for an order of the user's own", async () => {
    await render();
    expect(host.textContent).toBe("");
    await changeFilters({ order: "sequence_id" });
    expect(host.textContent).toBe("");
  });

  it("names each filter applied and each of its values, each value's button named as removing it", async () => {
    await render();
    await changeFilters({
      statuses: ["pending", "snoozed"],
      priorities: ["urgent", "high"],
      assignees: ["u-ann", "u-bob"],
      creators: ["u-ann"],
      labels: ["l-bug", "l-gone"],
      created: presetRange("today", now),
      updated: { from: "2026-11-01", to: "2026-11-20" },
    });
    const unknown = "common.filter_options.unavailable";
    const today = "inbox_issue.filters.presets.today";
    const between = "inbox_issue.filters.between Nov 01, 2026 Nov 20, 2026";
    expect(shown()).toEqual([
      [
        "inbox_issue.filters.status",
        [
          [status("pending"), removing(status("pending"))],
          [status("snoozed"), removing(status("snoozed"))],
        ],
      ],
      [
        "common.priority",
        [
          ["urgent", removing("urgent")],
          ["high", removing("high")],
        ],
      ],
      [
        "common.assignees",
        [
          ["Ann", removing("Ann")],
          ["Bob", removing("Bob")],
        ],
      ],
      ["common.created_by", [["Ann", removing("Ann")]]],
      [
        "common.labels",
        [
          ["Bug", removing("Bug")],
          [unknown, removing(unknown)],
        ],
      ],
      ["inbox_issue.filters.created_at", [[today, removing(today)]]],
      ["inbox_issue.filters.updated_at", [[between, removing(between)]]],
    ]);
  });

  it("names a range after a day, before a day, and of one day by that day", async () => {
    await render();
    await changeFilters({ created: { from: "2026-11-01" }, updated: { to: "2026-11-20" } });
    expect(shown().map(([, chips]) => (chips as unknown[][])[0]?.[0])).toEqual([
      "inbox_issue.status.pending.title",
      "inbox_issue.filters.after Nov 01, 2026",
      "inbox_issue.filters.before Nov 20, 2026",
    ]);
    await changeFilters({ created: { from: "2026-11-05", to: "2026-11-05" }, updated: undefined });
    expect(shown().map(([, chips]) => (chips as unknown[][])[0]?.[0])).toEqual([
      "inbox_issue.status.pending.title",
      "Nov 05, 2026",
    ]);
  });

  it("removes the value its native button names alone, by Enter and by Space, and submits no form", async () => {
    await render();
    await changeFilters({ assignees: ["u-ann", "u-bob"], created: { from: "2026-10-01" }, priorities: ["low"] });
    const buttons = [...host.querySelectorAll("button")];
    expect(buttons.every((found) => found.type === "button" && found.tabIndex === 0)).toBe(true);
    await press(button(removing("Ann"), group("common.assignees"))!, "Enter");
    expect(filterState.current.filters.assignees).toEqual(["u-bob"]);
    await press(button(removing("inbox_issue.filters.after Oct 01, 2026"))!, " ");
    expect(filterState.current.filters.created).toBeUndefined();
    await act(async () => button(removing("low"))!.click());
    expect(filterState.current.filters).toEqual({ ...defaultFilters("open"), assignees: ["u-bob"] });
    expect(submitted).not.toHaveBeenCalled();
  });

  it("gives the last status no button: a list names a status always (R9)", async () => {
    await render();
    await changeFilters({ statuses: ["pending", "snoozed"] });
    await act(async () => button(removing("inbox_issue.status.snoozed.title"))!.click());
    expect(filterState.current.filters.statuses).toEqual(["pending"]);
    // the open tab's first statuses: nothing applied
    expect(host.textContent).toBe("");
    await changeFilters({ statuses: ["snoozed"], priorities: ["urgent"] });
    expect(shown()[0]).toEqual(["inbox_issue.filters.status", [["inbox_issue.status.snoozed.title", null]]]);
  });

  it("clears all back to the tab's first filters, the order kept, in either tab", async () => {
    await render();
    await changeFilters({
      statuses: ["snoozed"],
      labels: ["l-bug"],
      order: "sequence_id",
      updated: { to: "2026-11-01" },
    });
    await act(async () => button("common.clear_all")!.click());
    expect(filterState.current.filters).toEqual({ ...defaultFilters("open"), order: "sequence_id" });
    expect(host.textContent).toBe("");

    await render("?currentTab=closed");
    await changeFilters({ statuses: ["declined"] });
    await act(async () => button("common.clear_all")!.click());
    expect(filterState.current.filters).toEqual(defaultFilters("closed"));
    expect(submitted).not.toHaveBeenCalled();
  });
});
