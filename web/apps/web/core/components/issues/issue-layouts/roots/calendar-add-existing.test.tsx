// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// A calendar day's "add existing" (P7b spec 2.10, 2.11, R6, R11): the day asks the layout's command host for the dialog,
// which searches the route's project for work items of no day, hiding those with a day or starting after it. On a
// cycle's or a module's page each chosen one gets the day and joins the page in one change, which nerve makes whole or
// refuses whole (a refused one keeps no day): the search takes nerve's most at once (100) and hides those already in
// the page's cycle or module, no more than two are out at once, and the batch reads the page's progress again once. On
// the project's page the search and each change are as they were (M4 I4): the day alone. The host holds the dialog
// while the list is checked, the calendar's days gone. The pages, days, menus, dialog, stores and HTTP are real.
import { act, createElement, type ReactNode } from "react";
import { MemoryRouter } from "react-router";
import { SWRConfig } from "swr";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { Issue, IssueDetail } from "@nerve/api-client";
import { EIssuesStoreType } from "@nerve/types";
import { json, problem } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { toasts } from "@/lib/fake-toast";
import { selectionButton } from "@/components/core/modals/fake-issue-selection";
import { issueDetailOf, issueOf } from "@/store/issue/fake-issues";
import { cycleRequests } from "@/store/planning/fake-cycles";
import { moduleRequests } from "@/store/planning/fake-modules";
import { preferencesOf } from "@/store/project/fake-projects";
import type { RootStore } from "@/store/root.store";
import { C1, M1, cyclePreferencesOf, layouts, modulePreferencesOf, planningPageTab } from "./fake-planning-page";
import { ProjectLayoutRoot } from "./project-layout-root";

const bound = vi.hoisted(() => ({ app: undefined as RootStore | undefined, params: {} as Record<string, string> }));
vi.mock("@/lib/store-context", async () => ({
  get rootStore() {
    return bound.app;
  },
  StoreContext: (await import("react")).createContext<RootStore | undefined>(undefined),
}));
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "one" } }, publicClient: {} }));
vi.mock("@/lib/auth/use-session", () => ({ useSession: () => ({ status: "signed-in", loginId: "one" }) }));
vi.mock("react-router", async (original) => ({
  ...(await original<typeof import("react-router")>()),
  useParams: () => bound.params,
}));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
// jsdom has no viewport geometry; days, cards, menus and HTTP are real.
vi.mock("@atlaskit/pragmatic-drag-and-drop-auto-scroll/element", () => ({ autoScrollForElements: () => () => {} }));
vi.mock("@/components/core/render-if-visible-HOC", () => ({
  default: ({ children, as }: { children: ReactNode; as?: string }) =>
    as ? createElement(as, null, children) : children,
}));

/** The day added to: in the page's week (Dec 27–Jan 2), with no work item. */
const DAY = "2027-01-01";
const C2 = "019a0000-0000-7000-8000-0000000000c2";
const BETA = "019a0000-0000-7000-8000-0000000000d2";
const GAMMA = "019a0000-0000-7000-8000-0000000000d3";
const LATER = "2027-01-01T09:00:00Z";
let page: Awaited<ReturnType<typeof planningPageTab>>;
beforeEach(() => {
  toasts.length = 0;
});
afterEach(async () => page.close());

/** A work item of the project, numbered, with fields. */
const item = (name: string, sequence: number, fields: Partial<Issue> = {}): Issue => ({
  ...issueOf(name),
  sequence_id: sequence,
  ...fields,
});
/** The calendar's day: its tile, which its button for small screens (named by the day) is in. */
const tileOf = (day: string) => document.querySelector(`button[aria-label="${day}"]`)?.parentElement ?? null;
/** The cards the day shows. */
const cardsOf = (day: string) => [...(tileOf(day)?.querySelectorAll('[id^="issue-"]') ?? [])].map((node) => node.id);
const dialog = () => document.querySelector('[role="dialog"]');
/** The work items the dialog lists, by name. */
const listed = () =>
  [...document.querySelectorAll('[role="option"]')].map((node) => node.lastElementChild?.textContent ?? "");
const chosen = (name: string) => document.querySelector(`[aria-label="common.remove ${name}"]`) !== null;
const writes = () => page.nerve.calls.filter((call) => call.method === "PATCH");
/** nerve's confirmation of id's change: the day, and fields. */
const confirmed = (id: string, sequence: number, fields: Partial<Issue> = {}) =>
  json(200, { ...issueDetailOf(id), sequence_id: sequence, target_date: DAY, updated_at: LATER, ...fields });

