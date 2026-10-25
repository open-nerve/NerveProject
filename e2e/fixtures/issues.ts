import type { components, operations } from "@nerve/api-client";
import { expect } from "@playwright/test";

import { answer, type Api } from "./api";
import { bearer } from "./auth";

// The work items' calls of the stories (M4 design 2).

export type Issue = components["schemas"]["IssueDetail"];
export type IssueCreate = components["schemas"]["IssueCreate"];
export type IssueUpdate = components["schemas"]["IssueUpdate"];
export type IssueQuery = NonNullable<operations["listIssues"]["parameters"]["query"]>;
export type MemberIssueQuery = operations["listMemberIssues"]["parameters"]["query"];
export type IssuePage = components["schemas"]["IssueListPage"];
type IssueRelation = components["schemas"]["IssueRelation"];
type IssueRelationsCreate = components["schemas"]["IssueRelationsCreate"];

/** The real project's list response, retaining refusal fields for negative stories. */
export async function issuesListed(api: Api, token: string, projectId: string, query: IssueQuery = {}) {
  const { data, error, response } = await api.GET("/api/v0/projects/{project_id}/issues", {
    params: { path: { project_id: projectId }, query },
    headers: bearer(token),
  });
  return data ? { status: response.status, page: data } : answer(response, error);
}

/** A successful project list; branch assertions belong to its caller. */
export async function listIssues(
  api: Api,
  token: string,
  projectId: string,
  query: IssueQuery = {}
): Promise<IssuePage> {
  const result = await issuesListed(api, token, projectId, query);
  expect(result.status, JSON.stringify(result)).toBe(200);
  if (!("page" in result)) throw new Error("listIssues answered without a page");
  return result.page;
}

/** The target account's issues, always scoped by the caller. */
export async function memberIssuesListed(
  api: Api,
  token: string,
  slug: string,
  userId: string,
  query: MemberIssueQuery
) {
  const { data, error, response } = await api.GET("/api/v0/workspaces/{slug}/members/{user_id}/issues", {
    params: { path: { slug, user_id: userId }, query },
    headers: bearer(token),
  });
  return data ? { status: response.status, page: data } : answer(response, error);
}

export async function listMemberIssues(
  api: Api,
  token: string,
  slug: string,
  userId: string,
  query: MemberIssueQuery
): Promise<IssuePage> {
  const result = await memberIssuesListed(api, token, slug, userId, query);
  expect(result.status, JSON.stringify(result)).toBe(200);
  if (!("page" in result)) throw new Error("listMemberIssues answered without a page");
  return result.page;
}

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

/** The answer of GET /api/v0/issues/{issue_id} by the caller of token: its status, the work item, or the problem's code. */
export async function issueRead(api: Api, token: string, issueId: string) {
  const { data, error, response } = await api.GET("/api/v0/issues/{issue_id}", {
    params: { path: { issue_id: issueId } },
    headers: bearer(token),
  });
  return data ? { status: response.status, issue: data } : answer(response, error);
}

/**
 * The answer of PATCH /api/v0/issues/{issue_id} by the caller of token: its status, the work item, or the problem's
 * code and fields.
 */
export async function issueUpdated(api: Api, token: string, issueId: string, body: IssueUpdate) {
  const { data, error, response } = await api.PATCH("/api/v0/issues/{issue_id}", {
    params: { path: { issue_id: issueId } },
    body,
    headers: bearer(token),
  });
  return data ? { status: response.status, issue: data } : answer(response, error);
}

/** Changes the work item of issueId with the bearer token given, a member's of its project, and returns it. */
export async function updateIssue(api: Api, token: string, issueId: string, body: IssueUpdate): Promise<Issue> {
  const updated = await issueUpdated(api, token, issueId, body);
  expect(updated.status, `change the work item ${issueId}: ${JSON.stringify(updated)}`).toBe(200);
  if (!("issue" in updated)) {
    throw new Error(`updateIssue ${issueId} answered 200 without the work item`);
  }
  return updated.issue;
}

/**
 * The answer of POST /api/v0/issues/{issue_id}/archive by the caller of token: its status, the work item, or the
 * problem's code.
 */
export async function issueArchived(api: Api, token: string, issueId: string) {
  const { data, error, response } = await api.POST("/api/v0/issues/{issue_id}/archive", {
    params: { path: { issue_id: issueId } },
    headers: bearer(token),
  });
  return data ? { status: response.status, issue: data } : answer(response, error);
}

/** The answer of POST /api/v0/issues/{issue_id}/unarchive by the caller of token, as issueArchived's. */
export async function issueUnarchived(api: Api, token: string, issueId: string) {
  const { data, error, response } = await api.POST("/api/v0/issues/{issue_id}/unarchive", {
    params: { path: { issue_id: issueId } },
    headers: bearer(token),
  });
  return data ? { status: response.status, issue: data } : answer(response, error);
}

