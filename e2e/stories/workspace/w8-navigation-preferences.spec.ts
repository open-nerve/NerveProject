import {
  createProject,
  createWorkspace,
  inviteAndAccept,
  slugFor,
  type Api,
  type WorkspacePreferences,
  type WorkspacePreferencesUpdate,
} from "../../fixtures/api";
import { expectPreferences } from "../../fixtures/assert/workspace";
import { bearer, createPAT, emailFor, register } from "../../fixtures/auth";
import { EMOJI_CHECK_WARNING, expectQuietConsole, watchPage } from "../../fixtures/browser";
import { answerTo, bodiesSentTo, holdAnswer, registerOnboarded, sentTo } from "../../fixtures/settings-pages";
import { expect, test } from "../../fixtures/test";

// W8, the project navigation's settings (M3 design 2, 3.18): through the API, and through the sidebar's "project
// navigation" dialog.

const defaults: WorkspacePreferences = { navigation_control_preference: "ACCORDION", navigation_project_limit: 10 };

/** The answer of GET /api/v0/me/workspaces/{slug}/preferences, which must be 200. */
async function read(api: Api, token: string, slug: string): Promise<unknown> {
  const { data, error, response } = await api.GET("/api/v0/me/workspaces/{slug}/preferences", {
    params: { path: { slug } },
    headers: bearer(token),
  });
  expect(response.status, `read the settings in ${slug}: ${JSON.stringify(error)}`).toBe(200);
  return data;
}

/** The answer of PATCH /api/v0/me/workspaces/{slug}/preferences, which must be 200. */
async function change(api: Api, token: string, slug: string, body: WorkspacePreferencesUpdate): Promise<unknown> {
  const { data, error, response } = await api.PATCH("/api/v0/me/workspaces/{slug}/preferences", {
    params: { path: { slug } },
    body,
    headers: bearer(token),
  });
  expect(response.status, `change the settings in ${slug}: ${JSON.stringify(error)}`).toBe(200);
  return data;
}

test("W8 (API): the settings are the defaults and nothing is stored until the first change, which stores one row; later changes change it; each member and each workspace has its own", async ({
  api,
  db,
}, testInfo) => {
  const email = emailFor(testInfo);
  const pat = (await createPAT(api, (await register(api, email)).access_token)).token;
  const slug = slugFor(testInfo);
  const other = slugFor(testInfo, "other");
  await createWorkspace(api, pat, { name: "Acme", slug });
  await createWorkspace(api, pat, { name: "Other", slug: other });

  expect(await read(api, pat, slug)).toEqual(defaults);
  await expectPreferences(db, slug, email, null);

  const tabbed: WorkspacePreferences = { navigation_control_preference: "TABBED", navigation_project_limit: 3 };
  expect(await change(api, pat, slug, tabbed)).toEqual(tabbed);
  const row = await expectPreferences(db, slug, email, tabbed);
  // After a refresh the page reads them again.
  expect(await read(api, pat, slug)).toEqual(tabbed);

  // Showing every project: one field changes, the other stays, on the same row.
  expect(await change(api, pat, slug, { navigation_project_limit: 0 })).toEqual({
    ...tabbed,
    navigation_project_limit: 0,
  });
  expect(await expectPreferences(db, slug, email, { ...tabbed, navigation_project_limit: 0 })).toBe(row);

  // Another member of the workspace reads the defaults, and nothing is stored for him.
  const memberEmail = emailFor(testInfo, "member");
  const member = (await createPAT(api, (await register(api, memberEmail)).access_token)).token;
  await inviteAndAccept(api, pat, slug, { email: memberEmail, token: member }, 15);
  expect(await read(api, member, slug)).toEqual(defaults);
  await expectPreferences(db, slug, memberEmail, null);

  // The other workspace keeps the defaults, and nothing is stored for it.
  expect(await read(api, pat, other)).toEqual(defaults);
  await expectPreferences(db, other, email, null);

  // Another account, not a member, reads nothing there, and nothing is stored for it.
  const strangerEmail = emailFor(testInfo, "stranger");
  const stranger = (await register(api, strangerEmail)).access_token;
  const refused = await api.GET("/api/v0/me/workspaces/{slug}/preferences", {
    params: { path: { slug } },
    headers: bearer(stranger),
  });
  expect(refused.response.status).toBe(404);
  expect(refused.error?.code).toBe("workspace.not_found");
  await expectPreferences(db, slug, strangerEmail, null);
});

