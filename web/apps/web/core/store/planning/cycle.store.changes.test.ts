/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { Cycle } from "@nerve/api-client";
import { ApiError } from "@/lib/api-error";
import type { Endpoint } from "@/lib/auth/fake-nerve";
import { FakeNerve, json, noContent, problem } from "@/lib/auth/fake-nerve";
import { settle, track, until } from "@/lib/auth/fake-time";
import { SessionChangedError } from "@/lib/auth/token-manager";
import { listTab, ops, web } from "@/store/fake-project-lists";
import { fetchedWhileChangeIsOut, inTurn, sent } from "@/store/fake-queue";
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

// The changes of the cycles (M6 design 3.1–3.8; P5a spec 2.2, R9; v0 design 7.7), against a fake nerve that answers
// each request when the test says. They go one at a time; a deletion or a transfer finds its cycle in its turn; nerve's
// answer is written into every copy that holds the cycle (its lists' rows, its own read, its progress's status), and a
// row keeps its counts, which only a list gives (a Cycle has none). The tab's address is web's, a project of acme;
// ops is acme's other.

const { list, at, progress, archive, unarchive, transfer } = cycleRequests;

const ended = cycleOf(web, "S1", { status: "completed", start_date: "2026-10-26", end_date: "2026-11-01" });
const current = cycleOf(web, "S2", {
  status: "current",
  start_date: "2026-11-09",
  end_date: "2026-11-15",
  sort_order: 55535,
});
const ideas = cycleOf(web, "Ideas", { sort_order: 35535 });
const shelved = cycleOf(web, "S0", {
  status: "completed",
  start_date: "2026-10-12",
  end_date: "2026-10-18",
  archived_at: "2026-10-20T09:00:00Z",
});
/** Times after the fixtures', of nerve's answers to the changes. */
const LATER = "2026-11-10T09:00:00Z";
const LATEST = "2026-11-10T10:00:00Z";
/** web's rows, each with counts of its own, which a change must keep (R9). */
const ideasRow = rowOf(ideas);
const currentRow = rowOf(current, { total_issues: 3, started_issues: 2, completed_issues: 1, assignee_ids: ["u-ann"] });
const endedRow = rowOf(ended, {
  total_issues: 4,
  unstarted_issues: 1,
  completed_issues: 3,
  assignee_ids: ["u-ann", "u-bob"],
  progress_snapshot: {
    total_issues: 4,
    backlog_issues: 0,
    unstarted_issues: 1,
    started_issues: 0,
    completed_issues: 3,
    cancelled_issues: 0,
    frozen_at: "2026-11-02T00:00:00Z",
  },
});
const shelvedRow = rowOf(shelved, { total_issues: 2, completed_issues: 2, assignee_ids: ["u-bob"] });
const currentProgress = progressOf(current, { total_issues: 3, started_issues: 2, completed_issues: 1 });

beforeEach(() => {
  vi.useFakeTimers({ now: 1_000_000 });
});
afterEach(() => {
  vi.useRealTimers();
});

/** A tab whose store holds web's two lists, and S2's own read and progress. */
async function loadedCycles() {
  const tab = await listTab(CycleStore);
  await loadCycles(tab.nerve, tab.store, [ideasRow, currentRow, endedRow]);
  await loadArchivedCycles(tab.nerve, tab.store, [shelvedRow]);
  await loadCycle(tab.nerve, tab.store, current);
  await loadProgress(tab.nerve, tab.store, current, currentProgress);
  return tab;
}

/** Every value the store holds of web's cycles: its two lists, and each cycle's own read and progress. */
const held = (store: CycleStore) => ({
  unarchived: store.unarchived.get(web.id),
  archived: store.archived.get(web.id),
  details: [ideas, current, ended, shelved].map((cycle) => store.details.get(cycle.id)),
  progress: [ideas, current, ended, shelved].map((cycle) => store.progress.get(cycle.id)),
});

