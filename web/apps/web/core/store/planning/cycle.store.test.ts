/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { Cycle, CycleProgress } from "@nerve/api-client";
import { ApiError } from "@/lib/api-error";
import { FakeNerve, json, noContent, problem } from "@/lib/auth/fake-nerve";
import { settle, track, until } from "@/lib/auth/fake-time";
import { listTab, ops, web } from "@/store/fake-project-lists";
import { sent } from "@/store/fake-queue";
import { CycleStore } from "@/store/planning/cycle.store";
import {
  cycleRequests,
  loadArchivedCycles,
  loadCycle,
  loadCycles,
  loadProgress,
  progressOf,
  rowOf,
} from "@/store/planning/fake-cycles";
import { cycleOf } from "@/store/project/fake-projects";

// The cycles of the caller's projects (M6 design 3.1–3.8, P5a spec 2.2), against a fake nerve that answers each
// request when the test says. The tab's address is web's, a project of acme; ops is acme's other; his other
// workspace, beta, has lab. The store's readers take the project from their caller (the route's), not the address.

const ended = cycleOf(web, "S1", { status: "completed", start_date: "2026-10-26", end_date: "2026-11-01" });
const current = cycleOf(web, "S2", {
  status: "current",
  start_date: "2026-11-09",
  end_date: "2026-11-15",
  sort_order: 55535,
});
const upcoming = cycleOf(web, "S3", {
  status: "upcoming",
  start_date: "2026-11-23",
  end_date: "2026-11-29",
  sort_order: 45535,
});
const ideas = cycleOf(web, "Ideas", { sort_order: 35535 });
const shelved = cycleOf(web, "S0", {
  status: "completed",
  start_date: "2026-10-12",
  end_date: "2026-10-18",
  archived_at: "2026-10-20T09:00:00Z",
});
const opsCycle = cycleOf(ops, "O1", { status: "current", start_date: "2026-11-09", end_date: "2026-11-15" });
const opsShelved = cycleOf(ops, "O0", { status: "completed", archived_at: "2026-10-20T09:00:00Z" });
/** A time after the fixtures', so that a later answer of nerve can be told from an earlier one. */
const LATER = "2026-11-10T09:00:00Z";
/** web's cycles that are not archived, in nerve's order: by sort order, the lowest first (not by name or status). */
const webRows = [
  rowOf(ideas),
  rowOf(upcoming),
  rowOf(current, { total_issues: 3, started_issues: 2, completed_issues: 1, assignee_ids: ["u-ann"] }),
  rowOf(ended, {
    total_issues: 2,
    completed_issues: 2,
    progress_snapshot: {
      total_issues: 2,
      backlog_issues: 0,
      unstarted_issues: 0,
      started_issues: 0,
      completed_issues: 2,
      cancelled_issues: 0,
      frozen_at: "2026-11-02T00:00:00Z",
    },
  }),
];
const ids = (cycles: Pick<Cycle, "id">[]) => cycles.map((cycle) => cycle.id);

beforeEach(() => {
  vi.useFakeTimers({ now: 1_000_000 });
});
afterEach(() => {
  vi.useRealTimers();
});

