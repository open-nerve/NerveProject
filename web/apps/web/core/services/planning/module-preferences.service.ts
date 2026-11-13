/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { ApiClient, ModulePreferences, ModulePreferencesUpdate } from "@nerve/api-client";
import { unwrap } from "@/lib/api-error";

/** The caller's own list settings on a module's page: its filters, display filters and display properties. */
export class ModulePreferencesService {
  /** api: the client bound to the session of the stores that build this service (RootStore). */
  constructor(private readonly api: ApiClient) {}

  /** The caller's settings on the module's page, nerve's defaults while he has changed none. */
  async get(moduleId: string): Promise<ModulePreferences> {
    return unwrap(
      await this.api.GET("/api/v0/me/modules/{module_id}/preferences", { params: { path: { module_id: moduleId } } })
    );
  }

  /** Each named field replaces that field whole; the answer is the page's settings as changed. */
  async update(moduleId: string, data: ModulePreferencesUpdate): Promise<ModulePreferences> {
    return unwrap(
      await this.api.PATCH("/api/v0/me/modules/{module_id}/preferences", {
        params: { path: { module_id: moduleId } },
        body: data,
      })
    );
  }
}
