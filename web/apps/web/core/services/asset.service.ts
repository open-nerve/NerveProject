/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import type {
  ApiClient,
  Asset,
  Content,
  DraftAssetPreparationPage,
  DuplicateAsset,
  NewAsset,
  PreparedAsset,
} from "@nerve/api-client";
import { unwrap } from "@/lib/api-error";

export class AssetService {
  constructor(private readonly api: ApiClient) {}

  async prepare(input: NewAsset, signal?: AbortSignal): Promise<PreparedAsset> {
    return unwrap(await this.api.POST("/api/v0/assets", { body: input, signal }));
  }

  async complete(id: string, signal?: AbortSignal): Promise<Asset> {
    return unwrap(await this.api.POST("/api/v0/assets/{id}/complete", { params: { path: { id } }, body: {}, signal }));
  }

  async read(id: string, signal?: AbortSignal): Promise<Asset> {
    return unwrap(await this.api.GET("/api/v0/assets/{id}", { params: { path: { id } }, signal }));
  }

  async remove(id: string, signal?: AbortSignal): Promise<void> {
    unwrap(await this.api.DELETE("/api/v0/assets/{id}", { params: { path: { id } }, signal }));
  }

  async duplicate(id: string, input: DuplicateAsset, signal?: AbortSignal): Promise<Asset> {
    return unwrap(
      await this.api.POST("/api/v0/assets/{id}/duplicate", { params: { path: { id } }, body: input, signal })
    );
  }

  async content(id: string, signal?: AbortSignal): Promise<Content> {
    return unwrap(await this.api.GET("/api/v0/assets/{id}/content", { params: { path: { id } }, signal }));
  }

  async listDraftPreparations(
    draftID: string,
    cursor?: string,
    signal?: AbortSignal
  ): Promise<DraftAssetPreparationPage> {
    return unwrap(
      await this.api.GET("/api/v0/drafts/{draft_id}/asset-preparations", {
        params: { path: { draft_id: draftID }, query: { limit: 50, cursor } },
        signal,
      })
    );
  }
}
