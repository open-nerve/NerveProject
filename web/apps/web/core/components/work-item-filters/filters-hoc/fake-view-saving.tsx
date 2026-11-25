// Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only
// A project's view's page against a fake nerve (P6a spec 2.7, R9, R29), for the tests of Update view and Save view on
// it and of its controls while Update view is out: project A with the caller, u-me, of a role (a member's unless the
// test says), its view First (the caller's, of high work items, unless the test says), the page's layout and page
// under the app's routes (its header, its small screen's header, its work items, its actions host), its read of First
// answered and its list. The stores, layouts, SWR and HTTP are real (fake-view-page.tsx); the test file binds the
// stores and the session and frames the app's header (vi.mock, as the view page's own test does).
import { act, type ComponentType } from "react";
import { MemoryRouter, Route, Routes, useParams } from "react-router";
import { SWRConfig } from "swr";
import type { View } from "@nerve/api-client";
import { EIssuesStoreType } from "@nerve/types";
import Layout from "@/app/(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/views/(detail)/layout";
import Page from "@/app/(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/views/(detail)/[viewId]/page";
import { Where, settle, where } from "@/components/common/fake-quick-actions";
import { viewPageTab } from "@/components/views/fake-view-page";
import { json } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { issueOf, issuePageOf } from "@/store/issue/fake-issues";
import { accountOf } from "@/store/member/fake-users";
import { seedAccount } from "@/store/user/fake-account";

type RouteProps = { params: Readonly<Record<string, string | undefined>> };
function RouteOf({ Component }: { Component: unknown }) {
  const Of = Component as ComponentType<RouteProps>;
  return <Of params={useParams()} />;
}

/** What a test says of the page: the caller's role, the project's fields, and First's fields. */
type Opened = Parameters<typeof viewPageTab>[0] & { view?: Partial<View> };

/** A tab of project A at the page of its view First, as opened says; nothing drawn until it opens. */
export async function viewSavingTab(opened: Opened = {}) {
  const { view: fields, ...caller } = opened;
  const tab = await viewPageTab(caller);
  // the caller, u-me, whom a view of the tab's is owned by unless the test says
  await seedAccount(tab.nerve, tab.app.user, accountOf("me"));
  const view = tab.view("First", { filters: { priority: ["high"] }, ...fields });
  const filter = () => tab.app.workItemFilters.getFilter(EIssuesStoreType.PROJECT_VIEW, view.id)!;
  return {
    ...tab,
    view,
    /** The page's work item filter of First. */
    filter,
    /** The page's draft of First, in nerve's words. */
    draft: () => tab.app.issue.projectViewIssuesFilter.confirmed(view.id),
    /** The changes of First sent to nerve. */
    patches: () => tab.nerve.to(`/api/v0/views/${view.id}`).filter((call) => call.method === "PATCH"),
    /** Opens First's page: nerve answers its read as First and its list; the page's filter is then made. */
    open: async () => {
      await tab.render(
        <SWRConfig value={{ provider: () => new Map() }}>
          <MemoryRouter initialEntries={[`/ws/projects/A/views/${view.id}`]}>
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
      await tab.answerRead(view);
      await tab.answerList();
      await settle();
      await until(
        () => tab.app.workItemFilters.getFilter(EIssuesStoreType.PROJECT_VIEW, view.id) !== undefined,
        "the filter"
      );
    },
    /** Takes the page's filter's condition on priority away, as its chip's close does; nerve lists again under it. */
    dropPriority: async () => {
      const lists = tab.lists().length;
      const priority = filter().allConditions.find((condition) => condition.property === "priority")!;
      await act(async () => filter().removeCondition(priority.id));
      await until(() => tab.lists().length === lists + 1, "the list again");
      await act(async () =>
        tab
          .lists()
          .at(-1)!
          .answer(json(200, issuePageOf([issueOf("one")])))
      );
      await settle();
    },
    /** Leaves the page for the project's views, as a link does. */
    leave: () => act(async () => void where.navigate!(`/ws/projects/A/views`)),
  };
}

/** The button of the page whose text is text. */
export const buttonOf = (text: string) =>
  [...document.querySelectorAll<HTMLButtonElement>("button")].find((node) => node.textContent?.trim() === text);
/** The filters row's Update view, as it says itself while it is offered or out. */
export const updateButton = () =>
  buttonOf("common.filter_options.update_view") ?? buttonOf("common.filter_options.confirming");
/** The filters row's Add filter: the add filter button outside the header. */
export const rowAddFilter = () =>
  [...document.querySelectorAll<HTMLButtonElement>("button")].find(
    (node) => node.textContent?.includes("common.filter_options.add") && !node.closest("header")
  );
/** Clicks the filters row's Update view with the focus on it, as a person does with Enter. */
export const clickUpdate = () =>
  act(async () => {
    updateButton()!.focus();
    updateButton()!.click();
  });
