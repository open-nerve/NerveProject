/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { ApiClient, IntakeCounts } from "@nerve/api-client";
import { unwrap } from "@/lib/api-error";

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
}
