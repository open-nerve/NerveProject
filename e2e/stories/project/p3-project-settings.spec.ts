import {
  addProjectMembers,
  amidAnotherWorkspace,
  changeProject,
  createProject,
  createWorkspace,
  inviteAndAccept,
  projectMemberWrites,
  slugFor,
} from "../../fixtures/api";
import { expectMember, projectSettingsOf } from "../../fixtures/assert/project";
import { accountId, bearer, createPAT, emailFor, newAccount, register } from "../../fixtures/auth";
import { EMOJI_CHECK_WARNING, expectQuietConsole, requestsElsewhere, watchPage } from "../../fixtures/browser";
import { shownNameOf } from "../../fixtures/project-pages";
import {
  answerTo,
  bodiesSentTo,
  closedByEscape,
  enabledWithin,
  moveWithinApp,
  registerOnboarded,
  sentHeld,
  sentTo,
} from "../../fixtures/settings-pages";
import { expect, test } from "../../fixtures/test";
import { anotherBrowser } from "../../fixtures/workspace-pages";

// P3, a project's settings (M3 design 2, 3.4, 3.5, 3.19, 7.6).

test("P3 (API): the project's admin adds a member and a guest, whom the member lists with him, then changes every setting, the member its lead and default assignee; its member may not; a lead or default assignee who is its guest or no member, and archive_in 13, change nothing", async ({
  api,
  db,
}, testInfo) => {
  const adminEmail = emailFor(testInfo, "admin");
  const admin = (await createPAT(api, (await register(api, adminEmail)).access_token)).token;
  const memberEmail = emailFor(testInfo, "member");
  const member = (await createPAT(api, (await register(api, memberEmail)).access_token)).token;
  const guestEmail = emailFor(testInfo, "guest");
  const guest = (await createPAT(api, (await register(api, guestEmail)).access_token)).token;
  const otherEmail = emailFor(testInfo, "other");
  const other = (await createPAT(api, (await register(api, otherEmail)).access_token)).token;
  const slug = slugFor(testInfo);
  await createWorkspace(api, admin, { name: "Acme", slug });
  await inviteAndAccept(api, admin, slug, { email: memberEmail, token: member }, 15);
  await inviteAndAccept(api, admin, slug, { email: guestEmail, token: guest }, 5);
  await inviteAndAccept(api, admin, slug, { email: otherEmail, token: other }, 15);
  const memberId = await accountId(api, member);
  const guestId = await accountId(api, guest);
  const otherId = await accountId(api, other);
  // other is a member of Acme and of Ops, not of Web: a membership of another project makes no one Web's assignee.
  // The member is Ops' member too, at 65535 in his sidebar: Web, added later, goes before it.
  const web = await amidAnotherWorkspace(api, admin, testInfo, async () => {
    const ops = await createProject(api, admin, slug, { name: "Ops", identifier: "OPS" });
    await addProjectMembers(api, admin, ops.id, [
      { member_id: otherId, role: 15 },
      { member_id: memberId, role: 15 },
    ]);
    return createProject(api, admin, slug, { name: "Web", identifier: "WEB" });
  });

  const added = await addProjectMembers(api, admin, web.id, [
    { member_id: memberId, role: 15 },
    { member_id: guestId, role: 5 },
  ]);
  expect(added.map((m) => ({ project_id: m.project_id, member_id: m.member_id, role: m.role }))).toEqual([
    { project_id: web.id, member_id: memberId, role: 15 },
    { project_id: web.id, member_id: guestId, role: 5 },
  ]);
  await expectMember(db, web.id, memberEmail, { role: 15, is_active: true, sort_order: 55535, by: adminEmail });
  await expectMember(db, web.id, guestEmail, { role: 5, is_active: true, sort_order: 65535, by: adminEmail });
  // Web lists exactly its own three; other, Ops' member, is not among them.
  const members = await api.GET("/api/v0/projects/{project_id}/members", {
    params: { path: { project_id: web.id } },
    headers: bearer(member),
  });
  expect(members.response.status, `Web's members: ${JSON.stringify(members.error)}`).toBe(200);
  expect(
    members.data?.data.map((m) => ({ member_id: m.member_id, role: m.role })).toSorted((a, b) => b.role - a.role)
  ).toEqual([
    { member_id: await accountId(api, admin), role: 20 },
    { member_id: memberId, role: 15 },
    { member_id: guestId, role: 5 },
  ]);
  // Web's member may not add to it: refused, nothing written.
  expect(await projectMemberWrites(api, web.id).add(member, [{ member_id: otherId, role: 15 }])).toEqual({
    status: 403,
    code: "forbidden",
  });
  await expectMember(db, web.id, otherEmail, null);

  const logo = { in_use: "icon", icon: { name: "rocket", color: "#46A758" } } as const;
  const settings = {
    name: "Site",
    identifier: "SITE",
    description: "The site",
    network: 0 as const,
    timezone: "Asia/Shanghai",
    logo_props: logo,
    cycle_view: false,
    module_view: false,
    issue_views_view: false,
    intake_view: true,
    guest_view_all_features: true,
    archive_in: 3,
  };
  expect(
    await changeProject(api, admin, web.id, {
      ...settings,
      identifier: "site",
      project_lead_id: memberId,
      default_assignee_id: memberId,
    })
  ).toMatchObject({ status: 200, project: { ...settings, project_lead_id: memberId, default_assignee_id: memberId } });
  const changed = { ...settings, lead: memberEmail, default_assignee: memberEmail, by: adminEmail };
  expect(await projectSettingsOf(db, web.id)).toEqual(changed);

  // Each refused, all at once: none writes.
  const refusals = [
    { token: member, body: { name: "Mine" }, want: { status: 403, code: "forbidden" } },
    {
      token: admin,
      body: { archive_in: 13 },
      want: { status: 422, code: "validation_failed", errors: [{ field: "archive_in", code: "out_of_range" }] },
    },
    {
      token: admin,
      body: { project_lead_id: guestId },
      want: { status: 422, code: "validation_failed", errors: [{ field: "project_lead_id", code: "not_allowed" }] },
    },
    {
      token: admin,
      body: { default_assignee_id: otherId },
      want: { status: 422, code: "validation_failed", errors: [{ field: "default_assignee_id", code: "not_allowed" }] },
    },
  ];
  expect(
    await Promise.all(refusals.map(({ token, body }) => changeProject(api, token, web.id, body))),
    "the refusals"
  ).toEqual(refusals.map((r) => r.want));
  expect(await projectSettingsOf(db, web.id)).toEqual(changed);

  // Both cleared with null.
  expect(await changeProject(api, admin, web.id, { project_lead_id: null, default_assignee_id: null })).toMatchObject({
    status: 200,
    project: { project_lead_id: null, default_assignee_id: null },
  });
  expect(await projectSettingsOf(db, web.id)).toEqual({ ...changed, lead: null, default_assignee: null });
});

