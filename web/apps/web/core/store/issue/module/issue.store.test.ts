/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// The module page's list (P6b spec 2.3, 2.5, R5, R6): its project's list under the page's own preferences and the
// module; a creation there joins the module in its one request, and a removal sends the modules the work item holds
// but the page's, archived ones too, worked out when its turn comes.
import { runInAction } from "mobx";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { Issue, IssueDetail, ModulePreferences } from "@nerve/api-client";
import { answered, json, problem } from "@/lib/auth/fake-nerve";
import { track, until } from "@/lib/auth/fake-time";
import { loadArchivedModules, loadModules, moduleRowOf } from "@/store/planning/fake-modules";
import { loadProjects, moduleOf, preferencesOf, projectOf } from "@/store/project/fake-projects";
import { issueDetailOf, issueOf, issuePageOf, issueStoreTab, requestedIssue } from "../fake-issues";

vi.stubGlobal("localStorage", { getItem: () => null, setItem: () => {}, removeItem: () => {} });
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: {}, publicClient: {} }));
const binding = vi.hoisted(() => ({ app: undefined as unknown }));
vi.mock("@/lib/store-context", () => ({
  get rootStore() {
    return binding.app;
  },
}));
/** The page's module, another module, and an archived module of the project. */
const MA = "019a0000-0000-7000-8000-0000000000d1";
const MB = "019a0000-0000-7000-8000-0000000000d2";
const MC = "019a0000-0000-7000-8000-0000000000d3";
const LATER = "2026-10-29T00:00:00Z";
const pageOf = (moduleID: string) => ({ projectID: "A", entityID: moduleID });
let tab: Awaited<ReturnType<typeof issueStoreTab>>;
beforeEach(async () => {
  vi.useFakeTimers();
  tab = await issueStoreTab();
  binding.app = tab.app;
  // The project page's own preferences differ, so a query that borrowed them would show it.
  runInAction(() => {
    tab.app.issue.projectIssuesFilter.filters.A = {
      richFilters: { and: [{ priority__in: "low" }] },
      displayProperties: undefined,
      kanbanFilters: undefined,
      displayFilters: { layout: "list", group_by: "state", order_by: "-updated_at" },
    };
  });
});
afterEach(() => {
  tab.app.dispose();
  vi.useRealTimers();
});

/** A module page's preferences as nerve answers them; its defaults are the project page's list defaults. */
function modulePreferencesOf(fields: Partial<ModulePreferences> = {}): ModulePreferences {
  const { display_filters, display_properties } = preferencesOf();
  return { filters: {}, display_filters, display_properties, ...fields };
}
const highFirst = (moduleID: string) =>
  answered(
    tab.nerve,
    () => tab.app.issue.moduleIssuesFilter.fetchFilters("ws", moduleID),
    ["GET", `/api/v0/me/modules/${moduleID}/preferences`],
    modulePreferencesOf({
      filters: { priority: ["high"] },
      display_filters: { ...modulePreferencesOf().display_filters, order_by: "sort_order" },
    }),
    "module preferences"
  );
const inModules = (id: string, ...module_ids: string[]): Issue => ({ ...issueOf(id), module_ids });
const answerOf = (id: string, ...module_ids: string[]): IssueDetail => ({
  ...issueDetailOf(id),
  module_ids,
  updated_at: LATER,
});
/** The ids the module page lists, in id order: the tests here are about which work items, not their order. */
const ids = () => tab.app.issue.moduleIssues.getIssueIds()?.toSorted();
/** Reads the module page's list and gives its binding, which stays registered after another becomes current. */
async function listed(moduleID: string, rows: Issue[]) {
  const store = tab.app.issue.moduleIssues;
  await answered(
    tab.nerve,
    () => store.fetchIssues("ws", pageOf(moduleID), "init-loader", { limit: 20 }),
    ["GET", "/api/v0/projects/A/issues"],
    issuePageOf(rows),
    "the module's list"
  );
  return store.currentList!;
}
const rowsOf = (list: Awaited<ReturnType<typeof listed>>) =>
  list
    .group("")
    ?.rows.map((row) => row.id)
    .toSorted();

