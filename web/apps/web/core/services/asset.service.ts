/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import type { ApiClient, Asset, NewAsset, PreparedAsset } from "@nerve/api-client";
import { unwrap } from "@/lib/api-error";

export class AssetService {
  constructor(private readonly api: ApiClient) {}

  async prepare(input: NewAsset, signal?: AbortSignal): Promise<PreparedAsset> {
    return unwrap(await this.api.POST("/api/v0/assets", { body: input, signal }));
  }

  async complete(id: string, signal?: AbortSignal): Promise<Asset> {
    return unwrap(await this.api.POST("/api/v0/assets/{id}/complete", { params: { path: { id } }, body: {}, signal }));
  }
}
