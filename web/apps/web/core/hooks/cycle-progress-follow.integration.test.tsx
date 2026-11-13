// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// A cycle page's progress is read again once nerve confirms a write its list sent (P5b spec 2.6, R10; re-66 L5): a
// creation, a change (its removal from the cycle too), a deletion, an archiving or a drop each read it once, as nerve
// counts it, never worked out by the page; of two reads, the newer answer is the one the page shows. The write's own
// answer or refusal is its caller's, as before. A refused write reads nothing, and so does one nerve confirms once the
// page is no longer the one that sent it: another cycle (and back), another session, a lost and regained permission,
// or the page gone. The stores and HTTP are real; the route's params are the test's.
import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { EIssuesStoreType } from "@nerve/types";
import { json, noContent, problem } from "@/lib/auth/fake-nerve";
import { track, until } from "@/lib/auth/fake-time";
import { toasts } from "@/lib/fake-toast";
import { StoreContext } from "@/lib/store-context";
import { issueDetailOf, issueOf, issuePageOf, issueStoreTab, requestedIssue } from "@/store/issue/fake-issues";
import { cycleRequests, loadCycle, progressOf } from "@/store/planning/fake-cycles";
import { cycleOf, loadProjects, preferencesOf, projectOf } from "@/store/project/fake-projects";
import { useIssuesActions, type IssueActions } from "./use-issues-actions";

