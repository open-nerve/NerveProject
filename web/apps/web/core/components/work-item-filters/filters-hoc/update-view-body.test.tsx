// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// Update view's body is reckoned whole from the page's draft as the change's turn comes (M7/P6b spec 2.9, R10, O1),
// never from the expression the filters row gave when it was clicked: a change of the draft made while an earlier change
// of the view is out is what is sent, on a project's view's page and on a workspace's own view's page alike, and nerve's
// answer is then the filters' expression and baseline. The two edges (re-97 m3): a turn that comes once the page has
// left sends the draft as it was before it left (its preferences are kept: discard leaves them), saying nothing; with no
// draft held, nothing is sent and the failure is said. The pages, the stores, SWR and HTTP are real (fake-view-saving.tsx,
// fake-workspace-view-saving.tsx); the app's header a frame.
import { act, createElement, type ReactNode } from "react";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { View, ViewFilters } from "@nerve/api-client";
import type { TUpdateViewOptions } from "@nerve/constants";
import type { IWorkItemFilterInstance } from "@nerve/shared-state";
import type { TWorkItemFilterExpression } from "@nerve/types";
import { EIssuesStoreType } from "@nerve/types";
import { deepCompareFilterExpressions } from "@nerve/utils";
import { settle } from "@/components/common/fake-quick-actions";
import { type Call, json } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { toasts } from "@/lib/fake-toast";
import { toFilterExpression } from "@/lib/issue-filters";
import type { IPagePreferencesFilter } from "@/store/issue/helpers/page-preferences-filter.store";
import type { RootStore } from "@/store/root.store";
import { clickUpdate, updateButton, viewSavingTab } from "./fake-view-saving";
import { workspaceSavingTab } from "./fake-workspace-view-saving";
import { useViewUpdate } from "./use-view-update";

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
vi.mock("@/components/breadcrumbs/common", () => ({ CommonProjectBreadcrumbs: () => <span>the project</span> }));
vi.mock("@/components/core/app-header", () => ({
  AppHeader: (props: { header: ReactNode }) => <header>{props.header}</header>,
}));

/** The draft's change made while Update view waits: urgent work items. */
const URGENT: ViewFilters = { priority: ["urgent"] };

let close: (() => Promise<void>) | undefined;
beforeEach(() => {
  toasts.length = 0;
});
afterEach(async () => {
  await close?.();
  close = undefined;
});

/** Whether the page's filters' expression and its baseline are both the conditions of filters, its change none. */
function bothAre(filter: IWorkItemFilterInstance, filters: ViewFilters) {
  const given = filter.adapter.toInternal(toFilterExpression(filters));
  return (
    deepCompareFilterExpressions(filter.expression, given) &&
    deepCompareFilterExpressions(filter.initialFilterExpression, given) &&
    !filter.hasChanges
  );
}

/**
 * Holds a change of the page's view before Update view (its name's, as the view's dialog sends it), clicks Update view,
 * changes the page's draft to urgent work items through its preferences (as the page's filters do), and lets the
 * change before it through: gives Update view's change. The page's stores are app's; patches gives the view's changes.
 */
async function updatedAfterAChangeOfDraft(
  app: RootStore,
  view: View,
  patches: () => Call[],
  drafts: IPagePreferencesFilter
) {
  const renamed = app.views.update(view.id, () => ({ name: "Renamed" }));
  await until(() => patches().length === 1, "the change before it");
  await clickUpdate();
  await act(async () => drafts.updateFilterExpression("ws", view.id, toFilterExpression(URGENT)));
  expect(patches(), "Update view waits for the change before it").toHaveLength(1);
  await act(async () => patches()[0]!.answer(json(200, { ...view, name: "Renamed" })));
  await act(async () => void (await renamed));
  await until(() => patches().length === 2, "Update view's change");
  return patches()[1]!;
}

it("sends a project's view's draft as Update view's turn comes, not the expression it was clicked with (O1)", async () => {
  const page = await viewSavingTab();
  bound.app = page.app;
  close = page.close;
  await page.open();
  await page.dropPriority();
  expect(updateButton()?.textContent).toBe("common.filter_options.update_view");
  const drafts = page.app.issue.projectViewIssuesFilter;
  const update = await updatedAfterAChangeOfDraft(page.app, page.view, page.patches, drafts);
  const draft = page.draft()!;
  expect(draft.filters, "the draft changed meanwhile").toEqual(URGENT);
  expect(update.body).toEqual({
    filters: URGENT,
    display_filters: draft.display_filters,
    display_properties: draft.display_properties,
  });
  await act(async () => update.answer(json(200, { ...page.view, name: "Renamed", filters: URGENT })));
  await settle();
  expect(toasts).toEqual([{ type: "success", title: "toast.success", message: "project_views.action.update.success" }]);
  expect(bothAre(page.filter(), URGENT), "the expression and its baseline are nerve's answer").toBe(true);
});

