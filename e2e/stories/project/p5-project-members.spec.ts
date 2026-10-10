import {
  addProjectMembers,
  amidAnotherWorkspace,
  createProject,
  createWorkspace,
  inviteAndAccept,
  membershipOf,
  projectMemberWrites,
  projectMembershipOf,
  slugFor,
} from "../../fixtures/api";
import { expectMembers, type MemberRow } from "../../fixtures/assert/project";
import { accountId, bearer, emailFor, newAccount, shownNameOf } from "../../fixtures/auth";
import { EMOJI_CHECK_WARNING, expectQuietConsole, watchPage } from "../../fixtures/browser";
import {
  answerTo,
  closedByEscape,
  closedWithin,
  enabledWithin,
  holdAnswer,
  registerOnboarded,
  sentHeld,
  sentTo,
} from "../../fixtures/settings-pages";
import { endProjectMembership, leavingOf, removalOf } from "../../fixtures/project-pages";
import { expect, test } from "../../fixtures/test";
import { anotherBrowser, memberRow } from "../../fixtures/workspace-pages";

// P5, a project's members (M3 design 2, 3.5, 3.7, 7.6): adding them, changing a role, removing a member, leaving.

test("P5 (API): the admin adds a member and a guest at once, and cannot leave, the only admin; another admin makes the member a guest, removes the guest, and removes a third admin, who joins again as a member, his row back; a member leaves, his membership of the workspace's other project kept, and the workspace's making him a guest makes his ended membership a guest's; an add of one who is no workspace member, of a workspace guest or admin as a member, a member's change of a role and an admin's change of another admin's change nothing", async ({
  api,
  db,
}, testInfo) => {
  const admin = await newAccount(api, testInfo, "admin");
  const slug = slugFor(testInfo);
  await createWorkspace(api, admin.token, { name: "Acme", slug });
  // acme's members mia, pam, ray and tom, its guest gus, its other admin wanda; olga is no member of it.
  const [mia, gus, pam, ray, tom, wanda, olga] = await Promise.all(
    ["mia", "gus", "pam", "ray", "tom", "wanda", "olga"].map((label) => newAccount(api, testInfo, label))
  );
  if (!mia || !gus || !pam || !ray || !tom || !wanda || !olga) {
    throw new Error("the accounts were not registered");
  }
  await Promise.all(
    (
      [
        [mia, 15],
        [gus, 5],
        [pam, 15],
        [ray, 15],
        [tom, 15],
        [wanda, 20],
      ] as const
    ).map(([who, role]) => inviteAndAccept(api, admin.token, slug, who, role))
  );
  const web = await amidAnotherWorkspace(api, admin.token, testInfo, () =>
    createProject(api, admin.token, slug, { name: "Web", identifier: "WEB" })
  );
  const { add, change, remove, leave } = projectMemberWrites(api, web.id);
  // Ops, acme's other project: wanda its other admin, none of Web's.
  const ops = await createProject(api, admin.token, slug, { name: "Ops", identifier: "OPS" });
  await addProjectMembers(api, admin.token, ops.id, [{ member_id: wanda.id, role: 20 }]);
  // Each membership of Web or Ops, as expectMembers reads it, with its display settings, the admin's writing, as the
  // adds and the creations made them: at 65535, or at 55535 in Ops for the admin and tom, whose second project of acme
  // it is (10000 before the first, M3 design 3.18).
  const row = (
    who: { email: string },
    role: number,
    is_active: boolean,
    by: { email: string },
    sort_order = 65535
  ): MemberRow => ({
    email: who.email,
    role,
    is_active,
    by: by.email,
    sort_order,
    settings_by: admin.email,
  });
  let members = [row(admin, 20, true, admin)];
  let opsMembers = [row(admin, 20, true, admin, 55535), row(wanda, 20, true, admin)];
  await expectMembers(db, web.id, members);
  await expectMembers(db, ops.id, opsMembers);

  // Refused, each adding nothing: olga, no member of acme; gus, its guest, as a member; wanda, its admin, as a member.
  expect(
    [
      await add(admin.token, [{ member_id: olga.id, role: 15 }]),
      await add(admin.token, [{ member_id: gus.id, role: 15 }]),
      await add(admin.token, [{ member_id: wanda.id, role: 15 }]),
    ],
    "the adds of olga, of gus and of wanda as members"
  ).toEqual([
    { status: 422, code: "validation_failed", errors: [{ field: "members[0].member_id", code: "not_allowed" }] },
    { status: 422, code: "validation_failed", errors: [{ field: "members[0].role", code: "not_allowed" }] },
    { status: 422, code: "validation_failed", errors: [{ field: "members[0].role", code: "not_allowed" }] },
  ]);
  await expectMembers(db, web.id, members);

  // The admin adds mia as a member and gus as a guest, at once: two memberships and two display settings, his.
  const added = await addProjectMembers(api, admin.token, web.id, [
    { member_id: mia.id, role: 15 },
    { member_id: gus.id, role: 5 },
  ]);
  expect(added.map((m) => [m.member_id, m.role])).toEqual([
    [mia.id, 15],
    [gus.id, 5],
  ]);
  const [mias, guss] = added.map((m) => m.id);
  if (!mias || !guss) {
    throw new Error("the add answered no memberships");
  }
  members = [...members, row(mia, 15, true, admin), row(gus, 5, true, admin)];
  await expectMembers(db, web.id, members);

  // Web's only admin cannot leave it; mia, its member, cannot change gus's role.
  expect(
    [await leave(admin.token), await change(mia.token, guss, 5)],
    "the admin's leaving, mia's change of gus"
  ).toEqual([
    { status: 409, code: "project.sole_admin" },
    { status: 403, code: "forbidden" },
  ]);
  await expectMembers(db, web.id, members);

  // The admin adds pam and ray as admins, tom as a member: pam, who is no workspace admin, cannot change ray's role,
  // another admin's.
  const more = await addProjectMembers(api, admin.token, web.id, [
    { member_id: pam.id, role: 20 },
    { member_id: ray.id, role: 20 },
    { member_id: tom.id, role: 15 },
  ]);
  const rays = more[1]?.id;
  if (!rays) {
    throw new Error("the add answered no membership of ray");
  }
  members = [...members, row(pam, 20, true, admin), row(ray, 20, true, admin), row(tom, 15, true, admin)];
  await expectMembers(db, web.id, members);
  // tom is Ops's member too.
  await addProjectMembers(api, admin.token, ops.id, [{ member_id: tom.id, role: 15 }]);
  opsMembers = [...opsMembers, row(tom, 15, true, admin, 55535)];
  await expectMembers(db, ops.id, opsMembers);
  expect(await change(pam.token, rays, 15), "pam's change of ray, an admin").toEqual({
    status: 403,
    code: "project.role_too_high",
  });
  await expectMembers(db, web.id, members);

  // pam makes mia a guest and removes gus: each row written by pam, gus's ended with its role and his display
  // settings kept.
  expect(
    [await change(pam.token, mias, 5), await remove(pam.token, guss)],
    "pam's change of mia, her removal of gus"
  ).toEqual([{ status: 200 }, { status: 204 }]);
  members = members.map((m) =>
    m.email === mia.email ? row(mia, 5, true, pam) : m.email === gus.email ? row(gus, 5, false, pam) : m
  );
  await expectMembers(db, web.id, members);

  // tom leaves Web: his membership ends, by him; his membership of Ops stays. The admin then makes him acme's guest:
  // his ended membership of Web becomes a guest's too, and so does his membership of Ops (M3 design 2, W7).
  expect(await leave(tom.token), "tom's leaving").toEqual({ status: 204 });
  members = members.map((m) => (m.email === tom.email ? row(tom, 15, false, tom) : m));
  await expectMembers(db, web.id, members);
  await expectMembers(db, ops.id, opsMembers);
  const demoted = await api.PATCH("/api/v0/workspace-members/{workspace_member_id}", {
    params: { path: { workspace_member_id: await membershipOf(api, admin.token, slug, tom.id) } },
    body: { role: 5 },
    headers: bearer(admin.token),
  });
  expect(demoted.response.status, "the admin's making tom acme's guest").toBe(200);
  members = members.map((m) => (m.email === tom.email ? row(tom, 5, false, admin) : m));
  await expectMembers(db, web.id, members);
  opsMembers = opsMembers.map((m) => (m.email === tom.email ? row(tom, 5, true, admin, 55535) : m));
  await expectMembers(db, ops.id, opsMembers);

  // pam removes ray, another admin and a member of acme; he joins Web again: his row is back, a member's, the 20 it
  // kept no more than his workspace role, 15 (M3 design 3.5).
  expect(await remove(pam.token, rays), "pam's removal of ray").toEqual({ status: 204 });
  members = members.map((m) => (m.email === ray.email ? row(ray, 20, false, pam) : m));
  await expectMembers(db, web.id, members);
  const joined = await api.POST("/api/v0/projects/{project_id}/join", {
    params: { path: { project_id: web.id } },
    headers: bearer(ray.token),
  });
  expect([joined.response.status, joined.data?.member_role], "ray's joining Web again").toEqual([200, 15]);
  members = members.map((m) => (m.email === ray.email ? row(ray, 15, true, ray) : m));
  await expectMembers(db, web.id, members);
  const listed = await api.GET("/api/v0/projects/{project_id}/members", {
    params: { path: { project_id: web.id } },
    headers: bearer(admin.token),
  });
  expect(
    listed.data?.data.map((m) => [m.member_id, m.role, m.id === rays]).toSorted(),
    "Web's members, ray's membership his row of before"
  ).toEqual(
    (
      [
        [admin.id, 20, false],
        [mia.id, 5, false],
        [pam.id, 20, false],
        [ray.id, 15, true],
      ] as const
    ).toSorted()
  );
});

