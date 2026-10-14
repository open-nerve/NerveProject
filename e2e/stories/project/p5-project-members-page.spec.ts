import {
  addProjectMembers,
  createProject,
  createWorkspace,
  inviteAndAccept,
  projectMemberWrites,
  projectMembershipOf,
  slugFor,
} from "../../fixtures/api";
import { expectMembers, type MemberRow } from "../../fixtures/assert/project";
import { accountId, emailFor, newAccount, shownNameOf } from "../../fixtures/auth";
import { EMOJI_CHECK_WARNING, expectQuietConsole, watchPage } from "../../fixtures/browser";
import {
  closedByEscape,
  closedWithin,
  enabledWithin,
  modalDialog,
  pageBehindHidden,
  sentWithin,
} from "../../fixtures/dialogs";
import { chooseByKeyboard } from "../../fixtures/menus";
import { answerTo, holdAnswer, registerOnboarded, sentHeld, sentTo } from "../../fixtures/settings-pages";
import { endProjectMembership, leavingOf, removalOf } from "../../fixtures/project-pages";
import { expect, test } from "../../fixtures/test";
import { anotherBrowser, memberRow } from "../../fixtures/workspace-pages";

// P5's page version (M3 design 2, 3.5, 3.7, 7.6): a project's members on its settings page, as a project admin who is
// no workspace admin adds, changes and removes them, and as its members leave. The API version is
// p5-project-members.spec.ts.