describe("CycleStore, the changes", () => {
  it("creates a cycle: first of its project's, of no work items, so of no assignees and not frozen", async () => {
    const { nerve, store } = await loadedCycles();
    // a draft names no dates; nerve gives a new cycle the lowest sort order of its project's
    const body = { name: "S4", description: "the next one" };
    const made = cycleOf(web, "S4", {
      description: "the next one",
      sort_order: 25535,
      created_at: LATER,
      updated_at: LATER,
    });
    const creating = await sent(nerve, () => store.create(web.id, body), ["POST", list(web)], json(201, made));
    expect(nerve.calls.at(-1)?.body).toEqual(body);
    expect(creating.value).toEqual(made);
    expect(store.getProjectCycleIds(web.id)).toEqual([made.id, ideas.id, current.id, ended.id]);
    expect(store.getCycleRow(made.id)).toEqual({
      ...made,
      total_issues: 0,
      backlog_issues: 0,
      unstarted_issues: 0,
      started_issues: 0,
      completed_issues: 0,
      cancelled_issues: 0,
      assignee_ids: [],
      progress_snapshot: null,
    });
    expect(store.getCycleRow(current.id)).toEqual(currentRow);

    // of a project whose cycles the store has not fetched: nothing is written, the fetch brings them
    const opsMade = cycleOf(ops, "O1");
    await sent(nerve, () => store.create(ops.id, { name: "O1" }), ["POST", list(ops)], json(201, opsMade));
    expect(store.getProjectCycleIds(ops.id)).toBeUndefined();
  });

  it("changes a cycle: nerve's answer, its status too, in its row, its read and its progress; the row keeps its counts (R9)", async () => {
    const { nerve, store } = await loadedCycles();
    // a week later: nerve answers it upcoming
    const body = { start_date: "2026-11-16", end_date: "2026-11-22" };
    const moved: Cycle = { ...current, ...body, status: "upcoming", updated_at: LATER };
    const changing = await sent(nerve, () => store.update(current.id, body), ["PATCH", at(current)], json(200, moved));
    expect(nerve.calls.at(-1)?.body).toEqual(body);
    expect(changing.value).toEqual(moved);
    expect(store.unarchived.get(web.id)).toEqual([ideasRow, { ...currentRow, ...moved }, endedRow]);
    expect(store.details.get(current.id)).toEqual(moved);
    expect(store.getCycleById(current.id)).toEqual(moved);
    expect(store.getCycleRow(current.id)).toEqual({ ...currentRow, ...moved });
    expect(store.getCycleProgress(current.id)).toEqual({ ...currentProgress, status: "upcoming" });
    expect(store.getActiveCycleId(web.id)).toBeNull();
  });

  it("writes a change into a copy of a list answered last, too: an archive fetched while it was archived elsewhere", async () => {
    const { nerve, store } = await loadedCycles();
    // archived in another tab, and this one fetched its archive; then unarchived there
    const between = rowOf({ ...current, archived_at: LATER }, { total_issues: 3, started_issues: 2 });
    await loadArchivedCycles(nerve, store, [between, shelvedRow]);
    const renamed: Cycle = { ...current, name: "Sprint two", updated_at: LATEST };
    const rename = () => store.update(current.id, { name: "Sprint two" });
    await sent(nerve, rename, ["PATCH", at(current)], json(200, renamed));
    // the newest copy, the archive's row, shows nerve's answer
    expect(store.archived.get(web.id)).toEqual([{ ...between, ...renamed }, shelvedRow]);
    expect(store.getCycleById(current.id)).toEqual({ ...between, ...renamed });
  });

  it("archives an ended cycle, which moves with its counts first into the archive, and unarchives it back", async () => {
    const { nerve, store } = await loadedCycles();
    const archived: Cycle = { ...ended, archived_at: LATER, updated_at: LATER };
    const archiving = await sent(nerve, () => store.archive(ended.id), ["POST", archive(ended)], json(200, archived));
    expect(archiving.value).toEqual(archived);
    expect(store.getProjectCycleIds(web.id)).toEqual([ideas.id, current.id]);
    expect(store.archived.get(web.id)).toEqual([{ ...endedRow, ...archived }, shelvedRow]);
    expect(store.getCycleRow(ended.id)).toEqual({ ...endedRow, ...archived });

    // unarchived: first of the list again, its counts kept; and so one nerve gave in the archive
    const restored: Cycle = { ...ended, updated_at: LATEST };
    const restore = () => store.unarchive(ended.id);
    const unarchiving = await sent(nerve, restore, ["POST", unarchive(ended)], json(200, restored));
    expect(unarchiving.value).toEqual(restored);
    const shelvedBack: Cycle = { ...shelved, archived_at: null, updated_at: LATEST };
    await sent(nerve, () => store.unarchive(shelved.id), ["POST", unarchive(shelved)], json(200, shelvedBack));
    expect(store.unarchived.get(web.id)).toEqual([
      { ...shelvedRow, ...shelvedBack },
      { ...endedRow, ...restored },
      ideasRow,
      currentRow,
    ]);
    expect(store.getArchivedCycleIds(web.id)).toEqual([]);
  });

  it("archives again a cycle archived elsewhere: nerve's answer is written, and the archive keeps its row in its place", async () => {
    const { nerve, store } = await loadedCycles();
    await loadCycle(nerve, store, ended);
    // another tab archived S1, then S-1 after it; this one's list still has S1, its archive, fetched since, has both
    const elsewhere: Cycle = { ...ended, archived_at: LATER, updated_at: LATER };
    const later = cycleOf(web, "S-1", { status: "completed", archived_at: LATEST });
    const archivedRows = [rowOf(later, { total_issues: 1 }), rowOf(elsewhere, { total_issues: 4 }), shelvedRow];
    await loadArchivedCycles(nerve, store, archivedRows);
    // nerve writes nothing and answers S1 as it holds it
    await sent(nerve, () => store.archive(ended.id), ["POST", archive(ended)], json(200, elsewhere));
    expect(store.getProjectCycleIds(web.id)).toEqual([ideas.id, current.id]);
    expect(store.archived.get(web.id)).toEqual(archivedRows);
    expect(store.details.get(ended.id)).toEqual(elsewhere);
  });

  it("deletes a cycle from every copy, its read and progress with it; a read out meanwhile answers null", async () => {
    const { nerve, store } = await loadedCycles();
    const k = nerve.calls.length;
    const reading = track(store.fetchCycle(current.id));
    await until(() => nerve.calls.length === k + 1, "the read");
    const removing = await sent(nerve, () => store.remove(current.id), ["DELETE", at(current)], noContent());
    expect(removing).toEqual({ settled: true, value: undefined });
    expect(store.getProjectCycleIds(web.id)).toEqual([ideas.id, ended.id]);
    expect(store.getActiveCycleId(web.id)).toBeNull();
    expect(store.getCycleById(current.id)).toBeUndefined();
    expect(store.getCycleRow(current.id)).toBeUndefined();
    expect(store.getCycleProgress(current.id)).toBeUndefined();
    // the read, which nerve answered before the deletion
    nerve.calls[k]?.answer(json(200, current));
    await until(() => reading.settled, "the read");
    expect(reading.value).toBeNull();
    expect(store.getCycleById(current.id)).toBeUndefined();

    // an archived one, which its read and the archive hold
    await loadCycle(nerve, store, shelved);
    await sent(nerve, () => store.remove(shelved.id), ["DELETE", at(shelved)], noContent());
    expect(store.getArchivedCycleIds(web.id)).toEqual([]);
    expect(store.getCycleById(shelved.id)).toBeUndefined();
  });

  it("transfers an ended cycle's unfinished work items: gives how many moved, then fetches the counts nerve holds now", async () => {
    const { nerve, store } = await loadedCycles();
    await loadProgress(nerve, store, ended, progressOf(ended, { total_issues: 4, unstarted_issues: 1 }));
    const k = nerve.calls.length;
    const moved = { transferred_issues: 1 };
    const move = () => store.transfer(ended.id, current.id);
    const moving = await sent(nerve, move, ["POST", transfer(ended)], json(200, moved));
    expect(nerve.calls[k]?.body).toEqual({ target_cycle_id: current.id });
    expect(moving.value).toEqual(moved);
    // the web reckons no counts: until nerve answers the fetches, the store shows what it answered before
    expect(store.getCycleRow(ended.id)).toEqual(endedRow);
    expect(store.getCycleProgress(current.id)).toEqual(currentProgress);
    await until(() => nerve.calls.length === k + 4, "the fetches");
    const fetches = nerve.calls.slice(k + 1);
    expect(fetches.map((call) => `${call.method} ${call.path}`).toSorted()).toEqual(
      [`GET ${progress(ended)}`, `GET ${progress(current)}`, `GET ${list(web)}`].toSorted()
    );
    expect(fetches.find((call) => call.path === list(web))?.query).toEqual({});

    const endedNow = progressOf(ended, { total_issues: 3 });
    const currentNow = progressOf(current, { total_issues: 4, unstarted_issues: 1, started_issues: 2 });
    const rowsNow = [ideasRow, rowOf(current, { total_issues: 4 }), rowOf(ended, { total_issues: 3 })];
    nerve.to(progress(ended))[1]?.answer(json(200, endedNow));
    nerve.to(progress(current))[1]?.answer(json(200, currentNow));
    nerve.calls.find((call, n) => n > k && call.path === list(web))?.answer(json(200, { data: rowsNow }));
    await until(() => store.getCycleRow(ended.id)?.total_issues === 3, "the list");
    await until(() => store.getCycleProgress(current.id)?.live.total_issues === 4, "the target's progress");
    await until(() => store.getCycleProgress(ended.id)?.live.total_issues === 3, "the source's progress");
    expect(store.getCycleProgress(current.id)).toEqual(currentNow);
    expect(store.getCycleProgress(ended.id)).toEqual(endedNow);
    expect(store.unarchived.get(web.id)).toEqual(rowsNow);
  });
});

