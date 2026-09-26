import { accountOf } from "../../fixtures/assert/identity";
import { bearer, createPAT, emailFor, register } from "../../fixtures/auth";
import { expect, test } from "../../fixtures/test";

// A8, changing the names and the time zone (M2 design 2). The page version
// joins in M2/P5.

test("A8 (API): a personal access token changes the names and the time zone; null is a 400", async ({
  api,
  db,
}, testInfo) => {
  const email = emailFor(testInfo);
  const pat = await createPAT(api, (await register(api, email)).access_token);
  const before = await accountOf(db, email);
  const change = { first_name: "Ada", last_name: "Lovelace", display_name: "ada", user_timezone: "Asia/Shanghai" };

  const { data, response } = await api.PATCH("/api/v0/me", { body: change, headers: bearer(pat.token) });

  expect(response.status).toBe(200);
  expect(data).toMatchObject(change);
  const after = await accountOf(db, email);
  expect(after).toMatchObject(change);
  expect(after.updated_at.getTime()).toBeGreaterThan(before.updated_at.getTime());

  // null breaks the contract: the platform's 400, and nothing changes.
  const nulled = await api.PATCH("/api/v0/me", {
    // @ts-expect-error -- null for a name breaks the contract on purpose
    body: { first_name: null },
    headers: bearer(pat.token),
  });
  expect(nulled.response.status).toBe(400);
  expect(nulled.error?.code).toBe("bad_request");
  expect(nulled.error?.errors?.map((e) => ({ field: e.field, code: e.code }))).toEqual([
    { field: "first_name", code: "invalid_format" },
  ]);
  expect(await accountOf(db, email)).toEqual(after);
});
