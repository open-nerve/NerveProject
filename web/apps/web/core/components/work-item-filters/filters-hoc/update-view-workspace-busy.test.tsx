// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// While Update view is out, a workspace's own view's page stops every way that changes its draft (M7/P6b spec 2.9, as
// P6a R29), so that nothing changed meanwhile is lost under nerve's answer: the filters row's conditions, its Add filter
// (gone meanwhile), Clear all and Save view; the header's display and its toggle's Add filter; a spreadsheet column's
// sort; the empty list's clearing of the filters. Once nerve answers or refuses, each works again. The page, the
// stores, SWR and HTTP are real (fake-workspace-view-saving.tsx); the app's header a frame.
import { act, createElement, type ReactNode } from "react";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { View } from "@nerve/api-client";
import { EIssueFilterType } from "@nerve/constants";
import { settle } from "@/components/common/fake-quick-actions";
import { json, problem } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import type { RootStore } from "@/store/root.store";
import { buttonOf, clickUpdate, rowAddFilter, updateButton } from "./fake-view-saving";
import { workspaceSavingTab } from "./fake-workspace-view-saving";

const bound = vi.hoisted(() => ({ app: undefined as RootStore | undefined }));
vi.mock("@/lib/store-context", async () => ({
  get rootStore() {
    return bound.app;
  },
  StoreContext: (await import("react")).createContext<RootStore | undefined>(undefined),
}));
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "one" } }, publicClient: {} }));
vi.mock("@/lib/auth/use-session", () => ({ useSession: () => ({ status: "signed-in", loginId: "one" }) }));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@atlaskit/pragmatic-drag-and-drop-auto-scroll/element", () => ({ autoScrollForElements: () => () => {} }));
vi.mock("@/components/core/render-if-visible-HOC", () => ({
  default: ({ children, as }: { children: ReactNode; as?: string }) =>
    as ? createElement(as, null, children) : children,
}));
vi.mock("@/components/issues/peek-overview", () => ({ IssuePeekOverview: () => null }));
vi.mock("@/components/core/app-header", () => ({
  AppHeader: (props: { header: ReactNode }) => <header>{props.header}</header>,
}));

let page: Awaited<ReturnType<typeof workspaceSavingTab>>;
beforeEach(() => {
  window.localStorage.clear();
});
afterEach(async () => page.close());

/** First's page, opened as opening says, its draft's display changed (no sub-work items): Update view offered. */
async function displayChanged(opening: Parameters<typeof workspaceSavingTab>[0] = {}, empty = false) {
  page = await workspaceSavingTab(opening);
  bound.app = page.app;
  await page.open(page.view.id, empty);
  await act(async () =>
    page.app.issue.globalIssuesFilter.updateFilters("ws", page.view.id, EIssueFilterType.DISPLAY_FILTERS, {
      sub_issue: false,
    })
  );
  await settle();
}
/** Sends Update view; nerve holds its answer. */
async function updating() {
  await clickUpdate();
  await until(() => page.patches().length === 1, "the change of the view");
}
/** The region's button named name (its aria-label). */
const named = (region: string, name: string) =>
  document.querySelector<HTMLButtonElement>(`${region} button[aria-label="${name}"]`);
/** The spreadsheet's columns' sorts. */
const sorts = () => [...document.querySelectorAll<HTMLButtonElement>("th button")];
/** The header's toggle's Add filter, while it offers one. */
const headerAddFilter = () =>
  [...document.querySelectorAll("header [data-filters-toggle] button")].find((button) =>
    button.textContent?.includes("common.filter_options.add")
  );
/** Whether each way that changes the draft is disabled (or gone), as busy says. */
function stopped(busy: boolean) {
  expect(document.querySelector("button[aria-label='Remove filter']") === null, "a condition's controls").toBe(busy);
  expect(rowAddFilter() === undefined, "the row's Add filter").toBe(busy);
  expect(buttonOf("common.clear_all")?.disabled, "Clear all").toBe(busy);
  expect(buttonOf("common.filter_options.save_view")?.disabled, "Save view").toBe(busy);
  expect(named("header", "common.display")?.disabled, "the header's display").toBe(busy);
}

it.each([
  [
    "nerve answers",
    (view: View) => json(200, { ...view, display_filters: { ...view.display_filters, sub_issue: false } }),
  ],
  ["nerve refuses", () => problem(404, "navigation.view_not_found")],
])("stops what changes the draft while Update view is out, and gives it back once %s", async (_, answer) => {
  await displayChanged();
  await until(() => updateButton() !== undefined, "Update view");
  stopped(false);
  await updating();
  expect(updateButton()?.textContent, "Update view, out").toBe("common.filter_options.confirming");
  stopped(true);
  await act(async () => page.patches()[0]!.answer(answer(page.view)));
  await act(async () => vi.advanceTimersByTimeAsync(300));
  stopped(false);
});

it("stops a spreadsheet column's sort while Update view is out", async () => {
  await displayChanged();
  await until(() => updateButton() !== undefined, "Update view");
  expect(sorts().length).toBeGreaterThan(0);
  expect(sorts().every((sort) => !sort.disabled)).toBe(true);
  await updating();
  expect(sorts().every((sort) => sort.disabled)).toBe(true);
  await act(async () => page.patches()[0]!.answer(problem(404, "navigation.view_not_found")));
  await settle();
  expect(sorts().every((sort) => !sort.disabled)).toBe(true);
});

it("stops the empty list's clearing of the filters while Update view is out", async () => {
  await displayChanged({}, true);
  await until(() => updateButton() !== undefined, "Update view");
  const clear = () => buttonOf("project_issues.empty_state.issues_empty_filter.secondary_button.text");
  expect(clear()?.disabled).toBe(false);
  await updating();
  expect(clear()?.disabled).toBe(true);
  await act(async () => page.patches()[0]!.answer(problem(404, "navigation.view_not_found")));
  await settle();
  expect(clear()?.disabled).toBe(false);
});

it("offers no Add filter in the header's toggle while Update view is out", async () => {
  // a view of no conditions: its row hidden, the header's toggle its Add filter
  await displayChanged({ view: { filters: {} } });
  expect(headerAddFilter(), "the toggle's Add filter").toBeDefined();
  await act(async () => page.filter().toggleVisibility(true));
  await until(() => updateButton() !== undefined, "Update view in the row shown");
  await updating();
  await act(async () => page.filter().toggleVisibility(false));
  await settle();
  expect(headerAddFilter(), "out: the toggle's row button alone").toBeUndefined();
  expect(named("header", "common.filters"), "the row's toggle").not.toBeNull();
  await act(async () => page.patches()[0]!.answer(problem(404, "navigation.view_not_found")));
  await settle();
  expect(headerAddFilter(), "its Add filter again").toBeDefined();
});
