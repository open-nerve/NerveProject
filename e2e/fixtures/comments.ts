import type { components } from "@nerve/api-client";
import { expect } from "@playwright/test";

import type { Api } from "./api";
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
