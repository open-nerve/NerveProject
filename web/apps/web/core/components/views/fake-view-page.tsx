// Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only
// A project's view's page against a fake nerve (P6a spec 2.6), for the tests of the page, of its read and of its work
// items: project A with the caller of a role (a member's unless the test says) and its fields, its views as nerve
// gives them, and what the page reads as nerve answers it. The stores, the layouts, SWR and HTTP are real; the test
// file binds the stores and the session (vi.mock) and renders what it tests.
import { act, type ReactNode } from "react";
import { createRoot } from "react-dom/client";
import { vi } from "vitest";
import type { Project, View } from "@nerve/api-client";
import { issueSelectionTab } from "@/components/core/modals/fake-issue-selection";
import { type FakeNerve, json } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { StoreContext } from "@/lib/store-context";
import { issueOf, issuePageOf, issueStoreTab } from "@/store/issue/fake-issues";
import { ME, viewOf } from "@/store/views/fake-views";

/** What a test says of the page: the caller's role and its project's fields. */
type PageFields = { role?: Project["member_role"]; project?: Partial<Project> };

/** Project A's views and what a view's page reads of them, as nerve answers it, over a tab's fake nerve. */
function viewsOf(project: Project, nerve: FakeNerve) {
  const gets = (path: string) => nerve.to(path).filter((call) => call.method === "GET");
  const reads = (view: View) => gets(`/api/v0/views/${view.id}`);
  return {
    /** A view of project A as nerve holds it, named name, the caller's, as changes say otherwise. */
    view: (name: string, changes: Partial<View> = {}) => viewOf(project, name, ME, changes),
    /** The page's reads of the view. */
    reads,
    /** The reads of project A's views. */
    viewLists: () => gets("/api/v0/projects/A/views"),
    /** The reads of project A's work items. */
    lists: () => gets("/api/v0/projects/A/issues"),
    /** The requests that change something. */
    writes: () => nerve.calls.filter((call) => call.method !== "GET"),
    /** nerve answers the view's last read as response says (the view, unless given). */
    answerRead: async (view: View, response: Response = json(200, view)) => {
      await until(() => reads(view).length > 0, `the read of ${view.name}`);
      await act(async () => reads(view).at(-1)!.answer(response));
    },
    /** nerve answers the last read of project A's work items with one of them, "one". */
    answerList: async () => {
      const list = () => gets("/api/v0/projects/A/issues").at(-1);
      await until(() => list() !== undefined, "the project's list");
      await act(async () => list()!.answer(json(200, issuePageOf([issueOf("one")]))));
    },
  };
}

/** A tab at project A, whose stores and layouts the page uses, with the caller of fields' role. */
export async function viewPageTab(fields: PageFields = {}) {
  const tab = await issueSelectionTab();
  vi.stubGlobal(
    "IntersectionObserver",
    class {
      observe() {}
      unobserve() {}
      disconnect() {}
    }
  );
  tab.app.projectRoot.project.confirmProject(tab.project, (project) => ({
    ...project,
    member_role: fields.role ?? 15,
    ...fields.project,
  }));
  await tab.app.state.fetchProjectStates("A");
  tab.nerve.calls.length = 0;
  return { ...tab, ...viewsOf(tab.project, tab.nerve) };
}

/** A tab at project A whose stores a part of the page uses that draws nothing of the layouts (no menu, no list). */
export async function viewReadTab() {
  globalThis.IS_REACT_ACT_ENVIRONMENT = true;
  vi.useFakeTimers();
  const tab = await issueStoreTab();
  const container = document.createElement("div");
  document.body.append(container);
  const renderer = createRoot(container);
  return {
    ...tab,
    ...viewsOf(tab.project, tab.nerve),
    render: (children: ReactNode) =>
      act(async () => renderer.render(<StoreContext.Provider value={tab.app}>{children}</StoreContext.Provider>)),
    close: async () => {
      await act(async () => renderer.unmount());
      container.remove();
      tab.app.dispose();
      vi.restoreAllMocks();
      vi.useRealTimers();
    },
  };
}
