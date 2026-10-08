/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { State, StateCreate } from "@nerve/api-client";
import { ApiError } from "@/lib/api-error";
import { FakeNerve, answered, json, noContent, problem } from "@/lib/auth/fake-nerve";
import { settle, track, until } from "@/lib/auth/fake-time";
import { inTurn, sent } from "@/store/fake-queue";
import { fakeRoot } from "@/store/fake-root";
import { projectOf, projectTab, stateOf } from "@/store/project/fake-projects";
import { StateStore } from "@/store/state.store";
import { workspaceOf } from "@/store/workspace/fake-workspaces";

// More of the states' changes and fetches (M3 design 3.17, 7.3; v0 design 7.7) than state.store.test.ts has: a change
// of a state of the caller's other workspace is made on that state's own lists, whatever the address; the workspace's
// list shows a creation once across a refetch, and keeps what it had when a refetch is refused or cut; and a change
// finds its state, and a move reckons its place, in its turn, from nerve's answers. The tab's address is web's, a
// project of acme; his other workspace, beta, has lab.

const acme = workspaceOf("acme");
const beta = workspaceOf("beta");
const web = projectOf("WEB", acme.id);
const lab = projectOf("LAB", beta.id);
const LIST = `/api/v0/projects/${web.id}/states`;
const ACME = "/api/v0/workspaces/acme/states";
const at = (state: State) => `/api/v0/states/${state.id}`;

const todo = stateOf(web, "Todo", "unstarted", 15000, { default: true });
const doing = stateOf(web, "Doing", "started", 30000);
const review = stateOf(web, "Review", "started", 45000);
const done = stateOf(web, "Done", "completed", 60000);
/** web's states by group, then sequence */
const listed = [todo, doing, review, done];
/** A state created in web's started group, as the page sends it and as nerve answers it: last of the group. */
const blocking: StateCreate = { name: "Blocked", color: "#60646C", group: "started" };
const blocked = stateOf(web, "Blocked", "started", 60000);
const labTodo = stateOf(lab, "Todo", "unstarted", 15000, { default: true });
const labDoing = stateOf(lab, "Doing", "started", 30000);
const labReview = stateOf(lab, "Review", "started", 45000);
/** A time after the fixtures', so that nerve's answer to a change can be told from the request. */
const LATER = "2026-10-08T09:00:00Z";

/** The store of a tab at web's address, whose caller's acme and beta nerve listed with web and lab: it fetched web's. */
async function loaded() {
  const { nerve, api, router, workspaceRoot, projectRoot } = await projectTab(
    { workspace: acme, projects: [web] },
    { workspace: beta, projects: [lab] }
  );
  const store = new StateStore(fakeRoot({ router, workspaceRoot, projectRoot }), api);
  await answered(nerve, () => store.fetchProjectStates(web.id), ["GET", LIST], { data: listed }, "web's states");
  return { nerve, api, router, projects: projectRoot.project, store };
}

beforeEach(() => {
  vi.useFakeTimers({ now: 1_000_000 });
});
afterEach(() => {
  vi.useRealTimers();
});