test("P3 (page): the project's admin changes its name, identifier, description, visibility, time zone and icon on its general page, which hold after a reload; an identifier another project has is said under it and nothing is sent; the page waits for nerve; its member sees them and can change nothing", async ({
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
  await createWorkspace(api, admin.access_token, { name: "Acme", slug });
  await inviteAndAccept(api, admin.access_token, slug, { email: memberEmail, token: member.access_token }, 15);
  await createProject(api, admin.access_token, slug, { name: "Ops", identifier: "OPS" });
  const web = await createProject(api, admin.access_token, slug, { name: "Web", identifier: "WEB" });
  const memberId = await accountId(api, member.access_token);
  await addProjectMembers(api, admin.access_token, web.id, [{ member_id: memberId, role: 15 }]);

  const page = await signedInPage(admin);
  const watch = await watchPage(page);
  await page.goto(`/${slug}/settings/projects/${web.id}`);
  await expect(page.locator("#name")).toHaveValue("Web");
  await page.locator("#name").fill("Site");
  await page.locator("#description").fill("The site");
  await page.getByRole("button", { name: "Public" }).click();
  await page.getByRole("option", { name: /Private/ }).click();
  // the workspace's time zone, which the project took, for Shanghai's
  await page.getByRole("button", { name: "UTC" }).click();
  await page.getByRole("combobox", { name: "Search" }).fill("Asia/Shanghai");
  await page.keyboard.press("ArrowDown");
  await page.keyboard.press("Enter");
  // The icon: the project has none, so the picker opens on its icons; an emoji from the emoji nerve serves.
  await page.getByRole("button", { name: "Project icon" }).click();
  await page.getByRole("tab", { name: "Emoji" }).click();
  await page.getByRole("searchbox").fill("rocket");
  await page.getByRole("gridcell", { name: "Rocket" }).click();
  await expect(page.getByRole("button", { name: "Project icon" })).toContainText("🚀");

  // An identifier another project has: the page asks nerve, says so under it, and sends no change.
  const changes = await bodiesSentTo(page, "PATCH", `/api/v0/projects/${web.id}`);
  const update = page.getByRole("button", { name: "Update project" });
  await page.locator("#identifier").fill("o.ps");
  await expect(page.locator("#identifier")).toHaveValue("OPS");
  const checked = await answerTo(page, "GET", `/api/v0/workspaces/${slug}/project-identifiers/OPS`, () =>
    update.click()
  );
  expect(await checked.json()).toEqual({ available: false });
  await expect(page.getByText("A project of this workspace already has this identifier.")).toBeVisible();
  await expect(update).toBeEnabled();
  expect(changes).toEqual([]);

  // Changed: the fields the page edits, the button busy until nerve answers.
  await page.locator("#identifier").fill("site");
  const { body, release } = await sentHeld(page, "PATCH", `/api/v0/projects/${web.id}`, () => update.click());
  const logo = { in_use: "emoji", emoji: { value: "128640" } };
  const settings = {
    name: "Site",
    identifier: "SITE",
    description: "The site",
    network: 0,
    logo_props: logo,
    timezone: "Asia/Shanghai",
  };
  expect(body).toEqual(settings);
  expect(await enabledWithin(page.getByRole("button", { name: "Updating" }))).toBe(false);
  expect((await release()).status()).toBe(200);
  await expect(page.getByText("Project updated successfully")).toBeVisible();
  await expect(update).toBeEnabled();
  expect(await projectSettingsOf(db, web.id)).toMatchObject({ ...settings, by: adminEmail });
  await page.reload();
  await expect(page.locator("#name")).toHaveValue("Site");
  await expect(page.locator("#identifier")).toHaveValue("SITE");
  await expect(page.locator("#description")).toHaveValue("The site");
  await expect(page.getByRole("button", { name: "Private" })).toBeVisible();
  await expect(page.getByRole("button", { name: "Beijing" })).toBeVisible();
  await expect(page.getByRole("button", { name: "Project icon" })).toContainText("🚀");

  // Its member sees what nerve holds, in a form he cannot change, and no archiving or deletion.
  const theMember = await anotherBrowser(browser, baseURL ?? "", member);
  const memberWatch = await watchPage(theMember.page);
  await theMember.page.goto(`/${slug}/settings/projects/${web.id}`);
  await expect(theMember.page.locator("#name")).toHaveValue("Site");
  await Promise.all(
    ["#name", "#identifier", "#description"].map((field) => expect(theMember.page.locator(field), field).toBeDisabled())
  );
  await Promise.all(
    ["Project icon", "Private", "Beijing", "Update project"].map((name) =>
      expect(theMember.page.getByRole("button", { name }), name).toBeDisabled()
    )
  );
  await expect(theMember.page.getByRole("button", { name: "Archive" })).toHaveCount(0);
  await expect(theMember.page.getByRole("button", { name: "Delete" })).toHaveCount(0);
  expect([memberWatch.apiFailures, memberWatch.oldApiRequests, memberWatch.pageErrors]).toEqual([[], [], []]);
  await expectQuietConsole(theMember.page, memberWatch, { warnings: [EMOJI_CHECK_WARNING] });
  await theMember.close();

  // The emoji came from nerve: nothing asked of another address, nothing blocked.
  expect(requestsElsewhere(page, watch)).toEqual([]);
  expect([watch.cspViolations, watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], [], []]);
  // the first load, and the reload
  await expectQuietConsole(page, watch, { warnings: [EMOJI_CHECK_WARNING, EMOJI_CHECK_WARNING] });
});