const C1 = "019a0000-0000-7000-8000-0000000000c1";
const C2 = "019a0000-0000-7000-8000-0000000000c2";
const LATER = "2026-10-29T00:00:00Z";
const fixture = vi.hoisted(() => ({
  app: undefined as unknown,
  session: { loginId: "one" },
  params: {} as Record<string, string>,
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
const cycle = () => cycleOf(tab.project, "Now", { id: C1, status: "current" });
const progressReads = () => tab.nerve.to(cycleRequests.progress({ id: C1 }));
/** The cycle's work item "old", as nerve answers a write of it with fields. */
const old = (fields: object = {}) => ({ ...issueDetailOf("old"), cycle_id: C1, updated_at: LATER, ...fields });
beforeEach(async () => {
  vi.useFakeTimers();
  tab = await issueStoreTab();
  fixture.app = tab.app;
  fixture.session.loginId = "one";
  fixture.params = { workspaceSlug: "ws", projectId: "A", cycleId: C1 };
  toasts.length = 0;
  await loadCycle(tab.nerve, tab.app.planning.cycles, cycle());
  const read = await requestedIssue(tab.nerve, () => tab.app.issue.cycleIssuesFilter.fetchFilters("ws", C1), [
    "GET",
    `/api/v0/me/cycles/${C1}/preferences`,
  ]);
  const { filters, display_filters, display_properties } = preferencesOf();
  read.call.answer(json(200, { filters, display_filters, display_properties }));
  await read.promise;
  const list = await requestedIssue(
    tab.nerve,
    () => tab.app.issue.cycleIssues.fetchIssues("ws", { projectID: "A", entityID: C1 }, "init-loader", { limit: 10 }),
    ["GET", "/api/v0/projects/A/issues"]
  );
  list.call.answer(json(200, issuePageOf([{ ...issueOf("old"), cycle_id: C1 }])));
  await list.promise;
  tab.nerve.calls.length = 0;
  view = createRoot(document.createElement("div"));
  await render();
});
afterEach(async () => {
  await act(async () => view.unmount());
  tab.app.dispose();
  vi.useRealTimers();
});

/** Each write the cycle page's list sends by its actions, the request it is, and nerve's confirmation of it. */
const writes: Record<string, { send: () => Promise<unknown>; endpoint: [string, string]; confirmed: () => Response }> =
  {
    "a creation": {
      send: () => fixture.actions!.createIssue!("A", { name: "Made" }),
      endpoint: ["POST", "/api/v0/projects/A/issues"],
      confirmed: () => json(201, { ...issueDetailOf("made"), name: "Made", cycle_id: C1 }),
    },
    "a change": {
      send: () => fixture.actions!.updateIssue!("A", "old", { priority: "high" }),
      endpoint: ["PATCH", "/api/v0/issues/old"],
      confirmed: () => json(200, old({ priority: "high" })),
    },
    "a removal from the cycle": {
      send: () => fixture.actions!.updateIssue!("A", "old", { cycle_id: null }),
      endpoint: ["PATCH", "/api/v0/issues/old"],
      confirmed: () => json(200, old({ cycle_id: null })),
    },
    "a deletion": {
      send: () => fixture.actions!.removeIssue("A", "old"),
      endpoint: ["DELETE", "/api/v0/issues/old"],
      confirmed: () => noContent(),
    },
    "an archiving": {
      send: () => fixture.actions!.archiveIssue!("A", "old"),
      endpoint: ["POST", "/api/v0/issues/old/archive"],
      confirmed: () => json(200, old({ archived_at: LATER })),
    },
    "a drop": {
      send: () => fixture.actions!.dropIssue!("A", "old", () => ({ priority: "high" }), { group: "" }),
      endpoint: ["PATCH", "/api/v0/issues/old"],
      confirmed: () => json(200, old({ priority: "high" })),
    },
  };
/** Sends the write by the page's actions; nerve holds it. */
async function sent(write: string) {
  let promise!: Promise<unknown>;
  const call = await act(async () => {
    const at = tab.nerve.calls.length;
    promise = writes[write]!.send();
    await until(() => tab.nerve.calls.length > at, write);
    return tab.nerve.calls[at]!;
  });
  expect([call.method, call.path]).toEqual(writes[write]!.endpoint);
  return { call, result: track(promise), promise };
}
/** nerve answers the held write; the page has done all it does about it once the fake time has moved on. */
async function answer(call: { answer: (response: Response) => void }, response: Response) {
  await act(async () => {
    call.answer(response);
    await vi.advanceTimersByTimeAsync(100);
  });
}

it.each(Object.keys(writes))(
  "reads the cycle's progress again, once, when nerve confirms %s of the page's list",
  async (write) => {
    const { call, result, promise } = await sent(write);
    expect(progressReads()).toEqual([]);
    await answer(call, writes[write]!.confirmed());
    expect(result.settled).toBe(true);
    await promise;
    expect(progressReads()).toHaveLength(1);
    expect(progressReads()[0]).toMatchObject({ method: "GET" });
    await act(async () => progressReads()[0]!.answer(json(200, progressOf(cycle(), { total_issues: 7 }))));
    expect(tab.app.planning.cycles.getCycleProgress(C1)?.live.total_issues).toBe(7);
    expect(toasts).toEqual([]);
  }
);

it("gives the caller nerve's answer to the write as before", async () => {
  const { call, promise } = await sent("a creation");
  await answer(call, writes["a creation"]!.confirmed());
  await expect(promise).resolves.toMatchObject({ id: "made", cycle_id: C1 });
});

it("shows the newer of two reads of the progress, whatever order nerve answers them in", async () => {
  const first = await sent("a change");
  await answer(first.call, writes["a change"]!.confirmed());
  const second = await sent("a removal from the cycle");
  await answer(second.call, writes["a removal from the cycle"]!.confirmed());
  expect(progressReads()).toHaveLength(2);
  await act(async () => progressReads()[1]!.answer(json(200, progressOf(cycle(), { total_issues: 0 }))));
  await act(async () => progressReads()[0]!.answer(json(200, progressOf(cycle(), { total_issues: 1 }))));
  expect(tab.app.planning.cycles.getCycleProgress(C1)?.live.total_issues).toBe(0);
});

it("reads nothing for a write nerve refuses, whose refusal reaches its caller", async () => {
  const { call, promise } = await sent("a removal from the cycle");
  const refused = expect(promise).rejects.toMatchObject({ status: 409 });
  await answer(call, problem(409, "planning.cycles_disabled"));
  await refused;
  expect(progressReads()).toEqual([]);
  expect(tab.app.issue.cycleIssues.getIssueIds()).toEqual(["old"]);
});

it.each([
  "in another project",
  "while the cycle has ended",
  "while the cycle is archived",
  "while the project's cycles are off",
])("reads nothing for a creation that does not join the cycle: %s", async (creation) => {
  const other = projectOf("B", tab.workspace.id, { id: "B" });
  await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [tab.project, other]);
  if (creation === "while the cycle has ended")
    await loadCycle(tab.nerve, tab.app.planning.cycles, cycleOf(tab.project, "Now", { id: C1, status: "completed" }));
  // nerve archives only ended cycles: a status other than completed shows that being archived alone is enough
  if (creation === "while the cycle is archived")
    await loadCycle(
      tab.nerve,
      tab.app.planning.cycles,
      cycleOf(tab.project, "Now", { id: C1, status: "current", archived_at: LATER })
    );
  if (creation === "while the project's cycles are off")
    tab.app.projectRoot.project.confirmProject(tab.project, (project) => ({ ...project, cycle_view: false }));
  tab.nerve.calls.length = 0;
  const project = creation === "in another project" ? "B" : "A";
  let promise!: Promise<unknown>;
  const call = await act(async () => {
    promise = fixture.actions!.createIssue!(project, { name: "Elsewhere" });
    await until(() => tab.nerve.calls.length === 1, "the creation");
    return tab.nerve.calls[0]!;
  });
  expect([call.method, call.path, call.body]).toEqual([
    "POST",
    `/api/v0/projects/${project}/issues`,
    { name: "Elsewhere" },
  ]);
  await answer(call, json(201, { ...issueDetailOf("elsewhere", project), name: "Elsewhere" }));
  await promise;
  expect(progressReads()).toEqual([]);
});

for (const ending of ["another cycle and back", "another session", "a permission lost and regained", "unmount"]) {
  it(`reads nothing when nerve confirms a write after ${ending}`, async () => {
    const { call, promise } = await sent("a removal from the cycle");
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
    await answer(call, writes["a removal from the cycle"]!.confirmed());
    await promise.catch(() => undefined);
    expect(progressReads()).toEqual([]);
    expect(toasts).toEqual([]);
    if (ending === "unmount") view = createRoot(document.createElement("div"));
  });
}
