/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { CommentReaction, IssueCommentPage } from "@nerve/api-client";
import { FakeNerve, json, noContent, problem, type Endpoint } from "@/lib/auth/fake-nerve";
import { SessionChangedError } from "@/lib/auth/token-manager";
import { issueCommentOf, requestedIssue } from "@/store/issue/fake-issues";
import { IssueCommentService } from "./issue_comment.service";

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
  run: (s: IssueCommentService) => Promise<unknown>;
  endpoint: Endpoint;
  body?: unknown;
  answer?: unknown;
  status: number;
  query?: [string, string][];
}[] = [
  {
    name: "list",
    run: (s) => s.getIssueComments("x", { limit: 2, cursor: "next+/=" }),
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
    run: (s) => s.createIssueComment("x", { comment_html: "<p>new</p>" }),
    endpoint: ["POST", "/api/v0/issues/x/comments"],
    body: { comment_html: "<p>new</p>" },
    answer: comment,
    status: 201,
  },
  {
    name: "update",
    run: (s) => s.patchIssueComment(comment.id, { comment_html: "<p>edit</p>" }),
    endpoint: ["PATCH", "/api/v0/issue-comments/comment-z"],
    body: { comment_html: "<p>edit</p>" },
    answer: comment,
    status: 200,
  },
  {
    name: "delete",
    run: (s) => s.deleteIssueComment(comment.id),
    endpoint: ["DELETE", "/api/v0/issue-comments/comment-z"],
    status: 204,
  },
  {
    name: "add reaction",
    run: (s) => s.addReaction(comment.id, { reaction: reaction.reaction }),
    endpoint: ["POST", "/api/v0/issue-comments/comment-z/reactions"],
    body: { reaction: reaction.reaction },
    answer: reaction,
    status: 201,
  },
  {
    name: "remove reaction",
    run: (s) => s.removeReaction(comment.id, reaction.reaction),
    endpoint: ["DELETE", "/api/v0/issue-comments/comment-z/reactions/128077"],
    status: 204,
  },
];
it.each(cases)("uses the native comment contract: $name", async ({ run, endpoint, body, answer, status, query }) => {
  const nerve = new FakeNerve();
  const sent = await requestedIssue(nerve, () => run(new IssueCommentService(nerve.client())), endpoint);
  expect(sent.call.body).toEqual(body);
  expect(sent.call.queryEntries).toEqual(query ?? []);
  sent.call.answer(status === 204 ? noContent() : json(status, answer));
  expect(await sent.promise).toEqual(answer);
  expect(nerve.calls).toHaveLength(1);
});
it.each(
  cases.flatMap(({ name, run, endpoint }) => [403, 404, 409, 422].map((status) => ({ name, run, endpoint, status })))
)("preserves refusal status/code for $name / $status", async ({ run, endpoint, status }) => {
  const nerve = new FakeNerve();
  const sent = await requestedIssue(nerve, () => run(new IssueCommentService(nerve.client())), endpoint);
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
  await expect(run(new IssueCommentService(api))).rejects.toBe(error);
  expect(nerve.calls).toHaveLength(0);
});
