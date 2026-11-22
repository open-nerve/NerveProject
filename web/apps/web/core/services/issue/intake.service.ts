/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { ApiClient, IntakeCounts, IntakeIssuePage, operations } from "@nerve/api-client";
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
}
