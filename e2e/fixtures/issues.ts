import type { components } from "@nerve/api-client";
import { expect } from "@playwright/test";

import { answer, type Api } from "./api";
import { bearer } from "./auth";

// The work items' calls of the stories (M4 design 2).

export type Issue = components["schemas"]["IssueDetail"];
export type IssueCreate = components["schemas"]["IssueCreate"];

/**
 * The answer of POST /api/v0/projects/{project_id}/issues by the caller of token: its status, the work item, or the
 * problem's code and fields.
 */
export async function issueCreated(api: Api, token: string, projectId: string, body: IssueCreate) {
  const { data, error, response } = await api.POST("/api/v0/projects/{project_id}/issues", {
    params: { path: { project_id: projectId } },
    body,
    headers: bearer(token),
  });
  return data ? { status: response.status, issue: data } : answer(response, error);
}

/** Creates a work item in the project of projectId with the bearer token given, a member's of it, and returns it. */
export async function createIssue(api: Api, token: string, projectId: string, body: IssueCreate): Promise<Issue> {
  const created = await issueCreated(api, token, projectId, body);
  expect(created.status, `create the work item ${body.name} in ${projectId}: ${JSON.stringify(created)}`).toBe(201);
  if (!("issue" in created)) {
    throw new Error(`createIssue ${body.name} answered 201 without the work item`);
  }
  return created.issue;
}
