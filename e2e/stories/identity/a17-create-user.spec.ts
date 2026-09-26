import { countIdentity, expectCreated, expectNothingAdded } from "../../fixtures/assert/identity";
import { emailFor, login, password, register } from "../../fixtures/auth";
import { expect, test } from "../../fixtures/test";
import { nerveUsers, nerveUsersFails } from "../../fixtures/users";

// A17, the administrator creates an account (M2 design 2, decision 2): how
// the first account comes to be while sign-up is closed. There is no page
// version.

test("A17: nerve users create makes an account with the password from stdin; a taken address or a weak password adds nothing", async ({
  api,
  db,
}, testInfo) => {
  const email = emailFor(testInfo);
  const taken = emailFor(testInfo, "taken");
  await register(api, taken);
  const before = await countIdentity(db);

  await nerveUsersFails(
    db,
    ["create", "--email", taken.toUpperCase()],
    "An account with this e-mail address already exists.",
    `${password}\n`
  );
  await nerveUsersFails(db, ["create", "--email", email], "the password is too common", "Password1!\n");
  await expectNothingAdded(db, before);

  expect(await nerveUsers(db, ["create", "--email", email.toUpperCase()], `${password}\n`)).toBe(
    `created user ${email}\n`
  );
  await expectCreated(db, email);
  await login(api, email);
});
