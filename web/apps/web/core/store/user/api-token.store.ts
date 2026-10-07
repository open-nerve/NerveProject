/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { action, makeObservable } from "mobx";
// nerve imports
import type { ApiClient, ApiToken, ApiTokenCreate, ApiTokenCreated } from "@nerve/api-client";
// lib
import { oneAtATime } from "@/lib/one-at-a-time";
import { Reconciled, dropped, prepended } from "@/lib/reconciled";
// services
import { ApiTokenService } from "@/services/api-token.service";

export interface IApiTokenStore {
  /** The account's tokens, newest first, as lists show them: never the token itself. Undefined until fetched. */
  tokens: ApiToken[] | undefined;
  fetchTokens: () => Promise<ApiToken[] | undefined>;
  createToken: (data: ApiTokenCreate) => Promise<ApiTokenCreated>;
  revokeToken: (tokenId: string) => Promise<void>;
}

/**
 * The personal access tokens of the account of a session (M2 design 7.5). Its service sends with the client of
 * that session, which the RootStore of the session hands down: a new session has a store of its own. Changes go one
 * at a time (v0 design 7.7); fetches do not queue.
 */
export class ApiTokenStore implements IApiTokenStore {
  /** The list, reconciled between its fetches and the creates and revokes nerve confirmed (reconciled.ts). */
  private readonly list = new Reconciled<ApiToken[]>();
  private readonly service: ApiTokenService;
  /** The creates and revokes, sent one at a time. */
  private readonly changes = oneAtATime();

  constructor(api: ApiClient) {
    makeObservable(this, {
      fetchTokens: action,
      createToken: action,
      revokeToken: action,
    });
    this.service = new ApiTokenService(api);
  }

  get tokens(): ApiToken[] | undefined {
    return this.list.value;
  }

  /**
   * @description fetches every page of the account's tokens and shows them all at once, with the creates and
   * revokes that finished while they loaded; gives what it shows, or undefined for a fetch a newer one overtook or
   * a change of session cut (Reconciled.fetch).
   * @returns {Promise<ApiToken[] | undefined>}
   */
  fetchTokens = (): Promise<ApiToken[] | undefined> => this.list.fetch(() => this.listFrom(undefined));

  /** The page after cursor (undefined: the first) and every page after it, one after another. */
  private async listFrom(cursor: string | undefined): Promise<ApiToken[]> {
    const page = await this.service.list(cursor);
    return page.next_cursor === null ? page.data : [...page.data, ...(await this.listFrom(page.next_cursor))];
  }

  /**
   * @description creates a token and returns it with the token itself, for the one time it is shown; the list
   * gets the token as lists show it, without the token itself, first. Fails when nerve refuses.
   * @returns {Promise<ApiTokenCreated>}
   */
  createToken = (data: ApiTokenCreate): Promise<ApiTokenCreated> =>
    this.changes(async () => {
      const created = await this.service.create(data);
      const listed: ApiToken = {
        id: created.id,
        label: created.label,
        description: created.description,
        expired_at: created.expired_at,
        last_used: created.last_used,
        created_at: created.created_at,
      };
      this.list.confirm(prepended([listed]));
      return created;
    });

  /**
   * @description revokes a token, which then leaves the list; fails when nerve refuses
   * @returns {Promise<void>}
   */
  revokeToken = (tokenId: string): Promise<void> =>
    this.changes(async () => {
      await this.service.revoke(tokenId);
      this.list.confirm(dropped(tokenId));
    });
}
