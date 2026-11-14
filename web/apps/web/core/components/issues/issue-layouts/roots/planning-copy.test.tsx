// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// What a new work item's caller gives its form of the cycle and the modules (M6 P7a spec 2.6, 2.7, R15; re-72 F7, F8):
// a row's copy, the first such caller, gives its source's. As the form opens with its project it reads the project's
// two lists; until both answer, its fields show loading and hold what was given. Once they have, it starts with those
// the work item may join, its options (an ended cycle, an archived module are none), the page's module put beside
// them, as its defaults, not as an edit; what it holds is what it sends. A form whose project was changed meanwhile
// keeps its own. A list nerve refuses shows its refusal under its field, with a retry: the form holds none of what was
// given of it, which the modal keeps until a retry answers (P7a T12, the controller's ruling on a failed read). The
// pages, the rows, the dialog's form and its fields, the stores and HTTP are real.
import { act, createElement, type ReactNode } from "react";
import { MemoryRouter } from "react-router";
import { SWRConfig } from "swr";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { Issue, IssueDetail } from "@nerve/api-client";
import { EIssuesStoreType } from "@nerve/types";
import { WorkItemLevelModals } from "@/components/modals/work-item-level";
import { json, problem, type Call } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { issueDetailOf, listIssue } from "@/store/issue/fake-issues";
import { cycleRequests, loadCycle, rowOf } from "@/store/planning/fake-cycles";
import { loadModule, moduleRequests, moduleRowOf } from "@/store/planning/fake-modules";
import { cycleOf, loadProjects, moduleOf, preferencesOf, projectOf } from "@/store/project/fake-projects";
import type { RootStore } from "@/store/root.store";
import {
  C1,
  M1,
  choose,
  creates,
  cycleField,
  cyclePreferencesOf,
  fieldOf,
  form,
  layouts,
  loading,
  moduleField,
  modulePreferencesOf,
  pick,
  planningPageTab,
  projectField,
} from "./fake-planning-page";
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
// jsdom has no viewport geometry, dialog transitions, editor or font loading; the form and its fields are real.
vi.mock("@atlaskit/pragmatic-drag-and-drop-auto-scroll/element", () => ({ autoScrollForElements: () => () => {} }));
vi.mock("@/components/core/render-if-visible-HOC", () => ({
  default: ({ children, as }: { children: ReactNode; as?: string }) =>
    as ? createElement(as, null, children) : children,
}));
vi.mock("@nerve/propel/emoji-icon-picker", () => ({ Logo: () => null }));
vi.mock("@nerve/ui", async (original) => ({
  ...(await original<typeof import("@nerve/ui")>()),
  ...(await import("@/components/issues/issue-modal/fake-dialog")).modal,
}));
vi.mock(
  "@/components/editor/rich-text",
  async () => (await import("@/components/issues/issue-modal/fake-dialog")).editor
);
vi.mock("@/components/issues/create-issue-toast-action-items", () => ({ CreateIssueToastActionItems: () => null }));

/** The project's cycle that has ended; its modules Edge, and Gone, archived. */
const DONE = "019a0000-0000-7000-8000-0000000000c3";
const EDGE = "019a0000-0000-7000-8000-0000000000d2";
const GONE = "019a0000-0000-7000-8000-0000000000d9";
const shownFields = { state_id: "state", priority: "high", target_date: "2026-12-31" } as const;
/** The module page's work item "first", in the ended cycle and in the page's module, Edge and Gone. */
const held: Issue = listIssue("first", 1000, { cycle_id: DONE, module_ids: [M1, EDGE, GONE], ...shownFields });
/** The project page's work item "first", in the cycle Now, under way, and in Edge and Gone. */
const own: Issue = listIssue("first", 1000, { cycle_id: C1, module_ids: [EDGE, GONE], ...shownFields });

let page: Awaited<ReturnType<typeof planningPageTab>>;
afterEach(async () => page.close());
beforeEach(() => {
  bound.params = { workspaceSlug: "ws", projectId: "A", moduleId: M1 };
});

