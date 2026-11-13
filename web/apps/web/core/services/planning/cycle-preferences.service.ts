/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { ApiClient, CyclePreferences, CyclePreferencesUpdate } from "@nerve/api-client";
import { unwrap } from "@/lib/api-error";

/** The caller's own list settings on a cycle's page: its filters, display filters and display properties. */
export class CyclePreferencesService {
  /** api: the client bound to the session of the stores that build this service (RootStore). */
  constructor(private readonly api: ApiClient) {}

  /** The caller's settings on the cycle's page, nerve's defaults while he has changed none. */
  async get(cycleId: string): Promise<CyclePreferences> {
    return unwrap(
      await this.api.GET("/api/v0/me/cycles/{cycle_id}/preferences", { params: { path: { cycle_id: cycleId } } })
    );
  }

  /** Each named field replaces that field whole; the answer is the page's settings as changed. */
  async update(cycleId: string, data: CyclePreferencesUpdate): Promise<CyclePreferences> {
    return unwrap(
      await this.api.PATCH("/api/v0/me/cycles/{cycle_id}/preferences", {
        params: { path: { cycle_id: cycleId } },
        body: data,
      })
    );
  }
}
