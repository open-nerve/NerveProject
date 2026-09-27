import { accountOf } from "../../fixtures/assert/identity";
import { bearer, createPAT, emailFor, register } from "../../fixtures/auth";
import { EMOJI_CHECK_WARNING, expectQuietConsole, watchPage } from "../../fixtures/browser";
import { answerTo, expectListBesideButton, registerOnboarded } from "../../fixtures/settings-pages";
import { expect, test } from "../../fixtures/test";

// A9, changing the preferences (M2 design 2), with the theme's list beside
// its button in both languages (7.7, 9.6).

const profileOf = async (db: Parameters<typeof accountOf>[0], email: string) =>
  db.query("SELECT theme, language, start_of_the_week FROM profiles WHERE user_id = $1", [
    (await accountOf(db, email)).id,
  ]);

test("A9 (page): the preferences page changes the theme, the language and the first day of the week, each list beside its button; they hold after a reload", async ({
  api,
  db,
  signedInPage,
}, testInfo) => {
  const email = emailFor(testInfo);
  const page = await signedInPage(await registerOnboarded(api, email));
  const watch = await watchPage(page);
  await page.goto("/settings/profile/preferences");

  // In English, the theme's list opens beside its button. Dark: the page reloads itself to apply it.
  const theme = page.getByRole("button", { name: "System Preference" });
  await theme.click();
  await expectListBesideButton(page, theme);
  const reloaded = page.waitForEvent("load", { timeout: 10_000 });
  const themed = await answerTo(page, "PATCH", "/api/v0/me/profile", () =>
    page.getByRole("option", { name: "Dark", exact: true }).click()
  );
  expect(themed.status()).toBe(200);
  await reloaded;
  await expect(page.getByRole("button", { name: "Sunday" })).toBeVisible();

  // The language: the page is in Chinese at once, the first day of the week too.
  const language = page.getByRole("button", { name: "English" });
  await language.click();
  await expectListBesideButton(page, language);
  const spoken = await answerTo(page, "PATCH", "/api/v0/me/profile", () =>
    page.getByRole("option", { name: "简体中文" }).click()
  );
  expect(spoken.status()).toBe(200);
  await expect(page.getByText("语言和时间")).toBeVisible();
  await expect(page.getByText("一周的第一天", { exact: true })).toBeVisible();

  // In Chinese, the theme's list opens beside its button too; Escape closes it, and the next click opens it again.
  const themeZh = page.getByRole("button", { name: "深色" });
  await themeZh.click();
  await expectListBesideButton(page, themeZh);
  await page.keyboard.press("Escape");
  await expect(page.getByRole("listbox")).toHaveCount(0);
  await themeZh.click();
  await expectListBesideButton(page, themeZh);
  await page.keyboard.press("Escape");
  await expect(page.getByRole("listbox")).toHaveCount(0);

  // The time zone's list speaks Chinese: its search, and what it says when nothing matches.
  const timezone = page.getByRole("button", { name: "UTC" });
  await timezone.click();
  await expectListBesideButton(page, timezone);
  await page.getByPlaceholder("搜索").fill("no such place");
  await expect(page.getByText("未找到匹配项")).toBeVisible();
  await page.keyboard.press("Escape");
  await expect(page.getByRole("listbox")).toHaveCount(0);

  // The first day of the week, from the keyboard: Tab reaches its button from the language's; an arrow opens its list
  // beside it (Enter does not: it submits an enclosing form, as a native select's does), the arrows move, Enter picks
  // and closes it.
  const week = page.getByRole("button", { name: "星期日" });
  await page.getByRole("button", { name: "简体中文" }).focus();
  await page.keyboard.press("Tab");
  await expect(week).toBeFocused();
  await page.keyboard.press("ArrowDown");
  await expectListBesideButton(page, week);
  await expect(page.getByRole("option", { name: "星期日" })).toBeVisible();
  await page.keyboard.press("ArrowDown");
  const started = await answerTo(page, "PATCH", "/api/v0/me/profile", () => page.keyboard.press("Enter"));
  expect(started.status()).toBe(200);
  await expect(page.getByRole("listbox")).toHaveCount(0);
  // Space opens it again; Escape closes it, and the focus is back on its button.
  const weekMonday = page.getByRole("button", { name: "星期一" });
  await expect(weekMonday).toBeFocused();
  await page.keyboard.press("Space");
  await expectListBesideButton(page, weekMonday);
  await page.keyboard.press("Escape");
  await expect(page.getByRole("listbox")).toHaveCount(0);
  await expect(weekMonday).toBeFocused();
  // A click on the day already picked closes the list: it sends that day again.
  await weekMonday.click();
  await expectListBesideButton(page, weekMonday);
  const picked = page.getByRole("option", { selected: true });
  await expect(picked).toHaveText("星期一");
  const repicked = await answerTo(page, "PATCH", "/api/v0/me/profile", () => picked.click());
  expect(repicked.status()).toBe(200);
  expect(await repicked.json()).toMatchObject({ start_of_the_week: 1 });
  await expect(page.getByRole("listbox")).toHaveCount(0);

  const change = { theme: "dark", language: "zh-CN", start_of_the_week: 1 };
  expect(await profileOf(db, email)).toEqual([change]);
  // One change for each pick, the day picked again included; the time zone's search sent none.
  expect(watch.apiRequests.filter((request) => request.startsWith("PATCH "))).toEqual(
    Array(4).fill("PATCH /api/v0/me/profile")
  );
  // After a reload, the page shows them all, in Chinese.
  await page.reload();
  await expect(page.getByRole("button", { name: "深色" })).toBeVisible();
  await expect(page.getByRole("button", { name: "简体中文" })).toBeVisible();
  await expect(page.getByRole("button", { name: "星期一" })).toBeVisible();

  expect(watch.apiFailures).toEqual([]);
  expect(watch.oldApiRequests).toEqual([]);
  expect(watch.pageErrors).toEqual([]);
  // Three loads: the first, the theme's own reload, and the last.
  await expectQuietConsole(page, watch, {
    warnings: [EMOJI_CHECK_WARNING, EMOJI_CHECK_WARNING, EMOJI_CHECK_WARNING],
  });
});

