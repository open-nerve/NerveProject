/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { State } from "@nerve/api-client";
import { ApiError } from "@/lib/api-error";
import { FakeNerve, answered, json, noContent, problem } from "@/lib/auth/fake-nerve";
import { settle, track, until } from "@/lib/auth/fake-time";
import {
  acme,
  beta,
  blocking,
  changed,
  lab,
  listTab,
  loadStates,
  loadedStates,
  ops,
  stateRequests,
  web,
} from "@/store/fake-project-lists";
import { fetchedWhileChangeIsOut, inTurn, sent } from "@/store/fake-queue";
import { stateOf } from "@/store/project/fake-projects";
import { StateStore } from "@/store/state.store";

// The states of the caller's projects (M3 design 3.17, 7.3), against a fake nerve that answers each request when the
// test says. The tab's address is web's, a project of acme; ops is acme's other; his other workspace, beta, has lab.

const { LIST, ACME, at } = stateRequests;
const backlog = stateOf(web, "Backlog", "backlog", 15000, { default: true });
const todo = stateOf(web, "Todo", "unstarted", 30000);
const doing = stateOf(web, "Doing", "started", 45000);
const review = stateOf(web, "Review", "started", 50000);
const done = stateOf(web, "Done", "completed", 40000);
/** web's states by group, then sequence: the started group has two, whose sequences are above done's. */
const listed = [backlog, todo, doing, review, done];
const opsBacklog = stateOf(ops, "Backlog", "backlog", 15000, { default: true });
/** Doing moved before Todo, as nerve answers it. */
const moved = changed(doing, { group: "unstarted", sequence: 20000 });
/** The state created in web's started group, as nerve answers it. */
const blocked = stateOf(web, "Blocked", "started", 75000);
const ids = (states: State[]) => states.map((state) => state.id);

beforeEach(() => {
  vi.useFakeTimers({ now: 1_000_000 });
});
afterEach(() => {
  vi.useRealTimers();
});

