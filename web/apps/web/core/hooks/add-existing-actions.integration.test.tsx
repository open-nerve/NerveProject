// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// "Add existing" as the pages' actions send it (P7b spec 2.9, 2.11, R2, R6): one change of each work item through the
// write unit, two at most at once (nerve's write gate takes two writes of a project at once), each one's outcome in the
// order given; a cycle's page joins its cycle, a module's page its module, the set worked out in the write's turn from
// the set then held (archived modules, and one another write added meanwhile, kept). The batch reads the page's
// progress again once, when nerve confirmed any. The project's page sends what its caller gives and reads no progress.
// The stores and HTTP are real; the route's params are the test's.
import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { Issue } from "@nerve/api-client";
import { EIssuesStoreType } from "@nerve/types";
import { answered, json, problem, type Call } from "@/lib/auth/fake-nerve";
import { track } from "@/lib/auth/fake-time";
import { StoreContext } from "@/lib/store-context";
import { issueDetailOf, issueOf, issuePageOf, issueStoreTab } from "@/store/issue/fake-issues";
import { cycleRequests, loadCycle } from "@/store/planning/fake-cycles";
import { loadModule, loadModules, moduleRequests, moduleRowOf } from "@/store/planning/fake-modules";
import { cycleOf, moduleOf } from "@/store/project/fake-projects";
import { useIssuesActions, type IssueActions } from "./use-issues-actions";

const C1 = "019a0000-0000-7000-8000-0000000000c1";
const C2 = "019a0000-0000-7000-8000-0000000000c2";
/** The module page's module; Beta, of the project's list; Gamma, archived, of no list the page reads; Delta, new. */
const M1 = "019a0000-0000-7000-8000-0000000000d1";
const BETA = "019a0000-0000-7000-8000-0000000000d2";
const GAMMA = "019a0000-0000-7000-8000-0000000000d3";
const DELTA = "019a0000-0000-7000-8000-0000000000d4";
const LATER = "2026-10-29T00:00:00Z";
const LATEST = "2026-10-30T00:00:00Z";
const fixture = vi.hoisted(() => ({
  app: undefined as unknown,
  params: {} as Record<string, string>,
  storeType: undefined as EIssuesStoreType | undefined,
  actions: undefined as IssueActions | undefined,
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
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "one" } }, publicClient: {} }));
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
  fixture.actions = useIssuesActions(fixture.storeType!);
  return null;
}
/** The page of the store type with the route's params; the project's work items held as nerve listed them. */
async function opened(storeType: EIssuesStoreType, params: Record<string, string>, held: Issue[]) {
  fixture.storeType = storeType;
  fixture.params = { workspaceSlug: "ws", projectId: "A", ...params };
  await answered(
    tab.nerve,
    () => tab.app.issue.projectIssues.fetchIssues("ws", "A", "init-loader", { limit: 10 }),
    ["GET", "/api/v0/projects/A/issues"],
    issuePageOf(held),
    "the project's work items"
  );
  tab.nerve.calls.length = 0;
  view = createRoot(document.createElement("div"));
  await act(async () =>
    view.render(
      <StoreContext.Provider value={tab.app as never}>
        <Probe />
      </StoreContext.Provider>
    )
  );
}
beforeEach(async () => {
  vi.useFakeTimers();
  tab = await issueStoreTab();
  fixture.app = tab.app;
});
afterEach(async () => {
  await act(async () => view.unmount());
  tab.app.dispose();
  vi.useRealTimers();
});

const writes = () => tab.nerve.calls.filter((call) => call.method === "PATCH");
/** The writes nerve has not answered yet. */
const out = (answeredCalls: Set<Call>) => writes().filter((call) => !answeredCalls.has(call));
/** Adds the work items by the page's action; the batch's answer, tracked. */
async function added(ids: string[], extra?: Parameters<NonNullable<IssueActions["addIssuesToView"]>>[1]) {
  let promise!: ReturnType<NonNullable<IssueActions["addIssuesToView"]>>;
  await act(async () => {
    promise = fixture.actions!.addIssuesToView!(ids, extra);
    await vi.advanceTimersByTimeAsync(100);
  });
  return { promise, result: track(promise) };
}
/**
 * Answers the writes of the work items one at a time, in the order given, as answer says, checking before each answer
 * and after the last that nerve holds two writes at most; then that none is left out.
 */
async function answerEach(order: string[], answer: (id: string) => Response, answeredCalls = new Set<Call>()) {
  expect(out(answeredCalls).length, "the writes out at once").toBeLessThanOrEqual(2);
  const [id, ...rest] = order;
  if (id === undefined) return expect(out(answeredCalls), "the writes left out").toEqual([]);
  const call = writes().find((write) => write.path === `/api/v0/issues/${id}` && !answeredCalls.has(write));
  expect(call, `the write of ${id}, sent`).toBeDefined();
  answeredCalls.add(call!);
  await act(async () => {
    call!.answer(answer(id));
    await vi.advanceTimersByTimeAsync(100);
  });
  await answerEach(rest, answer, answeredCalls);
}
const cycleProgressReads = () => tab.nerve.to(cycleRequests.progress({ id: C1 }));
const moduleProgressReads = () => tab.nerve.to(moduleRequests.progress({ id: M1 }));