/** Shows the page of the store type in the calendar, its week, its one work item answered. */
async function calendarOf(type: EIssuesStoreType.CYCLE | EIssuesStoreType.MODULE) {
  bound.params = {
    workspaceSlug: "ws",
    projectId: "A",
    ...(type === EIssuesStoreType.CYCLE ? { cycleId: C1 } : { moduleId: M1 }),
  };
  page = await planningPageTab(type);
  bound.app = page.app;
  const preferences = type === EIssuesStoreType.CYCLE ? cyclePreferencesOf : modulePreferencesOf;
  await page.show(preferences(layouts.calendar.display), page.listOf("calendar"));
  await until(() => cardsOf("2026-12-31").length === 1, "the page's work item on its day");
}
/** Shows the project's page in the calendar, its week, of the same one work item. */
async function projectCalendar() {
  bound.params = { workspaceSlug: "ws", projectId: "A" };
  page = await planningPageTab(EIssuesStoreType.CYCLE);
  bound.app = page.app;
  await page.render(
    <SWRConfig value={{ provider: () => new Map() }}>
      <MemoryRouter>
        <ProjectLayoutRoot />
      </MemoryRouter>
    </SWRConfig>
  );
  const preferences = () => page.nerve.to("/api/v0/me/projects/A/preferences");
  await until(() => preferences().length === 1, "the project's preferences");
  const display = { ...preferencesOf().display_filters, ...layouts.calendar.display };
  await act(async () => preferences()[0]!.answer(json(200, preferencesOf({ display_filters: display }))));
  await until(() => page.lists().length === 1, "the project's list");
  await act(async () => page.lists()[0]!.answer(json(200, page.listOf("calendar"))));
  await until(() => cardsOf("2026-12-31").length === 1, "the project's work item on its day");
}
/** Opens the day's add existing from its menu; nerve answers the dialog's search with the work items found. */
async function searched(day: string, found: Issue[]) {
  const trigger = [...(tileOf(day)?.querySelectorAll<HTMLButtonElement>("button") ?? [])].find(
    (node) => node.textContent === "issue.add.label"
  );
  expect(trigger, `the day ${day}'s menu`).toBeDefined();
  vi.spyOn(trigger!, "getBoundingClientRect").mockReturnValue(new DOMRect(1, 1, 150, 30));
  await act(async () => trigger!.click());
  await act(async () =>
    [...document.querySelectorAll<HTMLButtonElement>('[role="menuitem"]')]
      .find((node) => node.textContent === "issue.add.existing")!
      .click()
  );
  await until(() => page.searches().length === 1, "the dialog's search");
  await act(async () => page.searches()[0]!.answer(json(200, { data: found })));
}
/** Chooses each work item listed and adds them. */
async function addAll() {
  await act(async () => selectionButton("issue.select.select_all").click());
  await act(async () => selectionButton("issue.select.add_selected").click());
}
const answer = (index: number, response: Response) =>
  act(async () => {
    writes()[index]!.answer(response);
    await vi.advanceTimersByTimeAsync(100);
  });

it("gives each chosen one the day and the cycle in one change, two at most at once, the progress read once", async () => {
  await calendarOf(EIssuesStoreType.CYCLE);
  await searched(DAY, [
    item("u", 1),
    item("v", 2, { cycle_id: C2 }),
    item("w", 3),
    item("in", 4, { cycle_id: C1 }),
    item("dated", 5, { target_date: "2027-01-02" }),
    item("late", 6, { start_date: "2027-01-02" }),
  ]);
  expect(page.searches()[0]!.query).toEqual({ project_id: "A", target_date: "none", limit: "100" });
  expect(listed(), "those of no day, not starting after it, not already in the cycle").toEqual(["u", "v", "w"]);
  await addAll();
  await until(() => writes().length >= 2, "the first writes");
  const sent = () => writes().map((call) => [call.path, call.body]);
  expect(sent(), "two out at once").toEqual(
    ["u", "v"].map((id) => [`/api/v0/issues/${id}`, { target_date: DAY, cycle_id: C1 }])
  );
  await answer(0, problem(409, "planning.cycle_completed"));
  expect(sent(), "the third, once one has settled").toEqual(
    ["u", "v", "w"].map((id) => [`/api/v0/issues/${id}`, { target_date: DAY, cycle_id: C1 }])
  );
  const progress = () => page.nerve.to(cycleRequests.progress({ id: C1 }));
  expect(progress(), "the progress, while the batch is out").toEqual([]);
  await answer(1, confirmed("v", 2, { cycle_id: C1 }));
  await answer(2, confirmed("w", 3, { cycle_id: C1 }));
  expect(writes(), "one change of each").toHaveLength(3);
  expect(progress(), "the batch's reads of the progress").toHaveLength(1);
  // the refused one keeps no day, nerve having made none of its change; it stays chosen, the dialog says why
  expect(page.app.issue.issues.getIssueById("u")?.target_date).toBeNull();
  expect(dialog()).not.toBeNull();
  expect([chosen("u"), chosen("v"), chosen("w")]).toEqual([true, false, false]);
  expect(toasts).toEqual([{ type: "error", title: "toast.error", message: "errors.planning_cycle_completed" }]);
  expect(cardsOf(DAY).toSorted(), "the day's cards, of those confirmed").toEqual(["issue-v", "issue-w"]);
});