/** The project's lists as nerve gives them: its cycles, Now and the ended Done; its modules, Core (the page's) and Edge. */
const lists = () => ({
  cycles: [
    rowOf(cycleOf(page.project, "Now", { id: C1, status: "current" })),
    rowOf(cycleOf(page.project, "Done", { id: DONE, status: "completed" })),
  ],
  modules: [
    moduleRowOf(moduleOf(page.project, "Core", { id: M1 })),
    moduleRowOf(moduleOf(page.project, "Edge", { id: EDGE })),
  ],
});
/** The reads of the project's list of what is not archived of a domain (path). */
const listReads = (path: string) =>
  page.nerve.to(path).filter((call) => call.method === "GET" && call.query.archived !== "true");
const cycleReads = () => listReads(cycleRequests.list(page.project));
const moduleReads = () => listReads(moduleRequests.list(page.project));
/** Answers the reads of the project's two lists that are out. */
const answerLists = () =>
  act(async () => {
    const { cycles, modules } = lists();
    for (const read of cycleReads()) read.answer(json(200, { data: cycles }));
    for (const read of moduleReads()) read.answer(json(200, { data: modules }));
    await vi.advanceTimersByTimeAsync(20);
  });

/** The tab holds the ended cycle and the modules Edge and Gone as nerve gives them, by their names. */
async function holdNames() {
  await loadCycle(
    page.nerve,
    page.app.planning.cycles,
    cycleOf(page.project, "Done", { id: DONE, status: "completed" })
  );
  await loadModule(page.nerve, page.app.planning.modules, moduleOf(page.project, "Edge", { id: EDGE }));
  const gone = { id: GONE, status: "completed" as const, archived_at: "2026-12-01T09:00:00Z" };
  await loadModule(page.nerve, page.app.planning.modules, moduleOf(page.project, "Gone", gone));
  page.nerve.calls.length = 0;
}
/** Shows the module's page, its one work item "first" (held) in the list, the names held. */
async function shown() {
  page = await planningPageTab(EIssuesStoreType.MODULE, { role: 15 });
  bound.app = page.app;
  await holdNames();
  const list = layouts.list.page(true, held);
  await page.show(modulePreferencesOf(layouts.list.display), list);
  answerDialogReads(held, list);
}
/** Shows the project's page, its one work item "first" (own) in the list, the names held; the cycle Now is the tab's. */
async function projectShown() {
  bound.params = { workspaceSlug: "ws", projectId: "A" };
  page = await planningPageTab(EIssuesStoreType.CYCLE, { role: 15 });
  bound.app = page.app;
  await holdNames();
  await page.render(
    <SWRConfig value={{ provider: () => new Map() }}>
      <MemoryRouter>
        <ProjectLayoutRoot />
      </MemoryRouter>
    </SWRConfig>
  );
  const preferences = () => page.nerve.to("/api/v0/me/projects/A/preferences");
  await until(() => preferences().length === 1, "the project's preferences");
  const display = { ...preferencesOf().display_filters, ...layouts.list.display };
  await act(async () => preferences()[0]!.answer(json(200, preferencesOf({ display_filters: display }))));
  await until(() => page.lists().length === 1, "the project's list");
  const list = layouts.list.page(true, own);
  await act(async () => page.lists()[0]!.answer(json(200, list)));
  answerDialogReads(own, list);
}
/**
 * nerve answers the reads a row's dialog makes of the work item row: the work item's, its resources', and the page's
 * list's again once the check of the work item took the row off.
 */
function answerDialogReads(row: Issue, list: unknown) {
  const detail: IssueDetail = { ...issueDetailOf("first"), ...row };
  const normal = page.nerve.respond;
  page.nerve.respond = (call: Call) => {
    if (call.method !== "GET") normal?.(call);
    else if (call.path === "/api/v0/issues/first") call.answer(json(200, detail));
    else if (/\/issues\/first\/(sub-issues|relations|links|reactions|activities|comments)$/.test(call.path))
      call.answer(json(200, { data: [], next_cursor: null }));
    else if (call.path === "/api/v0/projects/A/issues") call.answer(json(200, list));
    else normal?.(call);
  };
}
/** nerve refuses the reads of the project's two lists that are out. */
const refuseLists = () =>
  act(async () => {
    for (const read of [...cycleReads(), ...moduleReads()]) read.answer(problem(403, "forbidden"));
    await vi.advanceTimersByTimeAsync(20);
  });