describe("StateStore, the states", () => {
  it("keeps a project's states by group, then sequence, and each one's place in its group", async () => {
    const { nerve, store } = await listTab(StateStore);
    const fetched = await loadStates(nerve, store, [doing, done, review, backlog, todo]);
    expect(fetched.value).toEqual([doing, done, review, backlog, todo]);
    expect(store.getProjectStates(web.id)).toEqual(listed);
    expect(store.projectStates).toEqual(listed);
    expect(store.getProjectStateIds(web.id)).toEqual(ids(listed));
    expect(store.groupedProjectStates).toEqual({
      backlog: [backlog],
      unstarted: [todo],
      started: [doing, review],
      completed: [done],
      cancelled: [],
    });
    expect(store.getStatePercentageInGroup(doing.id)).toBe(50);
    expect(store.getStatePercentageInGroup(review.id)).toBe(100);
    expect(store.getStateById(todo.id)).toEqual(todo);
    expect(store.getProjectStates(ops.id)).toBeUndefined();
  });

  it("gives a project's states from its workspace's list until its own is fetched, and none of a project left", async () => {
    const { nerve, projects, store } = await listTab(StateStore);
    const fetch = () => store.fetchWorkspaceStates(acme);
    await answered(nerve, fetch, ["GET", ACME], { data: [opsBacklog, ...listed] }, "acme's states");
    // by group, then sequence; nerve's order where both are equal
    expect(store.workspaceStates).toEqual([opsBacklog, backlog, todo, doing, review, done]);
    expect(store.getProjectStates(ops.id)).toEqual([opsBacklog]);
    expect(store.getProjectStates(web.id)).toEqual(listed);
    await loadStates(nerve, store, [backlog, done]);
    expect(store.getProjectStates(web.id)).toEqual([backlog, done]);

    await sent(nerve, () => projects.leaveProject(ops), ["POST", `/api/v0/projects/${ops.id}/leave`], noContent());
    expect(store.getProjectStates(ops.id)).toBeUndefined();
    expect(store.getStateById(opsBacklog.id)).toBeUndefined();
    expect(store.workspaceStates).toEqual([backlog, todo, doing, review, done]);
  });

  it("keeps each workspace's states under its id: another workspace's list does not replace acme's", async () => {
    const { nerve, store } = await listTab(StateStore);
    await answered(nerve, () => store.fetchWorkspaceStates(acme), ["GET", ACME], { data: listed }, "acme's states");
    const labBacklog = stateOf(lab, "Backlog", "backlog", 15000, { default: true });
    const BETA = "/api/v0/workspaces/beta/states";
    await answered(
      nerve,
      () => store.fetchWorkspaceStates(beta),
      ["GET", BETA],
      { data: [labBacklog] },
      "beta's states"
    );
    expect(store.workspaceStates).toEqual(listed);
    expect(store.getProjectStates(lab.id)).toEqual([labBacklog]);
  });

  it("gives one copy of a state, its project's list's, which the project's pages fetch again: not its workspace's", async () => {
    const { nerve, store } = await listTab(StateStore);
    await answered(nerve, () => store.fetchWorkspaceStates(acme), ["GET", ACME], { data: listed }, "acme's states");
    await loadStates(nerve, store, [backlog, moved, todo, review, done]);
    expect(store.getStateById(doing.id)).toEqual(moved);
    expect(store.getProjectStates(web.id)).toContainEqual(moved);
  });

  it("fails when nerve refuses a project's or the workspace's, keeping none, and again, keeping the states it had", async () => {
    const { nerve, store } = await listTab(StateStore);
    const refused = track(store.fetchProjectStates(web.id));
    await until(() => nerve.calls.length === 1, "the states");
    nerve.calls[0]?.answer(problem(403, "forbidden"));
    await until(() => refused.settled, "the refusal");
    expect(refused.error).toBeInstanceOf(ApiError);
    expect(store.getProjectStates(web.id)).toBeUndefined();
    const workspace = track(store.fetchWorkspaceStates(acme));
    await until(() => nerve.calls.length === 2, "acme's states");
    nerve.calls[1]?.answer(problem(403, "forbidden"));
    await until(() => workspace.settled, "the refusal");
    expect(workspace.error).toBeInstanceOf(ApiError);
    expect(store.workspaceStates).toBeUndefined();

    await loadStates(nerve, store, listed);
    const again = track(store.fetchProjectStates(web.id));
    await until(() => nerve.calls.length === 4, "the refetch");
    nerve.calls[3]?.answer(problem(503, "server_busy"));
    await until(() => again.settled, "the refusal");
    expect(again.error).toBeInstanceOf(ApiError);
    expect(store.getProjectStates(web.id)).toEqual(listed);
  });

  it("keeps the states it had, gives nothing and does not fail, when the session changes as it fetches again", async () => {
    const { nerve, api, store } = await loadedStates(listed);
    FakeNerve.replaceSession(api);
    const fetched = await settle(store.fetchProjectStates(web.id), "the refetch");
    expect(fetched).toEqual({ settled: true, value: undefined });
    expect(nerve.calls).toHaveLength(1);
    expect(store.getProjectStates(web.id)).toEqual(listed);
  });
});

