/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { ApiClient, SlugAvailability, Workspace, WorkspaceCreate, WorkspaceUpdate } from "@nerve/api-client";
import { unwrap } from "@/lib/api-error";

/** The caller's workspaces (M3 design 5.1, 7.3). */
export class WorkspacesService {
  /** api: the client bound to the session of the stores that build this service (RootStore). */
  constructor(private readonly api: ApiClient) {}

  /** The workspaces of which the caller is an active member, each with the caller's role, by name and then by id. */
  async list(): Promise<Workspace[]> {
    return unwrap(await this.api.GET("/api/v0/workspaces")).data;
  }

  /** Creates a workspace with the caller as its admin. */
  async create(data: WorkspaceCreate): Promise<Workspace> {
    return unwrap(await this.api.POST("/api/v0/workspaces", { body: data }));
  }

  /** Changes the fields data names; the answer is the workspace as nerve now holds it. */
  async update(slug: string, data: WorkspaceUpdate): Promise<Workspace> {
    return unwrap(await this.api.PATCH("/api/v0/workspaces/{slug}", { params: { path: { slug } }, body: data }));
  }

  async delete(slug: string): Promise<void> {
    unwrap(await this.api.DELETE("/api/v0/workspaces/{slug}", { params: { path: { slug } } }));
  }

  /** Ends the caller's own membership of the workspace. */
  async leave(slug: string): Promise<void> {
    unwrap(await this.api.POST("/api/v0/workspaces/{slug}/leave", { params: { path: { slug } } }));
  }

  /** Whether slug can name a new workspace, or why not. */
  async checkSlug(slug: string): Promise<SlugAvailability> {
    return unwrap(await this.api.GET("/api/v0/workspace-slugs/{slug}", { params: { path: { slug } } }));
  }
}
