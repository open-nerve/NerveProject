/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { ApiToken, ApiTokenCreated } from "@nerve/api-client";
import { ApiError } from "@/lib/api-error";
import { FakeNerve, answered, json, noContent, problem } from "@/lib/auth/fake-nerve";
import { track, until } from "@/lib/auth/fake-time";
import { inTurn } from "@/store/fake-queue";
import { ApiTokenStore } from "@/store/user/api-token.store";

// The personal access tokens of an account (M2 design 7.5), against a fake nerve. That the store sends as the
// session of its RootStore, and that a new session starts with a store of its own, is store-context.test.ts.

const LIST = "/api/v0/me/api-tokens";

/** A token as lists show it, the n-th. */
function listed(n: number): ApiToken {
  return {
    id: `00000000-0000-4000-8000-00000000000${n}`,
    label: `token ${n}`,
    description: "",
    expired_at: null,
    last_used: null,
    created_at: `2026-09-2${n}T00:00:00Z`,
  };
}

/** The token itself of a new token: nrv_pat_ and 43 characters (M2 design 3.4). */
const SECRET = "nrv_pat_Q2hhcmxlcyBCYWJiYWdlIGFuZCBBZGEgTG92ZWxhY2U";

function created(n: number): ApiTokenCreated {
  return { ...listed(n), token: SECRET };
}

beforeEach(() => {
  vi.useFakeTimers({ now: 1_000_000 });
});
afterEach(() => {
  vi.useRealTimers();
});

describe("ApiTokenStore", () => {
  it("fetches the pages one after another, each after the cursor of the last, and shows the whole list at once", async () => {
    const nerve = new FakeNerve();
    const store = new ApiTokenStore(nerve.client());
    const fetched = track(store.fetchTokens());

    await until(() => nerve.calls.length === 1, "the first page");
    expect(nerve.calls[0]).toMatchObject({ method: "GET", path: LIST });
    expect(nerve.calls[0]?.query).toEqual({ limit: "100" });
    nerve.calls[0]?.answer(json(200, { data: [listed(3), listed(2)], next_cursor: "c-1" }));
    await until(() => nerve.calls.length === 2, "the second page");
    // Nothing shows until the list is whole.
    expect(store.tokens).toBeUndefined();
    expect(nerve.calls[1]).toMatchObject({ method: "GET", path: LIST });
    expect(nerve.calls[1]?.query).toEqual({ limit: "100", cursor: "c-1" });
    nerve.calls[1]?.answer(json(200, { data: [listed(1)], next_cursor: null }));
    await until(() => fetched.settled, "the list");

    expect(fetched.value).toEqual([listed(3), listed(2), listed(1)]);
    expect(store.tokens).toEqual([listed(3), listed(2), listed(1)]);
    expect(nerve.calls).toHaveLength(2);
  });

  it("fails when a page fails, and shows no part of the list", async () => {
    const nerve = new FakeNerve();
    const store = new ApiTokenStore(nerve.client());
    const fetched = track(store.fetchTokens());

    await until(() => nerve.calls.length === 1, "the first page");
    nerve.calls[0]?.answer(json(200, { data: [listed(2)], next_cursor: "c-1" }));
    await until(() => nerve.calls.length === 2, "the second page");
    nerve.calls[1]?.answer(problem(503, "server_busy"));
    await until(() => fetched.settled, "the failure");

    expect(fetched.error).toBeInstanceOf(ApiError);
    expect(store.tokens).toBeUndefined();
  });

  it("gives the token itself to the one who created it, and lists the new token without it", async () => {
    const nerve = new FakeNerve();
    const store = new ApiTokenStore(nerve.client());
    const fetched = track(store.fetchTokens());
    await until(() => nerve.calls.length === 1, "the list");
    nerve.calls[0]?.answer(json(200, { data: [listed(1)], next_cursor: null }));
    await until(() => fetched.settled, "the list");

    const body = { label: "token 2", description: "ci", expired_at: "2026-10-04T00:00:00Z" };
    const creating = track(store.createToken(body));
    await until(() => nerve.calls.length === 2, "the creation");
    expect(nerve.calls[1]).toMatchObject({ method: "POST", path: LIST, body });
    nerve.calls[1]?.answer(json(201, created(2)));
    await until(() => creating.settled, "the new token");

    expect(creating.value).toEqual(created(2));
    // The list has the new token first, with the fields lists show and no other (toEqual fails on a field
    // more): never the token itself.
    expect(store.tokens).toEqual([listed(2), listed(1)]);
    expect(JSON.stringify(store.tokens)).not.toContain(SECRET.slice("nrv_pat_".length));
  });

  it("leaves a list it has not fetched to the fetch, when it creates a token", async () => {
    const nerve = new FakeNerve();
    const store = new ApiTokenStore(nerve.client());
    const creating = track(store.createToken({ label: "token 1" }));
    await until(() => nerve.calls.length === 1, "the creation");
    nerve.calls[0]?.answer(json(201, created(1)));
    await until(() => creating.settled, "the new token");

    expect(creating.value).toEqual(created(1));
    // A list of the new token alone would show as the whole list.
    expect(store.tokens).toBeUndefined();
  });

  it("takes a token off the list once nerve revoked it, and keeps it when nerve refuses", async () => {
    const nerve = new FakeNerve();
    const store = new ApiTokenStore(nerve.client());
    const fetched = track(store.fetchTokens());
    await until(() => nerve.calls.length === 1, "the list");
    nerve.calls[0]?.answer(json(200, { data: [listed(3), listed(2), listed(1)], next_cursor: null }));
    await until(() => fetched.settled, "the list");

    const refused = track(store.revokeToken(listed(2).id));
    await until(() => nerve.calls.length === 2, "the first revocation");
    expect(nerve.calls[1]).toMatchObject({ method: "DELETE", path: `/api/v0/api-tokens/${listed(2).id}` });
    nerve.calls[1]?.answer(problem(404, "identity.api_token_not_found"));
    await until(() => refused.settled, "the refusal");
    expect(refused.error).toBeInstanceOf(ApiError);
    expect(store.tokens).toEqual([listed(3), listed(2), listed(1)]);

    // The token in the middle: only it leaves.
    const revoked = track(store.revokeToken(listed(2).id));
    await until(() => nerve.calls.length === 3, "the second revocation");
    // Until nerve answers, the token is still the account's.
    expect(store.tokens).toEqual([listed(3), listed(2), listed(1)]);
    nerve.calls[2]?.answer(noContent());
    await until(() => revoked.settled, "the revocation");
    expect(revoked.error).toBeUndefined();
    expect(store.tokens).toEqual([listed(3), listed(1)]);
  });

  it("sends each change once nerve has answered the one before it, refused or not", async () => {
    const nerve = new FakeNerve();
    const store = new ApiTokenStore(nerve.client());
    await loadList(nerve, store, [listed(2), listed(1)]);
    const refused = track(store.createToken({ label: "token 3" }));
    const revoked = track(store.revokeToken(listed(2).id));
    const made = track(store.createToken({ label: "token 4" }));
    await inTurn(nerve, 1, ["POST", LIST], problem(422, "validation_failed"));
    await inTurn(nerve, 2, ["DELETE", `/api/v0/api-tokens/${listed(2).id}`], noContent());
    await inTurn(nerve, 3, ["POST", LIST], json(201, created(4)));
    await until(() => made.settled, "the last change");
    expect(refused.error).toBeInstanceOf(ApiError);
    expect(revoked.error).toBeUndefined();
    expect(store.tokens).toEqual([listed(4), listed(1)]);
  });
});

