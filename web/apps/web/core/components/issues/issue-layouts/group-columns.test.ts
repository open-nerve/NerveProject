/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// The columns of a grouping by cycle and by module (P7b spec 2.4, R4): the project's cycles (modules) that are not
// archived in nerve's order, then the archived ones the answer has, named once the archive is read, then those the
// answer has that neither list does, "None" last; and the column of each that nerve refuses a work item's joining.
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { Cycle, Module } from "@nerve/api-client";
import type { IGroupByColumn } from "@nerve/types";
import { cycleOf, loadProjects, moduleOf, projectOf } from "@/store/project/fake-projects";
import { loadArchivedCycles, loadCycles, rowOf } from "@/store/planning/fake-cycles";
import { loadArchivedModules, loadModules, moduleRowOf } from "@/store/planning/fake-modules";
import { issueStoreTab } from "@/store/issue/fake-issues";
import { getGroupByColumns } from "./group-columns";

const bound = vi.hoisted(() => ({ app: undefined as unknown }));
vi.mock("@/lib/store-context", () => ({
  get rootStore() {
    return bound.app;
  },
}));
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "A" } }, publicClient: {} }));

let tab: Awaited<ReturnType<typeof issueStoreTab>>;
beforeEach(async () => {
  vi.useFakeTimers();
  tab = await issueStoreTab();
  bound.app = tab.app;
});
afterEach(() => {
  tab.app.dispose();
  vi.useRealTimers();
});

const archivedAt = "2026-10-20T09:00:00Z";
const columns = (groupBy: "cycle" | "module", groupIds: string[], isWorkspaceLevel = false) =>
  getGroupByColumns({ groupBy, includeNone: true, isWorkspaceLevel, groupIds, workspaceSlug: "ws", projectId: "A" });
const shown = (list: IGroupByColumn[] | undefined) =>
  list?.map(({ id, name, payload, archived, refusal }) => ({ id, name, payload, archived, refusal }));

function cycles() {
  const project = tab.project;
  return {
    sprint: cycleOf(project, "Sprint", { status: "current" }),
    next: cycleOf(project, "Next", { status: "upcoming" }),
    ended: cycleOf(project, "Ended", { status: "completed" }),
    old: cycleOf(project, "Old", { status: "completed", archived_at: archivedAt }),
    older: cycleOf(project, "Older", { status: "completed", archived_at: "2026-10-10T09:00:00Z" }),
  };
}
const cycleColumn = (cycle: Cycle, fields: Partial<IGroupByColumn> = {}) => ({
  id: cycle.id,
  name: cycle.name,
  payload: { cycle_id: cycle.id },
  archived: undefined,
  refusal: undefined,
  ...fields,
});
const unnamed = (id: string, payload: IGroupByColumn["payload"]) => ({
  id,
  name: "—",
  payload,
  archived: undefined,
  refusal: undefined,
});
const none = { id: "none", name: "None", payload: {}, archived: undefined, refusal: undefined };

it("orders a cycle's columns by the directory: the list, the answer's archived and unknown ones, None last", async () => {
  const { sprint, next, ended, old, older } = cycles();
  await loadCycles(
    tab.nerve,
    tab.app.planning.cycles,
    [sprint, next, ended].map((cycle) => rowOf(cycle)),
    tab.project
  );
  // nerve's groups by the text of their keys, "none" last: an id the web cannot name yet among them.
  const answer = [older.id, "X", old.id, sprint.id, "none"];
  expect(shown(columns("cycle", answer))).toEqual([
    cycleColumn(sprint),
    cycleColumn(next),
    cycleColumn(ended, { refusal: "planning.cycle_completed" }),
    unnamed(older.id, { cycle_id: older.id }),
    unnamed("X", { cycle_id: "X" }),
    unnamed(old.id, { cycle_id: old.id }),
    none,
  ]);
  // The archive, the latest archived first, names the archived ones and orders them; X, deleted meanwhile, after.
  await loadArchivedCycles(
    tab.nerve,
    tab.app.planning.cycles,
    [old, older].map((cycle) => rowOf(cycle)),
    tab.project
  );
  expect(shown(columns("cycle", answer))).toEqual([
    cycleColumn(sprint),
    cycleColumn(next),
    cycleColumn(ended, { refusal: "planning.cycle_completed" }),
    cycleColumn(old, { archived: true, refusal: "planning.archived" }),
    cycleColumn(older, { archived: true, refusal: "planning.archived" }),
    unnamed("X", { cycle_id: "X" }),
    none,
  ]);
  // An archived cycle the answer does not have has no column.
  expect(shown(columns("cycle", [sprint.id, "none"]))?.map(({ id }) => id)).toEqual([
    sprint.id,
    next.id,
    ended.id,
    "none",
  ]);
});

const moduleColumn = (item: Module, fields: Partial<IGroupByColumn> = {}) => ({
  id: item.id,
  name: item.name,
  payload: { module_ids: [item.id] },
  archived: undefined,
  refusal: undefined,
  ...fields,
});

it("orders a module's columns as a cycle's, whatever their status, an archived one refused", async () => {
  const project = tab.project;
  const alpha = moduleOf(project, "Alpha", { status: "completed" });
  const beta = moduleOf(project, "Beta", { status: "in-progress" });
  const gamma = moduleOf(project, "Gamma", { archived_at: archivedAt });
  await loadModules(
    tab.nerve,
    tab.app.planning.modules,
    [alpha, beta].map((item) => moduleRowOf(item)),
    project
  );
  const answer = [gamma.id, beta.id, "none"];
  expect(shown(columns("module", answer))).toEqual([
    moduleColumn(alpha),
    moduleColumn(beta),
    unnamed(gamma.id, { module_ids: [gamma.id] }),
    none,
  ]);
  await loadArchivedModules(tab.nerve, tab.app.planning.modules, [moduleRowOf(gamma)], project);
  expect(shown(columns("module", answer))).toEqual([
    moduleColumn(alpha),
    moduleColumn(beta),
    moduleColumn(gamma, { archived: true, refusal: "planning.archived" }),
    none,
  ]);
});

it("refuses every column with a value while the project has the domain turned off, None not", async () => {
  const { sprint, ended, old } = cycles();
  const alpha = moduleOf(tab.project, "Alpha");
  await loadCycles(
    tab.nerve,
    tab.app.planning.cycles,
    [sprint, ended].map((cycle) => rowOf(cycle)),
    tab.project
  );
  await loadArchivedCycles(tab.nerve, tab.app.planning.cycles, [rowOf(old)], tab.project);
  await loadModules(tab.nerve, tab.app.planning.modules, [moduleRowOf(alpha)], tab.project);
  await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [
    projectOf("A", tab.workspace.id, { id: "A", cycle_view: false, module_view: false }),
  ]);
  expect(shown(columns("cycle", [old.id, "none"]))).toEqual([
    cycleColumn(sprint, { refusal: "planning.cycles_disabled" }),
    cycleColumn(ended, { refusal: "planning.cycles_disabled" }),
    cycleColumn(old, { archived: true, refusal: "planning.cycles_disabled" }),
    none,
  ]);
  expect(shown(columns("module", ["none"]))).toEqual([
    moduleColumn(alpha, { refusal: "planning.modules_disabled" }),
    none,
  ]);
});

it("gives no cycle or module columns to a list of several projects", async () => {
  expect(columns("cycle", ["none"], true)).toBeUndefined();
  expect(columns("module", ["none"], true)).toBeUndefined();
});
