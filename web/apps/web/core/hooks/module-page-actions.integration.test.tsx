// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// A module's page's list commands (P6b spec 2.5, 2.6, 2.7, R5, R6, R9, R13): they act for the module the route's params
// name, never the router store's. A creation in the module's project joins the module in its one request while the page
// shows it taking work items (not archived, the project's modules on, whatever its status); otherwise it goes as the
// project's page sends it. The removal from the module sends the modules the work item holds but the page's. A write of
// the page's list reads the module's progress again once nerve confirms it, in the session and page it was sent from;
// a refused one reads nothing. The stores and HTTP are real; the route's params are the test's.
import { act } from "react";
import { createRoot } from "react-dom/client";
import { runInAction } from "mobx";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { Issue, IssueDetail, ModulePreferences } from "@nerve/api-client";
import { EIssueFilterType } from "@nerve/constants";
import { EIssuesStoreType } from "@nerve/types";
import { answered, json, noContent, problem } from "@/lib/auth/fake-nerve";
import { track, until } from "@/lib/auth/fake-time";
import { toasts } from "@/lib/fake-toast";
import { StoreContext } from "@/lib/store-context";
import { issueDetailOf, issueOf, issuePageOf, issueStoreTab, requestedIssue } from "@/store/issue/fake-issues";
import { loadModule, moduleProgressOf, moduleRequests } from "@/store/planning/fake-modules";
import { loadProjects, moduleOf, preferencesOf, projectOf } from "@/store/project/fake-projects";
import { useIssuesActions, type IssueActions } from "./use-issues-actions";