test("P3 (page): the project's admin turns its cycles, modules, views and intake on, each on its feature's page, and has its closed work items archived, after a range of his own that the page holds until nerve answers, then after 3 months", async ({
  api,
  db,
  signedInPage,
}, testInfo) => {
  const adminEmail = emailFor(testInfo, "admin");
  const admin = await registerOnboarded(api, adminEmail);
  const slug = slugFor(testInfo);
  await createWorkspace(api, admin.access_token, { name: "Acme", slug });
  const web = await createProject(api, admin.access_token, slug, { name: "Web", identifier: "WEB" });
  const settings = `/${slug}/settings/projects/${web.id}`;
  const projectApi = `/api/v0/projects/${web.id}`;

  const page = await signedInPage(admin);
  const watch = await watchPage(page);
  /** Opens the page of the feature of path and turns it on: the feature alone is sent, the switch then shows it. */
  const flip = async (path: string, name: string) => {
    await page.goto(`${settings}/features/${path}`);
    const toggle = page.getByRole("switch", { name });
    await expect(toggle).not.toBeChecked();
    const flipped = await sentTo(page, "PATCH", projectApi, () => toggle.click());
    await expect(toggle).toBeChecked();
    return [flipped.answer.status(), flipped.body];
  };
  expect(await flip("cycles", "Enable cycles")).toEqual([200, { cycle_view: true }]);
  expect(await flip("modules", "Enable modules")).toEqual([200, { module_view: true }]);
  expect(await flip("views", "Enable views")).toEqual([200, { issue_views_view: true }]);
  expect(await flip("intake", "Enable intake")).toEqual([200, { intake_view: true }]);

  // The auto-archiving: on, after a month.
  await page.goto(`${settings}/automations`);
  const turnedOn = await sentTo(page, "PATCH", projectApi, () =>
    page.getByRole("switch", { name: "Auto-archive closed work items" }).click()
  );
  expect([turnedOn.answer.status(), turnedOn.body]).toEqual([200, { archive_in: 1 }]);
  // A range of his own: the modal waits for nerve, busy and not to be closed; it closes once nerve has made it.
  await page.getByRole("button", { name: "1 month" }).click();
  await page.getByRole("button", { name: "Customize time range" }).click();
  await page.locator("#archive_in").fill("6");
  const { body, release } = await sentHeld(page, "PATCH", projectApi, () =>
    page.getByRole("button", { name: "Submit" }).click()
  );
  expect(body).toEqual({ archive_in: 6 });
  await expect(page.getByRole("button", { name: "Submitting..." })).toBeDisabled();
  await expect(page.getByRole("button", { name: "Cancel" })).toBeDisabled();
  expect(await closedByEscape(page)).toBe(false);
  expect((await release()).status()).toBe(200);
  await expect(page.locator("#archive_in")).toHaveCount(0);
  await expect(page.getByRole("button", { name: "6 months" })).toBeVisible();
  // Then 3 months, from the list.
  await page.getByRole("button", { name: "6 months" }).click();
  const three = await sentTo(page, "PATCH", projectApi, () => page.getByRole("option", { name: "3 months" }).click());
  expect([three.answer.status(), three.body]).toEqual([200, { archive_in: 3 }]);
  await expect(page.getByRole("button", { name: "3 months" })).toBeVisible();

  expect(await projectSettingsOf(db, web.id)).toMatchObject({
    cycle_view: true,
    module_view: true,
    issue_views_view: true,
    intake_view: true,
    archive_in: 3,
    by: adminEmail,
  });
  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], []]);
  // a load of each feature's page, and of the automations page
  await expectQuietConsole(page, watch, { warnings: Array.from({ length: 5 }, () => EMOJI_CHECK_WARNING) });
});