describe("CycleStore, a change in its turn", () => {
  it("sends each change once nerve has answered the one before it; a deletion or a transfer finds its cycle in its turn", async () => {
    const { nerve, projects, store } = await loadedCycles();
    const k = nerve.calls.length;
    // made while S2 is held: the change and the deletion go out one at a time, and the two behind them find S2 gone
    const renamed = track(store.update(current.id, { name: "Sprint two" }));
    const removed = track(store.remove(current.id));
    const behind = [track(store.transfer(current.id, ideas.id)), track(store.remove(current.id))];
    await inTurn(nerve, k, ["PATCH", at(current)], problem(503, "server_busy"));
    await inTurn(nerve, k + 1, ["DELETE", at(current)], noContent());
    await until(() => behind.every((change) => change.settled), "the changes behind them");
    expect(renamed.error).toBeInstanceOf(ApiError);
    expect(removed).toEqual({ settled: true, value: undefined });
    const notFound = new Error("Cycle not found");
    expect(behind.map((change) => change.error)).toEqual([notFound, notFound]);
    expect(nerve.calls).toHaveLength(k + 2);

    // of a project he left, the store shows none: nerve is asked nothing
    await sent(nerve, () => projects.leaveProject(web), ["POST", `/api/v0/projects/${web.id}/leave`], noContent());
    const left = await settle(store.transfer(ended.id, ideas.id), "a transfer in web");
    expect(left).toMatchObject({ settled: true, error: notFound });
    expect(nerve.calls).toHaveLength(k + 3);
  });
});