/** What the form says of its lists' reads nerve refused: each refusal, with its retry. */
const refusals = () => [...(form()?.querySelectorAll('[role="alert"]') ?? [])].map((node) => node.textContent);
/** Retries the read of the list whose refusal is the form's index'th. */
async function retry(index: number) {
  const button = form()?.querySelectorAll('[role="alert"]')[index]?.querySelector("button");
  expect(button?.textContent, "the refusal's retry").toBe("common.retry");
  await act(async () => button!.click());
}
/** The form's "discard", which closes it. */
const discard = () =>
  act(async () => [...form()!.querySelectorAll("button")].find((button) => button.textContent === "discard")!.click());
/** Opens the row's copy; resolves once its form has read the work item's description. */
async function copied() {
  await until(() => !!document.querySelector('[id^="issue-first"]'), "the row");
  await choose("common.actions.make_a_copy");
  await until(() => form()?.querySelector<HTMLButtonElement>('button[type="submit"]')?.disabled === false, "the form");
}

it("holds a copy's cycle and modules, its fields loading, until the project's two lists answer; then sends those it may join with the page's module", async () => {
  await shown();
  await copied();
  await until(() => cycleReads().length === 1 && moduleReads().length === 1, "the reads of the two lists");
  expect([loading(cycleField()), loading(moduleField())], "the fields, before the lists answer").toEqual([true, true]);
  expect(cycleField()?.textContent, "the source's ended cycle, held").toContain("Done");
  expect(moduleField()?.textContent, "the source's modules, held").toContain("Core; Edge; Gone, common.archived");
  await answerLists();
  expect([loading(cycleField()), loading(moduleField())], "the fields, once both answered").toEqual([false, false]);
  expect(cycleField()?.textContent).toBe("common.cycle: cycle.label");
  expect(moduleField()?.textContent).toBe("common.modules: module.count: Core; Edge");
  await act(async () => form()!.dispatchEvent(new Event("submit", { bubbles: true, cancelable: true })));
  await until(() => creates().length === 1, "the copy");
  expect(creates()[0]!.body).toMatchObject({ name: "first (copy)", module_ids: [M1, EDGE] });
  expect(creates()[0]!.body).not.toHaveProperty("cycle_id");
});

it("starts the copy with what it may join as its defaults, not as an edit: discarding it asks nothing", async () => {
  await shown();
  await copied();
  await until(() => cycleReads().length === 1 && moduleReads().length === 1, "the reads of the two lists");
  await answerLists();
  expect(moduleField()?.textContent).toBe("common.modules: module.count: Core; Edge");
  await discard();
  expect(form(), "the copy's dialog").toBeNull();
  expect(document.body.textContent).not.toContain("issue.draft.save_prompt.title");
  expect(creates()).toEqual([]);
});

