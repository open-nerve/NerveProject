// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// A row's quick actions on a cycle's page (P5b spec 2.6, R6, R10; re-66 L4, L5): the project page's, and the removal
// of the work item from the cycle, offered in each layout to a member while the cycle takes work items; none once it
// has ended or is archived, nor to a guest. The removal is one change of the work item's cycle to none: once nerve
// confirms it the row leaves the list, the page says so and reads the cycle's progress again; a refusal is said and
// the row stays. Neither follows once the page is another cycle's or another session's. The edit and copy dialogs
// are the cycle page's, whose creation joins the cycle (cycle-create.test.tsx) and whose confirmed writes read the
// progress again (the cycle actions' tests in core/hooks). The layouts, the menus, the stores and HTTP are real; the
// dialog is a stand-in that says what it was opened with.
import { act, createElement, type ReactNode } from "react";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { Cycle } from "@nerve/api-client";
import { EIssuesStoreType } from "@nerve/types";
import { json, problem } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { toasts } from "@/lib/fake-toast";
import { issueDetailOf } from "@/store/issue/fake-issues";
import { cycleRequests } from "@/store/planning/fake-cycles";
import type { RootStore } from "@/store/root.store";
import type { IssuesModalProps } from "../../issue-modal/modal";
import { C1, choose, planningPageTab, cyclePreferencesOf, layouts, type Layout } from "./fake-planning-page";

const bound = vi.hoisted(() => ({
  app: undefined as RootStore | undefined,
  params: {} as Record<string, string>,
  session: { loginId: "one" },
  dialog: undefined as Pick<IssuesModalProps, "storeType" | "sourceIssueId" | "data"> | undefined,
}));
vi.mock("@/lib/store-context", async () => ({
  get rootStore() {
    return bound.app;
  },
  StoreContext: (await import("react")).createContext<RootStore | undefined>(undefined),
}));
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: bound.session }, publicClient: {} }));
vi.mock("@/lib/auth/use-session", () => ({ useSession: () => ({ status: "signed-in", loginId: "one" }) }));
vi.mock("react-router", async (original) => ({
  ...(await original<typeof import("react-router")>()),
  useParams: () => bound.params,
}));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
// jsdom has no viewport geometry; rows, cards, days and HTTP are real.
vi.mock("@atlaskit/pragmatic-drag-and-drop-auto-scroll/element", () => ({ autoScrollForElements: () => () => {} }));
vi.mock("@/components/core/render-if-visible-HOC", () => ({
  default: ({ children, as }: { children: ReactNode; as?: string }) =>
    as ? createElement(as, null, children) : children,
}));
// The row's dialog: a stand-in that keeps what it was opened with.
vi.mock("@/components/issues/issue-modal/modal", () => ({
  CreateUpdateIssueModal: (props: IssuesModalProps) => {
    if (props.isOpen)
      bound.dialog = { storeType: props.storeType, sourceIssueId: props.sourceIssueId, data: props.data };
    return null;
  },
}));

const C2 = "019a0000-0000-7000-8000-0000000000c2";
const LATER = "2026-12-31T13:00:00Z";
let page: Awaited<ReturnType<typeof planningPageTab>>;
beforeEach(() => {
  bound.params = { workspaceSlug: "ws", projectId: "A", cycleId: C1 };
  bound.session.loginId = "one";
  bound.dialog = undefined;
  toasts.length = 0;
});
afterEach(async () => page.close());

/** Shows the page of the cycle as fields say to the caller of its role, its one work item in the layout. */
async function shown(fields: Parameters<typeof planningPageTab>[1] = {}, layout: Layout = "list") {
  page = await planningPageTab(EIssuesStoreType.CYCLE, { role: 15, ...fields });
  bound.app = page.app;
  await page.show(cyclePreferencesOf(layouts[layout].display), layouts[layout].page());
}
/** The row's quick actions, by their names, as its menu offers them. */
const offered = () =>
  [...document.querySelectorAll('[data-context-menu="true"] [role="menuitem"]')].map((node) => node.textContent);
