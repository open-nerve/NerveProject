/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import type { CommentReaction, IssueComment } from "@nerve/api-client";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { runInAction } from "mobx";
import { accountOf } from "@/store/member/fake-users";
import { json, noContent, problem } from "@/lib/auth/fake-nerve";
import { until, track } from "@/lib/auth/fake-time";
import { heldChange } from "@/lib/auth/fake-tab";
import { SessionChangedError } from "@/lib/auth/token-manager";
import { loadProjects } from "@/store/project/fake-projects";
import { issueCommentOf, issueStoreTab, requestedIssue } from "../fake-issues";
vi.stubGlobal("localStorage", { getItem: () => null, setItem: () => {}, removeItem: () => {} });
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: {}, publicClient: {} }));
vi.mock("@/lib/store-context", () => ({ rootStore: undefined }));
let tab: Awaited<ReturnType<typeof issueStoreTab>>;
const store = () => tab.app.issue.issueDetail.comment;
const reactions = () => tab.app.issue.issueDetail.commentReaction;
const read = (incremental = false) => store().fetchComments("ws", "A", "x", incremental);
const route = "/api/v0/issues/x/comments";
const row = issueCommentOf("c-z");
beforeEach(async () => {
  vi.useFakeTimers();
  tab = await issueStoreTab();
  vi.spyOn(store(), "refreshAfterChange").mockResolvedValue();
  runInAction(() => {
    tab.app.user.data = accountOf("actor");
  });
});
afterEach(() => {
  tab.app.dispose();
  vi.restoreAllMocks();
  vi.useRealTimers();
});
async function load(data: IssueComment[] = [row]) {
  const sent = await requestedIssue(tab.nerve, read, ["GET", route]);
  sent.call.answer(json(200, { data, next_cursor: null }));
  await sent.promise;
}
it("atomically reads all pages and resumes from the last requested cursor", async () => {
  const sent = await requestedIssue(tab.nerve, read, ["GET", route]);
  sent.call.answer(json(200, { data: [row], next_cursor: "second" }));
  await until(() => tab.nerve.to(route).length === 2 || sent.result.settled, "second comment page");
  expect(tab.nerve.to(route)).toHaveLength(2);
  expect(store().getCommentsByIssueId("x")).toBeUndefined();
  tab.nerve.to(route)[1]!.answer(json(200, { data: [issueCommentOf("c-a")], next_cursor: null }));
  await sent.promise;
  expect(store().getCommentsByIssueId("x")).toEqual(["c-a", "c-z"]);
  const again = await requestedIssue(tab.nerve, () => read(true), ["GET", route]);
  expect(again.call.queryEntries).toEqual([["cursor", "second"]]);
  again.call.answer(
    json(200, {
      data: [{ ...issueCommentOf("c-a"), comment_html: "changed", updated_at: "2026-10-30T01:00:00Z" }],
      next_cursor: null,
    })
  );
  await again.promise;
  expect(store().getCommentById("x", "c-a")?.comment_html).toBe("changed");
  expect(store().getCommentsByIssueId("x")).toEqual(["c-a", "c-z"]);
  expect(tab.nerve.calls.every((call) => call.path === route)).toBe(true);
});
it.each([false, true])("a page-two refusal preserves the last complete value (previous=%s)", async (previous) => {
  if (previous) await load();
  const sent = await requestedIssue(tab.nerve, read, ["GET", route]);
  sent.call.answer(json(200, { data: [issueCommentOf("partial")], next_cursor: "later" }));
  await until(() => tab.nerve.to(route).length === (previous ? 3 : 2) || sent.result.settled, "failed second page");
  tab.nerve.to(route).at(-1)!.answer(problem(500, "unavailable"));
  await expect(sent.promise).rejects.toMatchObject({ status: 500 });
  expect(store().getCommentsByIssueId("x")).toEqual(previous ? [row.id] : undefined);
  expect(store().getError("x")).toMatchObject({ status: 500 });
  expect(store().isLoading("x")).toBe(false);
});
it("a full reopen removes missing comments and keeps nanosecond order", async () => {
  await load();
  await load([
    { ...issueCommentOf("a"), created_at: "2026-10-30T00:00:00.000002Z" },
    { ...issueCommentOf("z"), created_at: "2026-10-30T00:00:00.000001Z" },
  ]);
  expect(store().getCommentsByIssueId("x")).toEqual(["z", "a"]);
  expect(store().getCommentById("x", row.id)).toBeUndefined();
});
it("a confirmed edit never replaces a newer comment version already read", async () => {
  await load([{ ...row, comment_html: "newer", updated_at: "2026-10-30T00:00:00.123457Z" }]);
  const sent = await requestedIssue(
    tab.nerve,
    () => store().updateComment("ws", "A", "x", row.id, { comment_html: "requested" }),
    ["PATCH", `/api/v0/issue-comments/${row.id}`]
  );
  sent.call.answer(json(200, row));
  await sent.promise;
  expect(store().getCommentById("x", row.id)?.comment_html).toBe("newer");
});
it.each(["issue", "reaction"] as const)("rejects a mismatched %s scope before exposing the comment", async (field) => {
  const sent = await requestedIssue(tab.nerve, read, ["GET", route]);
  const wrong =
    field === "issue"
      ? { ...row, issue_id: "other" }
      : { ...row, reactions: [{ ...reaction("wrong", "actor"), comment_id: "other" }] };
  sent.call.answer(json(200, { data: [wrong], next_cursor: null }));
  await expect(sent.promise).rejects.toBeInstanceOf(SessionChangedError);
  expect(store().getCommentsByIssueId("x")).toBeUndefined();
});
const kinds = ["create", "update", "delete"] as const;
function write(kind: (typeof kinds)[number]): Promise<IssueComment | void> {
  if (kind === "create") return store().createComment("ws", "A", "x", { comment_html: "new" });
  if (kind === "update") return store().updateComment("ws", "A", "x", row.id, { comment_html: "new" });
  return store().removeComment("ws", "A", "x", row.id);
}
const endpoint = (kind: (typeof kinds)[number]): ["POST" | "PATCH" | "DELETE", string] =>
  kind === "create" ? ["POST", route] : [kind === "update" ? "PATCH" : "DELETE", `/api/v0/issue-comments/${row.id}`];
