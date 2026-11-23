/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type {
  ApiClient,
  IntakeCounts,
  IntakeIssueCreate,
  IntakeIssuePage,
  IntakeIssueUpdate,
  IntakeTriage,
  IssueDetail,
  operations,
} from "@nerve/api-client";
import { unwrap } from "@/lib/api-error";

/**
 * The query of a page of a project's intake, as nerve takes it: each filter's values an array the client sends as one
 * CSV parameter (@nerve/api-client), a date one range of days, the order one of six values, the page size a number,
 * the cursor as nerve answered it (filters.ts listQuery).
 */
export type IntakeListQuery = NonNullable<operations["listIntakeIssues"]["parameters"]["query"]>;

/**
 * The intakes of the caller's projects (M7 design 3.5, 3.6; P5a spec 2.1): each operation of nerve's, its answer as
 * nerve gives it. The issue root store of a session builds it with the session's client (IssueRootStore.intakeApi).
 */
export class IntakeService {
  /** api: the client bound to the session of the stores that build this service. */
  constructor(private readonly api: ApiClient) {}

  /**
   * The pending work items of the intakes the caller counts in the workspace slug names: each project he is a member
   * of that is not archived and has its intake turned on, once, in id order, a snooze whose time has come counted as
   * pending (getIntakeCounts).
   */
  async counts(slug: string): Promise<IntakeCounts> {
    return unwrap(await this.api.GET("/api/v0/workspaces/{slug}/intake-counts", { params: { path: { slug } } }));
  }

  /**
   * A page of the project's intake: the work items the query selects, in its order, with the number it selects and the
   * cursor of the next page (null on the last) (listIntakeIssues).
   */
  async list(projectId: string, query: IntakeListQuery): Promise<IntakeIssuePage> {
    return unwrap(
      await this.api.GET("/api/v0/projects/{project_id}/intake-issues", {
        params: { path: { project_id: projectId }, query },
      })
    );
  }

  /**
   * Triages the work item of a project's intake: a status, with the time of a snooze or the work item it duplicates
   * alone (C8); nerve's answer, the work item triaged (triageIntakeIssue).
   */
  async triage(issueId: string, body: IntakeTriage): Promise<IssueDetail> {
    return unwrap(
      await this.api.POST("/api/v0/intake-issues/{issue_id}/triage", { params: { path: { issue_id: issueId } }, body })
    );
  }

  /**
   * Changes the content of the work item of a project's intake, the seven fields it takes (C9); nerve's answer, the
   * work item as changed (updateIntakeIssue).
   */
  async update(issueId: string, body: IntakeIssueUpdate): Promise<IssueDetail> {
    return unwrap(
      await this.api.PATCH("/api/v0/intake-issues/{issue_id}", { params: { path: { issue_id: issueId } }, body })
    );
  }

  /** Deletes the work item of a project's intake, or an accepted one's intake row alone (deleteIntakeIssue). */
  async delete(issueId: string): Promise<void> {
    return unwrap(
      await this.api.DELETE("/api/v0/intake-issues/{issue_id}", { params: { path: { issue_id: issueId } } })
    );
  }

  /**
   * Submits a work item to a project's intake: createIssue's body but for what a triage's acceptance leaves to the
   * project, a day due left out for none (C10); nerve's answer, the new work item, its intake pending
   * (submitIntakeIssue).
   */
  async submit(projectId: string, body: IntakeIssueCreate): Promise<IssueDetail> {
    return unwrap(
      await this.api.POST("/api/v0/projects/{project_id}/intake-issues", {
        params: { path: { project_id: projectId } },
        body,
      })
    );
  }
}
