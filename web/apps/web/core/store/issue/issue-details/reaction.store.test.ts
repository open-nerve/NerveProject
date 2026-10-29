/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import type { IssueReaction } from "@nerve/api-client";
import { runInAction } from "mobx";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { json, noContent, problem } from "@/lib/auth/fake-nerve";
import { until, track } from "@/lib/auth/fake-time";
import { heldChange } from "@/lib/auth/fake-tab";
import { SessionChangedError } from "@/lib/auth/token-manager";
import { accountOf } from "@/store/member/fake-users";
import { loadProjects } from "@/store/project/fake-projects";
import { issueReactionOf, issueStoreTab, requestedIssue } from "../fake-issues";
vi.stubGlobal("localStorage", { getItem: () => null, setItem: () => {}, removeItem: () => {} });
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: {}, publicClient: {} }));
vi.mock("@/lib/store-context", () => ({ rootStore: undefined }));
let tab: Awaited<ReturnType<typeof issueStoreTab>>;
const store = () => tab.app.issue.issueDetail.reaction;
const mine = issueReactionOf("mine");
const theirs = issueReactionOf("theirs", "x", "other");
const route = "/api/v0/issues/x/reactions";
const read = () => store().fetchReactions("ws", "A", "x");
beforeEach(async () => {
  vi.useFakeTimers();
  tab = await issueStoreTab();
  runInAction(() => {
    tab.app.user.data = accountOf("actor");
  });
});
afterEach(() => {
  tab.app.dispose();
  vi.restoreAllMocks();
  vi.useRealTimers();
});
async function load(data: IssueReaction[] = [mine, theirs]) {
  const sent = await requestedIssue(tab.nerve, read, ["GET", route]);
  sent.call.answer(json(200, { data }));
  await sent.promise;
}
it("keeps native actors and reads reactions only in the requested issue", async () => {
  await load();
  expect(store().getReactionsByIssueId("x")).toEqual({ "128077": [mine.id, theirs.id] });
  expect(store().getReactionById("x", mine.id)).toEqual(mine);
  expect(store().getReactionById("other", mine.id)).toBeUndefined();
  expect(store().reactionsByUser("x", mine.actor_id)).toEqual([mine]);
});
it("a read refusal exposes an error without replacing the complete collection", async () => {
  await load();
  const sent = await requestedIssue(tab.nerve, read, ["GET", route]);
  sent.call.answer(problem(500, "unavailable"));
  await expect(sent.promise).rejects.toMatchObject({ status: 500 });
  expect(store().getReactionsByIssueId("x")).toEqual({ "128077": [mine.id, theirs.id] });
  expect(store().getError("x")).toMatchObject({ status: 500 });
});
const kinds = ["add", "remove"] as const;
function change(kind: (typeof kinds)[number]): Promise<IssueReaction | void> {
  return kind === "add"
    ? store().createReaction("ws", "A", "x", "128077")
    : store().removeReaction("ws", "A", "x", "128077");
}
const endpoint = (kind: (typeof kinds)[number]): ["POST" | "DELETE", string] =>
  kind === "add" ? ["POST", route] : ["DELETE", route + "/128077"];
it.each(kinds)("%s waits for confirmation and leaves both actors after refusal", async (kind) => {
  await load();
  const sent = await requestedIssue(tab.nerve, () => change(kind), endpoint(kind));
  expect(store().getReactionsByIssueId("x")).toEqual({ "128077": [mine.id, theirs.id] });
  sent.call.answer(problem(403, "forbidden"));
  await expect(sent.promise).rejects.toMatchObject({ status: 403 });
  expect(store().getReactionsByIssueId("x")).toEqual({ "128077": [mine.id, theirs.id] });
  expect(tab.nerve.calls).toHaveLength(2);
});
it.each(kinds)("%s is idempotent and replays over an old GET without deleting another actor", async (kind) => {
  await load();
  const old = await requestedIssue(tab.nerve, read, ["GET", route]);
  const sent = await requestedIssue(tab.nerve, () => change(kind), endpoint(kind));
  sent.call.answer(kind === "add" ? json(201, mine) : noContent());
  await sent.promise;
  old.call.answer(json(200, { data: [mine, theirs] }));
  await old.promise;
  expect(store().getReactionsByIssueId("x")).toEqual({ "128077": kind === "add" ? [mine.id, theirs.id] : [theirs.id] });
  expect(store().getReactionById("x", theirs.id)).toEqual(theirs);
  expect(tab.nerve.calls).toHaveLength(3);
});
it("rejects a reaction collection from a different issue", async () => {
  const sent = await requestedIssue(tab.nerve, read, ["GET", route]);
  sent.call.answer(json(200, { data: [issueReactionOf("wrong", "other")] }));
  await expect(sent.promise).rejects.toBeInstanceOf(SessionChangedError);
  expect(store().getReactionsByIssueId("x")).toBeUndefined();
});
it.each(["read", "write"] as const)("permission ABA hides held reactions and discards an old %s", async (operation) => {
  await load();
  const old = await requestedIssue(
    tab.nerve,
    async () => (operation === "read" ? read() : change("add")),
    operation === "read" ? ["GET", route] : ["POST", route]
  );
  await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, []);
  expect(store().getReactionsByIssueId("x")).toBeUndefined();
  await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [tab.project]);
  old.call.answer(operation === "read" ? json(200, { data: [mine] }) : json(201, mine));
  await expect(old.promise).rejects.toBeInstanceOf(SessionChangedError);
  expect(store().getReactionById("x", mine.id)).toBeUndefined();
});
it("uses the issue queue and sends no queued reaction after retirement", async () => {
  const gate = heldChange<void>();
  const holding = tab.app.issue.enqueueIssueChanges(["x"], () => gate.sent);
  const writing = track(change("add"));
  await vi.advanceTimersByTimeAsync(10);
  expect(tab.nerve.calls).toHaveLength(0);
  tab.app.issue.retire();
  gate.answer();
  await holding;
  await until(() => writing.settled, "retired issue reaction");
  expect(writing.error).toBeInstanceOf(SessionChangedError);
  expect(tab.nerve.calls).toHaveLength(0);
});