/** The answer of DELETE /api/v0/issues/{issue_id} by the caller of token: its status, or the problem's code. */
export async function issueDeleted(api: Api, token: string, issueId: string) {
  const { error, response } = await api.DELETE("/api/v0/issues/{issue_id}", {
    params: { path: { issue_id: issueId } },
    headers: bearer(token),
  });
  return answer(response, error);
}

/**
 * The answer of POST /api/v0/issues/{issue_id}/sub-issues by the caller of token, putting the work items of
 * subIssueIds under the work item of issueId: its status, the work items that moved under it, or the problem's code
 * and fields.
 */
export async function subIssuesAdded(api: Api, token: string, issueId: string, subIssueIds: string[]) {
  const { data, error, response } = await api.POST("/api/v0/issues/{issue_id}/sub-issues", {
    params: { path: { issue_id: issueId } },
    body: { sub_issue_ids: subIssueIds },
    headers: bearer(token),
  });
  return data ? { status: response.status, issues: data.data } : answer(response, error);
}

/**
 * The answer of POST /api/v0/issues/{issue_id}/relations by the caller of token: its status, the relations made, as
 * the work item reads them, or the problem's code and fields.
 */
export async function issueRelationsCreated(api: Api, token: string, issueId: string, body: IssueRelationsCreate) {
  const { data, error, response } = await api.POST("/api/v0/issues/{issue_id}/relations", {
    params: { path: { issue_id: issueId } },
    body,
    headers: bearer(token),
  });
  return data ? { status: response.status, relations: data.data } : answer(response, error);
}

/**
 * Relates the work item of issueId with the bearer token given, a member's of its project, to the work items body
 * names, as its type, and returns the relations made, as the work item reads them.
 */
export async function createIssueRelations(
  api: Api,
  token: string,
  issueId: string,
  body: IssueRelationsCreate
): Promise<IssueRelation[]> {
  const created = await issueRelationsCreated(api, token, issueId, body);
  expect(created.status, `relate the work item ${issueId}: ${JSON.stringify(created)}`).toBe(201);
  if (!("relations" in created)) {
    throw new Error(`createIssueRelations ${issueId} answered 201 without the relations`);
  }
  return created.relations;
}

/** The relations of the work item of issueId, as it reads them, that the caller of token, who sees it, lists. */
export async function listIssueRelations(api: Api, token: string, issueId: string): Promise<IssueRelation[]> {
  const { data, error, response } = await api.GET("/api/v0/issues/{issue_id}/relations", {
    params: { path: { issue_id: issueId } },
    headers: bearer(token),
  });
  expect(response.status, `the relations of ${issueId}: ${JSON.stringify(error)}`).toBe(200);
  if (!data) {
    throw new Error(`listIssueRelations ${issueId} answered 200 without the relations`);
  }
  return data.data;
}

/** The answer of DELETE /api/v0/issue-relations/{relation_id} by the caller of token: its status, or the problem's code. */
export async function issueRelationDeleted(api: Api, token: string, relationId: string) {
  const { error, response } = await api.DELETE("/api/v0/issue-relations/{relation_id}", {
    params: { path: { relation_id: relationId } },
    headers: bearer(token),
  });
  return answer(response, error);
}

/** Reads a page of visible activities in ascending position order. */
export async function activityPage(api: Api, token: string, issueId: string, cursor?: string, limit = 2) {
  const { data, error, response } = await api.GET("/api/v0/issues/{issue_id}/activities", {
    params: { path: { issue_id: issueId }, query: { limit, cursor } },
    headers: bearer(token),
  });
  expect(response.status, JSON.stringify(error)).toBe(200);
  if (!data) throw new Error("activities answered 200 without a page");
  return data;
}

/** Reads a page of saved descriptions, newest save first. */
export async function descriptionPage(api: Api, token: string, issueId: string, cursor?: string, limit = 2) {
  const { data, error, response } = await api.GET("/api/v0/issues/{issue_id}/description-versions", {
    params: { path: { issue_id: issueId }, query: { limit, cursor } },
    headers: bearer(token),
  });
  expect(response.status, JSON.stringify(error)).toBe(200);
  if (!data) throw new Error("description versions answered 200 without a page");
  return data;
}

/** Reads a historical description's HTML through its independent detail resource. */
export async function descriptionVersion(api: Api, token: string, versionId: string) {
  const { data, error, response } = await api.GET("/api/v0/issue-description-versions/{version_id}", {
    params: { path: { version_id: versionId } },
    headers: bearer(token),
  });
  expect(response.status, JSON.stringify(error)).toBe(200);
  if (!data) throw new Error("description version answered 200 without its detail");
  return data;
}