describe("StateStore, the changes", () => {
  it("adds a created state as nerve answers it, last of its project's", async () => {
    const { nerve, store } = await loadedStates(listed);
    const created = await sent(nerve, () => store.createState(web.id, blocking), ["POST", LIST], json(201, blocked));
    expect(nerve.calls[1]?.body).toEqual(blocking);
    expect(created.value).toEqual(blocked);
    expect(store.groupedProjectStates?.started).toEqual([doing, review, blocked]);
    expect(store.getStatePercentageInGroup(review.id)).toBeCloseTo(66.67, 2);
  });

  it("moves a state where it was dropped, to the place nerve answers, in its project's list and its workspace's", async () => {
    const { nerve, store } = await loadedStates(listed);
    await answered(nerve, () => store.fetchWorkspaceStates(acme), ["GET", ACME], { data: listed }, "acme's states");
    // into unstarted, before Todo, its first: a step before it
    const move = track(store.moveState(doing.id, "unstarted", todo.id, false));
    await until(() => nerve.calls.length === 3, "the move");
    expect(nerve.calls[2]).toMatchObject({
      method: "PATCH",
      path: at(doing),
      body: { group: "unstarted", sequence: 15000 },
    });
    expect(store.getStateById(doing.id)).toEqual(doing);
    nerve.calls[2]?.answer(json(200, moved));
    await until(() => move.settled, "the answer");
    expect(store.getProjectStates(web.id)).toEqual([backlog, moved, todo, review, done]);
    expect(store.workspaceStates).toEqual([backlog, moved, todo, review, done]);
    expect(store.getStatePercentageInGroup(review.id)).toBe(100);
  });

  it("reckons a move's sequence in its turn, from the sequences nerve gave: two moves in a row", async () => {
    const { nerve, store } = await loadedStates(listed);
    // Review before Doing, the group's first: a step before it; then Doing after Review, between the two once nerve
    // placed Review first
    const reviewFirst: State = { ...review, sequence: 30000 };
    const doingAfter: State = { ...doing, sequence: 37500 };
    const first = track(store.moveState(review.id, "started", doing.id, false));
    const second = track(store.moveState(doing.id, "started", review.id, true));
    await inTurn(nerve, 1, ["PATCH", at(review)], json(200, reviewFirst));
    await inTurn(nerve, 2, ["PATCH", at(doing)], json(200, doingAfter));
    await until(() => second.settled, "the second move");
    expect(first.error).toBeUndefined();
    expect(nerve.calls.slice(1).map((call) => call.body)).toEqual([
      { group: "started", sequence: 30000 },
      { group: "started", sequence: 37500 },
    ]);
    expect(store.groupedProjectStates?.started).toEqual([reviewFirst, doingAfter]);
  });

  it("changes a state, and moves a state to the end of a group: a step past its last", async () => {
    const { nerve, store } = await loadedStates(listed);
    const renamed: State = { ...todo, name: "To do" };
    await sent(nerve, () => store.updateState(todo.id, { name: "To do" }), ["PATCH", at(todo)], json(200, renamed));
    expect(nerve.calls[1]?.body).toEqual({ name: "To do" });
    expect(store.getStateById(todo.id)).toEqual(renamed);
    const last: State = { ...doing, sequence: 65000 };
    await sent(
      nerve,
      () => store.moveState(doing.id, "started", undefined, false),
      ["PATCH", at(doing)],
      json(200, last)
    );
    expect(nerve.calls[2]?.body).toEqual({ group: "started", sequence: 65000 });
  });

  it("deletes a state, and makes another its project's default, the one that was no longer", async () => {
    const { nerve, store } = await loadedStates(listed);
    await answered(nerve, () => store.fetchWorkspaceStates(acme), ["GET", ACME], { data: [opsBacklog] }, "acme");
    const deleted = await sent(nerve, () => store.deleteState(review.id), ["DELETE", at(review)], noContent());
    expect(deleted.error).toBeUndefined();
    expect(store.getProjectStateIds(web.id)).toEqual(ids([backlog, todo, doing, done]));

    await sent(nerve, () => store.markStateAsDefault(todo.id), ["POST", `${at(todo)}/mark-default`], noContent());
    expect(store.getProjectStates(web.id)?.filter((state) => state.default)).toEqual([{ ...todo, default: true }]);
    // ops's default is its own
    expect(store.getStateById(opsBacklog.id)?.default).toBe(true);
  });

  const refusals: { change: string; send: (store: StateStore) => Promise<unknown>; refusal: Response }[] = [
    {
      change: "a creation",
      send: (store) => store.createState(web.id, { name: "Todo", color: "#60646C", group: "unstarted" }),
      refusal: problem(409, "project.state_name_taken"),
    },
    {
      change: "a change",
      send: (store) => store.updateState(todo.id, { name: "Doing" }),
      refusal: problem(409, "project.state_name_taken"),
    },
    {
      change: "a move",
      send: (store) => store.moveState(done.id, "started", undefined, false),
      refusal: problem(409, "project.state_last_in_group"),
    },
    {
      change: "a deletion",
      send: (store) => store.deleteState(backlog.id),
      refusal: problem(409, "project.state_default"),
    },
    {
      change: "a new default",
      send: (store) => store.markStateAsDefault(todo.id),
      refusal: problem(403, "forbidden"),
    },
  ];
  it.each(refusals)("fails, changing nothing, when nerve refuses $change", async ({ send, refusal }) => {
    const { nerve, store } = await loadedStates(listed);
    const refused = track(send(store));
    await until(() => nerve.calls.length === 2, "the change");
    nerve.calls[1]?.answer(refusal);
    await until(() => refused.settled, "the refusal");
    expect(refused.error).toBeInstanceOf(ApiError);
    expect(store.getProjectStates(web.id)).toEqual(listed);
  });

  it.each(refusals.slice(2))(
    "fails, asking nerve nothing, for $change of a state it does not have",
    async ({ send }) => {
      const { nerve, store } = await listTab(StateStore);
      const refused = await settle(send(store), "the change");
      expect(refused).toMatchObject({ settled: true, error: new Error("State not found") });
      expect(nerve.calls).toEqual([]);
    }
  );

  it("sends each change once nerve has answered the one before it, refused or not: two moves in a row among them", async () => {
    const { nerve, store } = await loadedStates(listed);
    const first = track(store.updateState(doing.id, { sequence: 1 }));
    const second = track(store.updateState(doing.id, { group: "unstarted", sequence: 20000 }));
    const marked = track(store.markStateAsDefault(todo.id));
    await inTurn(nerve, 1, ["PATCH", at(doing)], problem(503, "server_busy"));
    await inTurn(nerve, 2, ["PATCH", at(doing)], json(200, moved));
    await inTurn(nerve, 3, ["POST", `${at(todo)}/mark-default`], noContent());
    await until(() => marked.settled, "the last change");
    expect(first.error).toBeInstanceOf(ApiError);
    expect(second.value).toEqual(moved);
    expect(store.getStateById(todo.id)?.default).toBe(true);
  });

  it("fetches the states while a change is out: a fetch does not wait for it", async () => {
    const { nerve, store } = await loadedStates(listed);
    await fetchedWhileChangeIsOut(
      nerve,
      { send: () => store.updateState(doing.id, { sequence: 1 }), request: ["PATCH", at(doing)] },
      { send: () => store.fetchProjectStates(web.id), request: ["GET", LIST], body: { data: [backlog, done] } }
    );
    expect(store.getProjectStates(web.id)).toEqual([backlog, done]);
  });
});