it("sends a workspace's own view's draft as Update view's turn comes, not the expression it was clicked with (O1)", async () => {
  const page = await workspaceSavingTab();
  bound.app = page.app;
  close = page.close;
  await page.open();
  await page.dropPriority();
  expect(updateButton()?.textContent).toBe("common.filter_options.update_view");
  const drafts = page.app.issue.globalIssuesFilter;
  const update = await updatedAfterAChangeOfDraft(page.app, page.view, page.patches, drafts);
  const draft = page.draft()!;
  expect(draft.filters, "the draft changed meanwhile").toEqual(URGENT);
  expect(update.body).toEqual({
    filters: URGENT,
    display_filters: draft.display_filters,
    display_properties: draft.display_properties,
  });
  await act(async () => update.answer(json(200, { ...page.view, name: "Renamed", filters: URGENT })));
  await settle();
  expect(toasts).toEqual([
    { type: "success", title: "toast.success", message: "workspace_views.action.update.success" },
  ]);
  expect(bothAre(page.filter(), URGENT), "the expression and its baseline are nerve's answer").toBe(true);
});

it("sends the draft as it was before the page left when Update view's turn comes after, and says nothing (re-97 m3)", async () => {
  const page = await workspaceSavingTab();
  bound.app = page.app;
  close = page.close;
  await page.open();
  await page.dropPriority();
  expect(updateButton()?.textContent).toBe("common.filter_options.update_view");
  const renamed = page.app.views.update(page.view.id, () => ({ name: "Renamed" }));
  await until(() => page.patches().length === 1, "the change before it");
  await clickUpdate();
  const before = page.draft()!;
  await page.leave();
  await settle();
  expect(page.app.issue.globalIssuesFilter.getIssueFilters(page.view.id), "the page's filters ended").toBeUndefined();
  await act(async () => page.patches()[0]!.answer(json(200, { ...page.view, name: "Renamed" })));
  await act(async () => void (await renamed));
  await until(() => page.patches().length === 2, "Update view's change");
  expect(page.patches()[1]!.body).toEqual({
    filters: before.filters,
    display_filters: before.display_filters,
    display_properties: before.display_properties,
  });
  const answer = { ...page.view, name: "Renamed", filters: before.filters };
  await act(async () => page.patches()[1]!.answer(json(200, answer)));
  await settle();
  expect(toasts, "the page left: nothing said").toEqual([]);
  expect(page.app.views.getViewById(page.view.id), "the answer is the store's").toEqual(answer);
});

let options: TUpdateViewOptions<TWorkItemFilterExpression> | undefined;
/** Update view of the view of viewId on a page of storeType that holds no draft of it. */
function NoDraft(props: { storeType: EIssuesStoreType.PROJECT_VIEW | EIssuesStoreType.GLOBAL; viewId: string }) {
  options = useViewUpdate({
    storeType: props.storeType,
    workspaceSlug: "ws",
    viewId: props.viewId,
    enabled: true,
    writes: true,
    shown: undefined,
  });
  return null;
}

it.each([
  ["a project's view's", EIssuesStoreType.PROJECT_VIEW],
  ["a workspace's own view's", EIssuesStoreType.GLOBAL],
] as const)(
  "sends nothing on %s page that holds no draft of it, and says it failed (re-97 m3)",
  async (_, storeType) => {
    const page = await viewSavingTab();
    bound.app = page.app;
    close = page.close;
    options = undefined;
    await page.render(<NoDraft storeType={storeType} viewId={page.view.id} />);
    expect(page.app.issue.projectViewIssuesFilter.confirmed(page.view.id)).toBeUndefined();
    expect(page.app.issue.globalIssuesFilter.confirmed(page.view.id)).toBeUndefined();
    let refused: unknown;
    await act(async () => Promise.resolve(options!.onViewUpdate({})).catch((error: unknown) => void (refused = error)));
    await settle();
    expect(refused, "Update view fails").toBeInstanceOf(Error);
    expect(page.patches(), "nothing is sent").toEqual([]);
    expect(toasts).toEqual([{ type: "error", title: "toast.error", message: "errors.unknown" }]);
  }
);
