// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// A row of a cycle page's progress sidebar toggles its value in the page's filters (P5b spec 2.8, R8): a state group's,
// an assignee's, a label's, through the page's own filters (their conditions saved as its preferences, as the page's
// filters row saves them): chosen, the value joins the condition of its kind; chosen again, it leaves it; the last one
// gone, the condition goes. Each row says whether its value is chosen. The rows are plain while the page has no filters
// to show (before it has read them), in the list page's preview, and for none (no assignee, no labels). The stores,
// routes and HTTP are real.
import { act, createElement, type ReactNode } from "react";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { json } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { toasts } from "@/lib/fake-toast";
import type { ProjectIssueFilters } from "@nerve/api-client";
import { ANN, BUG, UI, openTab, partOf, rowButton, rowsIn } from "@/components/core/sidebar/fake-progress-rows";
import { C1, LIST, PAGE, answerPreferences, cycleRoutesTab } from "../fake-cycle-routes";

const fixture = vi.hoisted(() => ({ app: undefined as unknown }));
vi.mock("@/lib/store-context", async () => ({
  get rootStore() {
    return fixture.app;
  },
  StoreContext: (await import("react")).createContext(undefined),
}));
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "one" } }, publicClient: {} }));
vi.mock("@/lib/auth/use-session", () => ({ useSession: () => ({ status: "signed-in", loginId: "one" }) }));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
// jsdom has no viewport geometry; rows and HTTP are real.
vi.mock("@atlaskit/pragmatic-drag-and-drop-auto-scroll/element", () => ({ autoScrollForElements: () => () => {} }));
vi.mock("@/components/core/render-if-visible-HOC", () => ({
  default: ({ children, as }: { children: ReactNode; as?: string }) =>
    as ? createElement(as, null, children) : children,
}));
// What the rows do not reach: the cycle's list row, the chart, the sidebar's head and details, a work item's preview.
vi.mock("@/components/cycles/list/cycles-list-item", () => ({ CyclesListItem: () => null }));
vi.mock("@/components/core/sidebar/burndown-chart", () => ({ BurndownChart: () => null }));
vi.mock("@/components/cycles/progress-sidebar/sidebar-header", () => ({ CycleSidebarHeader: () => null }));
vi.mock("@/components/cycles/progress-sidebar/sidebar-details", () => ({ CycleSidebarDetails: () => null }));
vi.mock("@/components/issues/peek-overview", () => ({ IssuePeekOverview: () => null }));

let tab: Awaited<ReturnType<typeof cycleRoutesTab>>;
beforeEach(async () => {
  localStorage.clear();
  toasts.length = 0;
  tab = await cycleRoutesTab();
  fixture.app = tab.app;
});
afterEach(async () => tab.close());

const sidebar = () => partOf("sidebar")!;
const STATES = "project_settings.states.groups";

/** Opens the cycle's page; nerve answers its preferences, of filters, and its list. */
async function atPage(filters: ProjectIssueFilters = {}) {
  await tab.open(PAGE);
  await until(() => tab.reads().length === 1, "the page's preferences");
  await answerPreferences(tab.reads()[0]!, filters);
  await until(() => tab.lists().length === 1, "the page's list");
  await act(async () => tab.lists()[0]!.answer(json(200, { total: 0, groups: [] })));
}
/**
 * Chooses the row named name of the sidebar, a button: the filters of the change of the page's preferences it sent.
 * nerve confirms them as sent, and answers the page's list read again by them.
 */
async function choose(name: string) {
  const button = rowButton(name, sidebar());
  expect(button, `the row ${name} is a button`).toBeDefined();
  const from = tab.patches().length;
  await act(async () => button!.click());
  await until(() => tab.patches().length === from + 1, `the change of the row ${name}`);
  const sent = tab.patches()[from]!;
  const { filters } = sent.body as { filters: ProjectIssueFilters };
  const lists = tab.lists().length;
  await answerPreferences(sent, filters);
  await until(() => tab.lists().length === lists + 1, `the list again after the row ${name}`);
  await act(async () =>
    tab
      .lists()
      .at(-1)!
      .answer(json(200, { total: 0, groups: [] }))
  );
  return filters;
}
/** Whether the row named name says its value is chosen. */
const chosen = (name: string) => rowButton(name, sidebar())?.getAttribute("aria-pressed");