/** nerve's answer when it cannot make a change, as a problem of status and code: the page says why by the code. */
const refusal = (status: number, title: string, code: string) => ({
  status,
  contentType: "application/problem+json",
  body: JSON.stringify({ type: "about:blank", title, status, code }),
});

test("A9 (page): a refused change leaves the page as nerve has it", async ({ api, db, signedInPage }, testInfo) => {
  const email = emailFor(testInfo);
  const page = await signedInPage(await registerOnboarded(api, email));
  const watch = await watchPage(page);
  await page.goto("/settings/profile/preferences");
  const language = page.getByRole("button", { name: "English" });
  const theme = page.getByRole("button", { name: "System Preference" });
  const html = page.locator("html");
  await expect(language).toBeVisible();
  await expect(theme).toBeVisible();
  await expect(html).toHaveAttribute("lang", "en");
  await expect(html).toHaveAttribute("data-theme", "light");
  await page.evaluate(() => {
    (window as unknown as { beforeRefusals?: true }).beforeRefusals = true;
  });
  // nerve refuses the next two changes of the profile: first as a server error, then as busy.
  const refusals = [
    refusal(500, "Internal Server Error", "internal_error"),
    refusal(503, "Service Unavailable", "server_busy"),
  ];
  await page.route("**/api/v0/me/profile", (route) => {
    const next = route.request().method() === "PATCH" ? refusals.shift() : undefined;
    return next ? route.fulfill(next) : route.fallback();
  });

  // The language: 简体中文 refused; the page stays in English, its button says English, and the toast says why.
  await language.click();
  const spoken = await answerTo(page, "PATCH", "/api/v0/me/profile", () =>
    page.getByRole("option", { name: "简体中文" }).click()
  );
  expect(spoken.status()).toBe(500);
  await expect(page.getByText("Something went wrong on the server. Please try again.")).toBeVisible();
  await expect(language).toBeVisible();
  await expect(page.getByText("Language & Time")).toBeVisible();

  // The theme: Dark refused; the page keeps its theme, says why, and does not reload.
  await expect(theme).toBeVisible();
  await theme.click();
  const themed = await answerTo(page, "PATCH", "/api/v0/me/profile", () =>
    page.getByRole("option", { name: "Dark", exact: true }).click()
  );
  expect(themed.status()).toBe(503);
  await expect(page.getByText("The server is busy. Please try again later.")).toBeVisible();
  await expect(html).toHaveAttribute("data-theme", "light");
  await expect(theme).toBeVisible();

  // Both refusals behind it, the page is as nerve has it: in English, in its theme, never reloaded.
  await expect(html).toHaveAttribute("lang", "en");
  await expect(page.getByText("Language & Time")).toBeVisible();
  await expect(language).toBeVisible();
  expect(await page.evaluate(() => (window as unknown as { beforeRefusals?: true }).beforeRefusals)).toBe(true);
  expect(await profileOf(db, email)).toEqual([{ theme: "system", language: "en", start_of_the_week: 0 }]);
  expect(watch.apiFailures).toEqual(["500 PATCH /api/v0/me/profile", "503 PATCH /api/v0/me/profile"]);
  expect(watch.oldApiRequests).toEqual([]);
  expect(watch.pageErrors).toEqual([]);
  // One load; the browser's report of each refusal.
  await expectQuietConsole(page, watch, {
    warnings: [EMOJI_CHECK_WARNING],
    errors: [
      "Failed to load resource: the server responded with a status of 500 (Internal Server Error)",
      "Failed to load resource: the server responded with a status of 503 (Service Unavailable)",
    ],
  });
});

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
  expect(await profileOf(db, email)).toEqual([change]);
  const read = await api.GET("/api/v0/me/profile", { headers: bearer(pat.token) });
  expect(read.data).toEqual(data);
});
