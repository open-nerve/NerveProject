/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { ApiClient, CommentReaction, IssueCommentPage } from "@nerve/api-client";
import { FakeNerve, json, noContent, problem, type Endpoint } from "@/lib/auth/fake-nerve";
import { SessionChangedError } from "@/lib/auth/token-manager";
import { issueCommentOf, issueReactionOf, requestedIssue } from "@/store/issue/fake-issues";
import { IssueCommentService } from "./issue_comment.service";
import { IssueReactionService } from "./issue_reaction.service";

beforeEach(() => vi.useFakeTimers());
afterEach(() => vi.useRealTimers());
const comment = issueCommentOf("comment-z");
const reaction: CommentReaction = {
  id: "reaction-a",
  comment_id: comment.id,
  actor_id: "actor",
  reaction: "128077",
  created_at: comment.created_at,
};
const page: IssueCommentPage = { data: [comment], next_cursor: "next+/=" };
const cases: {
  name: string;
  run: (s: ApiClient) => Promise<unknown>;
  endpoint: Endpoint;
  body?: unknown;
  answer?: unknown;
  status: number;
  query?: [string, string][];
}[] = [
  {
    name: "list",
    run: (s) => new IssueCommentService(s).getIssueComments("x", { limit: 2, cursor: "next+/=" }),
    endpoint: ["GET", "/api/v0/issues/x/comments"],
    answer: page,
    status: 200,
    query: [
      ["limit", "2"],
      ["cursor", "next+/="],
    ],
  },
  {
    name: "create",
    run: (s) => new IssueCommentService(s).createIssueComment("x", { comment_html: "<p>new</p>" }),
    endpoint: ["POST", "/api/v0/issues/x/comments"],
    body: { comment_html: "<p>new</p>" },
    answer: comment,
    status: 201,
  },
  {
    name: "update",
    run: (s) => new IssueCommentService(s).patchIssueComment(comment.id, { comment_html: "<p>edit</p>" }),
    endpoint: ["PATCH", "/api/v0/issue-comments/comment-z"],
    body: { comment_html: "<p>edit</p>" },
    answer: comment,
    status: 200,
  },
  {
    name: "delete",
    run: (s) => new IssueCommentService(s).deleteIssueComment(comment.id),
    endpoint: ["DELETE", "/api/v0/issue-comments/comment-z"],
    status: 204,
  },
  {
    name: "add reaction",
    run: (s) => new IssueCommentService(s).addReaction(comment.id, { reaction: reaction.reaction }),
    endpoint: ["POST", "/api/v0/issue-comments/comment-z/reactions"],
    body: { reaction: reaction.reaction },
    answer: reaction,
    status: 201,
  },
  {
    name: "remove reaction",
    run: (s) => new IssueCommentService(s).removeReaction(comment.id, reaction.reaction),
    endpoint: ["DELETE", "/api/v0/issue-comments/comment-z/reactions/128077"],
    status: 204,
  },
  {
    name: "list issue reactions",
    run: (s) => new IssueReactionService(s).listIssueReactions("x"),
    endpoint: ["GET", "/api/v0/issues/x/reactions"],
    answer: { data: [issueReactionOf("reaction")] },
    status: 200,
  },
  {
    name: "add issue reaction",
    run: (s) => new IssueReactionService(s).createIssueReaction("x", { reaction: "128077" }),
    endpoint: ["POST", "/api/v0/issues/x/reactions"],
    body: { reaction: "128077" },
    answer: issueReactionOf("reaction"),
    status: 201,
  },
  {
    name: "remove issue reaction",
    run: (s) => new IssueReactionService(s).deleteIssueReaction("x", "128077"),
    endpoint: ["DELETE", "/api/v0/issues/x/reactions/128077"],
    status: 204,
  },
];
it.each(cases)(
  "uses the native collaboration contract: $name",
  async ({ run, endpoint, body, answer, status, query }) => {
    const nerve = new FakeNerve();
    const sent = await requestedIssue(nerve, () => run(nerve.client()), endpoint);
    expect(sent.call.body).toEqual(body);
    expect(sent.call.queryEntries).toEqual(query ?? []);
    sent.call.answer(status === 204 ? noContent() : json(status, answer));
    expect(await sent.promise).toEqual(answer);
    expect(nerve.calls).toHaveLength(1);
  }
);
it.each(
  cases.flatMap(({ name, run, endpoint }) => [403, 404, 409, 422].map((status) => ({ name, run, endpoint, status })))
)("preserves refusal status/code for $name / $status", async ({ run, endpoint, status }) => {
  const nerve = new FakeNerve();
  const sent = await requestedIssue(nerve, () => run(nerve.client()), endpoint);
  sent.call.answer(problem(status, "original.code"));
  await expect(sent.promise).rejects.toMatchObject({ status, problem: { code: "original.code" } });
});
it.each(cases)("preserves retired sessions for $name", async ({ run }) => {
  const nerve = new FakeNerve();
  const api = nerve.client();
  const error = new SessionChangedError();
  api.use({
    onRequest() {
      throw error;
    },
  });
  await expect(run(api)).rejects.toBe(error);
  expect(nerve.calls).toHaveLength(0);
});
