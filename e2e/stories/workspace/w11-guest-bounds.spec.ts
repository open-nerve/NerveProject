import {
  addProjectMembers,
  answer,
  createProject,
  createWorkspace,
  invite,
  inviteAndAccept,
  listMembers,
  slugFor,
} from "../../fixtures/api";
import { workspaceTables } from "../../fixtures/assert/catalog";
import { expectMembership, expectMembershipEnded } from "../../fixtures/assert/workspace";
import { accountId, bearer, emailFor, newAccount } from "../../fixtures/auth";
import { EMOJI_CHECK_WARNING, expectNoAddress, expectQuietConsole, watchPage } from "../../fixtures/browser";
import {
  announced,
  closedByEscape,
  closedWithin,
  modalDialog,
  pageBehindHidden,
  shownWithin,
} from "../../fixtures/dialogs";
import { answerTo, holdAnswer, moveWithinApp, registerOnboarded } from "../../fixtures/settings-pages";
import { expect, test } from "../../fixtures/test";
import { anotherBrowser, askToEndMembership } from "../../fixtures/workspace-pages";

// W11, a guest's bounds (M3 design 2, 9.2): the permission matrix's backend
// test holds every cell; this story samples four of them through the API, and
// its page version what a guest's pages show him, and his leaving.

test("W11 (API): a guest of a workspace may not list its invitations nor create a state in the project he is a guest of, does not see a private project he is not a member of, and reads no member's address, his own neither; none of it changes a row of the workspace's", async ({
  api,
  db,
}, testInfo) => {
  const admin = await newAccount(api, testInfo, "admin");
  const slug = slugFor(testInfo);
  await createWorkspace(api, admin.token, { name: "Acme", slug });
  const gus = await newAccount(api, testInfo, "gus");
  await inviteAndAccept(api, admin.token, slug, gus, 5);
  // Web, public, gus its guest; Secret, private, the admin's alone; an invitation to olga, pending.
  const web = await createProject(api, admin.token, slug, { name: "Web", identifier: "WEB" });
  await addProjectMembers(api, admin.token, web.id, [{ member_id: gus.id, role: 5 }]);
  const secret = await createProject(api, admin.token, slug, { name: "Secret", identifier: "SECRET", network: 0 });
  await invite(api, admin.token, slug, [{ email: emailFor(testInfo, "olga"), role: 15 }]);
  // The workspaces and every table whose rows belong to a workspace, whole: none of gus's calls writes a row of them.
  const tables = async () =>
    Object.fromEntries(
      await Promise.all(
        ["workspaces", ...(await workspaceTables(db))].map(async (table) => [
          table,
          await db.query(`SELECT * FROM ${table} ORDER BY id`),
        ])
      )
    );
  const before = await tables();

  const invitations = await api.GET("/api/v0/workspaces/{slug}/invitations", {
    params: { path: { slug } },
    headers: bearer(gus.token),
  });
  const created = await api.POST("/api/v0/projects/{project_id}/states", {
    params: { path: { project_id: web.id } },
    body: { name: "Review", color: "#8B5CF6", group: "started" },
    headers: bearer(gus.token),
  });
  const read = await api.GET("/api/v0/projects/{project_id}", {
    params: { path: { project_id: secret.id } },
    headers: bearer(gus.token),
  });
  expect(
    [
      answer(invitations.response, invitations.error),
      answer(created.response, created.error),
      answer(read.response, read.error),
    ],
    "gus's listing of acme's invitations, his creation of a state in Web, his reading of Secret"
  ).toEqual([
    { status: 403, code: "forbidden" },
    { status: 403, code: "forbidden" },
    { status: 404, code: "project.not_found" },
  ]);

  // acme's members as gus lists them: no address, his own neither; the admin reads both.
  const addresses = async (token: string) =>
    (await listMembers(api, token, slug)).map((m) => [m.member.id, m.member.email]).toSorted();
  expect(await addresses(gus.token), "acme's members as gus lists them").toEqual(
    [
      [admin.id, null],
      [gus.id, null],
    ].toSorted()
  );
  expect(await addresses(admin.token), "acme's members as the admin lists them").toEqual(
    [
      [admin.id, admin.email],
      [gus.id, gus.email],
    ].toSorted()
  );
  expect(await tables(), "the tables after gus's calls").toEqual(before);
});

