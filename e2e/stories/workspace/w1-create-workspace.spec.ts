import { isDeepStrictEqual } from "node:util";

import type { Response } from "@playwright/test";

import { createApi, createWorkspace, invitationTo, inviteAndAccept, slugFor, type Api } from "../../fixtures/api";
import {
  countWorkspaces,
  expectInvitations,
  expectNoWorkspaceAdded,
  expectWorkspaceCreated,
  lastWorkspaceOf,
} from "../../fixtures/assert/workspace";
import { bearer, createPAT, emailFor, register } from "../../fixtures/auth";
import { EMOJI_CHECK_WARNING, expectQuietConsole, watchPage } from "../../fixtures/browser";
import { saveProfileStep } from "../../fixtures/onboarding-pages";
import {
  answerTo,
  enabledWithin,
  holdAnswer,
  holdScripts,
  registerOnboarded,
  sentHeld,
  sentTo,
} from "../../fixtures/settings-pages";
import { expect, test } from "../../fixtures/test";
import { anotherBrowser, invitationLinkOf } from "../../fixtures/workspace-pages";

// W1, create a workspace (M3 design 2, 3.10, 3.11), at /create-workspace and in the onboarding (7.4).

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
  // The form stays busy until the workspace opens: while nerve's answer to the write of the one opened last is on its
  // way (holdAnswer), then while the workspace's page is (its code, held), no second click checks the new workspace's
  // slug again.
  const release = await holdAnswer(page, "PATCH", "/api/v0/me/profile");
  const sent = await sentTo(page, "POST", "/api/v0/workspaces", () =>
    page.getByRole("button", { name: "Create workspace" }).click()
  );
  expect([sent.answer.status(), sent.body]).toEqual([201, { name: "Acme Two", slug, organization_size: "2-10" }]);
  await expect(page.getByText("Workspace created successfully")).toBeVisible();
  await expect(page.getByRole("button", { name: "Creating workspace" })).toBeDisabled();
  const scripts = await holdScripts(page);
  expect((await answerTo(page, "PATCH", "/api/v0/me/profile", release)).status()).toBe(200);
  await scripts.requested;
  expect(await enabledWithin(page.getByRole("button", { name: /^Creat(e|ing) workspace$/ }))).toBe(false);
  scripts.release();
  await expect(page).toHaveURL(`/${slug}`);
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

/** What finishing the onboarding writes: every step done, and the account onboarded. */
const FINISHED = {
  onboarding_step: { profile_complete: true, workspace_join: true, workspace_create: true, workspace_invite: true },
  is_onboarded: true,
};

/** nerve's answer when it is busy, a problem the page says by its code. */
const BUSY = {
  status: 503,
  contentType: "application/problem+json",
  body: JSON.stringify({ type: "about:blank", title: "Service Unavailable", status: 503, code: "server_busy" }),
};

test("W1 (page): a newcomer's onboarding creates a workspace after the profile step, written as the one opened last; invites to it and shows the link to copy; then lands in it", async ({
  api,
  db,
  signedInPage,
}, testInfo) => {
  const email = emailFor(testInfo);
  const invitee = emailFor(testInfo, "invitee");
  const tokens = await register(api, email);
  const page = await signedInPage(tokens);
  await page.context().grantPermissions(["clipboard-read", "clipboard-write"]);
  const watch = await watchPage(page);
  await page.goto("/onboarding");
  await page.getByLabel("Name", { exact: true }).fill("Ada");
  expect(await saveProfileStep(page)).toBe(200);

  // The creation step: nerve's check, the creation, and the workspace created written with the step.
  const slug = slugFor(testInfo);
  await page.locator("#name").fill("Acme");
  await page.locator("#slug").fill(slug);
  await page.getByRole("button", { name: "2-10", exact: true }).click();
  let created: { body: unknown; answer: Response } | undefined;
  const stepped = await sentTo(page, "PATCH", "/api/v0/me/profile", async () => {
    created = await sentTo(page, "POST", "/api/v0/workspaces", () =>
      page.getByRole("button", { name: "Create workspace" }).click()
    );
  });
  expect([created?.answer.status(), created?.body]).toEqual([201, { name: "Acme", slug, organization_size: "2-10" }]);
  const id = await expectWorkspaceCreated(db, email, {
    name: "Acme",
    slug,
    organization_size: "2-10",
    timezone: "UTC",
  });
  expect([stepped.answer.status(), stepped.body]).toEqual([
    200,
    { onboarding_step: { workspace_create: true }, last_workspace_id: id },
  ]);

  // The invitation step invites to it, and shows the link of each invitation.
  await page.locator('[id="invitations.0.email"]').fill(invitee);
  const invited = await sentTo(page, "POST", `/api/v0/workspaces/${slug}/invitations`, () =>
    page.getByRole("button", { name: "Continue", exact: true }).click()
  );
  expect([invited.answer.status(), invited.body]).toEqual([201, { invitations: [{ email: invitee, role: 15 }] }]);
  await expect(page.getByText("Your invitations")).toBeVisible();
  await page.getByRole("button", { name: "Copy link" }).click();
  await expect(page.getByText("Invite link copied to clipboard")).toBeVisible();
  const invitation = await invitationTo(api, tokens.access_token, slug, invitee);
  expect(await page.evaluate(() => navigator.clipboard.readText())).toBe(
    new URL(invitationLinkOf(invitation), page.url()).href
  );
  await expectInvitations(db, slug, email, [
    { email: invitee, role: 15, accepted: false, responded: false, deleted: false },
  ]);

  // Done: the landing is the workspace created. Continue stays busy while nerve's answer to the end is on its way.
  const proceed = page.getByRole("button", { name: "Continue", exact: true });
  const finished = await sentHeld(page, "PATCH", "/api/v0/me/profile", () => proceed.click());
  expect(await enabledWithin(proceed)).toBe(false);
  expect([(await finished.release()).status(), finished.body]).toEqual([200, FINISHED]);
  await expect(page).toHaveURL(`/${slug}`);
  expect(await lastWorkspaceOf(db, email)).toBe(id);
  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], []]);
  // The workspace's home.
  await expectQuietConsole(page, watch, { warnings: [EMOJI_CHECK_WARNING] });
});

