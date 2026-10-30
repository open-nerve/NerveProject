/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import type { ApiClient, Draft, DraftPage, DraftCreate, DraftUpdate, IssueDetail, operations } from "@nerve/api-client";
import { draftPayload } from "./draft-input";
import { unwrap } from "@/lib/api-error";
export class WorkspaceDraftService {
  constructor(private readonly api: ApiClient) {}
  async list(slug: string, query?: operations["listDrafts"]["parameters"]["query"]): Promise<DraftPage> {
    return unwrap(await this.api.GET("/api/v0/workspaces/{slug}/drafts", { params: { path: { slug }, query } }));
  }
  async create(slug: string, input: DraftCreate): Promise<Draft> {
    return unwrap(
      await this.api.POST("/api/v0/workspaces/{slug}/drafts", {
        params: { path: { slug } },
        body: {
          project_id: input.project_id,
          payload: draftPayload(input.payload),
        },
      })
    );
  }
  async update(id: string, input: DraftUpdate): Promise<Draft> {
    return unwrap(
      await this.api.PATCH("/api/v0/drafts/{draft_id}", {
        params: { path: { draft_id: id } },
        body: {
          project_id: input.project_id,
          payload: input.payload === undefined ? undefined : draftPayload(input.payload),
        },
      })
    );
  }
  async delete(id: string): Promise<void> {
    unwrap(await this.api.DELETE("/api/v0/drafts/{draft_id}", { params: { path: { draft_id: id } } }));
  }
  async publish(id: string): Promise<IssueDetail> {
    return unwrap(await this.api.POST("/api/v0/drafts/{draft_id}/publish", { params: { path: { draft_id: id } } }));
  }
}
