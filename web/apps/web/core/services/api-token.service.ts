/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { ApiClient, ApiTokenCreate, ApiTokenCreated, ApiTokenPage } from "@nerve/api-client";
import { unwrap } from "@/lib/api-error";

/** The account's personal access tokens (M2 design 7.5). */
export class ApiTokenService {
  /** api: the client bound to the session of the stores that build this service (RootStore). */
  constructor(private readonly api: ApiClient) {}

  /** A page of the account's tokens, newest first, as large as nerve gives: the one after cursor, or the first. */
  async list(cursor?: string): Promise<ApiTokenPage> {
    return unwrap(await this.api.GET("/api/v0/me/api-tokens", { params: { query: { limit: 100, cursor } } }));
  }

  /** Creates a token; the answer carries the token itself, this once. */
  async create(data: ApiTokenCreate): Promise<ApiTokenCreated> {
    return unwrap(await this.api.POST("/api/v0/me/api-tokens", { body: data }));
  }

  async revoke(tokenId: string): Promise<void> {
    unwrap(await this.api.DELETE("/api/v0/api-tokens/{token_id}", { params: { path: { token_id: tokenId } } }));
  }
}