// A fetch's answer may be older than a change nerve confirmed while it was out: what the fetch shows carries the
// change (the order is forced: the fetch waits until the test answers it, after the change has finished). Of two
// fetches of one project's states, only the newer writes.
describe("StateStore, while a fetch is out", () => {
  it("shows once a state created during a refetch whose list has it, and the changes confirmed meanwhile", async () => {
    const { nerve, store } = await loadedStates(listed);
    const refetched = track(store.fetchProjectStates(web.id));
    await until(() => nerve.calls.length === 2, "the refetch");
    await sent(nerve, () => store.createState(web.id, blocking), ["POST", LIST], json(201, blocked));
    await sent(nerve, () => store.updateState(doing.id, { sequence: 20000 }), ["PATCH", at(doing)], json(200, moved));
    await sent(nerve, () => store.deleteState(done.id), ["DELETE", at(done)], noContent());
    // read after the creation, before the move and the deletion
    nerve.calls[1]?.answer(json(200, { data: [...listed, blocked] }));
    await until(() => refetched.settled, "the refetch");

    const shown = [backlog, moved, todo, review, blocked];
    expect(store.getProjectStates(web.id)).toEqual(shown);
    expect(refetched.value && ids(refetched.value)).toEqual(ids([backlog, todo, moved, review, blocked]));
  });

  it("keeps a state created during a fetch of the workspace's states on the list the fetch shows", async () => {
    const { nerve, store } = await loadedStates(listed);
    const fetched = track(store.fetchWorkspaceStates(acme));
    await until(() => nerve.calls.length === 2, "acme's states");
    await sent(nerve, () => store.createState(web.id, blocking), ["POST", LIST], json(201, blocked));
    // read before the creation
    nerve.calls[1]?.answer(json(200, { data: listed }));
    await until(() => fetched.settled, "acme's states");
    expect(store.workspaceStates).toEqual([backlog, todo, doing, review, blocked, done]);
  });

  it("lets each project's newer fetch write: an older one answering last writes nothing", async () => {
    const { nerve, store } = await listTab(StateStore);
    const older = track(store.fetchProjectStates(web.id));
    await until(() => nerve.calls.length === 1, "the older states");
    const opsStates = track(store.fetchProjectStates(ops.id));
    await until(() => nerve.calls.length === 2, "ops's states");
    const newer = track(store.fetchProjectStates(web.id));
    await until(() => nerve.calls.length === 3, "the newer states");
    nerve.calls[2]?.answer(json(200, { data: [backlog, moved, done] }));
    await until(() => newer.settled, "the newer states");
    // a newer fetch of web's states does not overtake one of ops's
    nerve.calls[1]?.answer(json(200, { data: [opsBacklog] }));
    await until(() => opsStates.settled, "ops's states");
    // read before Doing moved
    nerve.calls[0]?.answer(json(200, { data: listed }));
    await until(() => older.settled, "the older states");

    expect(older).toEqual({ settled: true, value: undefined });
    expect(store.getProjectStates(web.id)).toEqual([backlog, moved, done]);
    expect(store.getProjectStates(ops.id)).toEqual([opsBacklog]);
  });
});