const REMOVE = "issue.remove.cycle.label";
const row = () => document.querySelector('[id^="issue-first"]');
const writesOf = (id: string) => page.nerve.to(`/api/v0/issues/${id}`).filter((call) => call.method === "PATCH");
const progressReads = () => page.nerve.to(cycleRequests.progress({ id: C1 }));
/** The cycle's work item "first" as nerve answers a write of it with fields. */
const first = (fields: object = {}) => ({
  ...issueDetailOf("first"),
  cycle_id: C1,
  state_id: "state",
  priority: "high",
  target_date: "2026-12-31",
  sort_order: 1000,
  updated_at: LATER,
  ...fields,
});
/** Removes the row's work item from the cycle by its quick action; nerve holds the change. */
async function removal() {
  await choose(REMOVE);
  await until(() => writesOf("first").length === 1, "the removal");
  expect(writesOf("first")[0]!.body).toEqual({ cycle_id: null });
  return writesOf("first")[0]!;
}

it.each(Object.keys(layouts) as Layout[])(
  "offers a member the removal of a row from a cycle under way in the %s layout, with the project page's actions",
  async (layout) => {
    await shown({}, layout);
    expect(offered()).toEqual(expect.arrayContaining(["common.actions.edit", "common.actions.make_a_copy", REMOVE]));
  }
);

const ended: Partial<Cycle> = { status: "completed" };
// nerve archives only ended cycles: a status other than completed shows that being archived alone is enough
const archived: Partial<Cycle> = { status: "current", archived_at: "2026-12-01T09:00:00Z" };
it.each([
  { who: "a member", page: "an ended cycle's", fields: { cycle: ended } },
  { who: "a member", page: "an archived cycle's", fields: { cycle: archived } },
  { who: "a guest", page: "a cycle under way's", fields: { role: 5 as const } },
])("offers $who no removal from the cycle on $page page", async ({ fields }) => {
  await shown(fields);
  expect(offered()).toEqual(["common.actions.open_in_new_tab", "common.actions.copy_link"]);
});

it("removes a work item from the cycle in one change; once nerve confirms, the row leaves, the page says so and reads the progress again", async () => {
  await shown();
  const change = await removal();
  await act(async () => vi.advanceTimersByTimeAsync(100));
  expect(toasts, "nothing said before nerve confirms").toEqual([]);
  expect(progressReads()).toEqual([]);
  await act(async () => change.answer(json(200, first({ cycle_id: null }))));
  await until(() => row() === null, "the row off the list");
  expect(toasts).toEqual([
    expect.objectContaining({ type: "success", title: "common.success", message: "issue.remove.cycle.success" }),
  ]);
  await until(() => progressReads().length === 1, "the progress again");
  expect(progressReads()[0]).toMatchObject({ method: "GET" });
  expect(writesOf("first")).toHaveLength(1);
});

it.each([
  { status: 409, code: "planning.cycles_disabled" },
  { status: 403, code: "forbidden" },
])("says why nerve refused the removal ($code); the row stays and nothing is read", async ({ status, code }) => {
  await shown();
  const change = await removal();
  await act(async () => change.answer(problem(status, code)));
  await act(async () => vi.advanceTimersByTimeAsync(100));
  expect(toasts).toEqual([expect.objectContaining({ type: "error", message: `errors.${code.replace(".", "_")}` })]);
  expect(row()).not.toBeNull();
  expect(progressReads()).toEqual([]);
});

it.each(["another cycle", "another session"])(
  "neither says nor reads anything when nerve confirms the removal once the page is %s's",
  async (ending) => {
    await shown();
    const change = await removal();
    if (ending === "another cycle") {
      bound.params = { workspaceSlug: "ws", projectId: "A", cycleId: C2 };
      await page.mount();
    } else bound.session.loginId = "two";
    await act(async () => change.answer(json(200, first({ cycle_id: null }))));
    await act(async () => vi.advanceTimersByTimeAsync(100));
    expect(toasts).toEqual([]);
    expect(progressReads()).toEqual([]);
  }
);

it("opens the edit dialog of a row as the cycle page's, with its work item", async () => {
  await shown();
  await choose("common.actions.edit");
  expect(bound.dialog?.storeType).toBe(EIssuesStoreType.CYCLE);
  expect(bound.dialog?.data).toMatchObject({ id: "first", cycle_id: C1 });
  expect(bound.dialog?.sourceIssueId).toBeUndefined();
});

it("opens the copy dialog of a row as the cycle page's, whose creation then joins the cycle", async () => {
  await shown();
  await choose("common.actions.make_a_copy");
  expect(bound.dialog?.storeType).toBe(EIssuesStoreType.CYCLE);
  expect(bound.dialog?.sourceIssueId).toBe("first");
  expect(bound.dialog?.data).toMatchObject({ name: "first (copy)", project_id: "A" });
});
