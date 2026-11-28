// Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only
// A workspace's view's page against a fake nerve (M7/P6b spec 2.9, R10, R29), for the tests of Save as and Update view
// on it and of its controls while Update view is out: workspace ws with the caller u-me of a role there (a member's
// unless the test says), its projects A (his as a member) and B (as a guest), its own view First (the caller's, of
// high work items, unless the test says), and the page of First or of a default view at the app's routes: its layout,
// keyed by the view, with its header and its actions host, and its page. The stores, the layouts, SWR and HTTP are
// real (fake-view-page.tsx); the test file binds the stores and the session and frames the app's header (vi.mock, as
// the view page's own tests do).
import { act } from "react";
import type { View, WorkspaceRole } from "@nerve/api-client";
import { COLLECTION_OPERATOR, EIssuesStoreType, LOGICAL_OPERATOR } from "@nerve/types";
import Layout from "@/app/(all)/[workspaceSlug]/(projects)/workspace-views/(detail)/layout";
import Page from "@/app/(all)/[workspaceSlug]/(projects)/workspace-views/(detail)/[globalViewId]/page";
import { settle, where } from "@/components/common/fake-quick-actions";
import { WorkspaceViewRoutes, workspaceViewPageTab } from "@/components/workspace/views/fake-view-page";
import { until } from "@/lib/auth/fake-time";
import { issueOf } from "@/store/issue/fake-issues";
import { workspacePageId } from "@/store/issue/workspace/page-id";

/** What a test says of the page: the caller's role in ws, and First's fields. */
type Opened = { role?: WorkspaceRole; view?: Partial<View> };

/** A tab of ws at no page yet, as opened says; open draws the page of a view. */
export async function workspaceSavingTab(opened: Opened = {}) {
  const tab = await workspaceViewPageTab(opened.role);
  const view = tab.view("First", { filters: { priority: ["high"] }, ...opened.view });
  /** The page's work item filter of the view of globalViewId (First's unless given): a default view's key, or an id. */
  const filter = (globalViewId = view.id) =>
    tab.app.workItemFilters.getFilter(EIssuesStoreType.GLOBAL, workspacePageId("ws", globalViewId))!;
  /** nerve lists ws's work items again, after a change of the page's filters: "one" again. */
  const listedAgain = async (lists: number) => {
    await until(() => tab.lists().length === lists + 1, "the list again");
    await tab.answerList([issueOf("one")]);
    await settle();
  };
  return {
    ...tab,
    view,
    filter,
    /** The page's draft of First, in nerve's words. */
    draft: () => tab.app.issue.globalIssuesFilter.confirmed(view.id),
    /** The changes of First sent to nerve. */
    patches: () => tab.nerve.to(`/api/v0/views/${view.id}`).filter((call) => call.method === "PATCH"),
    /** The new views of ws's own sent to nerve. */
    creations: () => tab.nerve.to("/api/v0/workspaces/ws/views").filter((call) => call.method === "POST"),
    /**
     * Opens the page of globalViewId (First's unless given): nerve answers the read of First, for its page, and the
     * page's list, of the work item "one" unless empty; the page's filter is then made.
     */
    open: async (globalViewId: string = view.id, empty = false) => {
      await tab.render(
        <WorkspaceViewRoutes
          path={`/ws/workspace-views/${globalViewId}`}
          Layout={Layout}
          Page={Page}
          cache={new Map()}
        />
      );
      if (globalViewId === view.id) await tab.answerRead(view);
      await tab.answerList(empty ? [] : [issueOf("one")]);
      await settle();
      await until(
        () =>
          tab.app.workItemFilters.getFilter(EIssuesStoreType.GLOBAL, workspacePageId("ws", globalViewId)) !== undefined,
        "the filter"
      );
    },
    /** Takes First's page's condition on priority away, as its chip's close does; nerve lists again under it. */
    dropPriority: async () => {
      const lists = tab.lists().length;
      const priority = filter().allConditions.find((condition) => condition.property === "priority")!;
      await act(async () => filter().removeCondition(priority.id));
      await listedAgain(lists);
    },
    /**
     * Adds the condition of urgent work items to the page of globalViewId, as a person picks it from the header's Add
     * filter, which shows the filters row too; nerve lists again.
     */
    addUrgent: async (globalViewId: string = view.id) => {
      const lists = tab.lists().length;
      await act(async () => {
        filter(globalViewId).toggleVisibility(true);
        filter(globalViewId).addCondition(LOGICAL_OPERATOR.AND, {
          property: "priority",
          operator: COLLECTION_OPERATOR.IN,
          value: ["urgent"],
        });
      });
      await listedAgain(lists);
    },
    /** Leaves the page for the workspace's views, as a link does. */
    leave: () => act(async () => void where.navigate!("/ws/workspace-views")),
  };
}
