/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { action, makeObservable, observable, runInAction } from "mobx";
// nerve imports
import type { ApiClient, ApiToken, ApiTokenCreate, ApiTokenCreated } from "@nerve/api-client";
// lib
import { SessionChangedError } from "@/lib/auth/token-manager";
// services
import { ApiTokenService } from "@/services/api-token.service";

export interface IApiTokenStore {
  /** The account's tokens, newest first, as lists show them: never the token itself. Undefined until fetched. */
  tokens: ApiToken[] | undefined;
  fetchTokens: () => Promise<ApiToken[] | undefined>;
  createToken: (data: ApiTokenCreate) => Promise<ApiTokenCreated>;
  revokeToken: (tokenId: string) => Promise<void>;
}

/** A create or a revoke that nerve confirmed, as it changes a list of the account's tokens. */
type Change = { created: ApiToken } | { revoked: string };

/**
 * The list with the change made: a created token first, unless the list has it already (a fetch read after the
 * create); a revoked token gone.
 */
function withChange(tokens: ApiToken[], change: Change): ApiToken[] {
  if ("revoked" in change) return tokens.filter((token) => token.id !== change.revoked);
  return tokens.some((token) => token.id === change.created.id) ? tokens : [change.created, ...tokens];
}

/**
 * The personal access tokens of the account of a session (M2 design 7.5). Its service sends with the client of
 * that session, which the RootStore of the session hands down: a new session has a store of its own.
 */
export class ApiTokenStore implements IApiTokenStore {
  tokens: ApiToken[] | undefined = undefined;
  /** The sequence number of the newest fetch, the one fetch that may write the list. */
  private newestFetch = 0;
  /**
   * The creates and revokes nerve confirmed while the newest fetch is out, in the order they finished: its
   * pages may have been read before them, so its write makes them again. Undefined once the newest fetch has
   * ended (an older one still out writes nothing).
   */
  private changesDuringFetch: Change[] | undefined = undefined;
  private readonly service: ApiTokenService;

  constructor(api: ApiClient) {
    makeObservable(this, {
      tokens: observable.ref,
      fetchTokens: action,
      createToken: action,
      revokeToken: action,
    });
    this.service = new ApiTokenService(api);
  }

  /**
   * @description fetches every page of the account's tokens and shows them all at once, with the creates and
   * revokes that finished while they loaded; gives what it shows. Only the newest fetch writes: one that a newer
   * fetch overtook writes nothing and gives undefined, as its list is older than the one the store shows or
   * will. A change of session while they load is no failure either: the new session's store fetches its own
   * (store-context.tsx), and this one gives undefined.
   * @returns {Promise<ApiToken[] | undefined>}
   */
  fetchTokens = async (): Promise<ApiToken[] | undefined> => {
    const sequence = ++this.newestFetch;
    const changes: Change[] = [];
    this.changesDuringFetch = changes;
    try {
      const fetched = await this.listFrom(undefined);
      if (sequence !== this.newestFetch) return undefined;
      const tokens = changes.reduce(withChange, fetched);
      runInAction(() => {
        this.tokens = tokens;
      });
      return tokens;
    } catch (error) {
      if (error instanceof SessionChangedError) return undefined;
      throw error;
    } finally {
      if (sequence === this.newestFetch) this.changesDuringFetch = undefined;
    }
  };

  /** The page after cursor (undefined: the first) and every page after it, one after another. */
  private async listFrom(cursor: string | undefined): Promise<ApiToken[]> {
    const page = await this.service.list(cursor);
    return page.next_cursor === null ? page.data : [...page.data, ...(await this.listFrom(page.next_cursor))];
  }

  /**
   * @description creates a token and returns it with the token itself, for the one time it is shown; the list
   * gets the token as lists show it, without the token itself. Fails when nerve refuses.
   * @returns {Promise<ApiTokenCreated>}
   */
  createToken = async (data: ApiTokenCreate): Promise<ApiTokenCreated> => {
    const created = await this.service.create(data);
    const listed: ApiToken = {
      id: created.id,
      label: created.label,
      description: created.description,
      expired_at: created.expired_at,
      last_used: created.last_used,
      created_at: created.created_at,
    };
    this.commit({ created: listed });
    return created;
  };

  /**
   * @description revokes a token, which then leaves the list; fails when nerve refuses
   * @returns {Promise<void>}
   */
  revokeToken = async (tokenId: string): Promise<void> => {
    await this.service.revoke(tokenId);
    this.commit({ revoked: tokenId });
  };

  /**
   * A change nerve confirmed: it goes into the list, when there is one (a list of the new token alone would
   * show as the whole list), and into the record of the fetch that is out.
   */
  private commit(change: Change): void {
    runInAction(() => {
      if (this.tokens) this.tokens = withChange(this.tokens, change);
    });
    this.changesDuringFetch?.push(change);
  }
}