/** A cycle's page of the cycle C1, under way; its project holds x, y in no cycle and z in the cycle C2. */
async function cyclePage() {
  await loadCycle(tab.nerve, tab.app.planning.cycles, cycleOf(tab.project, "Now", { id: C1, status: "current" }));
  await opened(EIssuesStoreType.CYCLE, { cycleId: C1 }, [
    issueOf("x"),
    issueOf("y"),
    { ...issueOf("z"), cycle_id: C2 },
  ]);
}
const joinedCycle = (id: string) => json(200, { ...issueDetailOf(id), cycle_id: C1, updated_at: LATER });

it.each([
  ["a refusal", problem(409, "planning.cycle_completed")],
  ["nerve being busy", problem(503, "server_busy")],
])(
  "joins three work items to the cycle two at a time, keeps %s of the second as its outcome, and reads the progress once",
  async (_, refusal) => {
    await cyclePage();
    const { promise, result } = await added(["x", "y", "z"]);
    expect(writes().map((call) => [call.path, call.body])).toEqual([
      ["/api/v0/issues/x", { cycle_id: C1 }],
      ["/api/v0/issues/y", { cycle_id: C1 }],
    ]);
    await answerEach(["y", "x", "z"], (id) => (id === "y" ? refusal.clone() : joinedCycle(id)));
    expect(writes().map((call) => [call.path, call.body])).toEqual([
      ["/api/v0/issues/x", { cycle_id: C1 }],
      ["/api/v0/issues/y", { cycle_id: C1 }],
      ["/api/v0/issues/z", { cycle_id: C1 }],
    ]);
    expect(result.settled).toBe(true);
    const outcome = await promise;
    expect(outcome.confirmedIds).toEqual(["x", "z"]);
    expect(outcome.failures).toHaveLength(1);
    expect(outcome.failures[0]!.id).toBe("y");
    expect(outcome.failures[0]!.error).toMatchObject({ status: refusal.status });
    expect(cycleProgressReads(), "the batch's reads of the progress").toHaveLength(1);
    expect(tab.app.issue.issues.getIssueById("z")?.cycle_id).toBe(C1);
  }
);

it("reads no progress for a batch nerve refuses whole", async () => {
  await cyclePage();
  const { promise } = await added(["x", "y"]);
  await answerEach(["x", "y"], () => problem(409, "planning.cycle_completed"));
  const outcome = await promise;
  expect(outcome.confirmedIds).toEqual([]);
  expect(outcome.failures.map((failure) => failure.id)).toEqual(["x", "y"]);
  expect(cycleProgressReads()).toEqual([]);
});

it("joins the module, its set worked out in the write's turn from the set then held, archived and new ones kept", async () => {
  await loadModule(tab.nerve, tab.app.planning.modules, moduleOf(tab.project, "Core", { id: M1 }));
  await loadModules(
    tab.nerve,
    tab.app.planning.modules,
    [moduleRowOf(moduleOf(tab.project, "Core", { id: M1 })), moduleRowOf(moduleOf(tab.project, "Beta", { id: BETA }))],
    tab.project
  );
  await opened(EIssuesStoreType.MODULE, { moduleId: M1 }, [{ ...issueOf("old"), module_ids: [BETA, GAMMA] }]);
  // another write of the work item is out first: the batch's goes in its turn, after nerve answers that one
  const elsewhere = track(tab.app.issue.projectIssues.updateIssue("ws", "A", "old", { modules: { add: [DELTA] } }));
  await act(async () => vi.advanceTimersByTimeAsync(100));
  const { promise } = await added(["old"]);
  expect(writes().map((call) => call.body)).toEqual([{ module_ids: [BETA, GAMMA, DELTA] }]);
  const answer = (call: Call, module_ids: string[], updated_at: string) =>
    act(async () => {
      call.answer(json(200, { ...issueDetailOf("old"), module_ids, updated_at }));
      await vi.advanceTimersByTimeAsync(100);
    });
  await answer(writes()[0]!, [BETA, GAMMA, DELTA], LATER);
  expect(elsewhere.settled).toBe(true);
  expect(writes()[1]?.body, "the batch's write, in its turn").toEqual({ module_ids: [BETA, GAMMA, DELTA, M1] });
  await answer(writes()[1]!, [BETA, GAMMA, DELTA, M1], LATEST);
  expect(await promise).toEqual({ confirmedIds: ["old"], failures: [] });
  expect(moduleProgressReads()).toHaveLength(1);
});

it("sends on the project's page what its caller gives, two at a time, and reads no progress", async () => {
  await opened(EIssuesStoreType.PROJECT, {}, [issueOf("x"), issueOf("y"), issueOf("z")]);
  const { promise } = await added(["x", "y", "z"], { target_date: "2026-11-01" });
  await answerEach(["x", "y", "z"], (id) => json(200, { ...issueDetailOf(id), target_date: "2026-11-01" }));
  expect(writes().map((call) => [call.path, call.body])).toEqual([
    ["/api/v0/issues/x", { target_date: "2026-11-01" }],
    ["/api/v0/issues/y", { target_date: "2026-11-01" }],
    ["/api/v0/issues/z", { target_date: "2026-11-01" }],
  ]);
  expect(await promise).toEqual({ confirmedIds: ["x", "y", "z"], failures: [] });
  expect(tab.nerve.calls.filter((call) => call.path.endsWith("/progress"))).toEqual([]);
});