it("gives a chosen one the day and the module in one change, its other modules kept", async () => {
  await calendarOf(EIssuesStoreType.MODULE);
  await searched(DAY, [item("u", 1, { module_ids: [BETA, GAMMA] }), item("in", 2, { module_ids: [M1] })]);
  expect(page.searches()[0]!.query).toEqual({ project_id: "A", target_date: "none", limit: "100" });
  expect(listed()).toEqual(["u"]);
  await addAll();
  await until(() => writes().length === 1, "the write");
  expect(writes()[0]!.body).toEqual({ target_date: DAY, module_ids: [BETA, GAMMA, M1] });
  await answer(0, confirmed("u", 1, { module_ids: [BETA, GAMMA, M1] }));
  await act(async () => vi.advanceTimersByTimeAsync(300));
  expect(dialog(), "the dialog, all confirmed").toBeNull();
  expect(page.nerve.to(moduleRequests.progress({ id: M1 }))).toHaveLength(1);
  expect(cardsOf(DAY)).toEqual(["issue-u"]);
});

it("searches and sends on the project's page the day alone, two at most at once", async () => {
  await projectCalendar();
  await searched(DAY, [item("u", 1), item("v", 2, { cycle_id: C1 }), item("w", 3)]);
  expect(page.searches()[0]!.query).toEqual({ project_id: "A", target_date: "none" });
  expect(listed()).toEqual(["u", "v", "w"]);
  await addAll();
  await until(() => writes().length >= 2, "the first writes");
  await act(async () => vi.advanceTimersByTimeAsync(100));
  expect(writes(), "two out at once").toHaveLength(2);
  await answer(0, confirmed("u", 1));
  expect(writes().map((call) => [call.path, call.body])).toEqual(
    ["u", "v", "w"].map((id) => [`/api/v0/issues/${id}`, { target_date: DAY }])
  );
  await answer(1, confirmed("v", 2, { cycle_id: C1 }));
  await answer(2, confirmed("w", 3));
  await act(async () => vi.advanceTimersByTimeAsync(300));
  expect(dialog(), "the dialog, all confirmed").toBeNull();
  expect(page.nerve.to(cycleRequests.progress({ id: C1 })), "no page's progress").toEqual([]);
});

it("keeps the dialog a day's add existing opens while the list is checked, the calendar's days gone", async () => {
  await calendarOf(EIssuesStoreType.CYCLE);
  await searched(DAY, [item("u", 1)]);
  // the work item is checked against nerve (as its preview's read does): the calendar shows its loader meanwhile
  void page.app.issue.issues.refreshAuthoritative("first").catch(() => undefined);
  const checks = () => page.nerve.to("/api/v0/issues/first").filter((call) => call.method === "GET");
  await until(() => checks().length === 1, "the check of the work item");
  await act(async () => vi.advanceTimersByTimeAsync(20));
  expect(tileOf(DAY), "the day, while the list is checked").toBeNull();
  expect(listed(), "the dialog, while the list is checked").toEqual(["u"]);
  const [row] = (layouts.spreadsheet.page() as { data: IssueDetail[] }).data;
  await act(async () => checks()[0]!.answer(json(200, { ...issueDetailOf("first"), ...row })));
  await until(() => page.lists().length === 2, "the list's read again");
  await act(async () => page.lists()[1]!.answer(json(200, page.listOf("calendar"))));
  await until(() => tileOf(DAY) !== null, "the day again");
  await addAll();
  await until(() => writes().length === 1, "the write");
  expect(writes()[0]!.body).toEqual({ target_date: DAY, cycle_id: C1 });
});
