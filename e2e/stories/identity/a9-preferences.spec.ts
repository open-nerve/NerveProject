import { accountOf } from "../../fixtures/assert/identity";
import { bearer, createPAT, emailFor, register } from "../../fixtures/auth";
import { expect, test } from "../../fixtures/test";

// A9, changing the preferences (M2 design 2). The page version, in both
// languages, joins in M2/P5.

test("A9 (API): a personal access token changes the theme, the language and the first day of the week", async ({
  api,
  db,
}, testInfo) => {
  const email = emailFor(testInfo);
  const pat = await createPAT(api, (await register(api, email)).access_token);
  const change = { theme: "dark", language: "zh-CN", start_of_the_week: 1 } as const;

  const { data, response } = await api.PATCH("/api/v0/me/profile", { body: change, headers: bearer(pat.token) });

  expect(response.status).toBe(200);
  expect(data).toMatchObject(change);
  const { id } = await accountOf(db, email);
  expect(await db.query("SELECT theme, language, start_of_the_week FROM profiles WHERE user_id = $1", [id])).toEqual([
    change,
  ]);
  const read = await api.GET("/api/v0/me/profile", { headers: bearer(pat.token) });
  expect(read.data).toEqual(data);
});
