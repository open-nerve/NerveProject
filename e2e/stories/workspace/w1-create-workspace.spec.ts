import { createApi, createWorkspace, slugFor, type Api } from "../../fixtures/api";
import {
  countWorkspaces,
  expectNoWorkspaceAdded,
  expectWorkspaceCreated,
  lastWorkspaceOf,
} from "../../fixtures/assert/workspace";
import { bearer, createPAT, emailFor, register } from "../../fixtures/auth";
import { EMOJI_CHECK_WARNING, expectQuietConsole, watchPage } from "../../fixtures/browser";
import { answerTo, registerOnboarded, sentTo } from "../../fixtures/settings-pages";
import { expect, test } from "../../fixtures/test";
import { anotherBrowser } from "../../fixtures/workspace-pages";

// W1, create a workspace (M3 design 2, 3.10, 3.11).

/** The answer of GET /api/v0/workspace-slugs/{slug}. */
async function availability(api: Api, token: string, slug: string): Promise<unknown> {
  const { data, error, response } = await api.GET("/api/v0/workspace-slugs/{slug}", {
    params: { path: { slug } },
    headers: bearer(token),
  });
  expect(response.status, `check ${slug}: ${JSON.stringify(error)}`).toBe(200);
  return data;
}

test("W1 (API): creating a workspace makes the caller its admin and only member; a taken or reserved slug, or creation switched off, adds nothing", async ({
  api,
  db,
  nerveWith,
}, testInfo) => {
  const email = emailFor(testInfo);
  const pat = (await createPAT(api, (await register(api, email)).access_token)).token;
  const slug = slugFor(testInfo);
  expect(await availability(api, pat, slug)).toEqual({ available: true });

  const body = { name: "Acme", slug, organization_size: "2-10", timezone: "Asia/Shanghai" } as const;
  const created = await createWorkspace(api, pat, body);

  expect(created).toMatchObject({ ...body, logo_url: null, role: 20, total_members: 1 });
  expect(await expectWorkspaceCreated(db, email, body)).toBe(created.id);
  expect(await availability(api, pat, slug)).toEqual({ available: false, reason: "taken" });
  expect(await availability(api, pat, "settings")).toEqual({ available: false, reason: "reserved" });

  const before = await countWorkspaces(db);
  const taken = await api.POST("/api/v0/workspaces", { body: { name: "Acme again", slug }, headers: bearer(pat) });
  expect(taken.response.status).toBe(409);
  expect(taken.error?.code).toBe("workspace.slug_taken");
  const reserved = await api.POST("/api/v0/workspaces", {
    body: { name: "Settings", slug: "settings" },
    headers: bearer(pat),
  });
  expect(reserved.response.status).toBe(422);
  expect(reserved.error?.errors?.map((e) => ({ field: e.field, code: e.code }))).toEqual([
    { field: "slug", code: "not_allowed" },
  ]);
  // A nerve with creation switched off, on the same database: the personal
  // access token works there too.
  const closed = createApi((await nerveWith({ NERVE_WORKSPACE__CREATION_ENABLED: "false" })).baseURL);
  const refused = await closed.POST("/api/v0/workspaces", {
    body: { name: "Beta", slug: slugFor(testInfo, "beta") },
    headers: bearer(pat),
  });
  expect(refused.response.status).toBe(403);
  expect(refused.error?.code).toBe("workspace.creation_disabled");
  await expectNoWorkspaceAdded(db, before);
  // No refusal changed the workspace created first.
  expect(await expectWorkspaceCreated(db, email, body)).toBe(created.id);
});

test("W1 (page): at /create-workspace an onboarded account is told under the field of a taken, a reserved or an invalid slug, and nothing is created; then it creates a workspace with the slug the field shows, which opens, written as the one opened last; on a nerve with creation switched off the page says so", async ({
  api,
  browser,
  db,
  nerveWith,
  signedInPage,
}, testInfo) => {
  const email = emailFor(testInfo);
  const tokens = await registerOnboarded(api, email);
  const taken = slugFor(testInfo, "taken");
  await createWorkspace(api, tokens.access_token, { name: "Taken", slug: taken });
  const page = await signedInPage(tokens);
  const watch = await watchPage(page);
  await page.goto("/create-workspace");
  await page.locator("#workspaceName").fill("Acme Two");
  await page.getByRole("button", { name: "Select a range" }).click();
  await page.getByRole("option", { name: "2-10" }).click();
  const url = page.locator("#workspaceUrl");
  // The name gives the slug.
  await expect(url).toHaveValue("acme-two");

  // nerve's check of the slug decides: a refused one is said under the field, and the form is not sent.
  const refused = async (slug: string, message: string) => {
    await url.fill(slug);
    const checked = await answerTo(page, "GET", `/api/v0/workspace-slugs/${encodeURIComponent(slug)}`, () =>
      page.getByRole("button", { name: "Create workspace" }).click()
    );
    expect(checked.status()).toBe(200);
    await expect(page.getByText(message)).toBeVisible();
  };
  await refused(taken, "Workspace URL is already taken!");
  await refused("settings", "This URL is reserved: choose another.");
  await refused("café", "URLs can contain only lower-case letters, digits, '-' and '_'.");

  // The field shows the slug it sends: in lower case, a space as "-".
  const slug = slugFor(testInfo);
  await url.fill(slug.replace("-", " ").toUpperCase());
  await expect(url).toHaveValue(slug);
  const sent = await sentTo(page, "POST", "/api/v0/workspaces", () =>
    page.getByRole("button", { name: "Create workspace" }).click()
  );
  expect([sent.answer.status(), sent.body]).toEqual([201, { name: "Acme Two", slug, organization_size: "2-10" }]);
  await expect(page).toHaveURL(`/${slug}`);
  await expect(page.getByText("Workspace created successfully")).toBeVisible();
  const id = await expectWorkspaceCreated(db, email, {
    name: "Acme Two",
    slug,
    organization_size: "2-10",
    timezone: "UTC",
  });
  await expect.poll(() => lastWorkspaceOf(db, email)).toBe(id);
  expect(watch.apiRequests.filter((request) => request === "POST /api/v0/workspaces")).toHaveLength(1);
  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], []]);
  // The new workspace's home.
  await expectQuietConsole(page, watch, { warnings: [EMOJI_CHECK_WARNING] });

  // A nerve with creation switched off, on the same database: the page says so, and offers no form, nor a letter to
  // an instance admin, whom nerve does not have.
  const closed = await nerveWith({ NERVE_WORKSPACE__CREATION_ENABLED: "false" });
  const closedTokens = await registerOnboarded(createApi(closed.baseURL), emailFor(testInfo, "closed"));
  const there = await anotherBrowser(browser, closed.baseURL, closedTokens);
  await there.page.goto("/create-workspace");
  await expect(there.page.getByText("Creating workspaces is switched off", { exact: true })).toBeVisible();
  await expect(there.page.getByText("Ask a workspace's admin for an invitation link.")).toBeVisible();
  await expect(there.page.locator("#workspaceName")).toHaveCount(0);
  await expect(there.page.locator('a[href^="mailto:"]')).toHaveCount(0);
  await there.close();
});
