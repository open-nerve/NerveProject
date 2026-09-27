import { accountOf } from "../../fixtures/assert/identity";
import { bearer, createPAT, emailFor, register } from "../../fixtures/auth";
import { EMOJI_CHECK_WARNING, expectQuietConsole, watchPage } from "../../fixtures/browser";
import { answerTo, expectListBesideButton, fieldBlock, registerOnboarded } from "../../fixtures/settings-pages";
import { expect, test } from "../../fixtures/test";

// A8, changing the names and the time zone (M2 design 2).

test("A8 (page): the general page changes the names, the preferences page the time zone; both hold after a reload", async ({
  api,
  db,
  signedInPage,
}, testInfo) => {
  const email = emailFor(testInfo);
  const page = await signedInPage(await registerOnboarded(api, email));
  const watch = await watchPage(page);
  const before = await accountOf(db, email);
  await page.goto("/settings/profile/general");

  await page.locator("#first_name").fill("Ada");
  await page.locator("#last_name").fill("Lovelace");
  await page.locator("#display_name").fill("ada");
  const saved = await answerTo(page, "PATCH", "/api/v0/me", () =>
    page.getByRole("button", { name: "Save changes" }).click()
  );
  expect(saved.status()).toBe(200);

  // The time zone, from nerve's list (GET /api/v0/timezones), in the list that opens beside its button.
  await page.getByRole("button", { name: "Preferences" }).click();
  const timezone = page.getByRole("button", { name: "UTC" });
  await timezone.click();
  await expectListBesideButton(page, timezone);
  // Escape closes it, and the next click opens it again.
  await page.keyboard.press("Escape");
  await expect(page.getByRole("listbox")).toHaveCount(0);
  await timezone.click();
  await expectListBesideButton(page, timezone);
  await page.getByPlaceholder("Search").fill("Asia/Shanghai");
  const zoned = await answerTo(page, "PATCH", "/api/v0/me", () =>
    page.getByRole("option", { name: "Beijing" }).click()
  );
  expect(zoned.status()).toBe(200);
  // The pick closes the list.
  await expect(page.getByRole("listbox")).toHaveCount(0);

  const change = { first_name: "Ada", last_name: "Lovelace", display_name: "ada", user_timezone: "Asia/Shanghai" };
  const after = await accountOf(db, email);
  expect(after).toMatchObject(change);
  expect(after.updated_at.getTime()).toBeGreaterThan(before.updated_at.getTime());

  // After a reload, both pages show what nerve has.
  await page.reload();
  await expect(page.getByRole("button", { name: "Beijing" })).toBeVisible();
  await page.getByRole("button", { name: "Profile" }).click();
  await expect(page.locator("#first_name")).toHaveValue("Ada");
  await expect(page.locator("#last_name")).toHaveValue("Lovelace");
  await expect(page.locator("#display_name")).toHaveValue("ada");

  expect(watch.apiFailures).toEqual([]);
  expect(watch.oldApiRequests).toEqual([]);
  expect(watch.pageErrors).toEqual([]);
  await expectQuietConsole(page, watch, { warnings: [EMOJI_CHECK_WARNING, EMOJI_CHECK_WARNING] });
});

test("A8 (page): the general page saves the names nerve takes, an empty last name too, and says under the field why nerve refused one", async ({
  api,
  db,
  signedInPage,
}, testInfo) => {
  const email = emailFor(testInfo);
  const page = await signedInPage(await registerOnboarded(api, email));
  const watch = await watchPage(page);
  await page.goto("/settings/profile/general");
  const save = () => page.getByRole("button", { name: "Save changes" }).click();

  // The rules of the names are nerve's (M2 design 4.2): the last name may stay empty, as onboarding leaves it,
  // and a display name may have a space.
  await expect(page.locator("#last_name")).toHaveValue("");
  await page.locator("#first_name").fill("Ada");
  await page.locator("#display_name").fill("Ada Lovelace");
  expect((await answerTo(page, "PATCH", "/api/v0/me", save)).status()).toBe(200);
  await expect(page.getByText("Your profile is updated.")).toBeVisible();
  const saved = await accountOf(db, email);
  expect(saved).toMatchObject({ first_name: "Ada", last_name: "", display_name: "Ada Lovelace" });

  // A web address in the first name: nerve refuses it, the page says why under that field and nowhere else,
  // and nothing changes.
  await page.locator("#first_name").fill("Ada example.com");
  expect((await answerTo(page, "PATCH", "/api/v0/me", save)).status()).toBe(422);
  await expect(fieldBlock(page, "first_name").getByText("Must not contain a web address")).toBeVisible();
  await expect(page.getByText("Must not contain a web address")).toHaveCount(1);
  await expect(page.getByText("Some fields are not valid.")).toHaveCount(0);
  expect(await accountOf(db, email)).toEqual(saved);

  expect(watch.apiFailures).toEqual(["422 PATCH /api/v0/me"]);
  expect(watch.oldApiRequests).toEqual([]);
  expect(watch.pageErrors).toEqual([]);
  await expectQuietConsole(page, watch, {
    warnings: [EMOJI_CHECK_WARNING],
    errors: ["Failed to load resource: the server responded with a status of 422 (Unprocessable Entity)"],
  });
});

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
