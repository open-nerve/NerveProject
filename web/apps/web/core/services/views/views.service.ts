/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { ApiClient, View, ViewCreate, ViewUpdate } from "@nerve/api-client";
import { unwrap } from "@/lib/api-error";

/**
 * A project's views (M7 design 3.7, P6a spec 2.1) and a workspace's own (M7/P6b spec 2.6): a query of the project's,
 * or of the workspace's, work items saved with its display, owned by the account that created it, who alone changes
 * it. Its answers are nerve's, as nerve gives them.
 */
export class ViewsService {
  /** api: the client bound to the session of the stores that build this service (RootStore). */
  constructor(private readonly api: ApiClient) {}

  /**
   * The project's views, the newest first (by created_at, then by id), all at once; a guest who sees only his own work
   * items there, his own views alone.
   */
  async listProjectViews(projectId: string): Promise<View[]> {
    return unwrap(
      await this.api.GET("/api/v0/projects/{project_id}/views", { params: { path: { project_id: projectId } } })
    ).data;
  }

  /** Creates a view in the project, owned by the caller; what body leaves out takes the contract's defaults. */
  async createProjectView(projectId: string, body: ViewCreate): Promise<View> {
    return unwrap(
      await this.api.POST("/api/v0/projects/{project_id}/views", {
        params: { path: { project_id: projectId } },
        body,
      })
    );
  }

  /**
   * Creates a view of the workspace, of no project, owned by the caller (its admin or member); what body leaves out
   * takes the contract's defaults.
   */
  async createWorkspaceView(slug: string, body: ViewCreate): Promise<View> {
    return unwrap(await this.api.POST("/api/v0/workspaces/{slug}/views", { params: { path: { slug } }, body }));
  }

  /** The view as nerve holds it, while the project's views are turned off too. */
  async getView(viewId: string): Promise<View> {
    return unwrap(await this.api.GET("/api/v0/views/{view_id}", { params: { path: { view_id: viewId } } }));
  }

  /**
   * Replaces each field body names whole; the answer is the view as nerve holds it, a change of nothing's too, which
   * nerve does not write.
   */
  async updateView(viewId: string, body: ViewUpdate): Promise<View> {
    return unwrap(await this.api.PATCH("/api/v0/views/{view_id}", { params: { path: { view_id: viewId } }, body }));
  }

  /** Deletes the view, and everyone's favorites of it. */
  async deleteView(viewId: string): Promise<void> {
    unwrap(await this.api.DELETE("/api/v0/views/{view_id}", { params: { path: { view_id: viewId } } }));
  }
}
