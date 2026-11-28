// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// Update view on a workspace's own view's page (M7/P6b spec 2.7, 2.9, R10): offered to the view's owner while he is the
// workspace's admin or member (an owner made its guest is not offered it, nor is anyone another's view) once the page
// holds a change of the view; it waits for nerve: confirmed, it says so in the workspace's words, its button goes, the
// focus lands on the filters row's Add filter, and the view, the draft and the filters' baseline are nerve's answer;
// refused, it says why by nerve's code and leaves the button, the draft and the baseline; once the page is left, it
// says nothing. A view held as a list is saved back as a list (R8). The page, the stores, SWR and HTTP are real
// (fake-workspace-view-saving.tsx); the app's header a frame.
import { act, createElement, type ReactNode } from "react";
import { MemoryRouter, Route, Routes } from "react-router";
import { SWRConfig } from "swr";
import { afterEach, beforeEach, expect, it, onTestFinished, vi } from "vitest";
import type { View } from "@nerve/api-client";
import { deepCompareFilterExpressions } from "@nerve/utils";
import { WorkspaceViewLayoutRoot } from "@/components/issues/issue-layouts/roots/workspace-view-layout-root";
import { settle } from "@/components/common/fake-quick-actions";
import { json, problem } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { toasts } from "@/lib/fake-toast";
import { toFilterExpression } from "@/lib/issue-filters";
import type { RootStore } from "@/store/root.store";
import { refreshWorkspacesWhileMounted, workspaceOf } from "@/store/workspace/fake-workspaces";
import { clickUpdate, rowAddFilter, updateButton } from "./fake-view-saving";
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
/** Every rejection the run left unhandled. */
const unhandled: unknown[] = [];
const onUnhandled = (reason: unknown) => void unhandled.push(reason);
beforeEach(() => {
  window.localStorage.clear();
  toasts.length = 0;
  unhandled.length = 0;
  process.on("unhandledRejection", onUnhandled);
  onTestFinished(() => void process.off("unhandledRejection", onUnhandled));
});
afterEach(async () => page.close());