test("W11 (page): a guest signs in: his sidebar lists Web, the project he is a guest of, not Secret; the workspace's members page and Web's state and label settings tell him he may not see them; no page he opens shows an address, his own neither; the workspace menu offers him his leaving, reached by the keyboard and sent once by a double click, its dialog held while it is out, which ends his memberships and lands him on the page that creates a workspace; the only admin's leaving there is refused with the reason, its dialog open; his leaving's dialog, a removal's, a project's creation, the command palette and its shortcuts end as a Back takes him to another workspace", async ({
  api,
  baseURL,
  browser,
  db,
  signedInPage,
}, testInfo) => {
  const adminEmail = emailFor(testInfo, "admin");
  const admin = await registerOnboarded(api, adminEmail);
  const gusEmail = emailFor(testInfo, "gus");
  const gus = await registerOnboarded(api, gusEmail);
  const slug = slugFor(testInfo);
  await createWorkspace(api, admin.access_token, { name: "Acme", slug });
  await inviteAndAccept(api, admin.access_token, slug, { email: gusEmail, token: gus.access_token }, 5);
  const web = await createProject(api, admin.access_token, slug, { name: "Web", identifier: "WEB" });
  await addProjectMembers(api, admin.access_token, web.id, [
    { member_id: await accountId(api, gus.access_token), role: 5 },
  ]);
  await createProject(api, admin.access_token, slug, { name: "Secret", identifier: "SECRET", network: 0 });

  const page = await signedInPage(gus);
  const watch = await watchPage(page);
  /** Opens path, and checks that once it shows what is expected no member's address is in the page. */
  const opened = async (path: string, shown: string) => {
    await page.goto(path);
    await expect(page.getByText(shown, { exact: true }).first(), path).toBeVisible();
    await expectNoAddress(page, [adminEmail, gusEmail], path);
  };
  await opened(`/${slug}`, "Web");
  const joined = page.getByRole("complementary", { name: "Main sidebar" }).locator('[id^="sidebar-"][id$="-JOINED"]');
  await expect(joined).toHaveText([/Web/]);
  const denied = "Oops! You are not authorized to view this page";
  await opened(`/${slug}/settings/members`, denied);
  await opened(`/${slug}/settings/projects/${web.id}/states`, denied);
  await opened(`/${slug}/settings/projects/${web.id}/labels`, denied);

  // The admin's dialogs end with the workspace they were opened in, whose content is keyed by it (v0 design 7.7); he
  // has another, Beta, whose pages share acme's layout. From Beta's home his menu's acme entry, a link of the app's,
  // takes him to acme's, where he asks his leaving: a Back to Beta's home ends that dialog rather than ask it of Beta,
  // and a Forward to acme's does not bring it back. Then, from Beta's members page, the app takes him to acme's (the
  // acme entry, then its "Invite members"), where he asks gus's removal: a Back of two entries at once (the Back
  // button's list) to Beta's members page ends that dialog too.
  const betaSlug = slugFor(testInfo, "beta");
  await createWorkspace(api, admin.access_token, { name: "Beta", slug: betaSlug });
  const theAdmin = await anotherBrowser(browser, baseURL ?? "", admin);
  const adminsMenu = theAdmin.page.getByRole("button", { name: "Open workspace switcher" });
  /** Opens the admin's menu and follows the link of its entry named to the path given. */
  const throughMenu = async (role: "menuitem" | "link", name: string | RegExp, path: string) => {
    await adminsMenu.click();
    await theAdmin.page.getByRole("menu").getByRole(role, { name, exact: true }).click();
    await expect(theAdmin.page).toHaveURL(path);
  };
  await theAdmin.page.goto(`/${betaSlug}`);
  await throughMenu("menuitem", /Acme/, `/${slug}`);
  await adminsMenu.click();
  await theAdmin.page.getByRole("menuitem", { name: "Leave workspace" }).click();
  await expect(modalDialog(theAdmin.page)).toHaveCount(1);
  await theAdmin.page.goBack();
  await expect(theAdmin.page).toHaveURL(`/${betaSlug}`);
  expect(await closedWithin(theAdmin.page)).toBe(true);
  await theAdmin.page.goForward();
  await expect(theAdmin.page).toHaveURL(`/${slug}`);
  expect(await shownWithin(modalDialog(theAdmin.page))).toBe(false);
  await throughMenu("menuitem", /Beta/, `/${betaSlug}`);
  await throughMenu("link", "Invite members", `/${betaSlug}/settings/members`);
  await throughMenu("menuitem", /Acme/, `/${slug}`);
  await throughMenu("link", "Invite members", `/${slug}/settings/members`);
  await askToEndMembership(theAdmin.page, gusEmail, "Remove");
  await expect(modalDialog(theAdmin.page)).toHaveCount(1);
  await theAdmin.page.evaluate(() => window.history.go(-2));
  await expect(theAdmin.page).toHaveURL(`/${betaSlug}/settings/members`);
  expect(await closedWithin(theAdmin.page)).toBe(true);
  // So does the projects page's creation, whose open flag the command palette's store holds, above the key: its host
  // closes it as acme's content goes. From Beta's projects page to acme's (the router's own move), Add Project, and a
  // Back to Beta's: the dialog does not open there, to create the project in Beta.
  await moveWithinApp(theAdmin.page, `/${betaSlug}/projects`);
  await moveWithinApp(theAdmin.page, `/${slug}/projects`);
  await theAdmin.page.getByRole("button", { name: "Add Project", exact: true }).click();
  await expect(modalDialog(theAdmin.page)).toHaveCount(1);
  await theAdmin.page.goBack();
  await expect(theAdmin.page).toHaveURL(`/${betaSlug}/projects`);
  expect(await closedWithin(theAdmin.page)).toBe(true);
  // So do the command palette and its list of shortcuts, whose open flags the power-K store holds, each by its own
  // host: opened by its keys on acme's projects page, each ends with the Back to Beta's.
  /** From Beta's projects page to acme's, opens a dialog by keys, Backs to Beta's: whether the dialog ended. */
  const endsOnBack = async (keys: string) => {
    await moveWithinApp(theAdmin.page, `/${slug}/projects`);
    await expect(theAdmin.page.getByRole("button", { name: "Add Project", exact: true })).toBeVisible();
    await theAdmin.page.keyboard.press(keys);
    await expect(modalDialog(theAdmin.page)).toHaveCount(1);
    await theAdmin.page.goBack();
    await expect(theAdmin.page).toHaveURL(`/${betaSlug}/projects`);
    return closedWithin(theAdmin.page);
  };
  expect(await endsOnBack("ControlOrMeta+k"), "the palette").toBe(true);
  expect(await endsOnBack("ControlOrMeta+/"), "the shortcuts list").toBe(true);

  // The only admin's leaving, through the workspace menu: nerve refuses it, the reason read beside the dialog, which
  // stays until Cancel closes it and gives the focus back to the menu's button; he is acme's admin still. His menu
  // lists Beta after acme: the arrows mark its entry as they reach it, as the pointer does.
  const leaving = { method: "POST", path: `/api/v0/workspaces/${slug}/leave` };
  await throughMenu("menuitem", /Acme/, `/${slug}`);
  await adminsMenu.click();
  const beta = theAdmin.page.getByRole("menuitem", { name: "Beta" });
  const betaMarked = () => beta.evaluate((entry) => getComputedStyle(entry).backgroundColor !== "rgba(0, 0, 0, 0)");
  expect(await betaMarked()).toBe(false);
  await theAdmin.page.keyboard.press("ArrowDown");
  await theAdmin.page.keyboard.press("ArrowDown");
  await expect.poll(betaMarked).toBe(true);
  await theAdmin.page.getByRole("menuitem", { name: "Leave workspace" }).click();
  await expect(modalDialog(theAdmin.page)).toHaveAccessibleName("Leave workspace?");
  const adminLeaving = await answerTo(theAdmin.page, leaving.method, leaving.path, () =>
    modalDialog(theAdmin.page).getByRole("button", { name: "Leave", exact: true }).click()
  );
  expect(adminLeaving.status()).toBe(409);
  const reason = theAdmin.page.getByText(
    "The workspace would be left without an admin. Make another member an admin first."
  );
  await expect(reason).toBeVisible();
  expect(await announced(reason)).toBe(true);
  expect(await closedWithin(theAdmin.page)).toBe(false);
  await modalDialog(theAdmin.page).getByRole("button", { name: "Cancel" }).click();
  await expect(modalDialog(theAdmin.page)).toHaveCount(0);
  await expect(adminsMenu).toBeFocused();
  await expect(theAdmin.page).toHaveURL(`/${slug}`);
  await expectMembership(db, slug, adminEmail, { role: 20, is_active: true });
  await theAdmin.close();

  // gus's leaving, by the keyboard: the menu's button, its ring shown, opens the menu on its first item, acme's entry,
  // the current workspace's, marked a step darker than it rests; then his leaving, from the last item, signing out, one
  // up, past it to creating a workspace and back, each action marked as the arrows reach it, the others not; the
  // dialog hides the page behind it. He leaves with a double click, nerve's answer held: while it is out the dialog
  // cannot be dismissed (Cancel is disabled, Escape leaves it open), and the page sends the leaving once. Leaving ends
  // his memberships, Web's too, and the root lands him on the page that creates a workspace.
  const menu = page.getByRole("button", { name: "Open workspace switcher" });
  await menu.focus();
  await expect(menu).toHaveCSS("outline-style", "solid");
  await page.keyboard.press("Enter");
  const acme = page.getByRole("menuitem", { name: "Acme" });
  const acmeBackground = () => acme.evaluate((entry) => getComputedStyle(entry).backgroundColor);
  await expect(acme).toHaveAttribute("data-focus");
  const acmeMarked = await acmeBackground();
  const actions = ["Create workspace", "Leave workspace", "Sign out"].map((name) =>
    page.getByRole("menuitem", { name, exact: true })
  );
  const marked = () =>
    Promise.all(
      actions.map((action) => action.evaluate((item) => getComputedStyle(item).backgroundColor !== "rgba(0, 0, 0, 0)"))
    );
  await page.keyboard.press("End");
  await expect.poll(marked).toEqual([false, false, true]);
  expect(await acmeBackground()).not.toBe(acmeMarked);
  await page.keyboard.press("ArrowUp");
  await expect.poll(marked).toEqual([false, true, false]);
  await page.keyboard.press("ArrowUp");
  await expect.poll(marked).toEqual([true, false, false]);
  await page.keyboard.press("ArrowDown");
  await expect.poll(marked).toEqual([false, true, false]);
  await page.keyboard.press("Enter");
  const leave = modalDialog(page).getByRole("button", { name: "Leave", exact: true });
  await expect(modalDialog(page).getByRole("heading", { name: "Leave workspace?" })).toBeVisible();
  await expect(modalDialog(page)).toHaveAccessibleName("Leave workspace?");
  expect(await pageBehindHidden(page)).toBe(true);
  const release = await holdAnswer(page, leaving.method, leaving.path);
  const left = answerTo(page, leaving.method, leaving.path, () => leave.dblclick());
  await expect(modalDialog(page).getByRole("button", { name: "Cancel" })).toBeDisabled();
  expect(await closedByEscape(page)).toBe(false);
  await release();
  expect((await left).status()).toBe(204);
  await expect(page).toHaveURL("/create-workspace");
  expect(watch.apiRequests.filter((request) => request === `POST ${leaving.path}`)).toEqual([`POST ${leaving.path}`]);
  await expectMembershipEnded(db, slug, gusEmail, gusEmail, 5, [{ identifier: "WEB", role: 5 }]);
  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], []]);
  await expectQuietConsole(page, watch, { warnings: Array(4).fill(EMOJI_CHECK_WARNING) });
});
