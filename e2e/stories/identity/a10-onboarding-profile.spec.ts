import { accountOf } from "../../fixtures/assert/identity";
import { bearer, createPAT, emailFor, register } from "../../fixtures/auth";
import { expect, test } from "../../fixtures/test";

// A10, the profile step of onboarding (M2 design 2). The page version, with
// the first visit of /onboarding, joins in M2/P4.

test("A10 (API): the profile step sets the name and one step, which the others keep beside", async ({
  api,
  db,
}, testInfo) => {
  const email = emailFor(testInfo);
  const pat = await createPAT(api, (await register(api, email)).access_token);
  const { id } = await accountOf(db, email);
  const steps = async () =>
    (await db.query<{ onboarding_step: unknown }>("SELECT onboarding_step FROM profiles WHERE user_id = $1", [id]))[0]
      ?.onboarding_step;

  const named = await api.PATCH("/api/v0/me", { body: { first_name: "Ada" }, headers: bearer(pat.token) });
  expect(named.response.status).toBe(200);
  expect((await accountOf(db, email)).first_name).toBe("Ada");

  // Another step is done already, so keeping it differs from resetting it
  // to its default.
  const joined = await api.PATCH("/api/v0/me/profile", {
    body: { onboarding_step: { workspace_join: true } },
    headers: bearer(pat.token),
  });
  expect(joined.response.status).toBe(200);

  // One key: it is merged in, the other three keep their values (M2 design 3.14).
  const stepped = await api.PATCH("/api/v0/me/profile", {
    body: { onboarding_step: { profile_complete: true } },
    headers: bearer(pat.token),
  });
  expect(stepped.response.status).toBe(200);
  const merged = { profile_complete: true, workspace_create: false, workspace_invite: false, workspace_join: true };
  expect(await steps()).toEqual(merged);

  // An unknown key, here misspelt, breaks the contract: the platform's 400,
  // and nothing changes.
  const unknown = await api.PATCH("/api/v0/me/profile", {
    body: { onboarding_step: { profile_completed: true } },
    headers: bearer(pat.token),
  });
  expect(unknown.response.status).toBe(400);
  expect(unknown.error?.errors?.map((e) => ({ field: e.field, code: e.code }))).toEqual([
    { field: "onboarding_step.profile_completed", code: "not_allowed" },
  ]);
  expect(await steps()).toEqual(merged);
});
