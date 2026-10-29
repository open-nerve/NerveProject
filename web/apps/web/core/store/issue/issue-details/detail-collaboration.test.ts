/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { runInAction } from "mobx";
import { json, problem } from "@/lib/auth/fake-nerve";
import { until, track } from "@/lib/auth/fake-time";
import { accountOf } from "@/store/member/fake-users";
import {
  issueActivityOf,
  issueCommentOf,
  issueDetailOf,
  issueLinkOf,
  issueReactionOf,
  issueStoreTab,
  requestedIssue,
} from "../fake-issues";
vi.stubGlobal("localStorage", { getItem: () => null, setItem: () => {}, removeItem: () => {} });
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: {}, publicClient: {} }));
vi.mock("@/lib/store-context", () => ({ rootStore: undefined }));
let tab: Awaited<ReturnType<typeof issueStoreTab>>;
const kinds = ["core", "link", "comment", "commentReaction", "issueReaction", "relation", "subIssues"] as const;
type Kind = (typeof kinds)[number];
const comment = issueCommentOf("comment");
const item = issueDetailOf("x");
const updated = { ...item, updated_at: "2026-10-30T02:00:00Z" };
const child = { ...issueDetailOf("child"), parent_id: "x", updated_at: updated.updated_at };
const afterComment = { ...comment, comment_html: "confirmed", updated_at: updated.updated_at };
const oldActivity = issueActivityOf("before");
const newActivity = { ...issueActivityOf("after"), created_at: updated.updated_at };
async function open() {
  const detail = tab.app.issue.issueDetail;
  const sent = await requestedIssue(tab.nerve, () => detail.activity.fetchActivities("ws", "A", "x"), [
    "GET",
    "/api/v0/issues/x/activities",
  ]);
  sent.call.answer(json(200, { data: [oldActivity], next_cursor: "activity-last" }));
  await until(() => tab.nerve.to("/api/v0/issues/x/activities").length === 2, "initial activity page");
  tab.nerve.to("/api/v0/issues/x/activities")[1]!.answer(json(200, { data: [], next_cursor: null }));
  await sent.promise;
  const comments = await requestedIssue(tab.nerve, () => detail.comment.fetchComments("ws", "A", "x"), [
    "GET",
    "/api/v0/issues/x/comments",
  ]);
  comments.call.answer(json(200, { data: [comment], next_cursor: "comment-last" }));
  await until(() => tab.nerve.to("/api/v0/issues/x/comments").length === 2, "initial comment page");
  tab.nerve.to("/api/v0/issues/x/comments")[1]!.answer(json(200, { data: [], next_cursor: null }));
  await comments.promise;
  tab.nerve.calls.length = 0;
}
function operate(kind: Kind): Promise<unknown> {
  const root = tab.app.issue;
  const detail = root.issueDetail;
  switch (kind) {
    case "core":
      return root.updateIssue("x", () => ({ name: "new" }));
    case "link":
      return detail.createLink("ws", "A", "x", { url: "https://example.com" });
    case "comment":
      return detail.updateComment("ws", "A", "x", comment.id, { comment_html: "confirmed" });
    case "commentReaction":
      return detail.createCommentReaction("ws", "A", "x", comment.id, "128077");
    case "issueReaction":
      return detail.createReaction("ws", "A", "x", "128077");
    case "relation":
      return detail.createRelation("ws", "A", "x", "blocking", ["child"]);
    case "subIssues":
      return detail.createSubIssues("ws", "A", "x", ["child"]);
  }
}
function replies(kind: Kind, readRefused = false, writeRefused = false) {
  tab.nerve.respond = (call) => {
    if (call.method === "GET") {
      if (call.path.endsWith("/activities"))
        call.answer(readRefused ? problem(500, "read-failed") : json(200, { data: [newActivity], next_cursor: null }));
      else if (call.path.endsWith("/comments")) call.answer(json(200, { data: [afterComment], next_cursor: null }));
      else if (call.path.endsWith("/states")) call.answer(json(200, { data: [] }));
      else if (call.path === "/api/v0/issues/x") call.answer(json(200, updated));
      else if (call.path === "/api/v0/issues/child") call.answer(json(200, child));
      return;
    }
    if (writeRefused) {
      call.answer(problem(403, "forbidden"));
      return;
    }
    const body =
      kind === "core"
        ? updated
        : kind === "link"
          ? issueLinkOf("link", "x")
          : kind === "comment"
            ? afterComment
            : kind === "issueReaction"
              ? issueReactionOf("r")
              : kind === "commentReaction"
                ? {
                    id: "r",
                    comment_id: comment.id,
                    actor_id: "u-actor",
                    reaction: "128077",
                    created_at: updated.updated_at,
                  }
                : kind === "relation"
                  ? {
                      data: [
                        { id: "relation", relation_type: "blocking", issue: child, created_at: updated.updated_at },
                      ],
                    }
                  : { data: [child] };
    call.answer(json(call.method === "POST" ? 201 : 200, body));
  };
}
beforeEach(async () => {
  vi.useFakeTimers();
  tab = await issueStoreTab();
  runInAction(() => {
    tab.app.user.data = accountOf("actor");
  });
  tab.app.issue.issues.acceptRead(item, tab.app.issue.captureRead());
  tab.app.issue.issues.acceptRead(issueDetailOf("child"), tab.app.issue.captureRead());
  await open();
});
afterEach(() => {
  tab.app.dispose();
  vi.useRealTimers();
});
it.each(kinds)("confirmed %s refreshes both opened collections using their last request cursors", async (kind) => {
  replies(kind);
  await operate(kind);
  const activities = tab.nerve.to("/api/v0/issues/x/activities");
  const comments = tab.nerve.to("/api/v0/issues/x/comments");
  expect(activities).toHaveLength(1);
  expect(comments).toHaveLength(1);
  expect(activities[0]!.queryEntries).toEqual([["cursor", "activity-last"]]);
  expect(comments[0]!.queryEntries).toEqual([["cursor", "comment-last"]]);
  expect(tab.app.issue.issueDetail.activity.getActivitiesByIssueId("x")).toEqual(["before", "after"]);
  expect(tab.app.issue.issueDetail.comment.getCommentById("x", comment.id)?.comment_html).toBe("confirmed");
  if (kind !== "commentReaction") expect(tab.app.issue.issues.getIssueById("x")?.updated_at).toBe(updated.updated_at);
});
it.each(kinds)("a refused %s sends no post-write reads", async (kind) => {
  replies(kind, false, true);
  await expect(operate(kind)).rejects.toMatchObject({ status: 403 });
  expect(tab.nerve.calls.filter((call) => call.method === "GET")).toHaveLength(0);
});
it.each(kinds)("a confirmed %s remains successful when an incremental activity read fails", async (kind) => {
  replies(kind, true);
  const command = track(operate(kind));
  await until(() => command.settled, "write and read settle");
  expect(command.error).toBeUndefined();
  expect(tab.app.issue.issueDetail.activity.getError("x")).toMatchObject({ status: 500 });
  expect(tab.app.issue.issueDetail.activity.getActivitiesByIssueId("x")).toEqual(["before"]);
  expect(tab.nerve.calls.filter((call) => call.method !== "GET")).toHaveLength(1);
});

