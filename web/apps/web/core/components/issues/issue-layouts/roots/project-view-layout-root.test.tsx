// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// A project's view's page's work items (P6a spec 2.5, 2.6, R10, R18): the project's list under the page's draft of the
// view, by the view's store type, with the page's own preview host. A change of the page's filters is the draft's
// alone: the list is read again under it, nothing is sent to nerve, and the filters' saved baseline stays the view's
// (so that Update view, T8, has a change to save); neither Save view nor Update view is offered yet (T8 gives them). A
// condition on something nerve no longer has is sent as the view holds it. The stores, the layouts, SWR and HTTP are
// real (fake-view-page.tsx); the preview a stand-in that says its host and its store type.
import { act, type ReactNode } from "react";
import { MemoryRouter } from "react-router";
import { SWRConfig } from "swr";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { View } from "@nerve/api-client";
import { EIssuesStoreType } from "@nerve/types";
import { viewPageTab } from "@/components/views/fake-view-page";
import { until } from "@/lib/auth/fake-time";
import { toProjectFilters } from "@/lib/issue-filters";
import type { RootStore } from "@/store/root.store";
import { ProjectViewLayoutRoot } from "./project-view-layout-root";

const bound = vi.hoisted(() => ({
  app: undefined as RootStore | undefined,
  params: {} as Record<string, string>,
  peek: undefined as { hostScope: string; store: unknown } | undefined,
}));
vi.mock("@/lib/store-context", async () => ({
  get rootStore() {
    return bound.app;
  },
  StoreContext: (await import("react")).createContext<RootStore | undefined>(undefined),
}));
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "one" } }, publicClient: {} }));
vi.mock("@/lib/auth/use-session", () => ({ useSession: () => ({ status: "signed-in", loginId: "one" }) }));
vi.mock("react-router", async (original) => ({
  ...(await original<typeof import("react-router")>()),
  useParams: () => bound.params,
}));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@atlaskit/pragmatic-drag-and-drop-auto-scroll/element", () => ({ autoScrollForElements: () => () => {} }));
vi.mock("@/components/core/render-if-visible-HOC", () => ({
  default: ({ children }: { children: ReactNode }) => children,
}));
vi.mock("../../peek-overview", async () => {
  const { useContext } = await import("react");
  const { IssuesStoreContext } = await import("@/hooks/use-issue-layout-store");
  return {
    IssuePeekOverview: (props: { hostScope: string }) => {
      bound.peek = { hostScope: props.hostScope, store: useContext(IssuesStoreContext) };
      return null;
    },
  };
});

const GONE = "019a0000-0000-7000-8000-0000000000ff";
let page: Awaited<ReturnType<typeof viewPageTab>>;
let view: View;
beforeEach(async () => {
  page = await viewPageTab();
  bound.app = page.app;
  // the view's conditions: a priority, and a label nerve no longer has
  view = page.view("First", { filters: { priority: ["high"], label_id: [GONE] } });
  bound.params = { workspaceSlug: "ws", projectId: "A", viewId: view.id };
  bound.peek = undefined;
});
afterEach(async () => page.close());

/** The page's work items, once nerve has given the view and the page's list. */
async function shown() {
  const seeded = page.app.issue.projectViewIssuesFilter.fetchFilters("ws", view.id);
  await page.answerRead(view);
  await seeded;
  await page.render(
    <SWRConfig value={{ provider: () => new Map() }}>
      <MemoryRouter>
        <ProjectViewLayoutRoot workspaceSlug="ws" projectId="A" viewId={view.id} />
      </MemoryRouter>
    </SWRConfig>
  );
  await page.answerList();
}
const filter = () => page.app.workItemFilters.getFilter(EIssuesStoreType.PROJECT_VIEW, view.id)!;

it("lists the project's work items under the view's conditions, a gone label's too, in the view page's own host", async () => {
  await shown();
  expect(page.lists()).toHaveLength(1);
  expect(page.lists()[0]!.query).toMatchObject({ priority: "high", label_id: GONE });
  expect(bound.peek).toEqual({
    hostScope: JSON.stringify(["project-view", "ws", "A", view.id]),
    store: EIssuesStoreType.PROJECT_VIEW,
  });
});

it("changes only its draft as its filters change: the list again under it, nothing sent, the baseline the view's", async () => {
  await shown();
  expect(filter().hasChanges).toBe(false);
  const priority = filter().allConditions.find((condition) => condition.property === "priority")!;
  await act(async () => filter().removeCondition(priority.id));
  await until(() => page.lists().length === 2, "the list again");
  expect(page.lists()[1]!.query).toMatchObject({ label_id: GONE });
  expect(page.lists()[1]!.query.priority).toBeUndefined();
  const draft = page.app.issue.projectViewIssuesFilter.getIssueFilters(view.id)?.richFilters;
  expect(toProjectFilters(draft ?? {})).toEqual({ label_id: [GONE] });
  await act(async () => vi.advanceTimersByTimeAsync(1000));
  expect(page.writes()).toEqual([]);
  // the saved baseline is still the view's: the page holds a change of it
  expect(filter().hasChanges).toBe(true);
  // neither Save view nor Update view is the page's yet (T8)
  expect(filter().hasActiveFilters).toBe(true);
  expect([filter().canSaveView, filter().canUpdateView]).toEqual([false, false]);
});