/** First's page, opened as opening says, its filters changed: its priority's condition taken away. */
async function changed(opening: Parameters<typeof workspaceSavingTab>[0] = {}) {
  page = await workspaceSavingTab(opening);
  bound.app = page.app;
  await page.open();
  await page.dropPriority();
  expect(updateButton()?.textContent, "Update view").toBe("common.filter_options.update_view");
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

it.each([
  ["its owner, the workspace's member", 15, "u-me", true],
  ["its owner, the workspace's admin", 20, "u-me", true],
  ["a member, another's", 15, "u-other", false],
  ["an admin, another's", 20, "u-other", false],
] as const)("offers %s Update view once the page's filters change: %s", async (_, role, owner, offered) => {
  page = await workspaceSavingTab({ role, view: { owned_by_id: owner } });
  bound.app = page.app;
  await page.open();
  expect(updateButton(), "the view as it is").toBeUndefined();
  await page.dropPriority();
  expect(page.filter().hasChanges, "the page holds a change of the view").toBe(true);
  expect(updateButton() !== undefined).toBe(offered);
  expect(page.filter().canUpdateView).toBe(offered);
});

it("offers its owner no Update view once he is made the workspace's guest", async () => {
  page = await workspaceSavingTab();
  bound.app = page.app;
  // the page's work items alone, its draft read: no page that sends the guest away
  const drafts = page.app.issue.globalIssuesFilter;
  const seeded = drafts.fetchFilters("ws", page.view.id);
  await page.answerRead(page.view);
  await seeded;
  await page.render(
    <SWRConfig value={{ provider: () => new Map() }}>
      <MemoryRouter initialEntries={[`/ws/workspace-views/${page.view.id}`]}>
        <Routes>
          <Route
            path=":workspaceSlug/workspace-views/:globalViewId"
            element={<WorkspaceViewLayoutRoot workspaceSlug="ws" globalViewId={page.view.id} />}
          />
        </Routes>
      </MemoryRouter>
    </SWRConfig>
  );
  await page.answerList();
  await settle();
  await page.dropPriority();
  expect(page.filter().canUpdateView, "his own, the workspace's member").toBe(true);
  await act(async () => {
    await refreshWorkspacesWhileMounted(page.nerve, page.app.workspaceRoot, [workspaceOf("ws", { role: 5 })]);
  });
  await settle();
  expect(updateButton()).toBeUndefined();
  expect(page.filter().canUpdateView).toBe(false);
});

it("saves the page's change once nerve answers: says so, the view, the draft and the baseline are the answer", async () => {
  await changed();
  const draft = page.draft()!;
  await clickUpdate();
  await until(() => page.patches().length === 1, "the change of the view");
  expect(page.patches()[0]!.body).toEqual({
    filters: {},
    display_filters: draft.display_filters,
    display_properties: draft.display_properties,
  });
  expect(updateButton()?.textContent, "while nerve answers").toBe("common.filter_options.confirming");
  const answer: View = {
    ...page.view,
    filters: { priority: ["urgent"] },
    display_filters: { ...draft.display_filters, sub_issue: false },
    updated_at: "2026-11-28T09:00:00Z",
  };
  await act(async () => page.patches()[0]!.answer(json(200, answer)));
  await settle();
  expect(toasts).toEqual([
    { type: "success", title: "toast.success", message: "workspace_views.action.update.success" },
  ]);
  expect(updateButton(), "its button goes").toBeUndefined();
  expect(document.activeElement, "the focus on the row's Add filter").toBe(rowAddFilter());
  expect(page.app.views.getViewById(page.view.id)).toEqual(answer);
  expect(page.draft()).toEqual({
    filters: answer.filters,
    display_filters: answer.display_filters,
    display_properties: answer.display_properties,
  });
  expect(bothAre(answer.filters), "the conditions and their baseline are nerve's").toBe(true);
  expect(page.filter().hasChanges).toBe(false);
  expect(page.patches()).toHaveLength(1);
});

it.each([
  [404, "navigation.view_not_found", "errors.navigation_view_not_found"],
  [403, "forbidden", "errors.forbidden"],
])(
  "keeps the change when nerve refuses it (%s %s): says why; the button, draft and baseline stay",
  async (status, code, message) => {
    await changed();
    const draft = page.draft();
    await clickUpdate();
    await until(() => page.patches().length === 1, "the change of the view");
    await act(async () => page.patches()[0]!.answer(problem(status, code)));
    await act(async () => vi.advanceTimersByTimeAsync(300));
    expect(toasts).toEqual([{ type: "error", title: "toast.error", message }]);
    expect(updateButton()?.textContent, "its button stays").toBe("common.filter_options.update_view");
    expect(page.draft()).toEqual(draft);
    expect(bothAre({}), "the change stays the page's").toBe(false);
    expect(page.filter().hasChanges).toBe(true);
    expect(unhandled, "the refusal said, not left unhandled").toEqual([]);
  }
);

it("says nothing of nerve's answer once the page is left, nor reads the view again", async () => {
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
  expect(page.app.issue.globalIssuesFilter.getIssueFilters(page.view.id), "no draft shown").toBeUndefined();
  expect(page.reads(page.view)).toHaveLength(reads);
  expect(page.app.views.getViewById(page.view.id), "the answer is the store's").toEqual(answer);
});

it("saves a view held as a list back as a list, the page drawing it as the spreadsheet (R8)", async () => {
  page = await workspaceSavingTab();
  bound.app = page.app;
  expect(page.view.display_filters.layout, "First, as nerve holds it").toBe("list");
  await page.open();
  expect(document.querySelector("table thead"), "drawn as the spreadsheet").not.toBeNull();
  await page.dropPriority();
  expect(updateButton()?.textContent, "Update view").toBe("common.filter_options.update_view");
  await clickUpdate();
  await until(() => page.patches().length === 1, "the change of the view");
  expect(page.patches()[0]!.body).toMatchObject({ display_filters: { layout: "list" } });
});
