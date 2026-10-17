import { changeRole, createWorkspace, slugFor } from "../../fixtures/api";
import { expectWorkspaceDeleted } from "../../fixtures/assert/workspace";
import { accountId, anotherTabSignsIn, emailFor } from "../../fixtures/auth";
import { EMOJI_CHECK_WARNING, expectQuietConsole, watchPage } from "../../fixtures/browser";
import { closedByEscape } from "../../fixtures/dialogs";
import { furnishWorkspace } from "../../fixtures/furnish";
import { answerTo, holdAnswer, registerOnboarded, sentTo } from "../../fixtures/settings-pages";
import { expect, test } from "../../fixtures/test";
import { anotherBrowser, confirmDeletion, deleteFromGeneralPage, pickTimeZone } from "../../fixtures/workspace-pages";

// W3, the workspace's settings (M3 design 2): the page versions, with the session switch of 7.1. The API version is
// in w3-workspace-settings.spec.ts.

test("W3 (page): the admin changes the name, size and time zone, which hold after a reload; a member sees them and can change nothing; demoted, the admin is refused and told why; the admin deletes the workspace by its name and lands on his other one", async ({
  api,
  baseURL,
  browser,
  db,
  signedInPage,
}, testInfo) => {
  const adminEmail = emailFor(testInfo, "admin");
  const admin = await registerOnboarded(api, adminEmail);
  const memberEmail = emailFor(testInfo, "member");
  const member = await registerOnboarded(api, memberEmail);
  const slug = slugFor(testInfo);
  const other = slugFor(testInfo, "other");
  await createWorkspace(api, admin.access_token, { name: "Acme", slug });
  await createWorkspace(api, admin.access_token, { name: "Zeta", slug: other });
  const memberAccount = { email: memberEmail, token: member.access_token };
  await furnishWorkspace(api, admin.access_token, slug, memberAccount, emailFor(testInfo, "invitee"));
  const memberId = await accountId(api, member.access_token);

  const page = await signedInPage(admin);
  const watch = await watchPage(page);
  await page.goto(`/${slug}/settings`);
  await expect(page.locator("#name")).toHaveValue("Acme");
  // Sent empty, the name's rule fails: the name takes the focus, its message its description.
  await page.locator("#name").fill("");
  await page.getByRole("button", { name: "Update workspace" }).click();
  await expect(page.locator("#name")).toBeFocused();
  await expect(page.locator("#name")).toHaveAccessibleDescription("Workspace name is required");
  await page.locator("#name").fill("Acme Corp");
  await page.getByRole("button", { name: "Select organization size" }).click();
  await page.getByRole("option", { name: "11-50" }).click();
  // The time zone by the keyboard alone (M3 design 7.7): the button then names the zone picked, and keeps the focus.
  await pickTimeZone(page, "UTC", "Asia/Shanghai");
  await expect(page.getByRole("button", { name: "Beijing" })).toBeFocused();
  // The page sends the fields its form edits, the size as nerve's OrganizationSize names it.
  const updated = await sentTo(page, "PATCH", `/api/v0/workspaces/${slug}`, () =>
    page.getByRole("button", { name: "Update workspace" }).click()
  );
  expect([updated.answer.status(), updated.body]).toEqual([
    200,
    { name: "Acme Corp", organization_size: "11-50", timezone: "Asia/Shanghai" },
  ]);
  await expect(page.getByText("Workspace updated successfully")).toBeVisible();
  expect(
    await db.query(
      `SELECT w.name, w.organization_size, w.timezone, w.updated_by_id = u.id AS by_the_admin
         FROM workspaces w JOIN users u ON u.email = $2 WHERE w.slug = $1`,
      [slug, adminEmail]
    )
  ).toEqual([{ name: "Acme Corp", organization_size: "11-50", timezone: "Asia/Shanghai", by_the_admin: true }]);
  await page.reload();
  await expect(page.locator("#name")).toHaveValue("Acme Corp");
  await expect(page.getByRole("button", { name: "11-50" })).toBeVisible();
  await expect(page.getByRole("button", { name: "Beijing" })).toBeVisible();

  // The member sees what nerve holds, in a form he cannot change; he has no update and no deletion.
  const theMember = await anotherBrowser(browser, baseURL ?? "", member);
  const memberWatch = await watchPage(theMember.page);
  await theMember.page.goto(`/${slug}/settings`);
  await expect(theMember.page.locator("#name")).toHaveValue("Acme Corp");
  await expect(theMember.page.locator("#name")).toBeDisabled();
  await expect(theMember.page.getByRole("button", { name: "Update workspace" })).toHaveCount(0);
  await expect(theMember.page.getByRole("button", { name: "Delete", exact: true })).toHaveCount(0);
  expect([memberWatch.apiFailures, memberWatch.oldApiRequests, memberWatch.pageErrors]).toEqual([[], [], []]);
  await expectQuietConsole(theMember.page, memberWatch, { warnings: [EMOJI_CHECK_WARNING] });
  await theMember.close();

  // Made an admin, the member demotes the admin while his page is open: nerve refuses his next update, and the page
  // says why and lets him try again. Made an admin again, he goes on.
  const adminId = await accountId(api, admin.access_token);
  await changeRole(api, admin.access_token, slug, memberId, 20);
  await changeRole(api, member.access_token, slug, adminId, 15);
  const refused = await answerTo(page, "PATCH", `/api/v0/workspaces/${slug}`, () =>
    page.getByRole("button", { name: "Update workspace" }).click()
  );
  expect(refused.status()).toBe(403);
  await expect(page.getByText("Your role does not allow this.")).toBeVisible();
  await expect(page.getByRole("button", { name: "Update workspace" })).toBeEnabled();
  await changeRole(api, member.access_token, slug, adminId, 20);

  // The admin deletes it; the root lands him on his other workspace, and the page says so.
  expect((await deleteFromGeneralPage(page, slug, "Acme Corp")).status()).toBe(204);
  await expect(page).toHaveURL(`/${other}`);
  await expect(page.getByText("Workspace deleted.")).toBeVisible();
  await expectWorkspaceDeleted(db, slug, adminEmail, ["workspace_member_invites"]);
  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([
    [`403 PATCH /api/v0/workspaces/${slug}`],
    [],
    [],
  ]);
  // Two loads of the settings: the first, and the reload; the deletion's landing navigates within the app. The
  // browser's report of the refusal.
  await expectQuietConsole(page, watch, {
    warnings: [EMOJI_CHECK_WARNING, EMOJI_CHECK_WARNING],
    errors: ["Failed to load resource: the server responded with a status of 403 (Forbidden)"],
  });
});

