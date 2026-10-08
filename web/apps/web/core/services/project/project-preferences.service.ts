/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { ApiClient, ProjectPreferences, ProjectPreferencesUpdate } from "@nerve/api-client";
import { unwrap } from "@/lib/api-error";

/** The caller's own display settings in a project: its tab bar and its place in his sidebar (M3 design 3.18, 5.1). */
export class ProjectPreferencesService {
  /** api: the client bound to the session of the stores that build this service (RootStore). */
  constructor(private readonly api: ApiClient) {}

  /** The caller's settings in the project, nerve's defaults while he has changed none. */
  async get(projectId: string): Promise<ProjectPreferences> {
    return unwrap(
      await this.api.GET("/api/v0/me/projects/{project_id}/preferences", {
        params: { path: { project_id: projectId } },
      })
    );
  }

  /** Changes the settings data names; the answer is all of them as nerve now holds them. */
  async update(projectId: string, data: ProjectPreferencesUpdate): Promise<ProjectPreferences> {
    return unwrap(
      await this.api.PATCH("/api/v0/me/projects/{project_id}/preferences", {
        params: { path: { project_id: projectId } },
        body: data,
      })
    );
  }
}