it.each(kinds)("%s changes no held comment before confirmation or after refusal", async (kind) => {
  await load();
  const sent = await requestedIssue(tab.nerve, () => write(kind), endpoint(kind));
  expect(store().getCommentById("x", row.id)).toEqual(row);
  sent.call.answer(problem(403, "forbidden"));
  await expect(sent.promise).rejects.toMatchObject({ status: 403 });
  expect(store().getCommentsByIssueId("x")).toEqual([row.id]);
  expect(store().getCommentById("x", row.id)).toEqual(row);
});
it.each(kinds)("%s is replayed over an older in-flight full read", async (kind) => {
  await load();
  const old = await requestedIssue(tab.nerve, read, ["GET", route]);
  const sent = await requestedIssue(tab.nerve, () => write(kind), endpoint(kind));
  const answer = {
    ...row,
    id: kind === "create" ? "new" : row.id,
    comment_html: "confirmed",
    updated_at: "2026-10-30T01:00:00Z",
  };
  sent.call.answer(kind === "delete" ? noContent() : json(kind === "create" ? 201 : 200, answer));
  await sent.promise;
  old.call.answer(json(200, { data: [row], next_cursor: null }));
  await old.promise;
  expect(store().getCommentsByIssueId("x")).toEqual(
    kind === "delete" ? [] : kind === "create" ? [row.id, "new"] : [row.id]
  );
  if (kind !== "delete") expect(store().getCommentById("x", answer.id)).toEqual(answer);
});
function reaction(id: string, actorId: string): CommentReaction {
  return { id, comment_id: row.id, actor_id: actorId, reaction: "128077", created_at: row.created_at };
}
it("uses only embedded reactions, confirms idempotently, and removes only the current actor", async () => {
  const mine = reaction("mine", tab.app.issue.currentUserId!);
  const theirs = reaction("theirs", "someone-else");
  await load([{ ...row, reactions: [mine, theirs] }]);
  expect(reactions().getCommentReactionsByCommentId("x", row.id)).toEqual({ "128077": ["mine", "theirs"] });
  expect(tab.nerve.calls).toHaveLength(1);
  const old = await requestedIssue(tab.nerve, read, ["GET", route]);
  const add = await requestedIssue(
    tab.nerve,
    () => reactions().createCommentReaction("ws", "A", "x", row.id, "128077"),
    ["POST", `/api/v0/issue-comments/${row.id}/reactions`]
  );
  add.call.answer(json(201, mine));
  await add.promise;
  expect(store().getCommentById("x", row.id)?.reactions).toEqual([mine, theirs]);
  const remove = await requestedIssue(
    tab.nerve,
    () => reactions().removeCommentReaction("ws", "A", "x", row.id, "128077"),
    ["DELETE", `/api/v0/issue-comments/${row.id}/reactions/128077`]
  );
  expect(store().getCommentById("x", row.id)?.reactions).toEqual([mine, theirs]);
  remove.call.answer(noContent());
  await remove.promise;
  old.call.answer(json(200, { data: [{ ...row, reactions: [mine, theirs] }], next_cursor: null }));
  await old.promise;
  expect(store().getCommentById("x", row.id)?.reactions).toEqual([theirs]);
  expect(reactions().getCommentReactionById("x", row.id, "mine")).toBeUndefined();
});
it("reaction refusal preserves the embedded record and does not request a fake reaction collection", async () => {
  const mine = reaction("mine", tab.app.issue.currentUserId!);
  await load([{ ...row, reactions: [mine] }]);
  const sent = await requestedIssue(
    tab.nerve,
    () => reactions().removeCommentReaction("ws", "A", "x", row.id, "128077"),
    ["DELETE", `/api/v0/issue-comments/${row.id}/reactions/128077`]
  );
  sent.call.answer(problem(422, "validation_failed"));
  await expect(sent.promise).rejects.toMatchObject({ status: 422 });
  expect(store().getCommentById("x", row.id)?.reactions).toEqual([mine]);
  expect(tab.nerve.calls).toHaveLength(2);
});
it("a delayed reaction does not resurrect a comment removed by a full read", async () => {
  await load();
  const sent = await requestedIssue(
    tab.nerve,
    () => reactions().createCommentReaction("ws", "A", "x", row.id, "128077"),
    ["POST", `/api/v0/issue-comments/${row.id}/reactions`]
  );
  await load([]);
  sent.call.answer(json(201, reaction("new", tab.app.issue.currentUserId!)));
  await sent.promise;
  expect(store().getCommentsByIssueId("x")).toEqual([]);
  expect(reactions().getCommentReactionsByCommentId("x", row.id)).toBeUndefined();
});
it("permission ABA revokes held comments and every outstanding page", async () => {
  await load();
  const old = await requestedIssue(tab.nerve, read, ["GET", route]);
  await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, []);
  expect(store().getCommentsByIssueId("x")).toBeUndefined();
  await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [tab.project]);
  old.call.answer(json(200, { data: [row], next_cursor: "forbidden-next" }));
  await expect(old.promise).rejects.toBeInstanceOf(SessionChangedError);
  expect(tab.nerve.to(route)).toHaveLength(2);
  expect(store().getCommentById("x", row.id)).toBeUndefined();
});
it("comments and their reactions share the parent issue queue and cannot write after retirement", async () => {
  await load();
  const gate = heldChange<void>();
  const holding = tab.app.issue.enqueueIssueChanges(["x"], () => gate.sent);
  const updating = track(write("update"));
  const adding = track(reactions().createCommentReaction("ws", "A", "x", row.id, "128077"));
  await vi.advanceTimersByTimeAsync(10);
  expect(tab.nerve.calls).toHaveLength(1);
  tab.app.issue.retire();
  gate.answer();
  await holding;
  await until(() => updating.settled && adding.settled, "retired collaboration queue");
  expect(updating.error).toBeInstanceOf(SessionChangedError);
  expect(adding.error).toBeInstanceOf(SessionChangedError);
  expect(tab.nerve.calls).toHaveLength(1);
});