/** The page's module, another one, and an archived one the work item is in too. */
const M1 = "019a0000-0000-7000-8000-0000000000d1";
const M2 = "019a0000-0000-7000-8000-0000000000d2";
const M3 = "019a0000-0000-7000-8000-0000000000d3";
const LATER = "2026-10-29T00:00:00Z";
const fixture = vi.hoisted(() => ({
  app: undefined as unknown,
  session: { loginId: "one" },
  params: {} as Record<string, string>,
  storeType: undefined as EIssuesStoreType.MODULE | EIssuesStoreType.CYCLE | undefined,
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
  fixture.actions = useIssuesActions(fixture.storeType!);
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
const page = (id = M1) => ({ projectID: "A", entityID: id });
const moduleNamed = (id: string, fields: object = {}) => moduleOf(tab.project, id, { id, ...fields });
const progressReads = (id = M1) => tab.nerve.to(moduleRequests.progress({ id }));
/** The page's work item "old", in the page's module, another one and an archived one. */
const held: Issue = { ...issueOf("old"), module_ids: [M1, M2, M3] };
/** "old" as nerve answers a write of it with fields. */
const old = (fields: Partial<IssueDetail> = {}): IssueDetail => ({
  ...issueDetailOf("old"),
  module_ids: [M1, M2, M3],
  updated_at: LATER,
  ...fields,
});
function modulePreferencesOf(): ModulePreferences {
  const { display_filters, display_properties } = preferencesOf();
  return { filters: {}, display_filters, display_properties };
}
beforeEach(async () => {
  vi.useFakeTimers();
  tab = await issueStoreTab();
  fixture.app = tab.app;
  fixture.session.loginId = "one";
  fixture.params = { workspaceSlug: "ws", projectId: "A", moduleId: M1 };
  fixture.storeType = EIssuesStoreType.MODULE;
  toasts.length = 0;
  runInAction(() => tab.app.router.setQuery({ workspaceSlug: "ws", projectId: "A", moduleId: "router-module" }));
  await loadModule(tab.nerve, tab.app.planning.modules, moduleNamed(M1));
  await answered(
    tab.nerve,
    () => tab.app.issue.moduleIssuesFilter.fetchFilters("ws", M1),
    ["GET", `/api/v0/me/modules/${M1}/preferences`],
    modulePreferencesOf(),
    "the module page's preferences"
  );
  await answered(
    tab.nerve,
    () => tab.app.issue.moduleIssues.fetchIssues("ws", page(), "init-loader", { limit: 10 }),
    ["GET", "/api/v0/projects/A/issues"],
    issuePageOf([held]),
    "the module's list"
  );
  tab.nerve.calls.length = 0;
  view = createRoot(document.createElement("div"));
  await render();
});
afterEach(async () => {
  await act(async () => view.unmount());
  tab.app.dispose();
  vi.useRealTimers();
});

/** Each write the module page's list sends by its actions, the request it is, and nerve's confirmation of it. */
const writes: Record<
  string,
  { send: () => Promise<unknown>; endpoint: [string, string]; body?: object; confirmed: () => Response }
> = {
  "a creation": {
    send: () => fixture.actions!.createIssue!("A", { name: "Made" }),
    endpoint: ["POST", "/api/v0/projects/A/issues"],
    body: { name: "Made", module_ids: [M1] },
    confirmed: () => json(201, { ...issueDetailOf("made"), name: "Made", module_ids: [M1] }),
  },
  "a change": {
    send: () => fixture.actions!.updateIssue!("A", "old", { priority: "high" }),
    endpoint: ["PATCH", "/api/v0/issues/old"],
    body: { priority: "high" },
    confirmed: () => json(200, old({ priority: "high" })),
  },
  "a removal from the module": {
    send: () => fixture.actions!.removeIssueFromView!("A", "old"),
    endpoint: ["PATCH", "/api/v0/issues/old"],
    body: { module_ids: [M2, M3] },
    confirmed: () => json(200, old({ module_ids: [M2, M3] })),
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

it("lists, creates and keeps preferences for the route's module, never the router store's", async () => {
  const list = await requestedIssue(tab.nerve, () => fixture.actions!.fetchIssues("init-loader", { limit: 1 }), [
    "GET",
    "/api/v0/projects/A/issues",
  ]);
  expect(list.call.query.module_id).toBe(M1);
  list.call.answer(json(200, { ...issuePageOf([held], "next"), total: 2 }));
  await list.promise;
  fixture.params = { workspaceSlug: "ws", projectId: "A", moduleId: M2 };
  await render();
  // The first module's list is still the store's current one; the next page belongs to the module left.
  await expect(fixture.actions!.fetchNextIssues()).resolves.toBeUndefined();
  expect(tab.nerve.calls).toHaveLength(1);
  await loadModule(tab.nerve, tab.app.planning.modules, moduleNamed(M2));
  const created = await requestedIssue(tab.nerve, () => fixture.actions!.createIssue!("A", { name: "New" }), [
    "POST",
    "/api/v0/projects/A/issues",
  ]);
  expect(created.call.body).toEqual({ name: "New", module_ids: [M2] });
  created.call.answer(json(201, { ...issueDetailOf("new"), module_ids: [M2] }));
  await created.promise;
  await until(() => progressReads(M2).length === 1, "M2's progress");
  progressReads(M2)[0]!.answer(json(200, moduleProgressOf()));
  // The change is followed as the project page's: the store's list being the module left's, not the page's, the route
  // module's list is read again with the pages it has.
  const kept = fixture.actions!.updateFilters("A", EIssueFilterType.KANBAN_FILTERS, {
    group_by: ["done"],
    sub_group_by: [],
  });
  expect(tab.app.issue.moduleIssuesFilter.filters[M2]?.kanbanFilters?.group_by).toEqual(["done"]);
  expect(tab.app.issue.moduleIssuesFilter.filters["router-module"]).toBeUndefined();
  expect(tab.app.issue.projectIssuesFilter.filters.A).toBeUndefined();
  expect(tab.app.issue.cycleIssuesFilter.filters[M2]).toBeUndefined();
  const followed = await requestedIssue(tab.nerve, () => kept, ["GET", "/api/v0/projects/A/issues"]);
  expect(followed.call.query.module_id).toBe(M2);
  followed.call.answer(json(200, issuePageOf([])));
  await followed.promise;
  expect(progressReads(M1)).toEqual([]);
});

it.each(Object.keys(writes))(
  "reads the module's progress again, once, when nerve confirms %s of the page's list",
  async (write) => {
    const { call, result, promise } = await sent(write);
    if (writes[write]!.body) expect(call.body).toEqual(writes[write]!.body);
    expect(progressReads()).toEqual([]);
    await answer(call, writes[write]!.confirmed());
    expect(result.settled).toBe(true);
    await promise;
    expect(progressReads()).toHaveLength(1);
    expect(progressReads()[0]).toMatchObject({ method: "GET" });
    await act(async () => progressReads()[0]!.answer(json(200, moduleProgressOf({ total_issues: 7 }))));
    expect(tab.app.planning.modules.getModuleProgress(M1)?.total_issues).toBe(7);
    expect(toasts).toEqual([]);
  }
);

it("takes the work item off the page once nerve confirms its removal, keeping its other modules", async () => {
  const { call, promise } = await sent("a removal from the module");
  await answer(call, writes["a removal from the module"]!.confirmed());
  await promise;
  expect(tab.app.issue.moduleIssues.getIssueIds()).toEqual([]);
  expect(tab.app.issue.issues.getIssueById("old")?.module_ids).toEqual([M2, M3]);
});

it("reads nothing for a write nerve refuses, whose refusal reaches its caller", async () => {
  const { call, promise } = await sent("a removal from the module");
  const refused = expect(promise).rejects.toMatchObject({ status: 409 });
  await answer(call, problem(409, "planning.modules_disabled"));
  await refused;
  expect(progressReads()).toEqual([]);
  expect(tab.app.issue.moduleIssues.getIssueIds()).toEqual(["old"]);
});

it.each(["completed", "cancelled"] as const)("joins a %s module, its status no matter", async (status) => {
  await loadModule(tab.nerve, tab.app.planning.modules, moduleNamed(M1, { status, updated_at: LATER }));
  tab.nerve.calls.length = 0;
  const { call, promise } = await sent("a creation");
  expect(call.body).toEqual({ name: "Made", module_ids: [M1] });
  await answer(call, writes["a creation"]!.confirmed());
  await promise;
  expect(progressReads()).toHaveLength(1);
});

it.each([
  "in another project",
  "while the module is archived",
  "while the project's modules are off",
  "before the page has the module",
])(
  "creates as the project's page does, reading nothing, for a creation that does not join the module: %s",
  async (creation) => {
    const other = projectOf("B", tab.workspace.id, { id: "B" });
    await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [tab.project, other]);
    if (creation === "while the module is archived")
      await loadModule(
        tab.nerve,
        tab.app.planning.modules,
        moduleNamed(M1, { status: "in-progress", archived_at: LATER, updated_at: LATER })
      );
    if (creation === "while the project's modules are off")
      tab.app.projectRoot.project.confirmProject(tab.project, (project) => ({ ...project, module_view: false }));
    if (creation === "before the page has the module") {
      fixture.params = { workspaceSlug: "ws", projectId: "A", moduleId: M2 };
      await render();
    }
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
    expect(progressReads(M2)).toEqual([]);
  }
);

it("reads nothing when nerve confirms a write after another module and back", async () => {
  const { call, promise } = await sent("a removal from the module");
  fixture.params = { workspaceSlug: "ws", projectId: "A", moduleId: M2 };
  await render();
  fixture.params = { workspaceSlug: "ws", projectId: "A", moduleId: M1 };
  await render();
  await answer(call, writes["a removal from the module"]!.confirmed());
  await promise;
  expect(progressReads()).toEqual([]);
  expect(progressReads(M2)).toEqual([]);
});

it("sends no removal of a work item of another project from the page's module", async () => {
  await expect(fixture.actions!.removeIssueFromView!("B", "old")).resolves.toBeUndefined();
  expect(tab.nerve.calls).toEqual([]);
});

// The per-domain removal is the cycle's too (P6b spec 2.2): its page's list takes a work item out of its cycle as a
// change to no cycle, and reads the cycle's progress once nerve confirms it.
it("takes a work item out of a cycle's page as a change to no cycle, then reads the cycle's progress", async () => {
  const C1 = "019a0000-0000-7000-8000-0000000000c1";
  await answered(
    tab.nerve,
    () => tab.app.issue.cycleIssues.fetchIssues("ws", page(C1), "init-loader", { limit: 10 }),
    ["GET", "/api/v0/projects/A/issues"],
    issuePageOf([{ ...issueOf("now"), cycle_id: C1 }]),
    "the cycle's list"
  );
  tab.nerve.calls.length = 0;
  fixture.params = { workspaceSlug: "ws", projectId: "A", cycleId: C1 };
  fixture.storeType = EIssuesStoreType.CYCLE;
  await render();
  const removal = await requestedIssue(tab.nerve, () => fixture.actions!.removeIssueFromView!("A", "now"), [
    "PATCH",
    "/api/v0/issues/now",
  ]);
  expect(removal.call.body).toEqual({ cycle_id: null });
  await answer(removal.call, json(200, { ...issueDetailOf("now"), cycle_id: null, updated_at: LATER }));
  await removal.promise;
  expect(tab.app.issue.cycleIssues.getIssueIds()).toEqual([]);
  expect(tab.nerve.to(`/api/v0/cycles/${C1}/progress`)).toHaveLength(1);
  expect(progressReads()).toEqual([]);
});