describe("CycleStore, the reads", () => {
  it("keeps a project's cycles, its archived ones, a cycle's read and its progress, each by its project or cycle", async () => {
    const { nerve, store } = await listTab(CycleStore);
    const listed = await loadCycles(nerve, store, webRows);
    expect(nerve.calls[0]?.query).toEqual({});
    expect(listed.value).toEqual(webRows);
    expect(store.getProjectCycleIds(web.id)).toEqual(ids(webRows));
    expect(store.getCycleRow(current.id)).toEqual(webRows[2]);
    expect(store.getCycleById(ended.id)).toEqual(webRows[3]);
    expect(store.getArchivedCycleIds(web.id)).toBeUndefined();
    expect(store.getProjectCycleIds(ops.id)).toBeUndefined();

    const archivedRows = [rowOf(shelved, { total_issues: 4, completed_issues: 4 })];
    await loadArchivedCycles(nerve, store, archivedRows);
    expect(nerve.calls[1]?.query).toEqual({ archived: "true" });
    expect(store.getArchivedCycleIds(web.id)).toEqual([shelved.id]);
    expect(store.getCycleRow(shelved.id)).toEqual(archivedRows[0]);
    expect(store.getProjectCycleIds(web.id)).toEqual(ids(webRows));

    const read = await loadCycle(nerve, store, shelved);
    expect(read.value).toEqual(shelved);
    expect(store.getCycleById(shelved.id)).toEqual(shelved);
    const progress = progressOf(current, { total_issues: 3, started_issues: 2, completed_issues: 1 });
    await loadProgress(nerve, store, current, progress);
    expect(store.getCycleProgress(current.id)).toEqual(progress);
    expect(store.getCycleProgress(ended.id)).toBeUndefined();

    const opsRows = [rowOf(opsCycle)];
    await loadCycles(nerve, store, opsRows, ops);
    expect(store.getProjectCycleIds(ops.id)).toEqual([opsCycle.id]);
    expect(store.getProjectCycleIds(web.id)).toEqual(ids(webRows));
  });

  it("shows a cycle's fields as nerve last answered them, its read's or its list's, and the counts of its list's row", async () => {
    const { nerve, store } = await listTab(CycleStore);
    const row = rowOf(current, { total_issues: 5, started_issues: 3, completed_issues: 2, assignee_ids: ["u-ann"] });
    await loadCycles(nerve, store, [row]);
    // another tab renames it, and its read, answered after the list, shows the name; the counts are the row's
    const renamed: Cycle = { ...current, name: "Sprint two", updated_at: LATER };
    await loadCycle(nerve, store, renamed);
    expect(store.getCycleById(current.id)).toEqual(renamed);
    expect(store.getCycleRow(current.id)).toEqual({ ...row, name: "Sprint two", updated_at: LATER });
    // the list, answered after the read, shows the cycle as it has it, counts and all
    const relisted = rowOf({ ...current, name: "Sprint 2b" }, { total_issues: 6, started_issues: 4 });
    await loadCycles(nerve, store, [relisted]);
    expect(store.getCycleById(current.id)).toEqual(relisted);
    expect(store.getCycleRow(current.id)).toEqual(relisted);

    // a cycle its read alone has, of no list the store fetched: it shows, without a row
    await loadCycle(nerve, store, shelved);
    expect(store.getCycleById(shelved.id)).toEqual(shelved);
    expect(store.getCycleRow(shelved.id)).toBeUndefined();
    // archived elsewhere: the archived list, answered last, has its row, with its counts
    const archivedRow = rowOf({ ...current, archived_at: LATER }, { total_issues: 7 });
    await loadArchivedCycles(nerve, store, [archivedRow]);
    expect(store.getCycleById(current.id)).toEqual(archivedRow);
    expect(store.getCycleRow(current.id)).toEqual(archivedRow);
  });

  it("names the project's cycle in progress: the first whose status nerve answered current; null for none", async () => {
    const { nerve, store } = await listTab(CycleStore);
    expect(store.getActiveCycleId(web.id)).toBeUndefined();
    await loadCycles(nerve, store, webRows);
    expect(store.getActiveCycleId(web.id)).toBe(current.id);
    await loadCycles(nerve, store, [rowOf(cycleOf(ops, "O2"))], ops);
    expect(store.getActiveCycleId(ops.id)).toBeNull();
    // its day has passed: its read, answered after the list, says it is completed
    await loadCycle(nerve, store, { ...current, status: "completed" });
    expect(store.getActiveCycleId(web.id)).toBeNull();
  });

  it("answers nothing of a project the caller no longer has, or never had: its cycles, rows, reads and progress", async () => {
    const { nerve, projects, store } = await listTab(CycleStore);
    await loadCycles(nerve, store, webRows);
    await loadCycles(nerve, store, [rowOf(opsCycle)], ops);
    await loadArchivedCycles(nerve, store, [rowOf(opsShelved)], ops);
    await loadCycle(nerve, store, opsShelved);
    await loadProgress(nerve, store, opsCycle, progressOf(opsCycle));
    expect(store.getProjectCycleIds(ops.id)).toEqual([opsCycle.id]);
    expect(store.getActiveCycleId(ops.id)).toBe(opsCycle.id);
    expect(store.getArchivedCycleIds(ops.id)).toEqual([opsShelved.id]);
    expect(store.getCycleById(opsShelved.id)).toEqual(opsShelved);
    expect(store.getCycleProgress(opsCycle.id)).toEqual(progressOf(opsCycle));

    await sent(nerve, () => projects.leaveProject(ops), ["POST", `/api/v0/projects/${ops.id}/leave`], noContent());
    expect(store.getProjectCycleIds(ops.id)).toBeUndefined();
    expect(store.getActiveCycleId(ops.id)).toBeUndefined();
    expect(store.getArchivedCycleIds(ops.id)).toBeUndefined();
    expect(store.getCycleById(opsCycle.id)).toBeUndefined();
    expect(store.getCycleById(opsShelved.id)).toBeUndefined();
    expect(store.getCycleRow(opsCycle.id)).toBeUndefined();
    expect(store.getCycleProgress(opsCycle.id)).toBeUndefined();
    // web's still show; a project he never had, or none, has nothing
    expect(store.getProjectCycleIds(web.id)).toEqual(ids(webRows));
    expect(store.getProjectCycleIds("p-none")).toBeUndefined();
    expect(store.getProjectCycleIds(undefined)).toBeUndefined();
    expect(store.getCycleById(undefined)).toBeUndefined();
  });

  /** A value of the store: how it is fetched, by which request, two answers of nerve, and how the store shows it. */
  type Fetched = {
    value: string;
    given?: (nerve: FakeNerve, store: CycleStore) => Promise<unknown>;
    fetch: (store: CycleStore) => Promise<unknown>;
    path: string;
    older: unknown;
    newer: unknown;
    shown: (store: CycleStore) => unknown;
    shows: unknown;
  };
  const newerProgress: CycleProgress = progressOf(current, { total_issues: 2, completed_issues: 1 });
  const fetches: Fetched[] = [
    {
      value: "a project's cycles",
      fetch: (store) => store.fetchCycles(web.id),
      path: cycleRequests.list(web),
      older: { data: [rowOf(ideas)] },
      newer: { data: [rowOf(ideas), rowOf(current)] },
      shown: (store) => store.getProjectCycleIds(web.id),
      shows: [ideas.id, current.id],
    },
    {
      value: "a project's archived cycles",
      fetch: (store) => store.fetchArchivedCycles(web.id),
      path: cycleRequests.list(web),
      older: { data: [] },
      newer: { data: [rowOf(shelved)] },
      shown: (store) => store.getArchivedCycleIds(web.id),
      shows: [shelved.id],
    },
    {
      value: "a cycle",
      fetch: (store) => store.fetchCycle(current.id),
      path: cycleRequests.at(current),
      older: current,
      newer: { ...current, name: "Sprint two" },
      shown: (store) => store.getCycleById(current.id)?.name,
      shows: "Sprint two",
    },
    {
      value: "a cycle's progress",
      given: (nerve, store) => loadCycle(nerve, store, current),
      fetch: (store) => store.fetchProgress(current.id),
      path: cycleRequests.progress(current),
      older: progressOf(current),
      newer: newerProgress,
      shown: (store) => store.getCycleProgress(current.id),
      shows: newerProgress,
    },
  ];
  it.each(fetches)(
    "writes nothing of an older fetch of $value that nerve answers after a newer one",
    async ({ given, fetch, path, older, newer, shown, shows }) => {
      const { nerve, store } = await listTab(CycleStore);
      await given?.(nerve, store);
      const at = nerve.calls.length;
      const first = track(fetch(store));
      const second = track(fetch(store));
      await until(() => nerve.calls.length === at + 2, "both fetches");
      expect(nerve.calls.slice(at).map((call) => call.path)).toEqual([path, path]);
      nerve.calls[at + 1]?.answer(json(200, newer));
      await until(() => second.settled, "the newer");
      nerve.calls[at]?.answer(json(200, older));
      await until(() => first.settled, "the older");
      expect(first).toEqual({ settled: true, value: undefined });
      expect(shown(store)).toEqual(shows);
    }
  );

  it("fails when nerve refuses a fetch, keeping what it had, and gives nothing when the session changes", async () => {
    const { nerve, api, store } = await listTab(CycleStore);
    const refused = track(store.fetchCycles(web.id));
    await until(() => nerve.calls.length === 1, "the cycles");
    nerve.calls[0]?.answer(problem(403, "forbidden"));
    await until(() => refused.settled, "the refusal");
    expect(refused.error).toBeInstanceOf(ApiError);
    expect(store.getProjectCycleIds(web.id)).toBeUndefined();

    await loadCycles(nerve, store, webRows);
    FakeNerve.replaceSession(api);
    const fetched = await settle(store.fetchCycles(web.id), "the refetch");
    expect(fetched).toEqual({ settled: true, value: undefined });
    expect(nerve.calls).toHaveLength(2);
    expect(store.getProjectCycleIds(web.id)).toEqual(ids(webRows));
  });
});
