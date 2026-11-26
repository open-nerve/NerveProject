// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// Update view waits for nerve (P6a spec 2.7, R9; re-96 I2, O1, m6, m11): it sends the page's filters and the draft's
// display as the change's turn comes, says "Confirming" while nerve answers; confirmed, it says so, its button goes, the
// focus lands on the filters row's Add filter, and the view, the page's draft and the filters' baseline are nerve's
// answer (the conditions as nerve gave them, the list read again under them); refused, it says why by nerve's code and
// leaves the button, the focus, the draft and the baseline as they were, no rejection left unhandled. Its follow-ups
// are the page's and its session's alone: nothing of an answer that comes once the page is left, or once another tab
// moved this one to another account. The page, the stores, SWR and HTTP are real (fake-view-saving.tsx).
import { act, type ReactNode } from "react";
import { afterEach, beforeEach, expect, it, onTestFinished, vi } from "vitest";
import type { View } from "@nerve/api-client";
import { EIssueFilterType } from "@nerve/constants";
import { deepCompareFilterExpressions } from "@nerve/utils";
import { settle } from "@/components/common/fake-quick-actions";
import { json, problem } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { toasts } from "@/lib/fake-toast";
import { toFilterExpression } from "@/lib/issue-filters";
import type { RootStore } from "@/store/root.store";
import { clickUpdate, rowAddFilter, updateButton, viewSavingTab } from "./fake-view-saving";

const bound = vi.hoisted(() => ({ app: undefined as RootStore | undefined, session: { loginId: "one" } }));
vi.mock("@/lib/store-context", async () => ({
  get rootStore() {
    return bound.app;
  },
  StoreContext: (await import("react")).createContext<RootStore | undefined>(undefined),
}));
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: bound.session }, publicClient: {} }));
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
/** Every rejection the run left unhandled. */
const unhandled: unknown[] = [];
const onUnhandled = (reason: unknown) => void unhandled.push(reason);
beforeEach(() => {
  bound.session.loginId = "one";
  toasts.length = 0;
  unhandled.length = 0;
  process.on("unhandledRejection", onUnhandled);
  onTestFinished(() => void process.off("unhandledRejection", onUnhandled));
});
afterEach(async () => page.close());

/** First's page, opened, its filters changed: its priority's condition taken away. */
async function changed() {
  page = await viewSavingTab();
  bound.app = page.app;
  await page.open();
  await page.dropPriority();
  expect(updateButton()?.textContent).toBe("common.filter_options.update_view");
}
/** Whether the filters' expression and its baseline are both the conditions of filters. */
const bothAre = (filters: View["filters"]) => {
  const filter = page.filter();
  const given = filter.adapter.toInternal(toFilterExpression(filters));
  return (
    deepCompareFilterExpressions(filter.expression, given) &&
    deepCompareFilterExpressions(filter.initialFilterExpression, given)
  );
};

it("saves the page's change as the view once nerve answers: the view, the draft and the baseline are the answer", async () => {
  await changed();
  const draft = page.draft()!;
  const lists = page.lists().length;
  await clickUpdate();
  await until(() => page.patches().length === 1, "the change of the view");
  expect(page.patches()[0]!.body).toEqual({
    filters: {},
    display_filters: draft.display_filters,
    display_properties: draft.display_properties,
  });
  expect(updateButton()?.textContent, "while nerve answers").toBe("common.filter_options.confirming");
  expect(updateButton()?.disabled).toBe(true);
  // nerve keeps urgent work items, and shows no empty groups
  const answer: View = {
    ...page.view,
    filters: { priority: ["urgent"] },
    display_filters: { ...draft.display_filters, show_empty_groups: false },
    updated_at: "2026-11-25T09:00:00Z",
  };
  await act(async () => page.patches()[0]!.answer(json(200, answer)));
  await settle();
  expect(toasts).toEqual([{ type: "success", title: "toast.success", message: "project_views.action.update.success" }]);
  expect(updateButton(), "its button goes").toBeUndefined();
  expect(document.activeElement, "the focus on the row's Add filter").toBe(rowAddFilter());
  expect(page.app.views.getViewById(page.view.id)).toEqual(answer);
  expect(page.draft()).toEqual({
    filters: answer.filters,
    display_filters: answer.display_filters,
    display_properties: answer.display_properties,
  });
  expect(page.app.issue.projectViewIssuesFilter.getIssueFilters(page.view.id)?.displayFilters?.show_empty_groups).toBe(
    false
  );
  expect(bothAre(answer.filters), "the conditions and their baseline are nerve's").toBe(true);
  expect(page.filter().hasChanges).toBe(false);
  // the list read again under nerve's conditions, once
  await until(() => page.lists().length === lists + 1, "the list under nerve's conditions");
  expect(page.lists().at(-1)!.query).toMatchObject({ priority: "urgent" });
  expect(page.patches()).toHaveLength(1);
});

