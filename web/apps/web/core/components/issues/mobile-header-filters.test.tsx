// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// The small screen's row of a list page's header (P7c spec 2.7, R8), by the page's store type: the page's layouts, the
// list, the board and the calendar, its trigger showing the one the page's preferences hold, and its display with the
// page's own options, both changing the page's preferences by the page's update, as the header's controls at a wider
// width do (HeaderFilters); its trigger follows what nerve confirmed. Both show the preferences of the page their address
// names, never the router store's project, which lags a commit when the address changes project (re-77 M2). Whether
// the page shows the row is its wrapper's (the project's: issues/(list)/layout.test.tsx). The stores and HTTP are real;
// the page is a project's.
import { act } from "react";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { ISSUE_GROUP_BY_OPTIONS, ISSUE_STORE_TO_FILTERS_MAP } from "@nerve/constants";
import { EIssuesStoreType } from "@nerve/types";
import { json } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { issueSelectionTab } from "@/components/core/modals/fake-issue-selection";
import { requestedIssue } from "@/store/issue/fake-issues";
import { preferencesOf } from "@/store/project/fake-projects";
import type { RootStore } from "@/store/root.store";
import { HeaderFilters } from "./filters";
import { MobileHeaderFilters } from "./mobile-header-filters";

const bound = vi.hoisted(() => ({ app: undefined as RootStore | undefined }));
vi.mock("@/lib/store-context", async () => ({
  get rootStore() {
    return bound.app;
  },
  StoreContext: (await import("react")).createContext<RootStore | undefined>(undefined),
}));
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "one" } }, publicClient: {} }));
vi.mock("react-router", async (original) => ({
  ...(await original<typeof import("react-router")>()),
  useParams: () => ({ workspaceSlug: "ws", projectId: "A" }),
}));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
// each layout's icon says which it is
vi.mock("@/components/issues/issue-layouts/layout-icon", () => ({
  IssueLayoutIcon: ({ layout }: { layout: string }) => <svg data-layout={layout} />,
}));

let tab: Awaited<ReturnType<typeof issueSelectionTab>>;
beforeEach(async () => {
  tab = await issueSelectionTab();
  bound.app = tab.app;
  const read = await requestedIssue(tab.nerve, () => tab.app.issue.projectIssuesFilter.fetchFilters("ws", "A"), [
    "GET",
    "/api/v0/me/projects/A/preferences",
  ]);
  read.call.answer(json(200, preferencesOf()));
  await read.promise;
  tab.nerve.calls.length = 0;
  await tab.render(<MobileHeaderFilters storeType={EIssuesStoreType.PROJECT} entityId="A" />);
});
afterEach(async () => tab.close());

const preferences = () => tab.nerve.to("/api/v0/me/projects/A/preferences");
const button = (name: string) =>
  [...document.querySelectorAll<HTMLButtonElement>("button")].find(
    (node) => (node.getAttribute("aria-label") ?? node.textContent?.trim()) === name
  );
const menuItems = () => [...document.querySelectorAll<HTMLElement>('[role="menuitem"]')];
/** The layouts the trigger shows. */
const shownLayout = () =>
  [...(button("common.layout")?.querySelectorAll("[data-layout]") ?? [])].map((icon) =>
    icon.getAttribute("data-layout")
  );

it("shows the page's layout on its trigger and offers the list, the board and the calendar, in the small screen's row", async () => {
  expect(shownLayout()).toEqual(["list"]);
  const row = button("common.layout")?.closest(".md\\:hidden");
  expect(row?.contains(button("common.display")!)).toBe(true);
  await act(async () => button("common.layout")!.click());
  expect(menuItems().map((item) => item.textContent?.trim())).toEqual([
    "issue.layouts.list",
    "issue.layouts.kanban",
    "issue.layouts.calendar",
  ]);
});

it("changes the page's layout by the page's update of its preferences, its trigger showing what nerve confirmed", async () => {
  await act(async () => button("common.layout")!.click());
  await act(async () => menuItems()[1]!.click());
  await until(() => preferences().length === 1, "the change of the page's layout");
  expect(preferences()[0]).toMatchObject({ method: "PATCH", body: { display_filters: { layout: "kanban" } } });
  expect(shownLayout()).toEqual(["list"]);
  // nerve confirms what the page sent
  await act(async () =>
    preferences()[0]!.answer(json(200, { ...preferencesOf(), ...(preferences()[0]!.body as object) }))
  );
  expect(shownLayout()).toEqual(["kanban"]);
});

it("offers the page's own display options, whose change is the page's update of its preferences", async () => {
  await act(async () => button("common.display")!.click());
  const options = ISSUE_STORE_TO_FILTERS_MAP[EIssuesStoreType.PROJECT]!.layoutOptions.list!.display_filters;
  const shown = [...document.querySelectorAll("button[aria-pressed]")].map((node) => node.textContent);
  const groupings = ISSUE_GROUP_BY_OPTIONS.filter(({ key }) => options.group_by?.includes(key));
  expect(groupings.length).toBeGreaterThan(0);
  for (const option of groupings) expect(shown, option.titleTranslationKey).toContain(option.titleTranslationKey);
  await act(async () => button("common.priority")!.click());
  await until(() => preferences().length === 1, "the change of the page's display");
  expect(preferences()[0]).toMatchObject({ method: "PATCH", body: { display_properties: { priority: false } } });
});

it("shows the layout of the page its address names, in the row and the wider controls, the router store naming another", async () => {
  // Project B's preferences, its layout the board, held beside A's; the router store still names A.
  const board = preferencesOf({ display_filters: { ...preferencesOf().display_filters, layout: "kanban" } });
  const read = await requestedIssue(tab.nerve, () => tab.app.issue.projectIssuesFilter.fetchFilters("ws", "B"), [
    "GET",
    "/api/v0/me/projects/B/preferences",
  ]);
  read.call.answer(json(200, board));
  await read.promise;
  await tab.render(
    <>
      <MobileHeaderFilters storeType={EIssuesStoreType.PROJECT} entityId="B" />
      <HeaderFilters storeType={EIssuesStoreType.PROJECT} entityId="B" />
    </>
  );
  expect(tab.app.router.projectId).toBe("A");
  // the row's trigger and the wider controls' narrow one show the board, as the wide ones' pressed layout does
  const triggers = [...document.querySelectorAll('button[aria-label="common.layout"]')];
  const pressed = [...document.querySelectorAll('button[aria-pressed="true"]')];
  expect({
    triggers: triggers.map((trigger) => trigger.querySelector("[data-layout]")?.getAttribute("data-layout")),
    pressed: pressed.map((layout) => layout.getAttribute("aria-label")),
  }).toEqual({ triggers: ["kanban", "kanban"], pressed: ["issue.layouts.title.kanban"] });
});
