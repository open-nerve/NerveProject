// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// A project's view's page (P6a spec 2.2, 2.6, R6; M3 handoff 1; re-96 I1, m8): its layout, keyed by the route's view,
// with its header, its small screen's header and its body. Each entry reads the view once, then the project's list
// under the view's filters and display; it says that nerve finds no such view, or why it could not read it, with a
// retry, and reads nothing while the project's views are off; the header draws its crumbs in each of those states. The
// switcher reads the project's views once opened; a new work item is offered to admins and members; deleting the view
// goes to the project's views. The stores, the layouts, SWR and HTTP are real (fake-view-page.tsx); the work item's
// preview is a stand-in, and the app's header a frame that shows the header and the small screen's header.
import { act, type ComponentType, type ReactNode } from "react";
import { MemoryRouter, Route, Routes, useParams } from "react-router";
import { SWRConfig } from "swr";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { View } from "@nerve/api-client";
import { Where, dialogButton, settle, where } from "@/components/common/fake-quick-actions";
import { viewPageTab } from "@/components/views/fake-view-page";
import { json, noContent, problem } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { toasts } from "@/lib/fake-toast";
import { loadProjects, projectOf } from "@/store/project/fake-projects";
import type { RootStore } from "@/store/root.store";
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

type RouteProps = { params: Readonly<Record<string, string | undefined>> };
function RouteOf({ Component }: { Component: unknown }) {
  const Of = Component as ComponentType<RouteProps>;
  return <Of params={useParams()} />;
}

let page: Awaited<ReturnType<typeof viewPageTab>>;
let first: View, second: View;
const consoleErrors = () => vi.mocked(console.error).mock.calls;
beforeEach(() => {
  toasts.length = 0;
  vi.spyOn(console, "error");
});
afterEach(async () => page.close());

/** A tab of project A with the caller of role and the project's fields, at the page of its view First. */
async function opened(fields: Parameters<typeof viewPageTab>[0] = {}) {
  page = await viewPageTab(fields);
  bound.app = page.app;
  first = page.view("First", { filters: { priority: ["high"] } });
  first.display_filters = { ...first.display_filters, order_by: "-updated_at" };
  second = page.view("Second");
  await page.render(
    <SWRConfig value={{ provider: () => new Map() }}>
      <MemoryRouter initialEntries={[`/ws/projects/A/views/${first.id}`]}>
        <Routes>
          <Route path=":workspaceSlug/projects/:projectId/views/:viewId" element={<RouteOf Component={Layout} />}>
            <Route index element={<RouteOf Component={Page} />} />
          </Route>
          <Route path=":workspaceSlug/projects/:projectId/views" element={<p>the list</p>} />
        </Routes>
        <Where />
      </MemoryRouter>
    </SWRConfig>
  );
}
/** What the page shows: all of it, its header's, and its small screen's header's (the app's header frames them). */
const text = () => document.body.textContent ?? "";
const header = () => document.querySelector("header")?.textContent ?? "";
const small = () => document.querySelector("aside")?.textContent ?? "";
/** The button of region whose text is name. */
const buttonIn = (region: string, name: string) =>
  [...document.querySelectorAll<HTMLButtonElement>(`${region} button`)].find((b) => b.textContent?.trim() === name);
/** The header's crumbs to the project's views: the project's, and the views' link to their list. */
const crumbs = () => {
  const link = document.querySelector<HTMLAnchorElement>("header a[href='/ws/projects/A/views']");
  return header().includes("the project") && link?.textContent === "views";
};
/** Opens the switcher of the header's crumb of the view. */
const openSwitcher = () =>
  act(async () => {
    buttonIn("header", first.name)?.click();
    await vi.advanceTimersByTimeAsync(20);
  });

it("reads its view once as it opens, then the project's work items under the view's filters and display", async () => {
  await opened();
  await until(() => page.reads(first).length === 1, "the view's read");
  expect(header()).not.toContain("common.display");
  await page.answerRead(first);
  await page.answerList();
  expect(page.reads(first)).toHaveLength(1);
  expect(page.lists()).toHaveLength(1);
  expect(page.lists()[0]!.query).toMatchObject({ priority: "high", order_by: "-updated_at" });
  expect(document.title).toBe("A - First");
  // the header in full: the view's crumb, its display and a new work item; the small screen's display
  expect(header()).toContain("First");
  expect(header()).toContain("common.display");
  expect(buttonIn("header", "issue.add.label")).toBeDefined();
  expect(small()).toContain("common.display");
  expect(page.writes()).toEqual([]);
});

it("reads the project's views only once its switcher opens, and another view mounts the page afresh", async () => {
  await opened();
  await page.answerRead(first);
  await page.answerList();
  await settle();
  expect(page.viewLists(), "the switcher closed").toEqual([]);
  await openSwitcher();
  await until(() => page.viewLists().length === 1, "the project's views");
  await act(async () => page.viewLists()[0]!.answer(json(200, { data: [first, second] })));
  const option = [...document.querySelectorAll<HTMLElement>('[role="option"]')].find((o) => o.textContent === "Second");
  expect(option, "the switcher's other view").toBeDefined();
  await act(async () => {
    option!.dispatchEvent(new MouseEvent("mousedown", { bubbles: true, cancelable: true }));
    option!.click();
    await vi.advanceTimersByTimeAsync(50);
  });
  expect(where.location?.pathname).toBe(`/ws/projects/A/views/${second.id}`);
  await page.answerRead(second);
  await page.answerList();
  await settle();
  expect(page.reads(second), "one read of the other view").toHaveLength(1);
  expect(page.app.issue.projectViewIssuesFilter.getIssueFilters(first.id), "the first view's draft").toBeUndefined();
  expect(header()).toContain("Second");
  // Back: the first view is read again, and listed as nerve now gives it
  await act(async () => void where.navigate?.(-1));
  await until(() => page.reads(first).length === 2, "the first view again");
  await page.answerRead(first, json(200, { ...first, filters: { priority: ["low"] } }));
  await until(() => page.lists().length === 3, "the first view's list again");
  expect(page.lists()[2]!.query).toMatchObject({ priority: "low" });
  expect(page.viewLists()).toHaveLength(1);
});

