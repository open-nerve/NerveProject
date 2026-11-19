import type { components, operations } from "@nerve/api-client";
import { expect } from "@playwright/test";

import { answer, type Api } from "./api";
import { bearer } from "./auth";
import type { Issue } from "./issues";

// The intake's calls of the stories (M7 design 3.5; P2a spec 2.4, 2.5; P2b spec 2.3–2.6): a work item submitted to a
// project's intake, its triage, its change and its deletion; a project's intake listed, and a workspace's intakes
// counted.

type IntakeIssueCreate = components["schemas"]["IntakeIssueCreate"];
export type IntakeTriage = components["schemas"]["IntakeTriage"];
type IntakeIssueUpdate = components["schemas"]["IntakeIssueUpdate"];
type IntakeIssuePage = components["schemas"]["IntakeIssuePage"];
type IntakeIssueQuery = NonNullable<operations["listIntakeIssues"]["parameters"]["query"]>;
type IntakeCounts = components["schemas"]["IntakeCounts"];

/**
 * The answer of POST /api/v0/projects/{project_id}/intake-issues by the caller of token: its status, the work item, or
 * the problem's code and fields.
 */
export async function intakeIssueSubmitted(api: Api, token: string, projectId: string, body: IntakeIssueCreate) {
  const { data, error, response } = await api.POST("/api/v0/projects/{project_id}/intake-issues", {
    params: { path: { project_id: projectId } },
    body,
    headers: bearer(token),
  });
  return data ? { status: response.status, issue: data } : answer(response, error);
}

/**
 * Submits a work item to the intake of the project of projectId with the bearer token given, a member's or a guest's
 * of it while its intake is turned on, and returns it: in the project's triage state, its intake row pending.
 */
export async function submitIntakeIssue(
  api: Api,
  token: string,
  projectId: string,
  body: IntakeIssueCreate
): Promise<Issue> {
  const submitted = await intakeIssueSubmitted(api, token, projectId, body);
  expect(submitted.status, `submit ${body.name} to the intake of ${projectId}: ${JSON.stringify(submitted)}`).toBe(201);
  if (!("issue" in submitted)) {
    throw new Error(`submitIntakeIssue ${body.name} answered 201 without the work item`);
  }
  return submitted.issue;
}

/**
 * The answer of POST /api/v0/intake-issues/{issue_id}/triage by the caller of token: its status, the work item, or the
 * problem's code and fields.
 */
export async function intakeIssueTriaged(api: Api, token: string, issueId: string, body: IntakeTriage) {
  const { data, error, response } = await api.POST("/api/v0/intake-issues/{issue_id}/triage", {
    params: { path: { issue_id: issueId } },
    body,
    headers: bearer(token),
  });
  return data ? { status: response.status, issue: data } : answer(response, error);
}

/**
 * Triages the work item of issueId, in its project's intake, with the bearer token given, an admin's of the project,
 * as body says, and returns it: accepted, it is in the project's default state, out of the intake.
 */
export async function triageIntakeIssue(api: Api, token: string, issueId: string, body: IntakeTriage): Promise<Issue> {
  const triaged = await intakeIssueTriaged(api, token, issueId, body);
  expect(triaged.status, `triage ${issueId} as ${body.status}: ${JSON.stringify(triaged)}`).toBe(200);
  if (!("issue" in triaged)) {
    throw new Error(`triageIntakeIssue ${issueId} answered 200 without the work item`);
  }
  return triaged.issue;
}

/**
 * The page of the intake of the project of projectId that query asks for, as the caller of token lists it, an admin, a
 * member or a guest of the project: its arrays go as CSV alternatives, one parameter each (the generated client's list
 * serialization).
 */
export async function listIntakeIssues(
  api: Api,
  token: string,
  projectId: string,
  query: IntakeIssueQuery = {}
): Promise<IntakeIssuePage> {
  const { data, error, response } = await api.GET("/api/v0/projects/{project_id}/intake-issues", {
    params: { path: { project_id: projectId }, query },
    headers: bearer(token),
  });
  expect(response.status, `the intake of ${projectId} ${JSON.stringify(query)}: ${JSON.stringify(error)}`).toBe(200);
  if (!data) {
    throw new Error(`listIntakeIssues of ${projectId} answered 200 without a page`);
  }
  return data;
}

/**
 * The answer of PATCH /api/v0/intake-issues/{issue_id} by the caller of token: its status, the work item, or the
 * problem's code and fields.
 */
export async function intakeIssueUpdated(api: Api, token: string, issueId: string, body: IntakeIssueUpdate) {
  const { data, error, response } = await api.PATCH("/api/v0/intake-issues/{issue_id}", {
    params: { path: { issue_id: issueId } },
    body,
    headers: bearer(token),
  });
  return data ? { status: response.status, issue: data } : answer(response, error);
}

/** The answer of DELETE /api/v0/intake-issues/{issue_id} by the caller of token: its status, or the problem's code. */
export async function intakeIssueDeleted(api: Api, token: string, issueId: string) {
  const { error, response } = await api.DELETE("/api/v0/intake-issues/{issue_id}", {
    params: { path: { issue_id: issueId } },
    headers: bearer(token),
  });
  return answer(response, error);
}

/**
 * The pending work items of each intake of the workspace of slug that the caller of token counts, an admin, a member
 * or a guest of it: each project of his with its intake turned on, in id order.
 */
export async function intakeCounts(api: Api, token: string, slug: string): Promise<IntakeCounts> {
  const { data, error, response } = await api.GET("/api/v0/workspaces/{slug}/intake-counts", {
    params: { path: { slug } },
    headers: bearer(token),
  });
  expect(response.status, `the intake counts of ${slug}: ${JSON.stringify(error)}`).toBe(200);
  if (!data) {
    throw new Error(`getIntakeCounts of ${slug} answered 200 without the counts`);
  }
  return data;
}
