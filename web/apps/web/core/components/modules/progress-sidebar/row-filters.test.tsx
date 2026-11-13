// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// A row of a module page's progress sidebar toggles its value in the page's filters (P6b spec 2.7, R14), as a cycle
// page's does: a state group's, an assignee's, a label's, through the page's own filters (their conditions saved as its
// preferences, the field of filters whole, as the page's filters row saves them): chosen, the value joins the
// condition of its kind; chosen again, it leaves it; the last one gone, the condition goes. Each row is named by its
// row and says whether its value is chosen. The rows are plain while the page has no filters to show (before it has
// read them, with the project's modules turned off), in the list page's preview, and for none (no assignee, no
// labels); they choose whatever the module's days (its tabs always show, P6a R8), and on an archived module's page,
// whose list is read-only but whose preferences are the caller's. The stores, routes and HTTP are real.
import { act, createElement, type ReactNode } from "react";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { json } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { toasts } from "@/lib/fake-toast";
import type { ModuleIssueFilters } from "@nerve/api-client";
import { ANN, BUG, UI, openTab, partOf, rowButton, rowsIn } from "@/components/core/sidebar/fake-progress-rows";
import { LIST, M1, PAGE, answerPreferences, moduleRoutesTab } from "../fake-module-routes";

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
// What the rows do not reach: the chart, the sidebar's head, details and links, a work item's preview.
vi.mock("@/components/core/sidebar/burndown-chart", () => ({ BurndownChart: () => null }));
vi.mock("@/components/modules/progress-sidebar/sidebar-header", () => ({ ModuleSidebarHeader: () => null }));
vi.mock("@/components/modules/progress-sidebar/sidebar-details", () => ({ ModuleSidebarDetails: () => null }));
vi.mock("@/components/modules/progress-sidebar/sidebar-links", () => ({ ModuleSidebarLinks: () => null }));
vi.mock("@/components/issues/peek-overview", () => ({ IssuePeekOverview: () => null }));

let tab: Awaited<ReturnType<typeof moduleRoutesTab>> | undefined;
/** A tab at project A's modules, M1 of the fields given (a backlog module of no days unless they say). */
async function at(fields: Parameters<typeof moduleRoutesTab>[0] = {}) {
  tab = await moduleRoutesTab(fields);
  fixture.app = tab.app;
  return tab;
}
beforeEach(() => {
  localStorage.clear();
  toasts.length = 0;
});
afterEach(async () => {
  await tab?.close();
  tab = undefined;
});

const sidebar = () => partOf("sidebar")!;
const STATES = "project_settings.states.groups";

/** Opens the module's page; nerve answers its preferences, of filters, and its list. */
async function atPage(filters: ModuleIssueFilters = {}) {
  await tab!.open(PAGE);
  await until(() => tab!.reads().length === 1, "the page's preferences");
  await answerPreferences(tab!.reads()[0]!, filters);
  await until(() => tab!.lists().length === 1, "the page's list");
  await act(async () => tab!.lists()[0]!.answer(json(200, { total: 0, groups: [] })));
}
/**
 * Chooses the row named name of the sidebar, a button: the change of the page's preferences it sent, its filters whole
 * and nothing else. nerve confirms them as sent, and answers the page's list read again by them.
 */
async function choose(name: string) {
  const button = rowButton(name, sidebar());
  expect(button, `the row ${name} is a button`).toBeDefined();
  const from = tab!.patches().length;
  await act(async () => button!.click());
  await until(() => tab!.patches().length === from + 1, `the change of the row ${name}`);
  const sent = tab!.patches()[from]!;
  expect(Object.keys(sent.body as object), "the change of the preferences is their filters").toEqual(["filters"]);
  const { filters } = sent.body as { filters: ModuleIssueFilters };
  const lists = tab!.lists().length;
  await answerPreferences(sent, filters);
  await until(() => tab!.lists().length === lists + 1, `the list again after the row ${name}`);
  await act(async () =>
    tab!
      .lists()
      .at(-1)!
      .answer(json(200, { total: 0, groups: [] }))
  );
  return filters;
}
/** Whether the row named name says its value is chosen. */
const chosen = (name: string) => rowButton(name, sidebar())?.getAttribute("aria-pressed");

it("toggles a label in the page's filters: in, another in, one out, the last out takes the condition away", async () => {
  await at();
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
  // each change is the page's: confirmed, its list (the module's) is read again by it
  expect(tab!.lists().map((call) => [call.query.module_id, call.query.label_id])).toEqual([
    [M1, undefined],
    [M1, BUG],
    [M1, `${BUG},${UI}`],
    [M1, UI],
    [M1, undefined],
  ]);
  expect(toasts).toEqual([]);
});

it("toggles a state group and an assignee alike, each row saying whether it is chosen", async () => {
  await at();
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
  await at();
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
  await at();
  await tab!.open(PAGE);
  await until(() => tab!.reads().length === 1, "the page's preferences");
  await openTab("common.labels", sidebar());
  expect(rowsIn(sidebar())).toEqual([
    [false, "Bug"],
    [false, "UI"],
    [false, "distribution.no_labels"],
  ]);
  await answerPreferences(tab!.reads()[0]!);
  expect(rowsIn(sidebar()).filter(([button]) => button)).toHaveLength(2);
});

it("leaves the rows plain while the project's modules are turned off: the page has no filters", async () => {
  await at({ project: { module_view: false } });
  await tab!.open(PAGE);
  expect(document.body.textContent).toContain("disabled_project.empty_state.module.title");
  await openTab("common.labels", sidebar());
  expect(rowsIn(sidebar())).toEqual([
    [false, "Bug"],
    [false, "UI"],
    [false, "distribution.no_labels"],
  ]);
  expect(tab!.preferences()).toEqual([]);
});

it("leaves the rows plain in the list page's preview of the module", async () => {
  await at();
  await tab!.open(`${LIST}?peekModule=${M1}`);
  const preview = () => partOf("preview")!;
  await until(() => preview().querySelector("li") !== null, "the preview's rows");
  await openTab("common.labels", preview());
  expect(rowsIn(preview())).toEqual([
    [false, "Bug"],
    [false, "UI"],
    [false, "distribution.no_labels"],
  ]);
  expect(tab!.patches()).toEqual([]);
});

it.each([
  ["no days", { start_date: null, target_date: null }],
  ["a start still to come", { start_date: "2027-02-01", target_date: "2027-02-26" }],
  ["days long past", { start_date: "2026-10-01", target_date: "2026-10-30" }],
])("toggles the rows whatever the module's days: %s (R14)", async (_, days) => {
  await at({ module: days });
  await atPage();
  await openTab("common.labels", sidebar());
  expect(await choose("Bug")).toEqual({ label_id: [BUG] });
  expect(chosen("Bug")).toBe("true");
});

it("toggles the rows on an archived module's page: its list is read-only, its preferences the caller's", async () => {
  await at({ module: { archived_at: "2026-11-11T09:00:00Z" } });
  await atPage();
  // the page is the archived module's: its list, of none, says only why it is read-only
  expect(document.body.textContent).toContain("project_module.empty_state.archived.title");
  await openTab("common.states", sidebar());
  expect(await choose(`${STATES}.completed`)).toEqual({ state_group: ["completed"] });
  expect(chosen(`${STATES}.completed`)).toBe("true");
  expect(await choose(`${STATES}.completed`)).toEqual({});
  expect(chosen(`${STATES}.completed`)).toBe("false");
});
