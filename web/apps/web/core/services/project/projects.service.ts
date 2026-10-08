/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { ApiClient, Project, ProjectCreate, ProjectUpdate } from "@nerve/api-client";
import { unwrap } from "@/lib/api-error";

/** The projects the caller sees, each as he sees it (M3 design 3.19, 5.1, 7.3). */
export class ProjectsService {
  /** api: the client bound to the session of the stores that build this service (RootStore). */
  constructor(private readonly api: ApiClient) {}

  /**
   * The workspace's projects that the caller sees: the archived ones alone when archived, else the others; by each
   * one's place in his sidebar, those without a place last, then by name.
   */
  async list(slug: string, archived: boolean): Promise<Project[]> {
    return unwrap(
      await this.api.GET("/api/v0/workspaces/{slug}/projects", { params: { path: { slug }, query: { archived } } })
    ).data;
  }

  /** Creates a project with the caller, and the lead when one is given, as its admins. */
  async create(slug: string, data: ProjectCreate): Promise<Project> {
    return unwrap(
      await this.api.POST("/api/v0/workspaces/{slug}/projects", { params: { path: { slug } }, body: data })
    );
  }

  /** The project as the caller sees it: member_role and sort_order null when he is not its member. */
  async get(projectId: string): Promise<Project> {
    return unwrap(await this.api.GET("/api/v0/projects/{project_id}", { params: { path: { project_id: projectId } } }));
  }

  /** Changes the fields data names; the answer is the project as nerve now holds it. */
  async update(projectId: string, data: ProjectUpdate): Promise<Project> {
    return unwrap(
      await this.api.PATCH("/api/v0/projects/{project_id}", { params: { path: { project_id: projectId } }, body: data })
    );
  }

  async delete(projectId: string): Promise<void> {
    unwrap(await this.api.DELETE("/api/v0/projects/{project_id}", { params: { path: { project_id: projectId } } }));
  }

  /** Archives the project; the answer is the project as archived. */
  async archive(projectId: string): Promise<Project> {
    return unwrap(
      await this.api.POST("/api/v0/projects/{project_id}/archive", { params: { path: { project_id: projectId } } })
    );
  }

  /** Unarchives the project; the answer is the project as unarchived. */
  async unarchive(projectId: string): Promise<Project> {
    return unwrap(
      await this.api.POST("/api/v0/projects/{project_id}/unarchive", { params: { path: { project_id: projectId } } })
    );
  }
}
