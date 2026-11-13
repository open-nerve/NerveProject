// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// A cycle page's changes of its preferences are followed as the project page's are (P5b spec 2.4): once nerve confirms
// one that changes the list's query, the route cycle's list is read again with the pages it has; a refusal is said.
// Neither follows once the page is no longer the one that sent it: another cycle (and back), another session, a lost
// and regained permission, or the page gone. The stores and HTTP are real; the route's params are the test's.
import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { EIssueFilterType } from "@nerve/constants";
import { EIssuesStoreType } from "@nerve/types";
import type { IIssueDisplayFilterOptions } from "@nerve/types";
import { json, problem } from "@/lib/auth/fake-nerve";
import { track, until } from "@/lib/auth/fake-time";
import { toasts } from "@/lib/fake-toast";
import { StoreContext } from "@/lib/store-context";
import { issueOf, issuePageOf, issueStoreTab, requestedIssue } from "@/store/issue/fake-issues";
import { loadProjects, preferencesOf } from "@/store/project/fake-projects";
import { useCyclePreferencesFollow } from "./use-cycle-preferences-follow";
import { useIssuesActions, type IssueActions } from "./use-issues-actions";

const C1 = "019a0000-0000-7000-8000-0000000000c1";
const C2 = "019a0000-0000-7000-8000-0000000000c2";
const fixture = vi.hoisted(() => ({
  app: undefined as unknown,
  session: { loginId: "one" },
  params: {} as Record<string, string>,
  actions: undefined as IssueActions | undefined,
  follow: undefined as ReturnType<typeof useCyclePreferencesFollow> | undefined,
}));
vi.mock("@/lib/store-context", async () => {
  const { createContext } = await import("react");
  return {
    StoreContext: createContext(undefined),
    get rootStore() {
      return fixture.app;
    },
  };
});
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: fixture.session }, publicClient: {} }));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("react-router", async (original) => ({
  ...(await original<typeof import("react-router")>()),
  useParams: () => fixture.params,
}));
globalThis.IS_REACT_ACT_ENVIRONMENT = true;
let tab: Awaited<ReturnType<typeof issueStoreTab>>;
let view: ReturnType<typeof createRoot>;
function Probe() {
  fixture.actions = useIssuesActions(EIssuesStoreType.CYCLE);
  fixture.follow = useCyclePreferencesFollow();
  return null;
}
const render = () =>
  act(async () =>
    view.render(
      <StoreContext.Provider value={tab.app as never}>
        <Probe />
      </StoreContext.Provider>
    )
  );
const lists = () => tab.nerve.to("/api/v0/projects/A/issues");
const writes = () => tab.nerve.calls.filter((call) => call.method === "PATCH");
const cyclePreferences = () => {
  const { filters, display_filters, display_properties } = preferencesOf({ filters: { priority: ["high"] } });
  return { filters, display_filters, display_properties };
};
beforeEach(async () => {
  vi.useFakeTimers();
  tab = await issueStoreTab();
  fixture.app = tab.app;
  fixture.session.loginId = "one";
  fixture.params = { workspaceSlug: "ws", projectId: "A", cycleId: C1 };
  toasts.length = 0;
  const read = await requestedIssue(tab.nerve, () => tab.app.issue.cycleIssuesFilter.fetchFilters("ws", C1), [
    "GET",
    `/api/v0/me/cycles/${C1}/preferences`,
  ]);
  read.call.answer(json(200, cyclePreferences()));
  await read.promise;
  const page = { projectID: "A", cycleID: C1 };
  const options = { limit: 10, query: { start_date: ["2026-01-01..2026-02-01"] } };
  const first = await requestedIssue(
    tab.nerve,
    () => tab.app.issue.cycleIssues.fetchIssues("ws", page, "init-loader", options),
    ["GET", "/api/v0/projects/A/issues"]
  );
  first.call.answer(json(200, issuePageOf([issueOf("old")])));
  await first.promise;
  tab.nerve.calls.length = 0;
  view = createRoot(document.createElement("div"));
  await render();
});
afterEach(async () => {
  await act(async () => view.unmount());
  tab.app.dispose();
  vi.useRealTimers();
});

