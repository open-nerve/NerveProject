/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { ApiClient, WorkspacePreferences, WorkspacePreferencesUpdate } from "@nerve/api-client";
import { unwrap } from "@/lib/api-error";

/** The caller's own settings of the sidebar's project navigation in a workspace (M3 design 3.18, 5.1). */
export class WorkspacePreferencesService {
  /** api: the client bound to the session of the stores that build this service (RootStore). */
  constructor(private readonly api: ApiClient) {}

  /** The caller's settings, or the defaults until he first changes them. */
  async get(slug: string): Promise<WorkspacePreferences> {
    return unwrap(await this.api.GET("/api/v0/me/workspaces/{slug}/preferences", { params: { path: { slug } } }));
  }

  /** Changes the settings data names; the answer is all of them as nerve now holds them. */
  async update(slug: string, data: WorkspacePreferencesUpdate): Promise<WorkspacePreferences> {
    return unwrap(
      await this.api.PATCH("/api/v0/me/workspaces/{slug}/preferences", { params: { path: { slug } }, body: data })
    );
  }
}