/** Fetches the list, and nerve answers it with one page of tokens. */
async function loadList(nerve: FakeNerve, store: ApiTokenStore, tokens: ApiToken[]) {
  await answered(nerve, () => store.fetchTokens(), ["GET", LIST], { data: tokens, next_cursor: null }, "the list");
  expect(store.tokens).toEqual(tokens);
}

/** Creates the n-th token, and nerve creates it. */
async function create(nerve: FakeNerve, store: ApiTokenStore, n: number) {
  const at = nerve.calls.length;
  const creating = track(store.createToken({ label: `token ${n}` }));
  await until(() => nerve.calls.length === at + 1, "the creation");
  expect(nerve.calls[at]).toMatchObject({ method: "POST", path: LIST, body: { label: `token ${n}` } });
  nerve.calls[at]?.answer(json(201, created(n)));
  await until(() => creating.settled, "the new token");
  expect(creating.value).toEqual(created(n));
}

// A fetch's pages may be older than a create or a revoke that finished while it was out: what the fetch
// shows carries them (the order is forced: the page waits until the test answers it, after the create or
// the revoke has finished). Of two fetches, only the newer writes.
describe("ApiTokenStore, while a fetch is out", () => {
  it("carries a token created during the first load into the list the load shows", async () => {
    const nerve = new FakeNerve();
    const store = new ApiTokenStore(nerve.client());
    const fetched = track(store.fetchTokens());
    await until(() => nerve.calls.length === 1, "the first page");

    await create(nerve, store, 4);
    expect(store.tokens).toBeUndefined();
    // The page was read before token 4 was made.
    nerve.calls[0]?.answer(json(200, { data: [listed(3), listed(2), listed(1)], next_cursor: null }));
    await until(() => fetched.settled, "the list");

    expect(store.tokens).toEqual([listed(4), listed(3), listed(2), listed(1)]);
    expect(fetched.value).toEqual(store.tokens);
    expect(JSON.stringify(store.tokens)).not.toContain(SECRET.slice("nrv_pat_".length));
  });

  it("carries a token created while a later page is out", async () => {
    const nerve = new FakeNerve();
    const store = new ApiTokenStore(nerve.client());
    const fetched = track(store.fetchTokens());
    await until(() => nerve.calls.length === 1, "the first page");
    nerve.calls[0]?.answer(json(200, { data: [listed(3), listed(2)], next_cursor: "c-1" }));
    await until(() => nerve.calls.length === 2, "the second page");

    // Token 4 is the newest: it belongs on the first page, which was read before it was made.
    await create(nerve, store, 4);
    nerve.calls[1]?.answer(json(200, { data: [listed(1)], next_cursor: null }));
    await until(() => fetched.settled, "the list");

    expect(store.tokens).toEqual([listed(4), listed(3), listed(2), listed(1)]);
  });

  it("makes the changes that finished during a load in their order: the newest token first, one created and revoked gone", async () => {
    const nerve = new FakeNerve();
    const store = new ApiTokenStore(nerve.client());
    const fetched = track(store.fetchTokens());
    await until(() => nerve.calls.length === 1, "the first page");

    await create(nerve, store, 4);
    await create(nerve, store, 5);
    const revoked = track(store.revokeToken(listed(4).id));
    await until(() => nerve.calls.length === 4, "the revocation");
    expect(nerve.calls[3]).toMatchObject({ method: "DELETE", path: `/api/v0/api-tokens/${listed(4).id}` });
    nerve.calls[3]?.answer(noContent());
    await until(() => revoked.settled, "the revocation");
    expect(revoked.error).toBeUndefined();
    // The page was read before tokens 4 and 5 were made.
    nerve.calls[0]?.answer(json(200, { data: [listed(3), listed(2), listed(1)], next_cursor: null }));
    await until(() => fetched.settled, "the list");

    expect(store.tokens).toEqual([listed(5), listed(3), listed(2), listed(1)]);
  });

  it("keeps a token revoked during a refetch off the list the refetch shows", async () => {
    const nerve = new FakeNerve();
    const store = new ApiTokenStore(nerve.client());
    await loadList(nerve, store, [listed(3), listed(2), listed(1)]);
    const refetched = track(store.fetchTokens());
    await until(() => nerve.calls.length === 2, "the refetch");

    const revoked = track(store.revokeToken(listed(2).id));
    await until(() => nerve.calls.length === 3, "the revocation");
    expect(nerve.calls[2]).toMatchObject({ method: "DELETE", path: `/api/v0/api-tokens/${listed(2).id}` });
    nerve.calls[2]?.answer(noContent());
    await until(() => revoked.settled, "the revocation");
    // The loaded list shows the revocation at once.
    expect(store.tokens).toEqual([listed(3), listed(1)]);
    // The page was read before token 2 was revoked.
    nerve.calls[1]?.answer(json(200, { data: [listed(3), listed(2), listed(1)], next_cursor: null }));
    await until(() => refetched.settled, "the refetch");

    expect(store.tokens).toEqual([listed(3), listed(1)]);
    expect(refetched.value).toEqual(store.tokens);
  });

  it("lists a created token once when the fetched page has it already, whichever answers first", async () => {
    const nerve = new FakeNerve();
    const store = new ApiTokenStore(nerve.client());
    // The create finishes while the list is out, and the page was read after token 2 was made.
    const fetched = track(store.fetchTokens());
    await until(() => nerve.calls.length === 1, "the list");
    await create(nerve, store, 2);
    nerve.calls[0]?.answer(json(200, { data: [listed(2), listed(1)], next_cursor: null }));
    await until(() => fetched.settled, "the list");
    expect(store.tokens).toEqual([listed(2), listed(1)]);

    // A refetch sent while a create is out, read after token 3 was made, answers before the create does.
    const creating = track(store.createToken({ label: "token 3" }));
    await until(() => nerve.calls.length === 3, "the creation");
    const refetched = track(store.fetchTokens());
    await until(() => nerve.calls.length === 4, "the refetch");
    nerve.calls[3]?.answer(json(200, { data: [listed(3), listed(2), listed(1)], next_cursor: null }));
    await until(() => refetched.settled, "the refetch");
    nerve.calls[2]?.answer(json(201, created(3)));
    await until(() => creating.settled, "the new token");

    expect(store.tokens).toEqual([listed(3), listed(2), listed(1)]);
  });

  it("lets the newer of two fetches write, and drops the older when it answers last", async () => {
    const nerve = new FakeNerve();
    const store = new ApiTokenStore(nerve.client());
    const older = track(store.fetchTokens());
    await until(() => nerve.calls.length === 1, "the older list");
    const newer = track(store.fetchTokens());
    await until(() => nerve.calls.length === 2, "the newer list");

    nerve.calls[1]?.answer(json(200, { data: [listed(2), listed(1)], next_cursor: null }));
    await until(() => newer.settled, "the newer list");
    expect(store.tokens).toEqual([listed(2), listed(1)]);
    // The older page, read before token 2 was made elsewhere, answers last: it writes nothing and gives
    // nothing.
    nerve.calls[0]?.answer(json(200, { data: [listed(1)], next_cursor: null }));
    await until(() => older.settled, "the older list");

    expect(older).toEqual({ settled: true, value: undefined });
    expect(newer.value).toEqual([listed(2), listed(1)]);
    expect(store.tokens).toEqual([listed(2), listed(1)]);
  });

  it("drops the older of two fetches when it answers first, and the newer still carries a create made meanwhile", async () => {
    const nerve = new FakeNerve();
    const store = new ApiTokenStore(nerve.client());
    const older = track(store.fetchTokens());
    await until(() => nerve.calls.length === 1, "the older list");
    const newer = track(store.fetchTokens());
    await until(() => nerve.calls.length === 2, "the newer list");

    nerve.calls[0]?.answer(json(200, { data: [listed(1)], next_cursor: null }));
    await until(() => older.settled, "the older list");
    // Only the newest fetch writes: the list shows nothing until it answers.
    expect(older).toEqual({ settled: true, value: undefined });
    expect(store.tokens).toBeUndefined();
    await create(nerve, store, 3);
    // The newer page was read before token 3 was made.
    nerve.calls[1]?.answer(json(200, { data: [listed(2), listed(1)], next_cursor: null }));
    await until(() => newer.settled, "the newer list");

    expect(store.tokens).toEqual([listed(3), listed(2), listed(1)]);
  });

  it("writes the page as nerve gave it when nerve refused a create and a revoke while it was out", async () => {
    const nerve = new FakeNerve();
    const store = new ApiTokenStore(nerve.client());
    await loadList(nerve, store, [listed(3), listed(2), listed(1)]);
    const refetched = track(store.fetchTokens());
    await until(() => nerve.calls.length === 2, "the refetch");

    const creating = track(store.createToken({ label: "token 5" }));
    await until(() => nerve.calls.length === 3, "the creation");
    expect(nerve.calls[2]).toMatchObject({ method: "POST", path: LIST, body: { label: "token 5" } });
    nerve.calls[2]?.answer(problem(422, "validation_failed"));
    await until(() => creating.settled, "the refused creation");
    expect(creating.error).toBeInstanceOf(ApiError);
    const revoking = track(store.revokeToken(listed(2).id));
    await until(() => nerve.calls.length === 4, "the revocation");
    expect(nerve.calls[3]).toMatchObject({ method: "DELETE", path: `/api/v0/api-tokens/${listed(2).id}` });
    nerve.calls[3]?.answer(problem(503, "server_busy"));
    await until(() => revoking.settled, "the refused revocation");
    expect(revoking.error).toBeInstanceOf(ApiError);
    expect(store.tokens).toEqual([listed(3), listed(2), listed(1)]);
    // Token 4 was made elsewhere meanwhile; token 2 is still the account's.
    nerve.calls[1]?.answer(json(200, { data: [listed(4), listed(3), listed(2), listed(1)], next_cursor: null }));
    await until(() => refetched.settled, "the refetch");

    expect(store.tokens).toEqual([listed(4), listed(3), listed(2), listed(1)]);
  });

  it("keeps the loaded list when a refetch fails, and fails", async () => {
    const nerve = new FakeNerve();
    const store = new ApiTokenStore(nerve.client());
    await loadList(nerve, store, [listed(3), listed(2), listed(1)]);
    const loaded = store.tokens;
    const refetched = track(store.fetchTokens());
    await until(() => nerve.calls.length === 2, "the refetch's first page");
    nerve.calls[1]?.answer(json(200, { data: [listed(4), listed(3)], next_cursor: "c-1" }));
    await until(() => nerve.calls.length === 3, "the refetch's second page");
    nerve.calls[2]?.answer(problem(503, "server_busy"));
    await until(() => refetched.settled, "the failure");

    expect(refetched.error).toBeInstanceOf(ApiError);
    expect(store.tokens).toBe(loaded);
    expect(store.tokens).toEqual([listed(3), listed(2), listed(1)]);
  });
});