/** The page changes its display filters by its actions, as a layout or its header does; nerve holds the change. */
async function change(update: IIssueDisplayFilterOptions) {
  let settled!: Promise<void>;
  await act(async () => {
    settled = fixture.actions!.updateFilters("A", EIssueFilterType.DISPLAY_FILTERS, update);
    await until(() => writes().length === 1, "the change of the page's preferences");
  });
  expect(writes()[0]).toMatchObject({ path: `/api/v0/me/cycles/${C1}/preferences` });
  return { settled, call: writes()[0]! };
}
const answered = (update: IIssueDisplayFilterOptions) => {
  const held = cyclePreferences();
  return json(200, { ...held, display_filters: { ...held.display_filters, ...update } });
};
/**
 * nerve answers the held change; the page has done all it does about it once the fake time has moved on. A follow-up
 * read would keep the change unsettled, so the test looks before it waits for the change to settle.
 */
async function settleWithout(settled: Promise<void>, call: { answer: (response: Response) => void }, answer: Response) {
  const result = track(settled);
  await act(async () => {
    call.answer(answer);
    await vi.advanceTimersByTimeAsync(100);
  });
  expect(lists(), "no read of the list").toEqual([]);
  expect(result.settled).toBe(true);
}

it("reads the route cycle's list again, with the pages it has, once nerve confirms a change of its query", async () => {
  const { settled, call } = await change({ sub_issue: false });
  expect(lists()).toEqual([]);
  await act(async () => {
    call.answer(answered({ sub_issue: false }));
    await until(() => lists().length === 1, "the list again");
  });
  expect(lists()[0]!.query).toMatchObject({
    cycle_id: C1,
    limit: "10",
    start_date: "2026-01-01..2026-02-01",
    priority: "high",
    sub_issue: "false",
  });
  // the change settles once its follow-up has: the page stays busy until the list is read again
  await act(async () => {
    lists()[0]!.answer(json(200, issuePageOf([issueOf("fresh")])));
    await settled;
  });
  expect(tab.app.issue.cycleIssues.getIssueIds()).toEqual(["fresh"]);
  expect(toasts).toEqual([]);
});

it("says why nerve refused a change of the page's preferences, and reads nothing", async () => {
  const { settled, call } = await change({ sub_issue: false });
  await settleWithout(settled, call, problem(403, "forbidden"));
  expect(toasts).toEqual([expect.objectContaining({ type: "error", message: "errors.forbidden" })]);
});

it("reads nothing again for a confirmed change that leaves the list's query as it was", async () => {
  const { settled, call } = await change({ show_empty_groups: false });
  await settleWithout(settled, call, answered({ show_empty_groups: false }));
  expect(toasts).toEqual([]);
});

it.each(["confirms", "refuses"] as const)(
  "follows only the route cycle's preferences: when nerve %s another cycle's, it neither reads nor says",
  async (outcome) => {
    let settled!: Promise<void>;
    const other = () => tab.nerve.to(`/api/v0/me/cycles/${C2}/preferences`);
    await act(async () => {
      settled = fixture.follow!("ws", C2, () =>
        tab.app.issue.cycleIssuesFilter.updateFilterExpression("ws", C2, { and: [{ priority__in: "low" }] })
      );
      await until(() => other().length === 1, "the change of another cycle's preferences");
    });
    const answer = outcome === "confirms" ? json(200, cyclePreferences()) : problem(403, "forbidden");
    await settleWithout(settled, other()[0]!, answer);
    expect(toasts).toEqual([]);
  }
);

for (const ending of ["another cycle and back", "another session", "a permission lost and regained", "unmount"]) {
  it.each(["confirms", "refuses"] as const)(
    `neither reads nor says anything when nerve %s after ${ending}`,
    async (outcome) => {
      const { settled, call } = await change({ sub_issue: false });
      if (ending === "another cycle and back") {
        fixture.params = { workspaceSlug: "ws", projectId: "A", cycleId: C2 };
        await render();
        fixture.params = { workspaceSlug: "ws", projectId: "A", cycleId: C1 };
        await render();
      } else if (ending === "another session") fixture.session.loginId = "two";
      else if (ending === "a permission lost and regained") {
        await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, []);
        await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [tab.project]);
      } else await act(async () => view.unmount());
      await settleWithout(
        settled,
        call,
        outcome === "confirms" ? answered({ sub_issue: false }) : problem(403, "forbidden")
      );
      expect(toasts).toEqual([]);
      if (ending === "unmount") view = createRoot(document.createElement("div"));
    }
  );
}