it("lists the module as its project's list under the module page's preferences, never the project page's", async () => {
  const { app, nerve } = tab;
  await highFirst(MA);
  const store = app.issue.moduleIssues;
  const first = await requestedIssue(nerve, () => store.fetchIssues("ws", pageOf(MA), "init-loader", { limit: 20 }), [
    "GET",
    "/api/v0/projects/A/issues",
  ]);
  expect(first.call.query).toEqual({
    priority: "high",
    order_by: "sort_order",
    sub_issue: "true",
    limit: "20",
    module_id: MA,
  });
  expect(store.currentList?.scope).toMatchObject({
    scope: "project",
    workspaceSlug: "ws",
    projectID: "A",
    query: { priority: ["high"], module_id: [MA] },
  });
  first.call.answer(json(200, issuePageOf([inModules("a", MA), inModules("b", MA, MB)])));
  await first.promise;
  expect(ids()).toEqual(["a", "b"]);
  expect(store.total).toBe(2);
  expect(app.issue.projectIssues.currentList).toBeUndefined();
  expect(app.issue.cycleIssues.currentList).toBeUndefined();
});

it("keeps one binding per module, so a late answer for the module left does not enter the module shown", async () => {
  const { app, nerve } = tab;
  const store = app.issue.moduleIssues;
  const one = await requestedIssue(nerve, () => store.fetchIssues("ws", pageOf(MA), "init-loader", { limit: 20 }), [
    "GET",
    "/api/v0/projects/A/issues",
  ]);
  const listOne = store.currentList!;
  const two = await requestedIssue(nerve, () => store.fetchIssues("ws", pageOf(MB), "init-loader", { limit: 20 }), [
    "GET",
    "/api/v0/projects/A/issues",
  ]);
  expect(two.call.query.module_id).toBe(MB);
  expect(store.currentList).not.toBe(listOne);
  two.call.answer(json(200, issuePageOf([inModules("b", MB)])));
  await two.promise;
  one.call.answer(json(200, issuePageOf([inModules("a", MA)])));
  await one.promise;
  expect(ids()).toEqual(["b"]);
  expect(store.isPreferenceQueryCurrent(pageOf(MB))).toBe(true);
  expect(store.isPreferenceQueryCurrent(pageOf(MA))).toBe(false);
  await answered(
    nerve,
    () => store.fetchIssues("ws", pageOf(MA), "init-loader", { limit: 20 }),
    ["GET", "/api/v0/projects/A/issues"],
    issuePageOf([inModules("a", MA)]),
    "back to the first module"
  );
  expect(store.currentList).toBe(listOne);
  expect(ids()).toEqual(["a"]);
});

it("counts a module page's work items as nerve does, only while the current list is that page's", async () => {
  const { nerve } = tab;
  const store = tab.app.issue.moduleIssues;
  expect(store.pageTotal(pageOf(MA)), "nothing read").toBeUndefined();
  await answered(
    nerve,
    () => store.fetchIssues("ws", pageOf(MA), "init-loader", { limit: 20 }),
    ["GET", "/api/v0/projects/A/issues"],
    { ...issuePageOf([inModules("a", MA)]), total: 7 },
    "the first module's list"
  );
  expect(store.pageTotal(pageOf(MA)), "nerve's count, though it sent one row").toBe(7);
  expect(store.pageTotal(pageOf(MB)), "another module's page").toBeUndefined();
  expect(store.pageTotal({ projectID: "B", entityID: MA }), "the module under another project").toBeUndefined();
  const two = await requestedIssue(nerve, () => store.fetchIssues("ws", pageOf(MB), "init-loader", { limit: 20 }), [
    "GET",
    "/api/v0/projects/A/issues",
  ]);
  expect(store.pageTotal(pageOf(MA)), "the first module's, its list no longer current").toBeUndefined();
  expect(store.pageTotal(pageOf(MB)), "the second module's, not answered yet").toBeUndefined();
  two.call.answer(json(200, { ...issuePageOf([inModules("b", MB)]), total: 1 }));
  await two.promise;
  expect(store.pageTotal(pageOf(MB))).toBe(1);
});

