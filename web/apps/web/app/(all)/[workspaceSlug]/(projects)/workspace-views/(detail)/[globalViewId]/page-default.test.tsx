// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// A workspace's default view's page (M7/P6b spec 2.5, 2.8, R3, R4; re-97 I2): it reads no view; it lists the
// workspace's work items under its own term and the page's preferences, the display the browser keeps and the filters
// of this entry alone, which end as the page leaves. Each entry reads its own preferences, not through SWR: entered
// again at once, within SWR's dedupe window, it draws its list again with the browser's display. It reads the
// workspace's label catalog once, for any role. The stores, the layouts, SWR (one cache for the tab, as the app's) and
// HTTP are real (fake-view-page.tsx), at the app's routes; the preview a marker, the app's header a frame.
import { act, type ReactNode } from "react";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { WorkspaceRole } from "@nerve/api-client";
import { EIssueFilterType } from "@nerve/constants";
import { settle, where } from "@/components/common/fake-quick-actions";
import { WorkspaceViewRoutes, shownNow, workspaceViewPageTab } from "@/components/workspace/views/fake-view-page";
import { json } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { issueOf, issuePageOf } from "@/store/issue/fake-issues";
import type { RootStore } from "@/store/root.store";
import { workspacePageId } from "@/store/issue/workspace/page-id";
import Layout from "../layout";
import Page from "./page";

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
vi.mock("@/components/issues/peek-overview", () => ({ IssuePeekOverview: () => <i data-peek="" /> }));
vi.mock("@/components/core/app-header", () => ({
  AppHeader: (props: { header: ReactNode }) => <header>{props.header}</header>,
}));

const ASSIGNED = workspacePageId("ws", "assigned");
let page: Awaited<ReturnType<typeof workspaceViewPageTab>>;
beforeEach(() => {
  window.localStorage.clear();
  vi.spyOn(console, "error");
});
afterEach(async () => page.close());

/** A tab of ws with the caller of role, at the page of the default view key; SWR's cache is the tab's. */
async function opened(key: string, role: WorkspaceRole = 15) {
  page = await workspaceViewPageTab(role);
  bound.app = page.app;
  await page.render(
    <WorkspaceViewRoutes path={`/ws/workspace-views/${key}`} Layout={Layout} Page={Page} cache={new Map()} />
  );
}
const header = () => document.querySelector("header")?.textContent ?? "";
/** The page's display and filters as its store holds them now. */
const held = () => page.app.issue.globalIssuesFilter.getIssueFilters(ASSIGNED);
/** Leaves for the workspace's views, then goes Back to the page at once, as a person may. */
async function leftAndBack() {
  // what follows nerve's last answer settles first (SWR forgets a list's read once it answered, in a timer)
  await act(async () => vi.advanceTimersByTimeAsync(100));
  await act(async () => void where.navigate?.("/ws/workspace-views"));
  expect(document.body.textContent).toContain("the list");
  await act(async () => void where.navigate?.(-1));
}

it("reads no view: it lists the workspace's work items under its own term, titled by its name", async () => {
  await opened("assigned");
  await page.answerList();
  expect(page.lists()).toHaveLength(1);
  expect(page.lists()[0]!.query).toMatchObject({ assignee_id: "me", order_by: "-created_at", limit: "100" });
  expect(page.lists()[0]!.queryEntries.map(([name]) => name)).not.toContain("group_by");
  expect(page.nerve.calls.filter((call) => call.path.startsWith("/api/v0/views/"))).toEqual([]);
  expect(shownNow()).toBe("the view");
  expect(document.title).toBe("ws - default_global_view.assigned");
  expect(header()).toContain("default_global_view.assigned");
});

it("ends the filters of its entry as it leaves, and keeps its display: entered again, it lists with the display alone", async () => {
  await opened("assigned");
  await page.answerList();
  const store = page.app.issue.globalIssuesFilter;
  await act(async () =>
    store.updateFilters("ws", ASSIGNED, EIssueFilterType.DISPLAY_FILTERS, { order_by: "priority" })
  );
  await act(async () => store.updateFilterExpression("ws", ASSIGNED, { and: [{ priority__in: "high" }] }));
  await act(async () => void where.navigate?.("/ws/workspace-views"));
  expect(held(), "the page's preferences once it left").toBeUndefined();
  // past SWR's dedupe window, back to the page
  await act(async () => vi.advanceTimersByTimeAsync(5000));
  const before = page.lists().length;
  await act(async () => void where.navigate?.(-1));
  await until(() => page.lists().length > before, "the list again");
  expect(page.lists().at(-1)!.query).toMatchObject({ assignee_id: "me", order_by: "priority" });
  expect(page.lists().at(-1)!.query.priority).toBeUndefined();
  expect(held()?.displayFilters?.order_by).toBe("priority");
});

it("draws its list again with the browser's display when entered again at once, its own read each time (re-97 I2)", async () => {
  window.localStorage.setItem(
    "workspace_view_display",
    JSON.stringify({ "u-me": { [ASSIGNED]: { display_filters: { order_by: "priority" } } } })
  );
  await opened("assigned");
  await page.answerList();
  expect(page.lists()[0]!.query).toMatchObject({ order_by: "priority" });
  // within SWR's dedupe window of the first entry's read
  await leftAndBack();
  await until(() => page.lists().length === 2, "the list again");
  await page.answerList();
  expect(page.lists()[1]!.query).toMatchObject({ assignee_id: "me", order_by: "priority" });
  expect(shownNow()).toBe("the view");
  expect(held()?.displayFilters?.order_by).toBe("priority");
});

it("lists its own rows when entered again while its list's first read is still out, another view's page between", async () => {
  await opened("all-issues");
  await until(() => page.lists().length === 1, "All work items' list");
  // Assigned, its list answered: none of its own; Back to All work items, whose read SWR joins, sending none again
  await act(async () => void where.navigate?.("/ws/workspace-views/assigned"));
  await until(() => page.lists().length === 2, "Assigned's list");
  await act(async () => page.lists()[1]!.answer(json(200, issuePageOf([]))));
  await act(async () => void where.navigate?.(-1));
  expect(header()).toContain("default_global_view.all_issues");
  await act(async () => page.lists()[0]!.answer(json(200, issuePageOf([issueOf("in-a")]))));
  await settle();
  expect(page.lists(), "the first read, joined").toHaveLength(2);
  expect(document.getElementById("issue-in-a"), "All work items' row, on its page").not.toBeNull();
});

it.each([
  ["its admin", 20],
  ["its member", 15],
  ["its guest", 5],
] as const)("lists for %s, and reads the workspace's label catalog once", async (_, role) => {
  await opened("all-issues", role);
  await page.answerList();
  await until(() => page.catalogReads().length > 0, "the label catalog");
  await settle();
  expect(shownNow()).toBe("the view");
  expect(page.catalogReads()).toHaveLength(1);
  expect(page.lists()).toHaveLength(1);
});
