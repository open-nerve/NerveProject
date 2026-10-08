/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { ApiClient, State, StateCreate, StateUpdate } from "@nerve/api-client";
import { unwrap } from "@/lib/api-error";

/** The states of a project's work items (M3 design 3.17, 5.1); the intake's triage state is none of them. */
export class StatesService {
  /** api: the client bound to the session of the stores that build this service (RootStore). */
  constructor(private readonly api: ApiClient) {}

  /** The project's states by sequence, the lowest first; none of an archived project. */
  async list(projectId: string): Promise<State[]> {
    return unwrap(
      await this.api.GET("/api/v0/projects/{project_id}/states", { params: { path: { project_id: projectId } } })
    ).data;
  }

  /** The states of the workspace's projects the caller is a member of, the archived ones' left out. */
  async listInWorkspace(slug: string): Promise<State[]> {
    return unwrap(await this.api.GET("/api/v0/workspaces/{slug}/states", { params: { path: { slug } } })).data;
  }

  /** Creates a state, last of the project's: not its default. */
  async create(projectId: string, data: StateCreate): Promise<State> {
    return unwrap(
      await this.api.POST("/api/v0/projects/{project_id}/states", {
        params: { path: { project_id: projectId } },
        body: data,
      })
    );
  }

  /** Changes the fields data names; the answer is the state as nerve now holds it. */
  async update(stateId: string, data: StateUpdate): Promise<State> {
    return unwrap(
      await this.api.PATCH("/api/v0/states/{state_id}", { params: { path: { state_id: stateId } }, body: data })
    );
  }

  /** Deletes a state that is neither its project's default nor the last of its group. */
  async delete(stateId: string): Promise<void> {
    unwrap(await this.api.DELETE("/api/v0/states/{state_id}", { params: { path: { state_id: stateId } } }));
  }

  /** Makes the state its project's default; the one that was is no longer. */
  async markDefault(stateId: string): Promise<void> {
    unwrap(await this.api.POST("/api/v0/states/{state_id}/mark-default", { params: { path: { state_id: stateId } } }));
  }
}
