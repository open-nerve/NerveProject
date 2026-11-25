// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// While Update view is out, a project's view's page stops every way that changes its draft (P6a spec 2.7, R29; O-6,
// re-96 I4), so that nothing changed meanwhile is lost under nerve's answer: the filters row's conditions, its Add
// filter (gone meanwhile), Clear all and Save view (Update view says Confirming, as ever); the header's layouts, at
// any width, its display and its toggle's Add filter; a spreadsheet column's sort; the empty list's clearing of the
// filters. Once nerve answers or refuses, each works again; the focus lands on Add filter once the row has it again
// (another change of the view queued after Update view keeps it stopped). The page, the stores, SWR and HTTP are real
// (fake-view-saving.tsx); the app's header frames the header and the small screen's header.
import { act, type ReactNode } from "react";
import { afterEach, expect, it, vi } from "vitest";
import type { View } from "@nerve/api-client";
import { EIssueFilterType } from "@nerve/constants";
import { settle } from "@/components/common/fake-quick-actions";
import { json, problem } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { preferencesOf } from "@/store/project/fake-projects";
import type { RootStore } from "@/store/root.store";
import { buttonOf, clickUpdate, rowAddFilter, updateButton, viewSavingTab } from "./fake-view-saving";

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
  default: ({ children }: { children: ReactNode }) => children,
}));
vi.mock("@/components/issues/peek-overview", () => ({ IssuePeekOverview: () => null }));
vi.mock("@/components/breadcrumbs/common", () => ({ CommonProjectBreadcrumbs: () => <span>the project</span> }));
vi.mock("@/components/core/app-header", () => ({
  AppHeader: (props: { header: ReactNode; mobileHeader?: ReactNode }) => (
    <>
      <header>{props.header}</header>
      <aside>{props.mobileHeader}</aside>
    </>
  ),
}));

let page: Awaited<ReturnType<typeof viewSavingTab>>;
afterEach(async () => page.close());

/** First's page, opened as opening says, its draft's display changed (no empty groups): Update view offered. */
async function changed(opening: Parameters<typeof viewSavingTab>[0] = {}, empty = false) {
  await displayChanged(opening, empty);
  await until(() => updateButton() !== undefined, "Update view");
}
/** First's page, opened as opening says, its draft's display changed (no empty groups). */
async function displayChanged(opening: Parameters<typeof viewSavingTab>[0] = {}, empty = false) {
  page = await viewSavingTab(opening);
  bound.app = page.app;
  await page.open(empty);
  await act(async () =>
    page.app.issue.projectViewIssuesFilter.updateFilters("ws", page.view.id, EIssueFilterType.DISPLAY_FILTERS, {
      show_empty_groups: false,
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
  expect(named("header", "issue.layouts.title.kanban")?.disabled, "the header's layouts").toBe(busy);
  expect(named("header", "common.layout")?.disabled, "the header's narrow layouts").toBe(busy);
  expect(named("header", "common.display")?.disabled, "the header's display").toBe(busy);
  expect(named("aside", "common.layout")?.disabled, "the small screen's layouts").toBe(busy);
  expect(named("aside", "common.display")?.disabled, "the small screen's display").toBe(busy);
}

it.each([
  [
    "nerve answers",
    (view: View) => json(200, { ...view, display_filters: { ...view.display_filters, show_empty_groups: false } }),
  ],
  ["nerve refuses", () => problem(409, "navigation.views_disabled")],
])("stops what changes the draft while Update view is out, and gives it back once %s", async (_, answer) => {
  await changed();
  stopped(false);
  await updating();
  expect(updateButton()?.textContent, "Update view, out").toBe("common.filter_options.confirming");
  stopped(true);
  await act(async () => page.patches()[0]!.answer(answer(page.view)));
  await act(async () => vi.advanceTimersByTimeAsync(300));
  stopped(false);
});

it("stops a spreadsheet column's sort while Update view is out", async () => {
  await changed({ view: { display_filters: { ...preferencesOf().display_filters, layout: "spreadsheet" } } });
  expect(sorts().length).toBeGreaterThan(0);
  expect(sorts().every((sort) => !sort.disabled)).toBe(true);
  await updating();
  expect(sorts().every((sort) => sort.disabled)).toBe(true);
  await act(async () => page.patches()[0]!.answer(problem(404, "navigation.view_not_found")));
  await settle();
  expect(sorts().every((sort) => !sort.disabled)).toBe(true);
});

it("stops the empty list's clearing of the filters while Update view is out", async () => {
  await changed({}, true);
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
  await act(async () => page.patches()[0]!.answer(problem(409, "navigation.views_disabled")));
  await settle();
  expect(headerAddFilter(), "its Add filter again").toBeDefined();
});

it("lands the focus on the row's Add filter once the view's change queued after Update view is answered too", async () => {
  await changed();
  await updating();
  // a change of the view's name, queued after it
  const renamed = page.app.views.update(page.view.id, () => ({ name: "Renamed" }));
  const answer = { ...page.view, display_filters: { ...page.view.display_filters, show_empty_groups: false } };
  await act(async () => page.patches()[0]!.answer(json(200, answer)));
  await settle();
  await until(() => page.patches().length === 2, "the change after it");
  expect(rowAddFilter(), "the draft still being saved: no Add filter").toBeUndefined();
  expect(document.activeElement?.closest("header"), "the focus not moved to the header meanwhile").toBe(null);
  await act(async () => page.patches()[1]!.answer(json(200, { ...answer, name: "Renamed" })));
  await renamed;
  await settle();
  expect(document.activeElement, "the focus on the row's Add filter").toBe(rowAddFilter());
});