test("P3 (page): the project's admin makes a member its lead and its default assignee, each picked from its members who are not its guests, and lets its guests see every work item", async ({
  api,
  db,
  signedInPage,
}, testInfo) => {
  const adminEmail = emailFor(testInfo, "admin");
  const admin = await registerOnboarded(api, adminEmail);
  const slug = slugFor(testInfo);
  await createWorkspace(api, admin.access_token, { name: "Acme", slug });
  // ann is Web's member, gus its guest; otto is a member of acme, not of Web.
  const [ann, gus, otto] = await Promise.all(["ann", "gus", "otto"].map((label) => newAccount(api, testInfo, label)));
  if (!ann || !gus || !otto) throw new Error("the accounts were not registered");
  await inviteAndAccept(api, admin.access_token, slug, ann, 15);
  await inviteAndAccept(api, admin.access_token, slug, gus, 5);
  await inviteAndAccept(api, admin.access_token, slug, otto, 15);
  const web = await createProject(api, admin.access_token, slug, { name: "Web", identifier: "WEB" });
  await addProjectMembers(api, admin.access_token, web.id, [
    { member_id: ann.id, role: 15 },
    { member_id: gus.id, role: 5 },
  ]);

  const page = await signedInPage(admin);
  const watch = await watchPage(page);
  await page.goto(`/${slug}/settings/projects/${web.id}/members`);
  /** Picks ann in the member select under title: the select offers the admin and ann, and none; the page sends field. */
  const pickAnn = async (title: string) => {
    await page.getByRole("heading", { name: title }).locator("xpath=following::button[1]").click();
    await expect(page.getByRole("option")).toHaveCount(3);
    await expect(page.getByRole("option", { name: shownNameOf(adminEmail) })).toBeVisible();
    await expect(page.getByRole("option", { name: "None" })).toBeVisible();
    const picked = await sentTo(page, "PATCH", `/api/v0/projects/${web.id}`, () =>
      page.getByRole("option", { name: shownNameOf(ann.email) }).click()
    );
    await expect(page.getByRole("heading", { name: title }).locator("xpath=following::button[1]")).toHaveText(
      new RegExp(shownNameOf(ann.email))
    );
    return [picked.answer.status(), picked.body];
  };
  expect(await pickAnn("Project Lead")).toEqual([200, { project_lead_id: ann.id }]);
  expect(await pickAnn("Default Assignee")).toEqual([200, { default_assignee_id: ann.id }]);
  const guests = await sentTo(page, "PATCH", `/api/v0/projects/${web.id}`, () =>
    page.getByRole("switch", { name: "Guest access" }).click()
  );
  expect([guests.answer.status(), guests.body]).toEqual([200, { guest_view_all_features: true }]);
  await expect(page.getByRole("switch", { name: "Guest access" })).toBeChecked();

  expect(await projectSettingsOf(db, web.id)).toMatchObject({
    lead: ann.email,
    default_assignee: ann.email,
    guest_view_all_features: true,
    by: adminEmail,
  });
  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], []]);
  await expectQuietConsole(page, watch, { warnings: [EMOJI_CHECK_WARNING] });
});