test("W1 (page): a newcomer who puts off the invitations at their step lands in the workspace he created, having invited no one; for one who creates a workspace for himself alone the creation ends the onboarding, the form busy until nerve answers, and an end nerve refuses leads to that step", async ({
  api,
  baseURL,
  browser,
  signedInPage,
}, testInfo) => {
  const page = await signedInPage(await register(api, emailFor(testInfo)));
  const watch = await watchPage(page);
  await page.goto("/onboarding");
  await page.getByLabel("Name", { exact: true }).fill("Ada");
  expect(await saveProfileStep(page)).toBe(200);
  const slug = slugFor(testInfo);
  await page.locator("#name").fill("Acme");
  await page.locator("#slug").fill(slug);
  await page.getByRole("button", { name: "2-10", exact: true }).click();
  const stepped = await sentTo(page, "PATCH", "/api/v0/me/profile", () =>
    page.getByRole("button", { name: "Create workspace" }).click()
  );
  expect(stepped.answer.status()).toBe(200);

  // The invitation step, put off: the onboarding ends, and he lands in the workspace he created. "Later" stays busy
  // while nerve's answer to the end is on its way (sentHeld).
  const putOff = page.getByRole("button", { name: "I’ll do it later", exact: true });
  const finished = await sentHeld(page, "PATCH", "/api/v0/me/profile", () => putOff.click());
  expect(await enabledWithin(putOff)).toBe(false);
  expect([(await finished.release()).status(), finished.body]).toEqual([200, FINISHED]);
  await expect(page).toHaveURL(`/${slug}`);
  expect(watch.apiRequests.filter((request) => request.endsWith(`/api/v0/workspaces/${slug}/invitations`))).toEqual([]);
  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], []]);
  // The workspace's home.
  await expectQuietConsole(page, watch, { warnings: [EMOJI_CHECK_WARNING] });

  // Bob creates a workspace for himself alone: the onboarding ends with the creation. The form stays busy until nerve
  // has answered the end: while its answer to the write of the creation's step is on its way (holdAnswer), no second
  // click checks the new workspace's slug again.
  const there = await anotherBrowser(browser, baseURL ?? "", await register(api, emailFor(testInfo, "bob")));
  await there.page.goto("/onboarding");
  await there.page.getByLabel("Name", { exact: true }).fill("Bob");
  expect(await saveProfileStep(there.page)).toBe(200);
  const own = slugFor(testInfo, "own");
  await there.page.locator("#name").fill("Solo");
  await there.page.locator("#slug").fill(own);
  await there.page.getByRole("button", { name: "Just myself", exact: true }).click();
  const release = await holdAnswer(there.page, "PATCH", "/api/v0/me/profile");
  // nerve is busy when the end comes (its body is FINISHED), once.
  let busy = true;
  await there.page.route("**/api/v0/me/profile", (route) => {
    const end = route.request().method() === "PATCH" && isDeepStrictEqual(route.request().postDataJSON(), FINISHED);
    if (!busy || !end) return route.fallback();
    busy = false;
    return route.fulfill(BUSY);
  });
  const created = await answerTo(there.page, "POST", "/api/v0/workspaces", () =>
    there.page.getByRole("button", { name: "Create workspace" }).click()
  );
  expect(created.status()).toBe(201);
  await expect(there.page.getByText("Workspace created successfully")).toBeVisible();
  // The step's button shows its spinner, whose status gives the button no name.
  await expect(there.page.getByRole("button").filter({ hasText: "Loading..." })).toBeDisabled();
  expect((await answerTo(there.page, "PATCH", "/api/v0/me/profile", release)).status()).toBe(200);

  // The end refused: the page says why, and the creation, over, gives way to the workspace's invitation step, whose
  // "later" ends the onboarding; he lands in his workspace.
  await expect(there.page.getByText("The server is busy. Please try again later.")).toBeVisible();
  await expect(there.page.getByText("Invite your teammates")).toBeVisible();
  const later = await sentTo(there.page, "PATCH", "/api/v0/me/profile", () =>
    there.page.getByRole("button", { name: "I’ll do it later", exact: true }).click()
  );
  expect([later.answer.status(), later.body, busy]).toEqual([200, FINISHED, false]);
  await expect(there.page).toHaveURL(`/${own}`);
  await there.close();
});

