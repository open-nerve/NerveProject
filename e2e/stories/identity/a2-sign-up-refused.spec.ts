import { createApi, type Api } from "../../fixtures/api";
import { countIdentity, expectNothingAdded } from "../../fixtures/assert/identity";
import { emailFor, password, register } from "../../fixtures/auth";
import { expect, test } from "../../fixtures/test";

// A2, sign-up refused (M2 design 2). The page version joins in M2/P4.

async function expectRefused(
  api: Api,
  body: { email: string; password: string },
  want: { status: number; code: string; fields?: { field: string; code: string }[] }
): Promise<void> {
  const { response, error } = await api.POST("/api/v0/auth/register", { body });
  const label = `${body.email} ${body.password}`;
  expect(response.status, label).toBe(want.status);
  expect(error?.code, label).toBe(want.code);
  if (want.fields) {
    expect(
      error?.errors?.map((e) => ({ field: e.field, code: e.code })),
      label
    ).toEqual(want.fields);
  }
}

test("A2 (API): a refused sign-up answers why and adds nothing", async ({ api, db, nerveWith }, testInfo) => {
  const email = emailFor(testInfo);
  await register(api, email);
  const before = await countIdentity(db);
  const newEmail = emailFor(testInfo, "new");
  const invalid = (pw: string, code: string) =>
    expectRefused(
      api,
      { email: newEmail, password: pw },
      { status: 422, code: "validation_failed", fields: [{ field: "password", code }] }
    );

  await Promise.all([
    // The address is taken, whatever its case.
    expectRefused(api, { email: email.toUpperCase(), password }, { status: 409, code: "identity.email_taken" }),
    // A weak password, and common ones.
    invalid("password", "weak_password"),
    invalid("Password1!", "common_password"),
    invalid("Password1!~", "common_password"),
  ]);

  // With sign-up off, every address gets the same answer, a taken one too.
  const closed = createApi((await nerveWith({ NERVE_AUTH__SIGNUP_ENABLED: "false" })).baseURL);
  await Promise.all(
    [newEmail, email].map((address) =>
      expectRefused(closed, { email: address, password }, { status: 403, code: "identity.signup_disabled" })
    )
  );

  await expectNothingAdded(db, before);
});