test("P3 (page): another project's general page, reached without leaving the route, shows that project's values", async ({
  api,
  signedInPage,
}, testInfo) => {
  const admin = await registerOnboarded(api, emailFor(testInfo, "admin"));
  const slug = slugFor(testInfo);
  await createWorkspace(api, admin.access_token, { name: "Acme", slug });
  const ops = await createProject(api, admin.access_token, slug, { name: "Ops", identifier: "OPS" });
  const web = await createProject(api, admin.access_token, slug, { name: "Web", identifier: "WEB" });
  const page = await signedInPage(admin);
  const watch = await watchPage(page);
  // the router's own move, as a switcher on the settings pages would make it: no link of the app does it yet
  await page.goto(`/${slug}/settings/projects/${web.id}`);
  await expect(page.locator("#name")).toHaveValue("Web");
  await moveWithinApp(page, `/${slug}/settings/projects/${ops.id}`);
  await expect(page.locator("#name")).toHaveValue("Ops");
  // Web's read is in the session's cache: the wrapper shows its page at once, the route mounted throughout
  await moveWithinApp(page, `/${slug}/settings/projects/${web.id}`);
  await expect(page.locator("#identifier")).toHaveValue("WEB");

  expect([watch.cspViolations, watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], [], []]);
  await expectQuietConsole(page, watch, { warnings: [EMOJI_CHECK_WARNING] });
});
