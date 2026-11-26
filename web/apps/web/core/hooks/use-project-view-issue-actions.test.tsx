// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// A project's view's route lists the project's work items under the page's draft of the view (P6a spec 2.5, R1; re-96
// I3): the route is the view page's, whose list and draft are its own, never the project page's; its commands are all
// of a project list page's, which write as the project page's do, into the view page's list; a change of its display
// is the route view's draft's, whatever project its caller names (the spreadsheet's and the calendar's name the
// project), followed by a read of its list when its query changed. The session's stores, hooks and HTTP are real.
import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { View } from "@nerve/api-client";
import { EIssueFilterType } from "@nerve/constants";
import { EIssuesStoreType, type IIssueDisplayFilterOptions } from "@nerve/types";
import { StoreContext } from "@/lib/store-context";
import { json, noContent } from "@/lib/auth/fake-nerve";
import { issueDetailOf, issueOf, issuePageOf, issueStoreTab, requestedIssue } from "@/store/issue/fake-issues";
import type { RootStore } from "@/store/root.store";
import { viewOf } from "@/store/views/fake-views";
import { useIssues } from "./store/use-issues";
import { useIssueStoreType } from "./use-issue-layout-store";
import { useIssuesActions, type IssueActions } from "./use-issues-actions";

const fixture = vi.hoisted(() => ({ params: {} as Record<string, string>, app: undefined as RootStore | undefined }));
vi.mock("react-router", () => ({ useParams: () => fixture.params }));
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "actions" } }, publicClient: {} }));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@/lib/store-context", async () => ({
  StoreContext: (await import("react")).createContext<RootStore | undefined>(undefined),
  get rootStore() {
    return fixture.app;
  },
}));

globalThis.IS_REACT_ACT_ENVIRONMENT = true;
let tab: Awaited<ReturnType<typeof issueStoreTab>>;
let view: View;
beforeEach(async () => {
  vi.stubGlobal("localStorage", { getItem: () => null, setItem: () => {}, removeItem: () => {} });
  vi.useFakeTimers();
  tab = await issueStoreTab();
  fixture.app = tab.app;
  view = viewOf(tab.project, "Urgent", "u-me", { filters: { priority: ["urgent"] } });
  fixture.params = { workspaceSlug: "ws", projectId: "A", viewId: view.id };
});
afterEach(() => {
  tab.app.dispose();
  vi.useRealTimers();
  vi.unstubAllGlobals();
});

const LIST: [string, string] = ["GET", "/api/v0/projects/A/issues"];
const CREATE: [string, string] = ["POST", "/api/v0/projects/A/issues"];
/** The page's commands, as its route gives them, rendered in the session's stores; run uses them. */
async function commands(run: (actions: IssueActions) => Promise<void>) {
  const root = createRoot(document.createElement("div"));
  let actions!: IssueActions;
  const Probe = () => {
    actions = useIssuesActions(useIssueStoreType());
    return null;
  };
  await act(async () =>
    root.render(
      <StoreContext.Provider value={tab.app}>
        <Probe />
      </StoreContext.Provider>
    )
  );
  try {
    await run(actions);
  } finally {
    await act(async () => root.unmount());
  }
}
/** The page reads its draft of the view, which nerve answers with the view. */
async function seeded() {
  const read = await requestedIssue(
    tab.nerve,
    () => tab.app.issue.projectViewIssuesFilter.fetchFilters("ws", view.id),
    ["GET", `/api/v0/views/${view.id}`]
  );
  read.call.answer(json(200, view));
  await read.promise;
}
/** The page's list: nerve answers its read with rows. */
async function listed(actions: IssueActions, rows = [issueOf("x")], next: string | null = null) {
  const read = await requestedIssue(tab.nerve, () => actions.fetchIssues("init-loader", { limit: 20 }), LIST);
  read.call.answer(json(200, issuePageOf(rows, next)));
  await read.promise;
  return read.call;
}

it("takes a view's route for the view page's list and draft, never the project page's, with all its commands", async () => {
  const root = createRoot(document.createElement("div"));
  let selected: EIssuesStoreType | undefined;
  let store: ReturnType<typeof useIssues<EIssuesStoreType.PROJECT_VIEW>> | undefined;
  let actions: IssueActions | undefined;
  const Probe = () => {
    selected = useIssueStoreType();
    store = useIssues(EIssuesStoreType.PROJECT_VIEW);
    actions = useIssuesActions(selected);
    return null;
  };
  await act(async () =>
    root.render(
      <StoreContext.Provider value={tab.app}>
        <Probe />
      </StoreContext.Provider>
    )
  );
  await act(async () => root.unmount());
  expect(selected).toBe(EIssuesStoreType.PROJECT_VIEW);
  expect(store?.issues).toBe(tab.app.issue.projectViewIssues);
  expect(store?.issuesFilter).toBe(tab.app.issue.projectViewIssuesFilter);
  const all = [
    "addIssuesToView",
    "archiveIssue",
    "createIssue",
    "createIssueAsGiven",
    "dropIssue",
    "fetchIssues",
    "fetchNextIssues",
    "removeIssue",
    "updateFilters",
    "updateIssue",
  ];
  expect(Object.keys(actions ?? {}).toSorted()).toEqual(all);
  expect(all.filter((name) => typeof actions?.[name as keyof IssueActions] !== "function")).toEqual([]);
});