test("W1 (page): one who joined a workspace before onboarding is done after the profile step; one who comes back after creating a workspace invites to that one, whatever comes first in his list", async ({
  api,
  baseURL,
  browser,
  signedInPage,
}, testInfo) => {
  const admin = await registerOnboarded(api, emailFor(testInfo, "admin"));
  const alpha = await createWorkspace(api, admin.access_token, { name: "Alpha", slug: slugFor(testInfo, "alpha") });
  const newcomer = async (label: string) => {
    const email = emailFor(testInfo, label);
    const tokens = await register(api, email);
    await inviteAndAccept(api, admin.access_token, alpha.slug, { email, token: tokens.access_token }, 15);
    return tokens;
  };

  // Ada joined Alpha: her profile step is her last, and she lands there. The step stays busy while nerve's answer to
  // the end is on its way (sentHeld).
  const page = await signedInPage(await newcomer("ada"));
  const watch = await watchPage(page);
  await page.goto("/onboarding");
  await page.getByLabel("Name", { exact: true }).fill("Ada");
  const proceed = page.getByRole("button", { name: "Continue", exact: true });
  const finished = await sentHeld(page, "PATCH", "/api/v0/me/profile", () => proceed.click());
  expect(await enabledWithin(proceed)).toBe(false);
  expect([(await finished.release()).status(), finished.body]).toEqual([200, FINISHED]);
  await expect(page).toHaveURL(`/${alpha.slug}`);
  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], []]);
  await expectQuietConsole(page, watch, { warnings: [EMOJI_CHECK_WARNING] });

  // Bob joined Alpha too, and created Zeta, which comes after it, and left the onboarding at its invitation step.
  const bob = await newcomer("bob");
  const zeta = await createWorkspace(api, bob.access_token, { name: "Zeta", slug: slugFor(testInfo, "zeta") });
  const left = await api.PATCH("/api/v0/me/profile", {
    body: { onboarding_step: { profile_complete: true, workspace_create: true }, last_workspace_id: zeta.id },
    headers: bearer(bob.access_token),
  });
  expect(left.response.status).toBe(200);
  const there = await anotherBrowser(browser, baseURL ?? "", bob);
  await there.page.goto("/onboarding");
  await expect(there.page.getByText("Invite your teammates")).toBeVisible();
  await there.page.locator('[id="invitations.0.email"]').fill(emailFor(testInfo, "carol"));
  const invited = await answerTo(there.page, "POST", `/api/v0/workspaces/${zeta.slug}/invitations`, () =>
    there.page.getByRole("button", { name: "Continue", exact: true }).click()
  );
  expect(invited.status()).toBe(201);
  await there.close();
});

test("W1 (page): on a nerve with creation switched off, a newcomer's onboarding says to ask for an invitation link after the profile step", async ({
  browser,
  nerveWith,
}, testInfo) => {
  const closed = await nerveWith({ NERVE_WORKSPACE__CREATION_ENABLED: "false" });
  const tokens = await register(createApi(closed.baseURL), emailFor(testInfo));
  const there = await anotherBrowser(browser, closed.baseURL, tokens);
  await there.page.goto("/onboarding");
  await there.page.getByLabel("Name", { exact: true }).fill("Ada");
  expect(await saveProfileStep(there.page)).toBe(200);
  await expect(
    there.page.getByText(
      "Creating workspaces is switched off on this instance: ask a workspace's admin for an invitation link."
    )
  ).toBeVisible();
  await expect(there.page.locator("#name")).toHaveCount(0);
  await there.close();
});