it.each([
  [409, "navigation.views_disabled", "errors.navigation_views_disabled"],
  [404, "navigation.view_not_found", "errors.navigation_view_not_found"],
])(
  "keeps the change when nerve refuses it (%s %s): says why; the button, the focus, draft and baseline stay",
  async (status, code, message) => {
    await changed();
    const draft = page.draft();
    const button = updateButton()!;
    await clickUpdate();
    await until(() => page.patches().length === 1, "the change of the view");
    await act(async () => page.patches()[0]!.answer(problem(status, code)));
    // the row's button says Confirming for a moment more, against flickering
    await act(async () => vi.advanceTimersByTimeAsync(300));
    expect(toasts).toEqual([{ type: "error", title: "toast.error", message }]);
    expect(updateButton()?.textContent, "its button stays").toBe("common.filter_options.update_view");
    expect(updateButton()?.disabled).toBe(false);
    expect(document.activeElement, "the focus stays").toBe(button);
    expect(page.draft()).toEqual(draft);
    expect(bothAre({}), "the change stays the page's").toBe(false);
    expect(page.filter().hasChanges).toBe(true);
    expect(unhandled, "the refusal said, not left unhandled").toEqual([]);
  }
);

it("reads the draft's display for the body as its turn comes, after the view's change before it", async () => {
  await changed();
  const drafts = page.app.issue.projectViewIssuesFilter;
  // a change of the view out before it, held
  const renamed = page.app.views.update(page.view.id, () => ({ name: "Renamed" }));
  await until(() => page.patches().length === 1, "the change before it");
  await clickUpdate();
  await act(async () =>
    drafts.updateFilters("ws", page.view.id, EIssueFilterType.DISPLAY_FILTERS, { show_empty_groups: false })
  );
  expect(page.patches(), "waiting for the change before it").toHaveLength(1);
  await act(async () => page.patches()[0]!.answer(json(200, { ...page.view, name: "Renamed" })));
  await renamed;
  await until(() => page.patches().length === 2, "Update view's change");
  expect(page.patches()[1]!.body).toMatchObject({ display_filters: { show_empty_groups: false } });
});

it("follows nothing of nerve's answer once the page is left: no toast, no draft made again, no read of the view", async () => {
  await changed();
  await clickUpdate();
  await until(() => page.patches().length === 1, "the change of the view");
  const reads = page.reads(page.view).length;
  await page.leave();
  await settle();
  const answer = { ...page.view, filters: {} };
  await act(async () => page.patches()[0]!.answer(json(200, answer)));
  await settle();
  expect(toasts).toEqual([]);
  expect(page.app.issue.projectViewIssuesFilter.getIssueFilters(page.view.id), "no draft").toBeUndefined();
  expect(page.reads(page.view)).toHaveLength(reads);
  expect(page.app.views.getViewById(page.view.id), "the answer is the store's").toEqual(answer);
});

it("says nothing of nerve's answer once another tab moved this one to another account", async () => {
  await changed();
  await clickUpdate();
  await until(() => page.patches().length === 1, "the change of the view");
  bound.session.loginId = "two";
  await act(async () => page.patches()[0]!.answer(json(200, { ...page.view, filters: {} })));
  await settle();
  expect(toasts).toEqual([]);
});

it("lands the focus on the header's filters toggle once the filters row is gone as nerve answers", async () => {
  await changed();
  await clickUpdate();
  await until(() => page.patches().length === 1, "the change of the view");
  await act(async () => page.filter().toggleVisibility(false));
  await settle();
  expect(rowAddFilter(), "the row hidden").toBeUndefined();
  await act(async () => page.patches()[0]!.answer(json(200, { ...page.view, filters: {} })));
  await settle();
  // the toggle as it shows now: the row hidden, its conditions none, its Add filter
  expect(document.activeElement?.closest("header [data-filters-toggle]"), "the focus on the header's toggle").not.toBe(
    null
  );
});