it("toggles a label in the page's filters: in, another in, one out, the last out takes the condition away", async () => {
  await atPage({ priority: ["high"] });
  await openTab("common.labels", sidebar());
  expect([chosen("Bug"), chosen("UI")]).toEqual(["false", "false"]);
  expect(await choose("Bug")).toEqual({ priority: ["high"], label_id: [BUG] });
  expect([chosen("Bug"), chosen("UI")]).toEqual(["true", "false"]);
  expect(await choose("UI")).toEqual({ priority: ["high"], label_id: [BUG, UI] });
  expect([chosen("Bug"), chosen("UI")]).toEqual(["true", "true"]);
  expect(await choose("Bug")).toEqual({ priority: ["high"], label_id: [UI] });
  expect(await choose("UI")).toEqual({ priority: ["high"] });
  expect([chosen("Bug"), chosen("UI")]).toEqual(["false", "false"]);
  // each change is the page's: confirmed, its list is read again by it
  expect(tab.lists().map((call) => call.query.label_id)).toEqual([undefined, BUG, `${BUG},${UI}`, UI, undefined]);
  expect(toasts).toEqual([]);
});

it("toggles a state group and an assignee alike, each row saying whether it is chosen", async () => {
  await atPage({ state_group: ["backlog"] });
  await openTab("common.states", sidebar());
  expect(chosen(`${STATES}.backlog`)).toBe("true");
  expect(await choose(`${STATES}.started`)).toEqual({ state_group: ["backlog", "started"] });
  expect(await choose(`${STATES}.backlog`)).toEqual({ state_group: ["started"] });
  expect([chosen(`${STATES}.backlog`), chosen(`${STATES}.started`)]).toEqual(["false", "true"]);
  await openTab("common.assignees", sidebar());
  expect(await choose("distribution.former_member")).toEqual({ state_group: ["started"], assignee_id: [ANN] });
  expect(chosen("distribution.former_member")).toBe("true");
});

it("makes the rows of a value buttons named by the row, and leaves the rows of none plain", async () => {
  await atPage();
  expect(rowsIn(sidebar())).toEqual([
    [true, "distribution.former_member"],
    [false, "no_assignee"],
  ]);
  await openTab("common.labels", sidebar());
  expect(rowsIn(sidebar())).toEqual([
    [true, "Bug"],
    [true, "UI"],
    [false, "distribution.no_labels"],
  ]);
  expect(rowButton("Bug", sidebar())?.type).toBe("button");
});

it("leaves the rows plain until the page shows its filters", async () => {
  await tab.open(PAGE);
  await until(() => tab.reads().length === 1, "the page's preferences");
  await openTab("common.labels", sidebar());
  expect(rowsIn(sidebar())).toEqual([
    [false, "Bug"],
    [false, "UI"],
    [false, "distribution.no_labels"],
  ]);
  await answerPreferences(tab.reads()[0]!);
  expect(rowsIn(sidebar()).filter(([button]) => button)).toHaveLength(2);
});

it("leaves the rows plain in the list page's preview of the cycle", async () => {
  await tab.open(`${LIST}?peekCycle=${C1}`);
  const preview = () => partOf("preview")!;
  await until(() => preview().querySelector("li") !== null, "the preview's rows");
  await openTab("common.labels", preview());
  expect(rowsIn(preview())).toEqual([
    [false, "Bug"],
    [false, "UI"],
    [false, "distribution.no_labels"],
  ]);
  expect(tab.patches()).toEqual([]);
});