// A fetch's answer may be older than the changes nerve confirmed while it was out: what the fetch shows carries them
// (the order is forced: each fetch waits until the test answers it, after the changes have finished).
describe("CycleStore, while a fetch is out", () => {
  it("makes the changes nerve confirmed meanwhile on the answers of the fetches, read before them", async () => {
    const { nerve, store } = await loadedCycles();
    const k = nerve.calls.length;
    const listing = track(store.fetchCycles(web.id));
    await until(() => nerve.calls.length === k + 1, "the list");
    const archiving = track(store.fetchArchivedCycles(web.id));
    await until(() => nerve.calls.length === k + 2, "the archive");
    const reading = track(store.fetchCycle(current.id));
    await until(() => nerve.calls.length === k + 3, "the read");
    // meanwhile S4 is created, S2 renamed and S1 archived
    const made = cycleOf(web, "S4", { sort_order: 25535, created_at: LATER, updated_at: LATER });
    await sent(nerve, () => store.create(web.id, { name: "S4" }), ["POST", list(web)], json(201, made));
    const renamed: Cycle = { ...current, name: "Sprint two", updated_at: LATER };
    const rename = () => store.update(current.id, { name: "Sprint two" });
    await sent(nerve, rename, ["PATCH", at(current)], json(200, renamed));
    const archived: Cycle = { ...ended, archived_at: LATER, updated_at: LATER };
    await sent(nerve, () => store.archive(ended.id), ["POST", archive(ended)], json(200, archived));
    // each read before the three; S2 had a work item more by then
    const currentThen = rowOf(current, { total_issues: 4, started_issues: 3, completed_issues: 1 });
    nerve.calls[k]?.answer(json(200, { data: [ideasRow, currentThen, endedRow] }));
    nerve.calls[k + 1]?.answer(json(200, { data: [shelvedRow] }));
    nerve.calls[k + 2]?.answer(json(200, current));
    await until(() => [listing, archiving, reading].every((fetch) => fetch.settled), "the fetches");

    expect(store.unarchived.get(web.id)).toEqual([rowOf(made), ideasRow, { ...currentThen, ...renamed }]);
    expect(store.archived.get(web.id)).toEqual([{ ...endedRow, ...archived }, shelvedRow]);
    expect(store.details.get(current.id)).toEqual(renamed);
    expect(listing.value).toEqual(store.unarchived.get(web.id));
  });

  it("fetches the cycles while a change is out: a fetch does not wait for it, and a refused change leaves its answer", async () => {
    const { nerve, store } = await loadedCycles();
    const relisted = [ideasRow, rowOf(current, { total_issues: 5 }), endedRow];
    await fetchedWhileChangeIsOut(
      nerve,
      { send: () => store.archive(ended.id), request: ["POST", archive(ended)] },
      { send: () => store.fetchCycles(web.id), request: ["GET", list(web)], body: { data: relisted } }
    );
    expect(store.unarchived.get(web.id)).toEqual(relisted);
    expect(store.archived.get(web.id)).toEqual([shelvedRow]);
  });
});

