import type { components } from "@nerve/api-client";
import { expect } from "@playwright/test";

import type { Api } from "./api";
import { bearer } from "./auth";
import type { Issue } from "./issues";

// The intake's calls of the stories (M7 design 3.5; P2a spec 2.4, 2.5): a work item submitted to a project's intake,
// and its triage.

type IntakeIssueCreate = components["schemas"]["IntakeIssueCreate"];
type IntakeTriage = components["schemas"]["IntakeTriage"];

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
  const { data, error, response } = await api.POST("/api/v0/projects/{project_id}/intake-issues", {
    params: { path: { project_id: projectId } },
    body,
    headers: bearer(token),
  });
  expect(response.status, `submit ${body.name} to the intake of ${projectId}: ${JSON.stringify(error)}`).toBe(201);
  if (!data) {
    throw new Error(`submitIntakeIssue ${body.name} answered 201 without the work item`);
  }
  return data;
}

/**
 * Triages the work item of issueId, in its project's intake, with the bearer token given, an admin's of the project,
 * as body says, and returns it: accepted, it is in the project's default state, out of the intake.
 */
export async function triageIntakeIssue(api: Api, token: string, issueId: string, body: IntakeTriage): Promise<Issue> {
  const { data, error, response } = await api.POST("/api/v0/intake-issues/{issue_id}/triage", {
    params: { path: { issue_id: issueId } },
    body,
    headers: bearer(token),
  });
  expect(response.status, `triage ${issueId} as ${body.status}: ${JSON.stringify(error)}`).toBe(200);
  if (!data) {
    throw new Error(`triageIntakeIssue ${issueId} answered 200 without the work item`);
  }
  return data;
}
