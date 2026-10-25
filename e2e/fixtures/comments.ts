import type { components } from "@nerve/api-client";
import { expect } from "@playwright/test";

import { answer, type Api } from "./api";
import { bearer } from "./auth";

/** Creates a comment through the issue API, furnishing deletion stories with a real comment row. */
export async function createIssueComment(
  api: Api,
  token: string,
  issueId: string,
  html: string
): Promise<components["schemas"]["IssueComment"]> {
  const { data, error, response } = await api.POST("/api/v0/issues/{issue_id}/comments", {
    params: { path: { issue_id: issueId } },
    body: { comment_html: html },
    headers: bearer(token),
  });
  expect(response.status, `create a comment on ${issueId}: ${JSON.stringify(error)}`).toBe(201);
  if (!data) {
    throw new Error(`createIssueComment on ${issueId} answered 201 without a comment`);
  }
  return data;
}

/** Creates a live work item reaction through the API for deletion and archival stories. */
export async function createIssueReaction(api: Api, token: string, issueId: string): Promise<void> {
  const { error, response } = await api.POST("/api/v0/issues/{issue_id}/reactions", {
    params: { path: { issue_id: issueId } },
    body: { reaction: "128077" },
    headers: bearer(token),
  });
  expect(response.status, `create a reaction on ${issueId}: ${JSON.stringify(error)}`).toBe(201);
}

/** Creates a comment reaction through the API for directory-based stories. */
export async function createCommentReaction(api: Api, token: string, commentId: string): Promise<void> {
  const { error, response } = await api.POST("/api/v0/issue-comments/{comment_id}/reactions", {
    params: { path: { comment_id: commentId } },
    body: { reaction: "128077" },
    headers: bearer(token),
  });
  expect(response.status, `create a reaction on comment ${commentId}: ${JSON.stringify(error)}`).toBe(201);
}

/** Edits a comment as its author; refused calls return the common problem answer. */
export async function commentUpdated(api: Api, token: string, commentId: string, html: string) {
  const { data, error, response } = await api.PATCH("/api/v0/issue-comments/{comment_id}", {
    params: { path: { comment_id: commentId } },
    body: { comment_html: html },
    headers: bearer(token),
  });
  return data ? { status: response.status, comment: data } : answer(response, error);
}

/** Deletes a comment through the API, retaining a rejected operation's problem code. */
export async function commentDeleted(api: Api, token: string, commentId: string) {
  const { error, response } = await api.DELETE("/api/v0/issue-comments/{comment_id}", {
    params: { path: { comment_id: commentId } },
    headers: bearer(token),
  });
  return answer(response, error);
}

/** Reads one ascending page; callers retain the request cursor of the final page for incremental reads. */
export async function commentPage(api: Api, token: string, issueId: string, cursor?: string, limit = 2) {
  const { data, error, response } = await api.GET("/api/v0/issues/{issue_id}/comments", {
    params: { path: { issue_id: issueId }, query: { limit, cursor } },
    headers: bearer(token),
  });
  expect(response.status, JSON.stringify(error)).toBe(200);
  if (!data) throw new Error("comments answered 200 without a page");
  return data;
}
