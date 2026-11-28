// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// A workspace's own view's page (M7/P6b spec 2.8, R8, R11, R30; replaces the old layout.test.tsx's three cells): its
// layout, keyed by the route's view, with its header and its body. An entry reads the view once, then the workspace's
// list under the view's filters, the spreadsheet whatever layout the view holds; nerve finding no such view, refusing
// it (an owner made the workspace's guest), giving a project's view of that id, and the workspace's guest (who reads
// nothing) are each "the view does not exist", with the way to the workspace's views; any other failure says why, with
// a retry. The header draws its crumbs in each state. The stores, the layouts, SWR and HTTP are real
// (fake-view-page.tsx); the preview a marker, the app's header a frame that shows the header.
import { act, type ReactNode } from "react";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { View, WorkspaceRole } from "@nerve/api-client";
import { settle, where } from "@/components/common/fake-quick-actions";
import { WorkspaceViewRoutes, shownNow, workspaceViewPageTab } from "@/components/workspace/views/fake-view-page";
import { json, problem } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import type { RootStore } from "@/store/root.store";
import { ME, viewOf } from "@/store/views/fake-views";
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

let page: Awaited<ReturnType<typeof workspaceViewPageTab>>;
let first: View;
beforeEach(() => {
  vi.spyOn(console, "error");
});
afterEach(async () => page.close());

/** A tab of ws with the caller of role, at the page of its view First, held as nerve gives a view: as a list. */
async function opened(role: WorkspaceRole = 15) {
  page = await workspaceViewPageTab(role);
  bound.app = page.app;
  first = page.view("First", { filters: { priority: ["high"] } });
  await page.render(
    <WorkspaceViewRoutes path={`/ws/workspace-views/${first.id}`} Layout={Layout} Page={Page} cache={new Map()} />
  );
}
const header = () => document.querySelector("header")?.textContent ?? "";
/** The button of region whose text is name. */
const buttonIn = (region: string, name: string) =>
  [...document.querySelectorAll<HTMLButtonElement>(`${region} button`)].find((b) => b.textContent?.trim() === name);
/** The header's crumb of the workspace's views, a link to their list, and no crumb of the view. */
const crumbs = () =>
  document.querySelector("header a[href='/ws/workspace-views']")?.textContent === "views" &&
  !header().includes("First");
/** Whether the page asked nerve for any view. */
const viewReads = () => page.nerve.calls.filter((call) => call.path.startsWith("/api/v0/views/"));

it("reads its view once as it opens, then the workspace's list under the view's filters, titled by the view", async () => {
  await opened();
  await until(() => page.reads(first).length === 1, "the view's read");
  expect(shownNow()).toBe("reading");
  await page.answerRead(first);
  await page.answerList();
  expect(page.reads(first)).toHaveLength(1);
  expect(page.lists()).toHaveLength(1);
  expect(page.lists()[0]!.query).toMatchObject({ priority: "high", order_by: "-created_at" });
  expect(page.lists()[0]!.queryEntries.map(([name]) => name)).not.toContain("group_by");
  expect(shownNow()).toBe("the view");
  expect(document.title).toBe("ws - First");
  expect(header()).toContain("First");
  expect(header()).toContain("common.display");
  expect(page.writes()).toEqual([]);
});

it("draws a view held as a list as the spreadsheet, and keeps the layout it holds in its draft", async () => {
  await opened();
  await page.answerRead(first, json(200, { ...first, display_filters: { ...first.display_filters, layout: "list" } }));
  await page.answerList();
  expect(document.querySelector("table thead")?.textContent).toContain("common.priority");
  expect(page.app.issue.globalIssuesFilter.confirmed(first.id)?.display_filters.layout).toBe("list");
});

it("draws the crumbs to the workspace's views while it reads the view, then the view's own header", async () => {
  await opened();
  await until(() => page.reads(first).length === 1, "the view's read");
  expect(crumbs()).toBe(true);
  await page.answerRead(first);
  expect(header()).toContain("First");
});

it.each([
  ["has none", problem(404, "navigation.view_not_found")],
  ["refuses it, its owner made the workspace's guest", problem(403, "forbidden")],
])("says the view does not exist when nerve %s, under the crumbs, with the way to the views", async (_, answer) => {
  await opened();
  await page.answerRead(first, answer);
  expect(shownNow()).toBe("not found");
  expect(crumbs()).toBe(true);
  expect(page.lists()).toEqual([]);
  await act(async () => buttonIn("body", "workspace_views.not_found.action")?.click());
  expect(where.location?.pathname).toBe("/ws/workspace-views");
});

it("says the view does not exist when nerve gives a project's view of that id, and lists nothing", async () => {
  await opened();
  await page.answerRead(first, json(200, viewOf(page.project, "Bugs", ME, { id: first.id })));
  await settle();
  expect(page.app.views.getViewById(first.id)?.project_id, "the store gives A's view").toBe("A");
  expect(shownNow()).toBe("not found");
  expect(crumbs()).toBe(true);
  expect(page.lists()).toEqual([]);
});

it("says why nerve could not read the view, under the crumbs, and reads it again on retry", async () => {
  await opened();
  await page.answerRead(first, problem(503, "server_busy"));
  expect(shownNow()).toBe("failed");
  expect(document.querySelector('[role="alert"]')).not.toBeNull();
  expect(crumbs()).toBe(true);
  await act(async () => buttonIn("body", "common.retry")?.click());
  await page.answerRead(first);
  await page.answerList();
  expect(page.reads(first)).toHaveLength(2);
  expect(shownNow()).toBe("the view");
});

it("tells the workspace's guest the view does not exist, under the crumbs, asking nerve for no view", async () => {
  await opened(5);
  await settle();
  expect(shownNow()).toBe("not found");
  expect(crumbs()).toBe(true);
  expect(viewReads()).toEqual([]);
  expect(page.lists()).toEqual([]);
  expect(vi.mocked(console.error).mock.calls).toEqual([]);
});