it("reads the next page only for the module page whose list is current", async () => {
  const { app, nerve } = tab;
  const store = app.issue.moduleIssues;
  await answered(
    nerve,
    () => store.fetchIssues("ws", pageOf(MA), "init-loader", { limit: 1 }),
    ["GET", "/api/v0/projects/A/issues"],
    { ...issuePageOf([inModules("a", MA)], "next-a"), total: 2 },
    "first page"
  );
  nerve.calls.length = 0;
  await expect(store.fetchNextIssues("ws", pageOf(MB))).resolves.toBeUndefined();
  await expect(store.fetchNextIssues("ws", { projectID: "B", entityID: MA })).resolves.toBeUndefined();
  expect(nerve.calls).toEqual([]);
  const next = await requestedIssue(nerve, () => store.fetchNextIssues("ws", pageOf(MA)), [
    "GET",
    "/api/v0/projects/A/issues",
  ]);
  expect(next.call.query).toMatchObject({ module_id: MA, cursor: "next-a" });
  next.call.answer(json(200, { ...issuePageOf([inModules("b", MA)]), total: 2 }));
  await next.promise;
  expect(ids()).toEqual(["a", "b"]);
});

it("knows when a confirmed change of the module page's preferences leaves its list behind, and reads it again", async () => {
  const { app, nerve } = tab;
  await highFirst(MA);
  const store = app.issue.moduleIssues;
  await listed(MA, [inModules("a", MA)]);
  expect(store.isPreferenceQueryCurrent(pageOf(MA))).toBe(true);
  await answered(
    nerve,
    () => app.issue.moduleIssuesFilter.updateFilterExpression("ws", MA, { and: [{ priority__in: "urgent" }] }),
    ["PATCH", `/api/v0/me/modules/${MA}/preferences`],
    modulePreferencesOf({ filters: { priority: ["urgent"] } }),
    "filters"
  );
  expect(store.isPreferenceQueryCurrent(pageOf(MA))).toBe(false);
  const again = await requestedIssue(nerve, () => store.fetchIssuesWithExistingPagination("ws", pageOf(MA)), [
    "GET",
    "/api/v0/projects/A/issues",
  ]);
  expect(again.call.query).toMatchObject({ priority: "urgent", limit: "20", module_id: MA });
  again.call.answer(json(200, issuePageOf([])));
  await again.promise;
  expect(store.isPreferenceQueryCurrent(pageOf(MA))).toBe(true);
});

it("creates in the module's project with the module joined to any modules given, in the one request", async () => {
  const { app, nerve } = tab;
  await loadProjects(nerve, app.projectRoot.project, tab.workspace, [
    tab.project,
    projectOf("B", tab.workspace.id, { id: "B" }),
  ]);
  nerve.calls.length = 0;
  const store = app.issue.moduleIssues;
  await listed(MA, [inModules("a", MA)]);
  const here = await requestedIssue(nerve, () => store.createIssue("ws", "A", { name: "Here" }, pageOf(MA)), [
    "POST",
    "/api/v0/projects/A/issues",
  ]);
  expect(here.call.body).toEqual({ name: "Here", module_ids: [MA] });
  here.call.answer(json(201, { ...answerOf("here", MA), sort_order: 10 }));
  await here.promise;
  expect(ids()).toEqual(["a", "here"]);
  const joined = await requestedIssue(
    nerve,
    () => store.createIssue("ws", "A", { name: "Joined", module_ids: [MB] }, pageOf(MA)),
    ["POST", "/api/v0/projects/A/issues"]
  );
  expect(joined.call.body, "the modules given, and the page's").toEqual({ name: "Joined", module_ids: [MB, MA] });
  joined.call.answer(json(201, { ...answerOf("joined", MA, MB), sort_order: 20 }));
  await joined.promise;
  const once = await requestedIssue(
    nerve,
    () => store.createIssue("ws", "A", { name: "Once", module_ids: [MA, MB] }, pageOf(MA)),
    ["POST", "/api/v0/projects/A/issues"]
  );
  expect(once.call.body, "the page's module named once").toEqual({ name: "Once", module_ids: [MA, MB] });
  once.call.answer(json(201, { ...answerOf("once", MA, MB), sort_order: 30 }));
  await once.promise;
  expect(ids()).toEqual(["a", "here", "joined", "once"]);
  expect(nerve.calls.filter((call) => call.method === "POST")).toHaveLength(3);
});

