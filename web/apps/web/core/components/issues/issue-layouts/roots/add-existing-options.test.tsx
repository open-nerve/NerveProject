// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// What the dialog of "add existing" lists on a planning page (P7b spec 2.9, R2, R21; the controller's A′): on a cycle's
// page each work item in another cycle shows that cycle beside its name, by the name the planning store has for it
// (usePlanningNames, which reads the project's archive once for an archived one), an archived one captioned so, so
// that the move shows before it is sent; a module's page shows none (joining a module leaves no other). Once each work
// item nerve found is already in the page's cycle or module the dialog says so, not that it found none. The page, its
// empty state, the dialog, the stores and HTTP are real.
import { act, createElement, type ReactNode } from "react";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { Issue } from "@nerve/api-client";
import { EIssuesStoreType } from "@nerve/types";
import { json } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { toasts } from "@/lib/fake-toast";
import { issueOf } from "@/store/issue/fake-issues";
import { cycleRequests, rowOf } from "@/store/planning/fake-cycles";
import { moduleRowOf } from "@/store/planning/fake-modules";
import { cycleOf, moduleOf } from "@/store/project/fake-projects";
import type { RootStore } from "@/store/root.store";
import {
  C1,
  M1,
  answeringPlanningLists,
  buttonOf,
  cyclePreferencesOf,
  layouts,
  modulePreferencesOf,
  planningPageTab,
} from "./fake-planning-page";

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
// jsdom has no viewport geometry; rows, cards and HTTP are real.
vi.mock("@atlaskit/pragmatic-drag-and-drop-auto-scroll/element", () => ({ autoScrollForElements: () => () => {} }));
vi.mock("@/components/core/render-if-visible-HOC", () => ({
  default: ({ children, as }: { children: ReactNode; as?: string }) =>
    as ? createElement(as, null, children) : children,
}));

const ARCHIVED = "2026-10-20T09:00:00Z";
let page: Awaited<ReturnType<typeof planningPageTab>>;
beforeEach(() => {
  toasts.length = 0;
});
afterEach(async () => page.close());

const pages = {
  cycle: {
    type: EIssuesStoreType.CYCLE,
    params: { cycleId: C1 },
    preferences: cyclePreferencesOf,
    empty: "project_empty_state.cycle_work_items",
  },
  module: {
    type: EIssuesStoreType.MODULE,
    params: { moduleId: M1 },
    preferences: modulePreferencesOf,
    empty: "project_empty_state.module_work_items",
  },
} as const;
type PageName = keyof typeof pages;

/**
 * The page with none of its work items, its project's cycles Now (the page's C1) and Next, Old and Older archived, its
 * modules the page's M1, as nerve lists them whenever they are read.
 */
async function emptyPage(name: PageName) {
  bound.params = { workspaceSlug: "ws", projectId: "A", ...pages[name].params };
  page = await planningPageTab(pages[name].type);
  bound.app = page.app;
  await page.show(pages[name].preferences(layouts.list.display), page.listOf("list", false));
  const cycles = {
    now: cycleOf(page.project, "Now", { id: C1, status: "current" }),
    next: cycleOf(page.project, "Next", { status: "upcoming" }),
    old: cycleOf(page.project, "Old", { status: "completed", archived_at: ARCHIVED }),
    older: cycleOf(page.project, "Older", { status: "completed", archived_at: ARCHIVED }),
  };
  answeringPlanningLists({
    cycles: [rowOf(cycles.now), rowOf(cycles.next)],
    archivedCycles: [rowOf(cycles.old), rowOf(cycles.older)],
    modules: [moduleRowOf(moduleOf(page.project, "Core", { id: M1 }))],
    archivedModules: [],
  });
  return cycles;
}
/** Opens the dialog from the empty state; nerve answers its search with the work items. */
async function searched(name: PageName, found: Issue[]) {
  await act(async () => buttonOf(`${pages[name].empty}.cta_secondary`)!.click());
  await until(() => page.searches().length === 1, "the dialog's search");
  await act(async () => page.searches()[0]!.answer(json(200, { data: found })));
}
/** A work item of the project, numbered, with fields. */
const item = (name: string, sequence: number, fields: Partial<Issue> = {}): Issue => ({
  ...issueOf(name),
  sequence_id: sequence,
  ...fields,
});
/** Each option of the dialog, as it reads: its identifier, its name and what it shows beside it. */
const options = () => [...document.querySelectorAll('[role="option"]')].map((node) => node.textContent);
/** The reads of the project's cycles: those not archived (false), the archive (true). */
const cycleReads = (archived: boolean) =>
  page.nerve
    .to(cycleRequests.list(page.project))
    .filter((call) => call.method === "GET" && (call.query.archived === "true") === archived);
const dialogText = () => document.querySelector('[role="dialog"]')?.textContent ?? "";

it("shows beside each work item listed on a cycle's page the cycle it is in, an archived one captioned, the archive read once", async () => {
  const { next, old, older } = await emptyPage("cycle");
  await searched("cycle", [
    item("x", 1),
    item("y", 2, { cycle_id: next.id }),
    item("v", 3, { cycle_id: old.id }),
    item("u", 4, { cycle_id: older.id }),
  ]);
  await until(() => options().some((option) => option?.includes("Older")), "the archived cycles' names");
  expect(options()).toEqual([
    "A-1x",
    "A-2y, common.cycle: Next",
    "A-3v, common.cycle: Old, common.archived",
    "A-4u, common.cycle: Older, common.archived",
  ]);
  expect(
    [cycleReads(false), cycleReads(true)].map((reads) => reads.length),
    "the project's cycles, each list once"
  ).toEqual([1, 1]);
});

it("shows no cycle beside a work item listed on a module's page", async () => {
  const { next } = await emptyPage("module");
  await searched("module", [item("x", 1), item("y", 2, { cycle_id: next.id })]);
  await act(async () => vi.advanceTimersByTimeAsync(100));
  expect(options()).toEqual(["A-1x", "A-2y"]);
  expect(cycleReads(false), "the project's cycles, not read for the dialog").toEqual([]);
});

it.each<{ name: PageName; fields: Partial<Issue>; hint: string }>([
  { name: "cycle", fields: { cycle_id: C1 }, hint: "cycle.found_already_in" },
  { name: "module", fields: { module_ids: [M1] }, hint: "module.found_already_in" },
])("says once each work item found is in the $name already that it is, not that none was found", async (each) => {
  await emptyPage(each.name);
  await searched(each.name, [item("x", 1, each.fields), item("y", 2, each.fields)]);
  expect(options()).toEqual([]);
  expect(dialogText()).toContain(each.hint);
  expect(dialogText()).not.toContain("issue_relation.empty_state.search.title");
});

it("says that none was found once nerve found none", async () => {
  await emptyPage("cycle");
  await searched("cycle", []);
  expect(dialogText()).toContain("issue_relation.empty_state.search.title");
  expect(dialogText()).not.toContain("cycle.found_already_in");
});

it("lists the work items found that are not in the cycle, without the saying, once some are", async () => {
  await emptyPage("cycle");
  await searched("cycle", [item("x", 1, { cycle_id: C1 }), item("y", 2)]);
  expect(options()).toEqual(["A-2y"]);
  expect(dialogText()).not.toContain("cycle.found_already_in");
});