it("reads the route project's list under the route view's draft, and its next page, as the view page's alone", async () => {
  await seeded();
  await commands(async (actions) => {
    const first = await listed(actions, [issueOf("x")], "after-x");
    expect(first.query).toMatchObject({ priority: "urgent" });
    const next = await requestedIssue(tab.nerve, () => actions.fetchNextIssues(), LIST);
    expect(next.call.query).toMatchObject({ priority: "urgent", cursor: "after-x" });
    // the newest first: the next page's is older
    next.call.answer(json(200, issuePageOf([{ ...issueOf("y"), created_at: "2026-10-27T00:00:00Z" }])));
    await next.promise;
  });
  expect(tab.app.issue.projectViewIssues.getIssueIds()).toEqual(["x", "y"]);
  expect(tab.app.issue.projectIssues.currentList, "the project page's list").toBeUndefined();
});

it("creates, changes and deletes as the project page does, into the view page's list", async () => {
  await seeded();
  await commands(async (actions) => {
    await listed(actions);
    // a creation, and a form's as it gives it: the project page's both
    const created = await requestedIssue(tab.nerve, () => actions.createIssue!("A", { name: "New" }), CREATE);
    expect(created.call.body).toEqual({ name: "New" });
    created.call.answer(json(201, { ...issueDetailOf("new"), name: "New" }));
    await created.promise;
    const given = await requestedIssue(tab.nerve, () => actions.createIssueAsGiven!("A", { name: "Given" }), CREATE);
    expect(given.call.body).toEqual({ name: "Given" });
    given.call.answer(json(201, { ...issueDetailOf("given"), name: "Given" }));
    await given.promise;
    const changed = await requestedIssue(tab.nerve, () => actions.updateIssue!("A", "x", { priority: "high" }), [
      "PATCH",
      "/api/v0/issues/x",
    ]);
    expect(changed.call.body).toEqual({ priority: "high" });
    changed.call.answer(json(200, { ...issueDetailOf("x"), priority: "high", updated_at: "2026-11-01T00:00:00Z" }));
    await changed.promise;
    expect(tab.app.issue.issues.getIssueById("x")?.priority).toBe("high");
    const removed = await requestedIssue(tab.nerve, () => actions.removeIssue("A", "x"), [
      "DELETE",
      "/api/v0/issues/x",
    ]);
    removed.call.answer(noContent());
    await removed.promise;
  });
  expect(tab.app.issue.projectViewIssues.getIssueIds(), "the view page's list").not.toContain("x");
});

const displays: [string, IIssueDisplayFilterOptions, object | undefined][] = [
  ["the spreadsheet's column sort", { order_by: "priority" }, { order_by: "priority" }],
  ["the calendar's layout", { calendar: { layout: "week", show_weekends: true } }, undefined],
];
it.each(displays)(
  "takes %s, its caller naming the project, into the route view's draft, sent nowhere",
  async (_, change, relisted) => {
    await seeded();
    await commands(async (actions) => {
      await listed(actions);
      const sent = tab.nerve.calls.length;
      const updated = actions.updateFilters("A", EIssueFilterType.DISPLAY_FILTERS, change);
      if (relisted) {
        // its list's query changed: the list is read again, under the draft
        const again = await requestedIssue(tab.nerve, () => updated, LIST);
        expect(again.call.query).toMatchObject({ ...relisted, priority: "urgent" });
        again.call.answer(json(200, issuePageOf([issueOf("x")])));
        await again.promise;
      } else await updated;
      expect(tab.nerve.calls.slice(sent).filter((call) => call.method !== "GET")).toEqual([]);
    });
    expect(tab.app.issue.projectViewIssuesFilter.filters[view.id]?.displayFilters).toMatchObject(change);
    expect(tab.app.issue.projectViewIssuesFilter.filters.A, "under the project's id").toBeUndefined();
    expect(tab.app.issue.projectIssuesFilter.filters.A, "the project page's").toBeUndefined();
  }
);
