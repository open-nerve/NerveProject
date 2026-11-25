// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// The ways deep in a list's layouts that change a page's draft (P6a spec 2.7, R29; O-6, re-96 I4) stop while the page
// says its draft is being saved (DraftBusyContext, a view's page's Update view out): a spreadsheet column's sort, the
// calendar's options (month or week, weekends), the empty list's clearing of the filters. Outside it, as on the
// project's, a cycle's and a module's pages, which never say so, each works as before. The parts, the stores and the
// filter are real; what a part changes is the test's.
import { act, type ReactNode } from "react";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { EIssuesStoreType } from "@nerve/types";
import { issueSelectionTab } from "@/components/core/modals/fake-issue-selection";
import type { RootStore } from "@/store/root.store";
import { CalendarOptionsDropdown } from "./calendar/dropdowns/options-dropdown";
import { DraftBusyContext } from "./draft-busy";
import { ProjectListEmptyState } from "./empty-states/project-issues";
import { SpreadsheetHeaderColumn } from "./spreadsheet/spreadsheet-header-column";

const bound = vi.hoisted(() => ({ params: {} as Record<string, string> }));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "one" } }, publicClient: {} }));
vi.mock("@/lib/store-context", async () => ({
  rootStore: undefined,
  StoreContext: (await import("react")).createContext<RootStore | undefined>(undefined),
}));
vi.mock("react-router", async (original) => ({
  ...(await original<typeof import("react-router")>()),
  useParams: () => bound.params,
}));

const V = "019a0000-0000-7000-8000-0000000000e1";
let tab: Awaited<ReturnType<typeof issueSelectionTab>>;
beforeEach(async () => {
  tab = await issueSelectionTab();
  bound.params = { workspaceSlug: "ws", projectId: "A", viewId: V };
});
afterEach(async () => tab.close());

/** Renders part in a page whose draft is being saved (busy), or in one that never says so (no context). */
const onPage = (busy: boolean | undefined, part: ReactNode) =>
  tab.render(
    <MemoryRouter>
      {busy === undefined ? part : <DraftBusyContext.Provider value={busy}>{part}</DraftBusyContext.Provider>}
    </MemoryRouter>
  );
const buttonOf = (text: string) =>
  [...document.querySelectorAll<HTMLButtonElement>("button")].find((node) => node.textContent?.trim() === text);
/** Clicks button and lets its menu or popover open. */
const press = (button: HTMLButtonElement) =>
  act(async () => {
    vi.spyOn(button, "getBoundingClientRect").mockReturnValue(new DOMRect(1, 1, 180, 40));
    button.dispatchEvent(new MouseEvent("mousedown", { bubbles: true, cancelable: true, button: 0 }));
    button.click();
    await vi.advanceTimersByTimeAsync(50);
  });

it.each([
  ["stops while the page's draft is being saved", true, []],
  ["sorts outside it", undefined, [{ order_by: "priority" }]],
] as const)("a spreadsheet column's sort %s", async (_, busy, sorted) => {
  const changes: unknown[] = [];
  await onPage(
    busy,
    <table>
      <thead>
        <tr>
          <SpreadsheetHeaderColumn
            property="priority"
            displayProperties={{ priority: true }}
            displayFilters={{ order_by: "-created_at" }}
            handleDisplayFilterUpdate={(change) => changes.push(change)}
          />
        </tr>
      </thead>
    </table>
  );
  const trigger = document.querySelector<HTMLButtonElement>("th button")!;
  expect(trigger.disabled).toBe(busy === true);
  await press(trigger);
  const first = document.querySelector<HTMLButtonElement>('[role="menuitem"]');
  if (first) await act(async () => first.click());
  expect(changes).toEqual(sorted);
});

it.each([
  ["stop while the page's draft is being saved", true, []],
  ["change it outside it", undefined, [{ layout: "week" }]],
] as const)("the calendar's options %s", async (_, busy, changed) => {
  const changes: unknown[] = [];
  await onPage(
    busy,
    <CalendarOptionsDropdown calendarLayout="month" showWeekends={false} updateCalendar={(c) => changes.push(c)} />
  );
  const trigger = buttonOf("common.options")!;
  expect(trigger.disabled).toBe(busy === true);
  await press(trigger);
  const week = buttonOf("Week layout");
  if (week) await act(async () => week.click());
  expect(changes).toEqual(changed);
});

it.each([
  ["stops while the page's draft is being saved", true, true],
  ["clears outside it", undefined, false],
] as const)("the empty list's clearing of the view page's filters %s", async (_, busy, applies) => {
  const view = tab.app.workItemFilters.getOrCreateFilter({
    entityType: EIssuesStoreType.PROJECT_VIEW,
    entityId: V,
    initialExpression: { and: [{ priority__in: "urgent" }] },
    onExpressionChange: () => {},
  });
  await onPage(busy, <ProjectListEmptyState storeType={EIssuesStoreType.PROJECT_VIEW} filterId={V} />);
  const clear = buttonOf("project_issues.empty_state.issues_empty_filter.secondary_button.text")!;
  expect(clear.disabled).toBe(busy === true);
  await act(async () => clear.click());
  expect(view.hasActiveFilters).toBe(applies);
});