describe("StateStore, the workspace's list and another workspace's", () => {
  it("makes the changes of a state of his other workspace on its project's list and its workspace's", async () => {
    const { nerve, router, store } = await loaded();
    const labs = [labTodo, labDoing, labReview];
    const BETA = "/api/v0/workspaces/beta/states";
    await answered(nerve, () => store.fetchWorkspaceStates(beta), ["GET", BETA], { data: labs }, "beta's states");
    const LAB = `/api/v0/projects/${lab.id}/states`;
    await answered(nerve, () => store.fetchProjectStates(lab.id), ["GET", LAB], { data: labs }, "lab's states");
    const renamed: State = { ...labDoing, name: "In progress", updated_at: LATER };
    const rename = () => store.updateState(labDoing.id, { name: "In progress" });
    await sent(nerve, rename, ["PATCH", at(labDoing)], json(200, renamed));
    // Review before Doing, among lab's started states: a step before Doing's 30000, whatever web's are
    const moved: State = { ...labReview, sequence: 15000, updated_at: LATER };
    const move = () => store.moveState(labReview.id, "started", labDoing.id, false);
    await sent(nerve, move, ["PATCH", at(labReview)], json(200, moved));
    expect(nerve.calls.at(-1)?.body).toEqual({ group: "started", sequence: 15000 });
    expect(store.getProjectStates(lab.id)).toEqual([labTodo, moved, renamed]);
    await sent(nerve, () => store.deleteState(labReview.id), ["DELETE", at(labReview)], noContent());
    await sent(
      nerve,
      () => store.markStateAsDefault(labDoing.id),
      ["POST", `${at(labDoing)}/mark-default`],
      noContent()
    );

    const shown = [
      { ...labTodo, default: false },
      { ...renamed, default: true },
    ];
    expect(store.getProjectStates(lab.id)).toEqual(shown);
    // the last of lab's started group, whatever web's has
    expect(store.getStatePercentageInGroup(labDoing.id)).toBe(100);
    expect(store.getProjectStates(web.id)).toEqual(listed);
    router.setQuery({ workspaceSlug: beta.slug, projectId: lab.id });
    expect(store.workspaceStates).toEqual(shown);
  });

  it("keeps a state created during a fetch of the workspace's states once, the list read after the creation", async () => {
    const { nerve, store } = await loaded();
    const fetched = track(store.fetchWorkspaceStates(acme));
    await until(() => nerve.calls.length === 2, "acme's states");
    await sent(nerve, () => store.createState(web.id, blocking), ["POST", LIST], json(201, blocked));
    nerve.calls[1]?.answer(json(200, { data: [...listed, blocked] }));
    await until(() => fetched.settled, "acme's states");
    expect(store.workspaceStates).toEqual([todo, doing, review, blocked, done]);
  });

  it("keeps the workspace's states it had when nerve refuses a refetch, and when the session changes as it fetches", async () => {
    const { nerve, api, store } = await loaded();
    await answered(nerve, () => store.fetchWorkspaceStates(acme), ["GET", ACME], { data: listed }, "acme's states");
    const refused = track(store.fetchWorkspaceStates(acme));
    await until(() => nerve.calls.length === 3, "the refetch");
    nerve.calls[2]?.answer(problem(503, "server_busy"));
    await until(() => refused.settled, "the refusal");
    expect(refused.error).toBeInstanceOf(ApiError);
    expect(store.workspaceStates).toEqual(listed);

    FakeNerve.replaceSession(api);
    const cut = await settle(store.fetchWorkspaceStates(acme), "the refetch the session cut");
    expect(cut).toEqual({ settled: true, value: undefined });
    expect(store.workspaceStates).toEqual(listed);
  });
});

describe("StateStore, a change in its turn", () => {
  it("fails in its turn, asking nerve nothing, for a change queued behind its state's deletion, or of a project he left", async () => {
    const { nerve, projects, store } = await loaded();
    // made while Review is held: the deletion and the creation go out one at a time, and the three changes behind
    // them find in their turn that Review is gone
    const deleted = track(store.deleteState(review.id));
    const created = track(store.createState(web.id, blocking));
    const behind = [
      track(store.moveState(review.id, "started", doing.id, false)),
      track(store.markStateAsDefault(review.id)),
      track(store.deleteState(review.id)),
    ];
    await inTurn(nerve, 1, ["DELETE", at(review)], noContent());
    await inTurn(nerve, 2, ["POST", LIST], json(201, blocked));
    await until(() => behind.every((change) => change.settled), "the changes behind them");
    expect([deleted.error, created.value]).toEqual([undefined, blocked]);
    const notFound = new Error("State not found");
    expect(behind.map((change) => change.error)).toEqual([notFound, notFound, notFound]);

    await sent(nerve, () => projects.leaveProject(web), ["POST", `/api/v0/projects/${web.id}/leave`], noContent());
    const left = await settle(store.markStateAsDefault(todo.id), "a change of a state of web");
    expect(left).toMatchObject({ settled: true, error: notFound });
    expect(nerve.calls).toHaveLength(4);
  });

  it("reckons a move from the states nerve gave, not from a refused move's request: two moves in a row", async () => {
    const { nerve, store } = await loaded();
    // Done last of the started group asks for 60000, a step past Review's 45000, and nerve refuses it: Done stays the
    // completed group's. Doing last of the started group then asks for 60000 too (from the refused request, 75000)
    const doingLast: State = { ...doing, sequence: 60000, updated_at: LATER };
    const first = track(store.moveState(done.id, "started", undefined, false));
    const second = track(store.moveState(doing.id, "started", undefined, false));
    await inTurn(nerve, 1, ["PATCH", at(done)], problem(409, "project.state_last_in_group"));
    await inTurn(nerve, 2, ["PATCH", at(doing)], json(200, doingLast));
    await until(() => second.settled, "the second move");
    expect(first.error).toBeInstanceOf(ApiError);
    expect(second.value).toEqual(doingLast);
    expect(nerve.calls.slice(1).map((call) => call.body)).toEqual([
      { group: "started", sequence: 60000 },
      { group: "started", sequence: 60000 },
    ]);
    expect(store.getProjectStates(web.id)).toEqual([todo, review, doingLast, done]);
  });
});
