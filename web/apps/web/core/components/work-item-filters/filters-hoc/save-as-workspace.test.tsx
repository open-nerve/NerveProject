// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// Save view on a workspace's pages (M7/P6b spec 2.7, 2.9, R8, R10): each of the four default views' pages and a view of
// the workspace's own's page offers it, once the page has a condition, to the workspace's admins and members, never
// its guest; its dialog starts with no name; the new view, of the workspace's own, keeps the page's conditions, a
// default view's own term besides them, and the display the page holds, in nerve's words, as the spreadsheet whatever
// layout a view held; once nerve has made it, the page goes to it. The profile's work items, under the same filters,
// offer neither Save view nor Update view. The pages, the stores, the dialog, SWR and HTTP are real; the work item's
// preview is left out, the app's header a frame.
import { act, createElement, type ReactNode } from "react";
import { createMemoryRouter, RouterProvider } from "react-router";
import { SWRConfig } from "swr";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { ViewFilters } from "@nerve/api-client";
import { COLLECTION_OPERATOR, EIssuesStoreType, LOGICAL_OPERATOR } from "@nerve/types";
import { settle, where } from "@/components/common/fake-quick-actions";
import { ProfileIssuesPage } from "@/components/profile/profile-issues";
import { field, fill, submit } from "@/components/views/fake-view-dialog";
import { json } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { toasts } from "@/lib/fake-toast";
import { issuePageOf } from "@/store/issue/fake-issues";
import { workspacePageId } from "@/store/issue/workspace/page-id";
import type { RootStore } from "@/store/root.store";
import { ME, workspaceViewOf } from "@/store/views/fake-views";
import { buttonOf, updateButton } from "./fake-view-saving";
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
beforeEach(() => {
  window.localStorage.clear();
  toasts.length = 0;
});
afterEach(async () => page.close());

/** Saves the page as "Mine" from its filters row: the dialog starts with no name; gives what was sent. */
async function saveAs() {
  await settle();
  const save = buttonOf("common.filter_options.save_view");
  expect(save, "Save view").toBeDefined();
  await act(async () => save!.click());
  await settle();
  expect(field("name").value, "a new view's name starts empty").toBe("");
  await fill("name", "Mine");
  await submit();
  await until(() => page.creations().length === 1, "the new view");
  return page.creations()[0]!;
}

it.each<[string, string, ViewFilters]>([
  ["All work items", "all-issues", {}],
  ["Assigned to me", "assigned", { assignee_id: ["me"] }],
  ["Created by me", "created", { created_by_id: ["me"] }],
  ["Subscribed", "subscribed", { subscriber_id: ["me"] }],
])(
  "saves %s's page as a view of the workspace's own: its conditions and its own term, as the spreadsheet",
  async (_, key, own) => {
    page = await workspaceSavingTab();
    bound.app = page.app;
    await page.open(key);
    expect(buttonOf("common.filter_options.save_view"), "no condition yet").toBeUndefined();
    await page.addUrgent(key);
    const held = page.app.issue.globalIssuesFilter.confirmed(workspacePageId("ws", key))!;
    const creation = await saveAs();
    expect(creation.body).toEqual({
      name: "Mine",
      description: "",
      logo_props: {},
      filters: { priority: ["urgent"], ...own },
      display_filters: { ...held.display_filters, layout: "spreadsheet" },
      display_properties: held.display_properties,
    });
    const made = workspaceViewOf(page.workspace, "Mine", ME, {
      id: "v-mine",
      filters: { priority: ["urgent"], ...own },
    });
    await act(async () => creation.answer(json(201, made)));
    await settle();
    expect(toasts).toContainEqual({
      type: "success",
      title: "toast.success",
      message: "workspace_views.action.create.success",
    });
    expect(where.location?.pathname, "the new view's page").toBe("/ws/workspace-views/v-mine");
  }
);

it("saves a view's page as a new view of its draft, as the spreadsheet though the view is held as a list", async () => {
  page = await workspaceSavingTab();
  bound.app = page.app;
  expect(page.view.display_filters.layout, "First, as nerve holds it").toBe("list");
  await page.open();
  const creation = await saveAs();
  expect(creation.body).toEqual({
    name: "Mine",
    description: "",
    logo_props: {},
    filters: { priority: ["high"] },
    display_filters: { ...page.view.display_filters, layout: "spreadsheet" },
    display_properties: page.view.display_properties,
  });
});

it("offers the workspace's guest no Save view, whatever the page's conditions", async () => {
  page = await workspaceSavingTab({ role: 5 });
  bound.app = page.app;
  await page.open("all-issues");
  await page.addUrgent("all-issues");
  expect(page.filter("all-issues").hasActiveFilters, "a condition").toBe(true);
  expect(buttonOf("common.clear_all"), "the filters row shown").toBeDefined();
  expect(buttonOf("common.filter_options.save_view")).toBeUndefined();
  expect(page.filter("all-issues").canSaveView).toBe(false);
});

it("offers neither Save view nor Update view on the profile's work items, its admin's under a condition", async () => {
  page = await workspaceSavingTab({ role: 20 });
  bound.app = page.app;
  page.app.router.setQuery({ workspaceSlug: "ws", userId: "u-me", profileViewId: "assigned" });
  const router = createMemoryRouter(
    [{ path: "/:workspaceSlug/profile/:userId/:profileViewId", element: <ProfileIssuesPage type="assigned" /> }],
    { initialEntries: ["/ws/profile/u-me/assigned"] }
  );
  const list = () => page.nerve.to("/api/v0/workspaces/ws/members/u-me/issues");
  await page.render(
    <SWRConfig value={{ provider: () => new Map() }}>
      <RouterProvider router={router} />
    </SWRConfig>
  );
  await until(() => list().length > 0, "the profile's list");
  await act(async () =>
    list()
      .at(-1)!
      .answer(json(200, issuePageOf([])))
  );
  const filter = () => page.app.workItemFilters.getFilter(EIssuesStoreType.PROFILE, "u-me");
  await until(() => filter() !== undefined, "the profile's filter");
  // as a person picks it from the header's Add filter, which shows the filters row too
  await act(async () => {
    filter()!.toggleVisibility(true);
    filter()!.addCondition(LOGICAL_OPERATOR.AND, {
      property: "priority",
      operator: COLLECTION_OPERATOR.IN,
      value: ["urgent"],
    });
  });
  await settle();
  expect(filter()!.hasActiveFilters, "a condition").toBe(true);
  expect(buttonOf("common.clear_all"), "the filters row shown").toBeDefined();
  expect(buttonOf("common.filter_options.save_view")).toBeUndefined();
  expect(updateButton()).toBeUndefined();
  expect([filter()!.canSaveView, filter()!.canUpdateView]).toEqual([false, false]);
  router.dispose();
});