// The session switch (M3 design 7.1, 9.6): a change the page sent before another tab moved it to another account may
// still succeed after, and the page is that account's then. The request reaches nerve before the switch, as X's: sent
// after it, it would be Y's, refused, and the page would stay put without any check of the session.
test("W3 (page): a deletion nerve made before another tab signed another account in, and answered after, neither moves the page nor says so: the page is that account's", async ({
  api,
  context,
  db,
  signedInPage,
}, testInfo) => {
  const x = emailFor(testInfo, "x");
  const y = emailFor(testInfo, "y");
  const xTokens = await registerOnboarded(api, x);
  const yTokens = await registerOnboarded(api, y);
  const slug = slugFor(testInfo);
  await createWorkspace(api, xTokens.access_token, { name: "Doomed", slug });
  await createWorkspace(api, xTokens.access_token, { name: "Kept", slug: slugFor(testInfo, "kept") });
  await createWorkspace(api, yTokens.access_token, { name: "Yours", slug: slugFor(testInfo, "yours") });
  const tabA = await signedInPage(xTokens);
  const watch = await watchPage(tabA);
  await tabA.goto(`/${slug}/settings`);
  await expect(tabA.locator("#name")).toHaveValue("Doomed");

  // X deletes Doomed: nerve deletes it at once, and its answer waits (holdAnswer: route.fetch, later route.fulfill).
  const release = await holdAnswer(tabA, "DELETE", `/api/v0/workspaces/${slug}`);
  await confirmDeletion(tabA, "Doomed");
  // While it is out the dialog cannot be dismissed: Cancel is disabled, and Escape leaves it open.
  await expect(tabA.getByRole("dialog").getByRole("button", { name: "Cancel" })).toBeDisabled();
  expect(await closedByEscape(tabA)).toBe(false);
  const deletedAt = async () =>
    (await db.query<{ deleted_at: Date | null }>(`SELECT deleted_at FROM workspaces WHERE slug = $1`, [slug]))[0]
      ?.deleted_at ?? null;
  await expect.poll(deletedAt).toBeInstanceOf(Date);

  // Tab B keeps a sign-in of Y as the token manager does; tab A follows, and Doomed is not one of Y's.
  await anotherTabSignsIn(context, api, y);
  await expect(tabA.getByText("Workspace not found")).toBeVisible();

  // nerve's answer reaches tab A, waited for from its release on (answerTo), so that the wait's deadline is the
  // answer's alone. A move would come with the deletion's continuation, as its answer settles it: the window of 2 s
  // after the release is orders of magnitude longer than the moment that takes.
  const moved = tabA
    .waitForURL((url) => url.pathname !== `/${slug}/settings`, { timeout: 2_000 })
    .then(
      () => true,
      () => false
    );
  expect((await answerTo(tabA, "DELETE", `/api/v0/workspaces/${slug}`, release)).status()).toBe(204);
  expect(await moved).toBe(false);
  // counted at once, as a retrying check would pass once a toast had gone
  expect(await tabA.getByText("Workspace deleted.").count()).toBe(0);
  await expect(tabA.getByText("Workspace not found")).toBeVisible();
  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], []]);
  // One load of tab A.
  await expectQuietConsole(tabA, watch, { warnings: [EMOJI_CHECK_WARNING] });
});
