import {
  changeRole,
  createLabel,
  createProject,
  createWorkspace,
  furnishWorkspace,
  invite,
  inviteAndAccept,
  slugFor,
  type Workspace,
} from "../../fixtures/api";
import { expectLabels, expectProjectCreated, expectProjectDeleted, type LabelRow } from "../../fixtures/assert/project";
import {
  deletedAloneTables,
  expectInvitations,
  expectMembership,
  expectPreferences,
  expectWorkspaceDeleted,
} from "../../fixtures/assert/workspace";
import { accountId, bearer, createPAT, emailFor, login, newRecord, register, writeRecord } from "../../fixtures/auth";
import { EMOJI_CHECK_WARNING, expectQuietConsole, watchPage } from "../../fixtures/browser";
import { answerTo, closedByEscape, holdAnswer, registerOnboarded, sentTo } from "../../fixtures/settings-pages";
import { expect, test } from "../../fixtures/test";
import { anotherBrowser, confirmDeletion, deleteFromGeneralPage } from "../../fixtures/workspace-pages";

// W3, the workspace's settings (M3 design 2), with the session switch of 7.1.

test("W3 (API): the admin changes the workspace and deletes it with its members, invitations, settings and projects at one moment; a member may do neither, and the slug never changes", async ({
  api,
  db,
}, testInfo) => {
  const adminEmail = emailFor(testInfo, "admin");
  const admin = (await createPAT(api, (await register(api, adminEmail)).access_token)).token;
  const memberEmail = emailFor(testInfo, "member");
  const member = (await createPAT(api, (await register(api, memberEmail)).access_token)).token;
  const declinerEmail = emailFor(testInfo, "decliner");
  const decliner = (await register(api, declinerEmail)).access_token;
  const inviteeEmail = emailFor(testInfo, "invitee");
  const slug = slugFor(testInfo);
  const other = slugFor(testInfo, "other");
  // Two workspaces alike, each with the member, a pending and a declined invitation, the member's settings and a
  // project the admin created with the member its lead, which the member labelled; only Acme is deleted. Both exist
  // before either is furnished, and the member is Other's admin and Acme's member: each answer, role and row must be
  // the workspace's own, whichever row the database reads first.
  const [adminId, memberId] = await Promise.all([admin, member].map((token) => accountId(api, token)));
  const web = { name: "Web", identifier: "WEB", description: "", network: 2, timezone: "UTC", logo_props: {} };
  const otherWorkspace = await createWorkspace(api, admin, { name: "Other", slug: other });
  const acme = await createWorkspace(api, admin, { name: "Acme", slug });
  const tokens: string[] = [];
  const furnish = async ({ slug: target, id }: Workspace, memberRole: 15 | 20) => {
    expect(await inviteAndAccept(api, admin, target, { email: memberEmail, token: member }, memberRole)).toMatchObject({
      slug: target,
      role: memberRole,
      total_members: 2,
    });
    const created = await invite(api, admin, target, [
      { email: inviteeEmail, role: 5 },
      { email: declinerEmail, role: 15 },
    ]);
    tokens.push(...created.map((i) => i.token));
    const [, toDecline] = created;
    if (!toDecline) {
      throw new Error(`the invitations to ${target} were not created`);
    }
    const declined = await api.POST("/api/v0/workspace-invitations/{invitation_id}/decline", {
      params: { path: { invitation_id: toDecline.id } },
      body: { token: toDecline.token },
      headers: bearer(decliner),
    });
    expect(declined.response.status).toBe(204);
    const settings = await api.PATCH("/api/v0/me/workspaces/{slug}/preferences", {
      params: { path: { slug: target } },
      body: { navigation_project_limit: 3 },
      headers: bearer(member),
    });
    expect(settings.response.status).toBe(200);
    // The answer is the workspace's own new project as its creator sees it, first in both admins' sidebars.
    const project = await createProject(api, admin, target, {
      name: "Web",
      identifier: "web",
      project_lead_id: memberId,
    });
    expect(project).toMatchObject({
      workspace_id: id,
      identifier: "WEB",
      member_role: 20,
      sort_order: 65535,
      member_ids: [adminId, memberId],
    });
    // The member, Web's lead and so its admin, labels it: Bug, and UI under Bug.
    const bug = await createLabel(api, member, project.id, { name: "Bug", color: "#EF4444" });
    await createLabel(api, member, project.id, { name: "UI", parent_id: bug.id });
    return project;
  };
  const otherWeb = await furnish(otherWorkspace, 20);
  const acmeWeb = await furnish(acme, 15);
  /** Web's labels, as the member made them; deleted by the admin, as Acme's deletion writes them. */
  const webLabels = (deleted: boolean): LabelRow[] => [
    { name: "Bug", color: "#EF4444", parent: null, sort_order: 65535, deleted, by: deleted ? adminEmail : memberEmail },
    { name: "UI", color: "", parent: "Bug", sort_order: 75535, deleted, by: deleted ? adminEmail : memberEmail },
  ];
  await expectLabels(db, acmeWeb.id, webLabels(false));
  // The member reads Other as its admin, whatever becomes of Acme.
  const readOther = async () => {
    const read = await api.GET("/api/v0/workspaces/{slug}", {
      params: { path: { slug: other } },
      headers: bearer(member),
    });
    expect([read.response.status, read.data]).toEqual([
      200,
      expect.objectContaining({ name: "Other", slug: other, role: 20, total_members: 2 }),
    ]);
  };

  // A second admin changes it, so that the deletion must record its own author.
  const coAdminEmail = emailFor(testInfo, "co-admin");
  const coAdmin = (await createPAT(api, (await register(api, coAdminEmail)).access_token)).token;
  await inviteAndAccept(api, admin, slug, { email: coAdminEmail, token: coAdmin }, 20);
  const renamed = await api.PATCH("/api/v0/workspaces/{slug}", {
    params: { path: { slug } },
    body: { name: "Acme Corp", organization_size: "11-50", timezone: "Europe/Berlin" },
    headers: bearer(coAdmin),
  });
  expect(renamed.response.status).toBe(200);
  expect(renamed.data).toMatchObject({
    slug,
    name: "Acme Corp",
    organization_size: "11-50",
    timezone: "Europe/Berlin",
    role: 20,
    total_members: 3,
  });
  expect(
    await db.query(
      `SELECT w.name, w.organization_size, w.timezone, w.updated_by_id = u.id AS updated_by_the_co_admin
         FROM workspaces w JOIN users u ON u.email = $2 WHERE w.slug = $1`,
      [slug, coAdminEmail]
    )
  ).toEqual([
    { name: "Acme Corp", organization_size: "11-50", timezone: "Europe/Berlin", updated_by_the_co_admin: true },
  ]);
  await readOther();

  // A member may not change it; a slug in the body is refused before anything is looked at.
  const byMember = await api.PATCH("/api/v0/workspaces/{slug}", {
    params: { path: { slug } },
    body: { name: "Mine" },
    headers: bearer(member),
  });
  expect(byMember.response.status).toBe(403);
  expect(byMember.error?.code).toBe("forbidden");
  const withSlug = await api.PATCH("/api/v0/workspaces/{slug}", {
    params: { path: { slug } },
    body: { slug: "other" } as never,
    headers: bearer(admin),
  });
  expect(withSlug.response.status).toBe(400);
  const memberDeletes = await api.DELETE("/api/v0/workspaces/{slug}", {
    params: { path: { slug } },
    headers: bearer(member),
  });
  expect(memberDeletes.response.status).toBe(403);
  expect(memberDeletes.error?.code).toBe("forbidden");

  // A project deleted before the workspace keeps its moment, and so do its rows: Old, with the member its lead, so
  // that each project table has a row of it.
  const old = await createProject(api, admin, slug, { name: "Old", identifier: "OLD", project_lead_id: memberId });
  // The member moves Old in his sidebar and labels it: his display settings in it and its label are his own writing,
  // which Old's deletion must write again as the admin's.
  await createLabel(api, member, old.id, { name: "Bug" });
  const moved = await api.PATCH("/api/v0/me/projects/{project_id}/preferences", {
    params: { path: { project_id: old.id } },
    body: { sort_order: 75535 },
    headers: bearer(member),
  });
  expect(moved.response.status).toBe(200);
  expect(
    await db.query(
      `SELECT s.sort_order, b.email AS by
         FROM project_user_properties s JOIN users u ON u.id = s.user_id JOIN users b ON b.id = s.updated_by_id
        WHERE s.project_id = $1 AND u.email = $2 AND s.deleted_at IS NULL`,
      [old.id, memberEmail]
    )
  ).toEqual([{ sort_order: 75535, by: memberEmail }]);
  await expectLabels(db, old.id, [
    { name: "Bug", color: "", parent: null, sort_order: 65535, deleted: false, by: memberEmail },
  ]);
  const oldDeleted = await api.DELETE("/api/v0/projects/{project_id}", {
    params: { path: { project_id: old.id } },
    headers: bearer(admin),
  });
  expect(oldDeleted.response.status).toBe(204);

  const deleted = await api.DELETE("/api/v0/workspaces/{slug}", { params: { path: { slug } }, headers: bearer(admin) });
  expect(deleted.response.status).toBe(204);
  await expectWorkspaceDeleted(db, slug, adminEmail, deletedAloneTables);
  await expectLabels(db, acmeWeb.id, webLabels(true));
  // Each of Web's labels carries Acme's deletion, its moment and its author: expectWorkspaceDeleted's counts cannot
  // tell a label stamped earlier, as Old's Bug, deleted before, gives the labels an earlier row.
  expect(
    await db.query(
      `SELECT l.name, l.deleted_at = w.deleted_at AS at_acmes_moment, l.updated_by_id = w.updated_by_id AS by_acmes_deleter
         FROM labels l JOIN workspaces w ON w.id = l.workspace_id WHERE l.project_id = $1 ORDER BY l.sort_order`,
      [acmeWeb.id]
    ),
    "when and by whom Web's labels were deleted"
  ).toEqual([
    { name: "Bug", at_acmes_moment: true, by_acmes_deleter: true },
    { name: "UI", at_acmes_moment: true, by_acmes_deleter: true },
  ]);
  // Old and every row under it keep its deletion: its moment and its author, the admin.
  await expectProjectDeleted(db, old.id, adminEmail);
  // The invitations accepted before keep the moment they were answered; the declined one goes with the pending
  // one; the other workspace keeps everything.
  const invitations = (memberRole: number, withTheWorkspace: boolean) => [
    { email: memberEmail, role: memberRole, accepted: true, responded: true, deleted: true },
    { email: inviteeEmail, role: 5, accepted: false, responded: false, deleted: withTheWorkspace },
    { email: declinerEmail, role: 15, accepted: false, responded: true, deleted: withTheWorkspace },
  ];
  await expectInvitations(
    db,
    slug,
    adminEmail,
    [...invitations(15, true), { email: coAdminEmail, role: 20, accepted: true, responded: true, deleted: true }],
    tokens
  );
  await expectInvitations(db, other, adminEmail, invitations(20, false), tokens);
  await expectMembership(db, other, memberEmail, { role: 20, is_active: true });
  await expectProjectCreated(db, other, web, adminEmail, memberEmail, [
    { email: adminEmail, sort_order: 65535 },
    { email: memberEmail, sort_order: 65535 },
  ]);
  await expectPreferences(db, other, memberEmail, {
    navigation_control_preference: "ACCORDION",
    navigation_project_limit: 3,
  });
  await expectLabels(db, otherWeb.id, webLabels(false));
  await readOther();
  const gone = await Promise.all(
    [admin, member].map((token) =>
      api.GET("/api/v0/workspaces/{slug}", { params: { path: { slug } }, headers: bearer(token) })
    )
  );
  expect(gone.map((g) => [g.response.status, g.error?.code])).toEqual([
    [404, "workspace.not_found"],
    [404, "workspace.not_found"],
  ]);
});

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
  await page.locator("#name").fill("Acme Corp");
  await page.getByRole("button", { name: "Select organization size" }).click();
  await page.getByRole("option", { name: "11-50" }).click();
  await page.getByRole("button", { name: "UTC" }).click();
  await page.getByRole("combobox", { name: "Search" }).fill("Asia/Shanghai");
  await page.getByRole("option", { name: "Beijing" }).click();
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
  const tabB = await context.newPage();
  await tabB.goto("/site.webmanifest.json");
  await writeRecord(tabB, newRecord(await login(api, y)));
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