test("W8 (page): in his workspace's sidebar the caller makes the project navigation tabs and limits it to 3 projects, the count sent once, as he leaves its field; so it stays after a refresh; a count nerve refuses is said, and the sidebar keeps its 3", async ({
  api,
  db,
  signedInPage,
}, testInfo) => {
  const email = emailFor(testInfo);
  const tokens = await registerOnboarded(api, email);
  const slug = slugFor(testInfo);
  await createWorkspace(api, tokens.access_token, { name: "Acme", slug });
  await Promise.all(
    ["ALPHA", "BETA", "GAMMA", "DELTA"].map((identifier) =>
      createProject(api, tokens.access_token, slug, { name: identifier, identifier })
    )
  );
  const path = `/api/v0/me/workspaces/${slug}/preferences`;
  const page = await signedInPage(tokens);
  const watch = await watchPage(page);
  await page.goto(`/${slug}`);

  // The home's tour comes first, which he skips: the page writes its end in his profile.
  const skipped = await sentTo(page, "PATCH", "/api/v0/me/profile", () =>
    page.getByRole("button", { name: "No thanks, I will explore it myself" }).click()
  );
  expect([skipped.answer.status(), skipped.body]).toEqual([200, { is_tour_completed: true }]);
  await expect(page.getByRole("button", { name: "Take a Product Tour" })).toHaveCount(0);

  // nerve's defaults: up to 10 projects, as an accordion, so the sidebar shows the 4.
  const sidebar = page.getByRole("complementary", { name: "Main sidebar" });
  const projects = sidebar.locator('[id^="sidebar-"][id$="-JOINED"]');
  await expect(projects).toHaveCount(4);
  const dialog = page.getByRole("dialog");
  const tabs = dialog.getByRole("radio", { name: /Tabbed Navigation/ });
  const limited = dialog.getByRole("checkbox", { name: "Show limited projects on sidebar" });
  const count = dialog.getByLabel("Enter number of projects");
  await sidebar.getByRole("button", { name: "Project navigation" }).click();
  await expect(limited).toBeChecked();
  await expect(count).toHaveValue("10");

  const tabbed = await sentTo(page, "PATCH", path, () => tabs.click());
  expect([tabbed.answer.status(), tabbed.body]).toEqual([200, { navigation_control_preference: "TABBED" }]);
  await expect(tabs).toBeChecked();
  // Every project, then a limit again, in two quick turns: the second is asked for before nerve answers the first, and
  // made to that answer (v0 design 7.7), so they end where they began, at nerve's default count.
  const release = await holdAnswer(page, "PATCH", path);
  const turns = await bodiesSentTo(page, "PATCH", path);
  await limited.click();
  await expect.poll(() => turns.length).toBe(1);
  await limited.click();
  await release();
  await expect.poll(() => turns).toEqual([{ navigation_project_limit: 0 }, { navigation_project_limit: 10 }]);
  await expect(limited).toBeChecked();
  await expect(count).toHaveValue("10");
  // He types 30 and corrects it to 3: the count goes once, as he leaves the field.
  const limit = await sentTo(page, "PATCH", path, async () => {
    await count.fill("30");
    await count.press("Backspace");
    await count.blur();
  });
  expect([limit.answer.status(), limit.body]).toEqual([200, { navigation_project_limit: 3 }]);
  await expect(count).toHaveValue("3");
  await dialog.getByRole("button", { name: "Close" }).click();
  await expect(projects).toHaveCount(3);
  const settings: WorkspacePreferences = { navigation_control_preference: "TABBED", navigation_project_limit: 3 };
  await expectPreferences(db, slug, email, settings);

  // After a refresh: 3 projects, and the dialog shows his settings, its count nerve's, not one taken as it mounted.
  await page.reload();
  await expect(projects).toHaveCount(3);
  await sidebar.getByRole("button", { name: "Project navigation" }).click();
  await expect(tabs).toBeChecked();
  await expect(count).toHaveValue("3");

  // A count beyond nerve's bounds: nerve refuses it, the page says why, and the field and the sidebar keep 3.
  const beyond = await answerTo(page, "PATCH", path, async () => {
    await count.fill("3000000000");
    await count.blur();
  });
  expect(beyond.status()).toBe(422);
  await expect(page.getByText("Some fields are not valid.")).toBeVisible();
  await expect(count).toHaveValue("3");
  await expect(projects).toHaveCount(3);
  await expectPreferences(db, slug, email, settings);
  expect(watch.apiRequests.filter((request) => request === `PATCH ${path}`)).toHaveLength(5);
  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[`422 PATCH ${path}`], [], []]);
  // Two loads: the first and the refresh.
  await expectQuietConsole(page, watch, {
    warnings: [EMOJI_CHECK_WARNING, EMOJI_CHECK_WARNING],
    errors: ["Failed to load resource: the server responded with a status of 422 (Unprocessable Entity)"],
  });
});