test("P5 (page): a project admin who is no workspace admin filters the members by their roles with the keyboard alone, adds a member and a guest from the workspace's members who are not the project's, Enter on a select opening its list, the modal held until nerve answers; he is offered only the roles below his own, none for another admin; he makes a member a guest, removes the other admin and a guest, the dialog held until nerve answers, and, its only admin now, is told why he may not leave, its dialog open; a member leaves, and the workspace's projects show once nerve has made it, not before; the guest leaves by the sidebar, its modal open after a refusal and held too", async ({
  api,
  baseURL,
  browser,
  db,
  signedInPage,
}, testInfo) => {
  const admin = await newAccount(api, testInfo, "admin");
  const slug = slugFor(testInfo);
  await createWorkspace(api, admin.token, { name: "Acme", slug });
  // pat, bob and ann open the pages; max and gus do not
  const [pat, bob, ann] = await Promise.all(
    ["pat", "bob", "ann"].map(async (label) => {
      const email = emailFor(testInfo, label);
      const tokens = await registerOnboarded(api, email);
      await inviteAndAccept(api, admin.token, slug, { email, token: tokens.access_token }, 15);
      return { email, tokens, id: await accountId(api, tokens.access_token) };
    })
  );
  const [max, gus] = await Promise.all(["max", "gus"].map((label) => newAccount(api, testInfo, label)));
  if (!pat || !bob || !ann || !max || !gus) throw new Error("the accounts were not registered");
  await inviteAndAccept(api, admin.token, slug, max, 15);
  await inviteAndAccept(api, admin.token, slug, gus, 5);
  // pat, a member of acme, makes Web, its admin; max its other admin, bob its member.
  const web = await createProject(api, pat.tokens.access_token, slug, { name: "Web", identifier: "WEB" });
  const [maxs] = await addProjectMembers(api, pat.tokens.access_token, web.id, [
    { member_id: max.id, role: 20 },
    { member_id: bob.id, role: 15 },
  ]);
  if (!maxs) throw new Error("the members were not added");
  const members = `/${slug}/settings/projects/${web.id}/members`;
  const leaving = leavingOf(web.id);

  const page = await signedInPage(pat.tokens);
  const watch = await watchPage(page);
  await page.goto(members);
  // The roles' filter, a popover, by the keyboard alone: Tab reaches its button from the search, and Enter opens it,
  // which the button says; Tab reaches its roles' heading, which says it shows them. Tab reaches Admin, which Space
  // picks, saying so, the popover open still and the list down to the admins; Space picks it no more. Escape closes
  // the popover, the focus back on its button; opened again, Tab past its last role closes it, the focus moving on.
  const filters = page.getByRole("button", { name: "Filters", exact: true });
  const search = page.getByPlaceholder("Search", { exact: true });
  await search.focus();
  // the search, focused, is marked
  await expect(search.locator("..")).not.toHaveCSS("box-shadow", "none");
  await page.keyboard.press("Tab");
  await expect(filters).toBeFocused();
  await page.keyboard.press("Enter");
  await expect(filters).toHaveAttribute("aria-expanded", "true");
  await page.keyboard.press("Tab");
  const rolesHeading = page.getByRole("button", { name: /^Roles/ });
  await expect(rolesHeading).toBeFocused();
  await expect(rolesHeading).toHaveAttribute("aria-expanded", "true");
  const roleOption = (role: string) =>
    rolesHeading.locator("xpath=../..").getByRole("button", { name: role, exact: true });
  // Enter on the heading hides the roles, which it says, and Enter again shows them
  await page.keyboard.press("Enter");
  await expect(rolesHeading).toHaveAttribute("aria-expanded", "false");
  await expect(roleOption("Admin")).toHaveCount(0);
  await page.keyboard.press("Enter");
  await expect(rolesHeading).toHaveAttribute("aria-expanded", "true");
  await expect(roleOption("Admin")).toBeVisible();
  await page.keyboard.press("Tab");
  await expect(roleOption("Admin")).toBeFocused();
  await page.keyboard.press("Space");
  await expect(roleOption("Admin")).toHaveAttribute("aria-pressed", "true");
  await expect(roleOption("Admin")).toBeFocused();
  await expect(memberRow(page, bob.email)).toHaveCount(0);
  await page.keyboard.press("Space");
  await expect(roleOption("Admin")).toHaveAttribute("aria-pressed", "false");
  await expect(memberRow(page, bob.email)).toBeVisible();
  await page.keyboard.press("Escape");
  // closed, the popover is out of the page, not hidden in it
  await expect(page.getByRole("button", { name: /^Roles/, includeHidden: true })).toHaveCount(0);
  await expect(filters).toBeFocused();
  await expect(filters).toHaveAttribute("aria-expanded", "false");
  await page.keyboard.press("Enter");
  await page.keyboard.press("Tab");
  await expect(rolesHeading).toBeFocused();
  await page.keyboard.press("Tab");
  await page.keyboard.press("Tab");
  await page.keyboard.press("Tab");
  await expect(roleOption("Guest")).toBeFocused();
  await page.keyboard.press("Tab");
  await expect(page.getByRole("button", { name: "Add member" })).toBeFocused();
  await expect(rolesHeading).toHaveCount(0);

  // He adds ann as a member and gus as a guest, from acme's active members who are not Web's (the admin, ann and gus),
  // gus with a guest's role alone; until nerve has added them the modal can neither be closed nor send them again.
  await page.getByRole("button", { name: "Add member" }).click();
  await expect(modalDialog(page)).toHaveAccessibleName("Add members");
  // the member select by the keyboard: Tab reaches it, Enter opens its list with the search focused
  const coWorker = page.getByRole("dialog").getByRole("button", { name: "Select co-worker" });
  await expect
    .poll(
      async () => {
        await page.keyboard.press("Tab");
        return coWorker.evaluate((button) => button === document.activeElement);
      },
      { timeout: 5_000 }
    )
    .toBe(true);
  await page.keyboard.press("Enter");
  await expect(page.getByRole("combobox", { name: "Search" })).toBeFocused();
  // the list opens outside the page behind the dialog, which is hidden: its options are picked all the same
  expect(await pageBehindHidden(page)).toBe(true);
  await expect(page.getByRole("option")).toHaveCount(3);
  await expect(page.getByRole("option", { name: shownNameOf(admin.email) })).toBeVisible();
  await page.getByRole("option", { name: shownNameOf(ann.email) }).click();
  await page.getByRole("button", { name: "Add more" }).click();
  await page.getByRole("button", { name: "Select co-worker" }).click();
  await page.getByRole("option", { name: shownNameOf(gus.email) }).click();
  // the role select by the keyboard: Enter opens its list, and sends no form
  await page.getByRole("button", { name: "Guest", exact: true }).focus();
  const adds = `/api/v0/projects/${web.id}/members`;
  expect(await sentWithin(page, "POST", adds, () => page.keyboard.press("Enter"))).toBe(false);
  await expect(page.getByRole("option")).toHaveText(["Guest"]);
  await page.getByRole("option", { name: "Guest" }).click();
  const adding = await sentHeld(page, "POST", adds, () => page.getByRole("button", { name: "Add members" }).click());
  expect(adding.body).toEqual({
    members: [
      { member_id: ann.id, role: 15 },
      { member_id: gus.id, role: 5 },
    ],
  });
  await expect(page.getByRole("dialog").getByRole("button", { name: "Cancel" })).toBeDisabled();
  expect(await enabledWithin(page.getByRole("button", { name: "Add members..." }))).toBe(false);
  expect(await closedByEscape(page)).toBe(false);
  expect((await adding.release()).status()).toBe(201);
  await expect(page.getByRole("heading", { name: "Add members" })).toHaveCount(0);
  // the page says so; the toast is closed, as it would cover the role lists below it
  const saidSo = page.getByRole("dialog").filter({ hasText: "Members added successfully." });
  await saidSo.locator("button").click();
  await expect(saidSo).toHaveCount(0);
  await expect(memberRow(page, gus.email)).toContainText("Guest");
  const membershipOfWeb = (id: string) => projectMembershipOf(api, pat.tokens.access_token, web.id, id);
  const [anns, guss] = await Promise.all([membershipOfWeb(ann.id), membershipOfWeb(gus.id)]);
  // His own row and max's, another admin's: the role as text, nothing to pick.
  await expect(memberRow(page, pat.email)).toContainText("Admin");
  await expect(memberRow(page, max.email)).toContainText("Admin");
  await expect(memberRow(page, max.email).getByRole("button", { name: "Admin" })).toHaveCount(0);
  // ann, a member: the roles below his own, not an admin's; the page sends the role's number.
  await memberRow(page, ann.email).getByRole("button", { name: "Member", exact: true }).click();
  await expect(page.getByRole("option")).toHaveText(["Guest", "Member"]);
  const demoted = await sentTo(page, "PATCH", `/api/v0/project-members/${anns}`, () =>
    page.getByRole("option", { name: "Guest", exact: true }).click()
  );
  expect([demoted.answer.status(), demoted.body]).toEqual([200, { role: 5 }]);
  await expect(memberRow(page, ann.email).getByRole("button", { name: "Guest", exact: true })).toBeVisible();

  // max's row menu, named by him, shows, and is marked, as Tab reaches it from his avatar.
  const maxsMenu = memberRow(page, max.email).getByRole("button", { name: `${shownNameOf(max.email)} options` });
  await memberRow(page, max.email).getByRole("link").first().focus();
  await page.keyboard.press("Tab");
  await expect(maxsMenu).toBeFocused();
  await expect(maxsMenu).toHaveCSS("opacity", "1");
  await expect(maxsMenu).not.toHaveCSS("outline-style", "none");
  // the keyboard chooses its Remove, whose dialog Escape closes, the focus back on the menu
  await chooseByKeyboard(page, maxsMenu, "Remove");
  await expect(page.getByRole("dialog").getByRole("button", { name: "Remove", exact: true })).toBeVisible();
  await expect(modalDialog(page)).toHaveAccessibleName(`Remove ${shownNameOf(max.email)}?`);
  expect(await closedByEscape(page)).toBe(true);
  await expect(maxsMenu).toBeFocused();

  // max is removed; then gus, while the removal is out the dialog that asked cannot be dismissed.
  expect((await endProjectMembership(page, max.email, "Remove", removalOf(maxs.id))).status()).toBe(204);
  await expect(memberRow(page, max.email)).toHaveCount(0);
  const release = await holdAnswer(page, "DELETE", removalOf(guss).path);
  const removed = endProjectMembership(page, gus.email, "Remove", removalOf(guss));
  await expect(page.getByRole("dialog").getByRole("button", { name: "Cancel" })).toBeDisabled();
  expect(await closedByEscape(page)).toBe(false);
  await release();
  expect((await removed).status()).toBe(204);
  await expect(page.getByRole("dialog")).toHaveCount(0);
  await expect(memberRow(page, gus.email)).toHaveCount(0);

  // His leaving, as the only admin: nerve refuses it, and the page says why and stays, the dialog open.
  expect((await endProjectMembership(page, pat.email, "Leave", leaving)).status()).toBe(409);
  await expect(modalDialog(page)).toHaveAccessibleName("Leave project?");
  await expect(page.getByText("The project would be left without an admin")).toBeVisible();
  await expect(page.getByRole("dialog").getByRole("button", { name: "Leave", exact: true })).toBeEnabled();
  await expect(page).toHaveURL(members);
  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([
    [`409 ${leaving.method} ${leaving.path}`],
    [],
    [],
  ]);
  await expectQuietConsole(page, watch, {
    warnings: [EMOJI_CHECK_WARNING],
    errors: ["Failed to load resource: the server responded with a status of 409 (Conflict)"],
  });

  // bob leaves: the page stays while nerve has not answered, the dialog busy; then the workspace's projects show.
  const theMember = await anotherBrowser(browser, baseURL ?? "", bob.tokens);
  const memberWatch = await watchPage(theMember.page);
  await theMember.page.goto(members);
  const releaseLeaving = await holdAnswer(theMember.page, leaving.method, leaving.path);
  const left = endProjectMembership(theMember.page, bob.email, "Leave", leaving);
  const busy = theMember.page.getByRole("dialog").getByRole("button", { name: "Leaving" });
  expect(await enabledWithin(busy)).toBe(false);
  await expect(theMember.page).toHaveURL(members);
  await releaseLeaving();
  expect((await left).status()).toBe(204);
  await expect(theMember.page).toHaveURL(`/${slug}/projects`);
  expect([memberWatch.apiFailures, memberWatch.oldApiRequests, memberWatch.pageErrors]).toEqual([[], [], []]);
  await expectQuietConsole(theMember.page, memberWatch, { warnings: [EMOJI_CHECK_WARNING] });
  await theMember.close();

  // ann, a guest now, leaves by the sidebar, which offers it to guests alone: its modal asks Web's name and "Leave
  // Project". pat has ended her membership meanwhile: nerve refuses the leaving, and the modal stays, to try again.
  // He adds her back; while the leaving is out the modal can neither be closed nor send it again; once nerve has made
  // it, Web leaves the sidebar.
  const theGuest = await anotherBrowser(browser, baseURL ?? "", ann.tokens);
  const guestWatch = await watchPage(theGuest.page);
  await theGuest.page.goto(`/${slug}/projects`);
  const sidebar = theGuest.page.getByRole("complementary", { name: "Main sidebar" });
  await sidebar.getByText("Web", { exact: true }).hover();
  await sidebar.getByRole("button", { name: "Web quick actions", exact: true }).click();
  await theGuest.page.getByRole("menuitem", { name: "Leave project" }).click();
  await expect(modalDialog(theGuest.page)).toHaveAccessibleName("Leave Project");
  await theGuest.page.locator("#projectName").fill("Web");
  // the hint names the words to type, as the modal compares them
  await expect(theGuest.page.locator("#confirmLeave")).toHaveAttribute("placeholder", "Enter “Leave Project”");
  await theGuest.page.locator("#confirmLeave").fill("Leave Project");
  expect((await projectMemberWrites(api, web.id).remove(pat.tokens.access_token, anns)).status).toBe(204);
  const leaveProject = theGuest.page.getByRole("dialog").getByRole("button", { name: "Leave Project" });
  expect((await answerTo(theGuest.page, leaving.method, leaving.path, () => leaveProject.click())).status()).toBe(403);
  expect(await closedWithin(theGuest.page)).toBe(false);
  await expect(leaveProject).toBeEnabled();
  await addProjectMembers(api, pat.tokens.access_token, web.id, [{ member_id: ann.id, role: 5 }]);
  const releaseGuest = await holdAnswer(theGuest.page, leaving.method, leaving.path);
  const guestLeft = answerTo(theGuest.page, leaving.method, leaving.path, () => leaveProject.click());
  await expect(theGuest.page.getByRole("dialog").getByRole("button", { name: "Cancel" })).toBeDisabled();
  expect(await enabledWithin(theGuest.page.getByRole("button", { name: "Leaving" }))).toBe(false);
  expect(await closedByEscape(theGuest.page)).toBe(false);
  await releaseGuest();
  expect((await guestLeft).status()).toBe(204);
  await expect(sidebar.getByText("Web", { exact: true })).toHaveCount(0);
  expect([guestWatch.apiFailures, guestWatch.oldApiRequests, guestWatch.pageErrors]).toEqual([
    [`403 ${leaving.method} ${leaving.path}`],
    [],
    [],
  ]);
  await expectQuietConsole(theGuest.page, guestWatch, {
    warnings: [EMOJI_CHECK_WARNING],
    errors: ["Failed to load resource: the server responded with a status of 403 (Forbidden)"],
  });
  await theGuest.close();

  const row = (email: string, role: number, is_active: boolean, by: string): MemberRow => ({
    email,
    role,
    is_active,
    by,
    sort_order: 65535,
    settings_by: pat.email,
  });
  await expectMembers(db, web.id, [
    row(pat.email, 20, true, pat.email),
    row(max.email, 20, false, pat.email),
    row(ann.email, 5, false, ann.email),
    row(bob.email, 15, false, bob.email),
    row(gus.email, 5, false, pat.email),
  ]);
});