/** A change of the store: what sends it, its request, and a refusal of nerve's. */
type Sending = { change: string; send: (store: CycleStore) => Promise<unknown>; request: Endpoint; refusal: Response };
const changes: Sending[] = [
  {
    change: "a creation",
    send: (store) => store.create(web.id, { name: "S4", start_date: "2026-11-12", end_date: "2026-11-14" }),
    request: ["POST", list(web)],
    refusal: problem(409, "planning.cycle_dates_overlap"),
  },
  {
    change: "a change",
    send: (store) => store.update(current.id, { name: "  " }),
    request: ["PATCH", at(current)],
    refusal: problem(422, "validation_failed"),
  },
  {
    change: "a deletion",
    send: (store) => store.remove(current.id),
    request: ["DELETE", at(current)],
    refusal: problem(403, "forbidden"),
  },
  {
    change: "an archiving",
    send: (store) => store.archive(current.id),
    request: ["POST", archive(current)],
    refusal: problem(409, "planning.cycle_not_completed"),
  },
  {
    change: "an unarchiving",
    send: (store) => store.unarchive(shelved.id),
    request: ["POST", unarchive(shelved)],
    refusal: problem(409, "planning.cycles_disabled"),
  },
  {
    change: "a transfer",
    send: (store) => store.transfer(ended.id, current.id),
    request: ["POST", transfer(ended)],
    refusal: problem(422, "validation_failed"),
  },
];

describe("CycleStore, a change refused or cut", () => {
  it.each(changes)("fails, changing nothing, when nerve refuses $change", async ({ send, request, refusal }) => {
    const { nerve, store } = await loadedCycles();
    const before = held(store);
    const k = nerve.calls.length;
    const refused = track(send(store));
    await inTurn(nerve, k, request, refusal);
    await until(() => refused.settled, "the refusal");
    await vi.advanceTimersByTimeAsync(1_000);
    expect(refused.error).toBeInstanceOf(ApiError);
    expect(held(store)).toEqual(before);
    expect(nerve.calls).toHaveLength(k + 1);
  });

  it("changes nothing of this generation once the session has changed: each change fails before it is sent", async () => {
    const { nerve, api, store } = await loadedCycles();
    const before = held(store);
    const k = nerve.calls.length;
    FakeNerve.replaceSession(api);
    const cut = changes.map(({ send }) => track(send(store)));
    await until(() => cut.every((change) => change.settled), "the changes");
    expect(cut.map((change) => change.error)).toEqual(changes.map(() => expect.any(SessionChangedError)));
    expect(nerve.calls).toHaveLength(k);
    expect(held(store)).toEqual(before);
  });
});