it.each(["comment", "issueReaction"] as const)(
  "%s retains confirmation and retries only reads after the core authority fails",
  async (kind) => {
    replies(kind);
    const fallback = tab.nerve.respond!;
    tab.nerve.respond = (call) =>
      call.method === "GET" && call.path === "/api/v0/issues/x"
        ? call.answer(problem(500, "core-failed"))
        : fallback(call);
    await operate(kind);
    const detail = tab.app.issue.issueDetail;
    const resource = kind === "comment" ? detail.comment : detail.reaction;
    expect(resource.getError("x")).toMatchObject({ status: 500 });
    if (kind === "comment") expect(detail.comment.getCommentById("x", comment.id)?.comment_html).toBe("confirmed");
    else expect(detail.reaction.getReactionsByIssueId("x")).toEqual({ "128077": ["r"] });
    expect(tab.app.issue.issues.getIssueById("x")).toBeUndefined();
    const before = tab.nerve.calls.length;
    replies(kind);
    await resource.retry("ws", "A", "x");
    expect(resource.getError("x")).toBeUndefined();
    expect(tab.app.issue.issues.getIssueById("x")?.updated_at).toBe(updated.updated_at);
    expect(tab.nerve.calls.slice(before).every((call) => call.method === "GET")).toBe(true);
    expect(tab.nerve.calls.filter((call) => call.method !== "GET")).toHaveLength(1);
  }
);

it("a comment reaction cannot erase a pending core-read failure from the prior comment write", async () => {
  replies("comment");
  const fallback = tab.nerve.respond!;
  tab.nerve.respond = (call) =>
    call.method === "GET" && call.path === "/api/v0/issues/x"
      ? call.answer(problem(500, "core-failed"))
      : fallback(call);
  await operate("comment");
  expect(tab.app.issue.issues.getIssueById("x")).toBeUndefined();
  replies("commentReaction");
  await operate("commentReaction");
  expect(tab.app.issue.issues.getIssueById("x")?.updated_at).toBe(updated.updated_at);
  expect(tab.nerve.to("/api/v0/issues/x")).toHaveLength(2);
  expect(tab.app.issue.issueDetail.comment.getError("x")).toBeUndefined();
});
