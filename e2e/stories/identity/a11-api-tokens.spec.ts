import type { components } from "@nerve/api-client";

import { expectTokenStored } from "../../fixtures/assert/identity";
import { bearer, createPAT, emailFor, register } from "../../fixtures/auth";
import { expect, test } from "../../fixtures/test";

// A11, personal access tokens (M2 design 2). The page version joins in
// M2/P5.

const weekMs = 7 * 24 * 60 * 60 * 1000;

test("A11 (API): a token creates, lists page by page and revokes another; a revoked or expired token fails", async ({
  api,
  db,
}, testInfo) => {
  const admin = await createPAT(api, (await register(api, emailFor(testInfo))).access_token, { label: "admin" });
  const expiredAt = new Date(Date.now() + weekMs).toISOString();

  const created = await createPAT(api, admin.token, { label: "deploy", description: "ci", expired_at: expiredAt });

  expect(created).toMatchObject({ label: "deploy", description: "ci", last_used: null });
  expect(created.token).toMatch(/^nrv_pat_[A-Za-z0-9_-]{43}$/);
  await expectTokenStored(db, created.id, created.token, expiredAt);

  // The list, a token a page, newest first, never shows a token itself.
  const list = async (cursor?: string): Promise<components["schemas"]["ApiToken"][]> => {
    const { data, response } = await api.GET("/api/v0/me/api-tokens", {
      params: { query: { limit: 1, cursor } },
      headers: bearer(admin.token),
    });
    expect(response.status).toBe(200);
    const page = data?.data ?? [];
    return data?.next_cursor ? [...page, ...(await list(data.next_cursor))] : page;
  };
  const listed = await list();
  expect(listed.map((t) => t.id)).toEqual([created.id, admin.id]);
  expect(JSON.stringify(listed)).not.toContain(created.token);

  // The token authenticates, and its use is recorded.
  expect((await api.GET("/api/v0/me", { headers: bearer(created.token) })).response.status).toBe(200);
  const [used] = await db.query<{ last_used: Date | null }>("SELECT last_used FROM api_tokens WHERE id = $1", [
    created.id,
  ]);
  expect(used?.last_used).not.toBeNull();

  // Revoked, it stops at once and leaves the list.
  const revoked = await api.DELETE("/api/v0/api-tokens/{token_id}", {
    params: { path: { token_id: created.id } },
    headers: bearer(admin.token),
  });
  expect(revoked.response.status).toBe(204);
  const [row] = await db.query<{ deleted_at: Date | null }>("SELECT deleted_at FROM api_tokens WHERE id = $1", [
    created.id,
  ]);
  expect(row?.deleted_at).not.toBeNull();
  expect((await api.GET("/api/v0/me", { headers: bearer(created.token) })).response.status).toBe(401);
  expect((await list()).map((t) => t.id)).toEqual([admin.id]);

  // A token past its expiry fails too.
  const old = await createPAT(api, admin.token, { label: "old" });
  await db.query("UPDATE api_tokens SET expired_at = now() - interval '1 minute' WHERE id = $1", [old.id]);
  expect((await api.GET("/api/v0/me", { headers: bearer(old.token) })).response.status).toBe(401);
});
