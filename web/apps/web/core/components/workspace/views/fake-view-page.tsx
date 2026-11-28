// Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only
// A workspace's view's page against a fake nerve (M7/P6b spec 2.8), for the tests of the page, its header and its work
// items: workspace ws, the caller u-me of a role in it (a member's unless the test says), its project A (his as a
// member) and B (his as a guest), its own views as nerve gives them, and what the page reads as nerve answers it. The
// stores, the layouts, SWR and HTTP are real; the test file binds the stores and the session (vi.mock) and renders the
// page's layout and page at the app's routes (pages).
import { act, type ComponentType } from "react";
import { MemoryRouter, Route, Routes, useParams } from "react-router";
import { SWRConfig, type Cache } from "swr";
import { vi } from "vitest";
import type { Issue, View, WorkspaceRole } from "@nerve/api-client";
import { Where } from "@/components/common/fake-quick-actions";
import { issueSelectionTab } from "@/components/core/modals/fake-issue-selection";
import { json } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { issueOf, issuePageOf } from "@/store/issue/fake-issues";
import { accountOf } from "@/store/member/fake-users";
import { loadProjects, projectOf } from "@/store/project/fake-projects";
import { seedAccount } from "@/store/user/fake-account";
import { ME, workspaceViewOf } from "@/store/views/fake-views";
import { loadWorkspaces, workspaceOf } from "@/store/workspace/fake-workspaces";

/** A tab at workspace ws whose caller has role there (a member's unless given), with its projects A and B. */
export async function workspaceViewPageTab(role: WorkspaceRole = 15) {
  const tab = await issueSelectionTab();
  vi.stubGlobal(
    "IntersectionObserver",
    class {
      observe() {}
      unobserve() {}
      disconnect() {}
    }
  );
  const workspace = workspaceOf("ws", { role });
  await loadWorkspaces(tab.nerve, tab.app.workspaceRoot, [workspace]);
  const b = projectOf("B", workspace.id, { id: "B", member_role: 5 });
  await loadProjects(tab.nerve, tab.app.projectRoot.project, workspace, [tab.project, b]);
  await seedAccount(tab.nerve, tab.app.user, accountOf("me"));
  tab.nerve.calls.length = 0;
  const gets = (path: string) => tab.nerve.to(path).filter((call) => call.method === "GET");
  const reads = (view: View) => gets(`/api/v0/views/${view.id}`);
  const lists = () => gets("/api/v0/workspaces/ws/issues");
  return {
    ...tab,
    workspace,
    /** A view of ws as nerve holds it, named name, the caller's, as changes say otherwise. */
    view: (name: string, changes: Partial<View> = {}) => workspaceViewOf(workspace, name, ME, changes),
    /** The page's reads of the view. */
    reads,
    /** The reads of ws's work items, the page's list. */
    lists,
    /** The reads of ws's own views. */
    viewLists: () => gets("/api/v0/workspaces/ws/views"),
    /** The reads of ws's label catalog. */
    catalogReads: () => gets("/api/v0/workspaces/ws/labels"),
    /** The requests that change something. */
    writes: () => tab.nerve.calls.filter((call) => call.method !== "GET"),
    /** nerve answers the view's last read as response says (the view, unless given). */
    answerRead: async (view: View, response: Response = json(200, view)) => {
      await until(() => reads(view).length > 0, `the read of ${view.name}`);
      await act(async () => reads(view).at(-1)!.answer(response));
    },
    /** nerve answers the last read of ws's work items with rows (one of project A, "one", unless given). */
    answerList: async (rows: Issue[] = [issueOf("one")]) => {
      await until(() => lists().length > 0, "the page's list");
      await act(async () =>
        lists()
          .at(-1)!
          .answer(json(200, issuePageOf(rows)))
      );
    },
  };
}

type RouteProps = { params: Readonly<Record<string, string | undefined>> };
function RouteOf({ Component }: { Component: unknown }) {
  const Of = Component as ComponentType<RouteProps>;
  return <Of params={useParams()} />;
}

/**
 * The tab's pages at path, at the app's routes (app/routes/core.ts): a workspace's view's page, its Layout and its
 * Page by the route's workspaceSlug and globalViewId, and the views list, a marker; SWR's cache is the tab's, one for
 * all its pages, as the app's.
 */
export function WorkspaceViewRoutes(props: { path: string; Layout: unknown; Page: unknown; cache: Cache }) {
  const { path, Layout, Page, cache } = props;
  return (
    <SWRConfig value={{ provider: () => cache }}>
      <MemoryRouter initialEntries={[path]}>
        <Routes>
          <Route path=":workspaceSlug/workspace-views/:globalViewId" element={<RouteOf Component={Layout} />}>
            <Route index element={<RouteOf Component={Page} />} />
          </Route>
          <Route path=":workspaceSlug/workspace-views" element={<p>the list</p>} />
        </Routes>
        <Where />
      </MemoryRouter>
    </SWRConfig>
  );
}

/** What the page's body shows now: its read's wait, nerve's having no such view, why it could not read it, the view. */
export function shownNow(): string {
  const text = document.body.textContent ?? "";
  if (text.includes("workspace_views.not_found.title")) return "not found";
  if (text.includes("common.retry")) return "failed";
  if (document.querySelector("[data-peek]")) return "the view";
  if (document.querySelector('[role="status"].p-5')) return "reading";
  return "blank";
}
