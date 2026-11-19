import type { components } from "@nerve/api-client";
import { expect } from "@playwright/test";

import { answer, type Api } from "./api";
import { bearer } from "./auth";
import type { Issue } from "./issues";

// The intake's calls of the stories (M7 design 3.5; P2a spec 2.4, 2.5): a work item submitted to a project's intake,
// and its triage.

type IntakeIssueCreate = components["schemas"]["IntakeIssueCreate"];
export type IntakeTriage = components["schemas"]["IntakeTriage"];

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