test("P5 (page): a project admin who is no workspace admin adds a member and a guest from the workspace's members who are not the project's, the modal held until nerve answers; he is offered only the roles below his own, none for another admin; he makes a member a guest, removes the other admin and a guest, the dialog held until nerve answers, and, its only admin now, is told why he may not leave, its dialog open; a member leaves, and the workspace's projects show once nerve has made it, not before; the guest leaves by the sidebar, its modal open after a refusal and held too", async ({
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
  // He adds ann as a member and gus as a guest, from acme's active members who are not Web's (the admin, ann and gus),
  // gus with a guest's role alone; until nerve has added them the modal can neither be closed nor send them again.
  await page.getByRole("button", { name: "Add member" }).click();
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
  await expect(page.getByRole("option")).toHaveCount(3);
  await expect(page.getByRole("option", { name: shownNameOf(admin.email) })).toBeVisible();
  await page.getByRole("option", { name: shownNameOf(ann.email) }).click();
  await page.getByRole("button", { name: "Add more" }).click();
  await page.getByRole("button", { name: "Select co-worker" }).click();
  await page.getByRole("option", { name: shownNameOf(gus.email) }).click();
  await page.getByRole("button", { name: "Guest", exact: true }).click();
  await expect(page.getByRole("option")).toHaveText(["Guest"]);
  await page.getByRole("option", { name: "Guest" }).click();
  const adding = await sentHeld(page, "POST", `/api/v0/projects/${web.id}/members`, () =>
    page.getByRole("button", { name: "Add members" }).click()
  );
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
  const busy = theMember.page.getByRole("dialog").getByRole("button", { name: "Leaving..." });
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
  await sidebar.getByRole("button", { name: "Toggle quick actions menu" }).last().click();
  await theGuest.page.getByRole("menuitem", { name: "Leave project" }).click();
  await theGuest.page.locator("#projectName").fill("Web");
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
  expect(await enabledWithin(theGuest.page.getByRole("button", { name: "Leaving..." }))).toBe(false);
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