it("draws the crumbs to the project's views while it reads the view, then the view's own header", async () => {
  await opened();
  await until(() => page.reads(first).length === 1, "the view's read");
  expect(crumbs()).toBe(true);
  expect(header()).not.toContain("First");
  expect(small()).toBe("");
  await page.answerRead(first);
  expect(header()).toContain("First");
});

it("says that nerve finds no such view, with the way back to the project's views, under the crumbs", async () => {
  await opened();
  await page.answerRead(first, problem(404, "navigation.view_not_found"));
  expect(text()).toContain("project_views.not_found.title");
  expect(text()).not.toContain("common.retry");
  expect(crumbs()).toBe(true);
  await act(async () => buttonIn("body", "project_views.not_found.action")?.click());
  expect(where.location?.pathname).toBe("/ws/projects/A/views");
});

it("says why nerve could not read the view, under the crumbs, and reads it again on retry", async () => {
  await opened();
  await page.answerRead(first, problem(503, "server_busy"));
  expect(document.querySelector('[role="alert"]')).not.toBeNull();
  expect(text()).not.toContain("project_views.not_found.title");
  expect(crumbs()).toBe(true);
  await act(async () => buttonIn("body", "common.retry")?.click());
  await page.answerRead(first);
  expect(page.reads(first)).toHaveLength(2);
  expect(header()).toContain("common.display");
});

it("is the views' turned-off state while the project's views are off, under the crumbs, and reads nothing", async () => {
  await opened({ project: { issue_views_view: false } });
  await settle();
  expect(text()).toContain("disabled_project.empty_state.view.title");
  expect(crumbs()).toBe(true);
  expect(page.nerve.calls).toEqual([]);
});

it("shows nothing of a view that the page's read found deleted here meanwhile: no failure, no retry, no toast", async () => {
  await opened({ role: 20 });
  // the project's views list the view, which the caller deletes while the page's read is out
  const listed = page.app.views.fetchProjectViews("A");
  await until(() => page.viewLists().length > 0, "the project's views");
  await act(async () =>
    page
      .viewLists()
      .at(-1)!
      .answer(json(200, { data: [first] }))
  );
  await listed;
  const deleted = page.app.views.remove(first.id);
  await until(() => page.writes().length === 1, "the deletion");
  await act(async () => page.writes()[0]!.answer(noContent()));
  await deleted;
  await page.answerRead(first);
  await settle();
  expect(document.querySelector('[role="alert"]')).toBeNull();
  expect(text()).not.toContain("common.retry");
  expect(text()).not.toContain("project_views.not_found.title");
  expect(crumbs()).toBe(true);
  expect(toasts).toEqual([]);
  expect(consoleErrors()).toEqual([]);
  expect(page.lists()).toEqual([]);
});

it("says that nerve finds no such view here when nerve gives a view of another project, under the crumbs", async () => {
  await opened();
  // the caller sees project B too
  await loadProjects(page.nerve, page.app.projectRoot.project, page.workspace, [
    page.project,
    projectOf("B", page.workspace.id, { id: "B" }),
  ]);
  await page.answerRead(first, json(200, { ...first, project_id: "B" }));
  expect(page.app.views.getViewById(first.id)?.project_id, "the store gives B's view").toBe("B");
  expect(text()).toContain("project_views.not_found.title");
  expect(crumbs()).toBe(true);
  expect(header()).not.toContain("common.display");
  expect(page.lists()).toEqual([]);
});

it.each([
  ["an admin", 20, true],
  ["a member", 15, true],
  ["a guest", 5, false],
] as const)("offers %s (role %i) a new work item of the page: %s", async (_who, role, offered) => {
  await opened({ role });
  await page.answerRead(first);
  await page.answerList();
  expect(header()).toContain("common.display");
  expect(buttonIn("header", "issue.add.label") !== undefined).toBe(offered);
});

it("goes to the project's views once nerve deletes the view from the header's quick actions", async () => {
  await opened({ role: 20 });
  await page.answerRead(first);
  await page.answerList();
  const menu = document.querySelector<HTMLButtonElement>('header button[aria-haspopup="menu"]');
  await act(async () => menu?.click());
  await settle();
  const item = [...document.querySelectorAll<HTMLButtonElement>('[role="menuitem"]')].find(
    (node) => node.querySelector("h5")?.textContent === "delete"
  );
  await act(async () => item?.click());
  await settle();
  await act(async () => dialogButton("delete")?.click());
  await until(() => page.writes().length === 1, "the deletion");
  expect(page.writes()[0]).toMatchObject({ method: "DELETE", path: `/api/v0/views/${first.id}` });
  await act(async () => page.writes()[0]!.answer(noContent()));
  await settle();
  expect(where.location?.pathname).toBe("/ws/projects/A/views");
  expect(text()).toContain("the list");
});
