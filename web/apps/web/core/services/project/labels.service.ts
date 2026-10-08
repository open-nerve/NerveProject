/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { ApiClient, Label, LabelCreate, LabelUpdate } from "@nerve/api-client";
import { unwrap } from "@/lib/api-error";

/** The labels of a project's work items, in two levels: at the top, or under a label at the top (M3 design 3.16). */
export class LabelsService {
  /** api: the client bound to the session of the stores that build this service (RootStore). */
  constructor(private readonly api: ApiClient) {}

  /** The project's labels, those at the top and those under them alike, by sort order, then id; an archived one's too. */
  async list(projectId: string): Promise<Label[]> {
    return unwrap(
      await this.api.GET("/api/v0/projects/{project_id}/labels", { params: { path: { project_id: projectId } } })
    ).data;
  }

  /** Creates a label after the project's others; data names a parent only for a label under one. */
  async create(projectId: string, data: LabelCreate): Promise<Label> {
    return unwrap(
      await this.api.POST("/api/v0/projects/{project_id}/labels", {
        params: { path: { project_id: projectId } },
        body: data,
      })
    );
  }

  /** Changes the fields data names (a null parent moves the label to the top); the answer is the label as nerve holds it. */
  async update(labelId: string, data: LabelUpdate): Promise<Label> {
    return unwrap(
      await this.api.PATCH("/api/v0/labels/{label_id}", { params: { path: { label_id: labelId } }, body: data })
    );
  }

  /** Deletes a label and the labels under it. */
  async delete(labelId: string): Promise<void> {
    unwrap(await this.api.DELETE("/api/v0/labels/{label_id}", { params: { path: { label_id: labelId } } }));
  }
}