it("keeps a form's own cycle once its project changed before the lists of the project it opened with answer", async () => {
  bound.params = { workspaceSlug: "ws", projectId: "A", cycleId: C1 };
  page = await planningPageTab(EIssuesStoreType.CYCLE, { role: 15 });
  bound.app = page.app;
  const other = projectOf("B", page.workspace.id, { id: "B" });
  await loadProjects(page.nerve, page.app.projectRoot.project, page.workspace, [page.project, other]);
  await page.show(
    cyclePreferencesOf(layouts.list.display),
    page.listOf("list"),
    <WorkItemLevelModals workItemIdentifier={undefined} />
  );
  // the app's own new work item, of no page, whose form starts with the page's cycle and lets the project be changed
  await act(async () => page.app.commandPalette.toggleCreateIssueModal(true));
  await until(() => cycleReads().length === 1 && moduleReads().length === 1, "the reads of the two lists");
  expect(cycleField()?.textContent).toContain("Now");
  await pick(projectField, "B");
  await until(() => fieldOf("common.project")?.textContent === "common.project: B", "the project B");
  expect(cycleField()?.textContent).toBe("common.cycle: cycle.label");
  await answerLists();
  expect(cycleField()?.textContent, "B's form, once A's lists answered").toBe("common.cycle: cycle.label");
  const elsewhere = () => page.nerve.to("/api/v0/projects/B/issues").filter((call) => call.method === "POST");
  await act(async () => {
    const input = form()!.querySelector<HTMLInputElement>("#name")!;
    Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, "value")!.set!.call(input, "Moved");
    input.dispatchEvent(new Event("input", { bubbles: true }));
  });
  await act(async () => form()!.dispatchEvent(new Event("submit", { bubbles: true, cancelable: true })));
  await until(() => elsewhere().length === 1, "the creation in B");
  expect(elsewhere()[0]!.body).not.toHaveProperty("cycle_id");
});

it("shows under each field nerve's refusal of its project's list, with a retry; the form holds none of the copy's cycle and modules", async () => {
  await projectShown();
  await copied();
  await until(() => cycleReads().length === 1 && moduleReads().length === 1, "the reads of the two lists");
  expect(cycleField()?.textContent, "the source's cycle, held while the lists are out").toContain("Now");
  await refuseLists();
  expect([loading(cycleField()), loading(moduleField())], "the fields, once nerve refused the lists").toEqual([
    false,
    false,
  ]);
  expect(refusals()).toEqual(["errors.forbidden common.retry", "errors.forbidden common.retry"]);
  expect(cycleField()?.textContent).toBe("common.cycle: cycle.label");
  expect(moduleField()?.textContent).toBe("common.modules: module.no_module");
});

it("starts the copy with what it may join once a retry of each list answers, as its defaults, not as an edit", async () => {
  await projectShown();
  await copied();
  await until(() => cycleReads().length === 1 && moduleReads().length === 1, "the reads of the two lists");
  await refuseLists();
  // the modules' first, then the cycles' (a list read again is out, and says nothing meanwhile)
  await retry(1);
  await retry(0);
  await until(() => cycleReads().length === 2 && moduleReads().length === 2, "the reads of the two lists again");
  await answerLists();
  expect(refusals(), "the refusals, once a retry answered").toEqual([]);
  expect(cycleField()?.textContent).toBe("common.cycle: Now");
  expect(moduleField()?.textContent).toBe("common.modules: Edge");
  await discard();
  expect(form(), "the copy's dialog").toBeNull();
  expect(document.body.textContent).not.toContain("issue.draft.save_prompt.title");
});

it("sends neither the cycle nor the modules of a copy whose project's lists nerve refused", async () => {
  await projectShown();
  await copied();
  await until(() => cycleReads().length === 1 && moduleReads().length === 1, "the reads of the two lists");
  await refuseLists();
  await act(async () => form()!.dispatchEvent(new Event("submit", { bubbles: true, cancelable: true })));
  await until(() => creates().length === 1, "the copy");
  expect(creates()[0]!.body).toMatchObject({ name: "first (copy)" });
  expect(creates()[0]!.body).not.toHaveProperty("cycle_id");
  expect(creates()[0]!.body).not.toHaveProperty("module_ids");
});

it("starts the copy with the modules it may join once its modules' field, opened, reads their list again", async () => {
  await projectShown();
  await copied();
  await until(() => cycleReads().length === 1 && moduleReads().length === 1, "the reads of the two lists");
  await refuseLists();
  await act(async () => {
    moduleField()!.click();
    await vi.advanceTimersByTimeAsync(20);
  });
  await until(() => moduleReads().length === 2, "the field's read of the modules");
  await answerLists();
  expect(refusals(), "the cycles' refusal alone").toEqual(["errors.forbidden common.retry"]);
  expect(moduleField()?.textContent).toBe("common.modules: Edge");
  expect(cycleField()?.textContent).toBe("common.cycle: cycle.label");
});