it("creates elsewhere, or without the page, with the body as given", async () => {
  const { app, nerve } = tab;
  await loadProjects(nerve, app.projectRoot.project, tab.workspace, [
    tab.project,
    projectOf("B", tab.workspace.id, { id: "B" }),
  ]);
  nerve.calls.length = 0;
  const store = app.issue.moduleIssues;
  await listed(MA, [inModules("a", MA)]);
  const elsewhere = await requestedIssue(nerve, () => store.createIssue("ws", "B", { name: "Elsewhere" }, pageOf(MA)), [
    "POST",
    "/api/v0/projects/B/issues",
  ]);
  expect(elsewhere.call.body).toEqual({ name: "Elsewhere" });
  elsewhere.call.answer(json(201, issueDetailOf("elsewhere", "B")));
  await elsewhere.promise;
  const plain = await requestedIssue(nerve, () => store.createIssue("ws", "A", { name: "Plain", module_ids: [MB] }), [
    "POST",
    "/api/v0/projects/A/issues",
  ]);
  expect(plain.call.body).toEqual({ name: "Plain", module_ids: [MB] });
  plain.call.answer(json(201, { ...answerOf("plain", MB), sort_order: 10 }));
  await plain.promise;
  expect(ids()).toEqual(["a"]);
  expect(nerve.calls).toHaveLength(3);
});

/** The project's modules as the page's caller knows them: two open, and one archived. */
async function modulesKnown() {
  const { app, nerve, project } = tab;
  const modules = app.planning.modules;
  await loadModules(
    nerve,
    modules,
    [moduleRowOf(moduleOf(project, "alpha", { id: MA })), moduleRowOf(moduleOf(project, "beta", { id: MB }))],
    project
  );
  await loadArchivedModules(
    nerve,
    modules,
    [moduleRowOf(moduleOf(project, "gamma", { id: MC, archived_at: "2026-10-20T00:00:00Z" }))],
    project
  );
  expect(modules.getArchivedModuleIds("A")).toEqual([MC]);
  nerve.calls.length = 0;
}

it("removes a work item from the page's module with the modules it holds but that one, worked out in its turn", async () => {
  const { app, nerve } = tab;
  await modulesKnown();
  const store = app.issue.moduleIssues;
  const pageB = await listed(MB, [inModules("x", MA, MB, MC)]);
  const pageA = await listed(MA, [inModules("x", MA, MB, MC), inModules("y", MA)]);
  nerve.calls.length = 0;
  const fromA = await requestedIssue(nerve, () => store.removeIssueFromModule("ws", pageOf(MA), "x"), [
    "PATCH",
    "/api/v0/issues/x",
  ]);
  expect(fromA.call.body, "the other module and the archived one stay").toEqual({ module_ids: [MB, MC] });
  // A second removal, from the other module's page, waits for the first: its body is worked out from nerve's answer.
  const fromB = track(store.removeIssueFromModule("ws", pageOf(MB), "x"));
  await vi.advanceTimersByTimeAsync(1_000);
  expect(nerve.calls).toHaveLength(1);
  fromA.call.answer(json(200, answerOf("x", MB, MC)));
  await fromA.promise;
  await until(() => nerve.calls.length === 2, "the second removal");
  expect(nerve.calls[1]).toMatchObject({ method: "PATCH", path: "/api/v0/issues/x", body: { module_ids: [MC] } });
  expect(rowsOf(pageA), "left the page's module once nerve confirmed it").toEqual(["y"]);
  expect(rowsOf(pageB), "still in the other module").toEqual(["x"]);
  nerve.calls[1]?.answer(json(200, { ...answerOf("x", MC), updated_at: "2026-10-30T00:00:00Z" }));
  await until(() => fromB.settled, "the second removal");
  expect(fromB.error).toBeUndefined();
  expect(rowsOf(pageB)).toEqual([]);
  expect(app.issue.issues.getIssueById("x")?.module_ids).toEqual([MC]);
});

it("keeps the modules a work item holds when nerve refuses its removal, and hands the refusal to the caller", async () => {
  const { app, nerve } = tab;
  await modulesKnown();
  const store = app.issue.moduleIssues;
  const page = await listed(MA, [inModules("x", MA, MB, MC)]);
  nerve.calls.length = 0;
  const refused = await requestedIssue(nerve, () => store.removeIssueFromModule("ws", pageOf(MA), "x"), [
    "PATCH",
    "/api/v0/issues/x",
  ]);
  expect(refused.call.body).toEqual({ module_ids: [MB, MC] });
  refused.call.answer(problem(409, "planning.modules_disabled"));
  await expect(refused.promise).rejects.toMatchObject({ problem: { code: "planning.modules_disabled" } });
  expect(app.issue.issues.getIssueById("x")?.module_ids).toEqual([MA, MB, MC]);
  expect(rowsOf(page)).toEqual(["x"]);
  expect(nerve.calls).toHaveLength(1);
});
