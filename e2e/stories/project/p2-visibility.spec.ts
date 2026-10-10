import {
  addProjectMembers,
  amidAnotherWorkspace,
  archiveProject,
  createProject,
  createWorkspace,
  inviteAndAccept,
  slugFor,
  type Api,
  type Project,
} from "../../fixtures/api";
import { expectMember } from "../../fixtures/assert/project";
import { accountId, bearer, createPAT, emailFor, register } from "../../fixtures/auth";
import { EMOJI_CHECK_WARNING, expectQuietConsole, refuseClipboardWrites, watchPage } from "../../fixtures/browser";
import { PROJECT_MEMBER, valued } from "../../fixtures/mounts";
import {
  answerTo,
  closedByEscape,
  closedWithin,
  enabledWithin,
  registerOnboarded,
  sentHeld,
} from "../../fixtures/settings-pages";
import { expect, test } from "../../fixtures/test";

// P2, the projects' list and who sees them, and joining (M3 design 2, 3.4, 3.5, 3.19); the page versions, with the
// "join the project" screen and the projects page's cards (7.6, P10).

/** The names of the projects of slug that the caller of token lists, in their order; archived: the archived ones. */
async function listed(api: Api, token: string, slug: string, archived = false): Promise<string[]> {
  const { data, error, response } = await api.GET("/api/v0/workspaces/{slug}/projects", {
    params: { path: { slug }, query: { archived } },
    headers: bearer(token),
  });
  expect(response.status, `list ${slug}'s projects: ${JSON.stringify(error)}`).toBe(200);
  return (data?.data ?? []).map((p) => p.name);
}

/** The answer of POST /api/v0/projects/{project_id}/join: its status, and the project or the problem's code. */
async function join(
  api: Api,
  token: string,
  id: string
): Promise<{ status: number; project?: Project; code?: string }> {
  const { data, error, response } = await api.POST("/api/v0/projects/{project_id}/join", {
    params: { path: { project_id: id } },
    headers: bearer(token),
  });
  return data ? { status: response.status, project: data } : { status: response.status, code: error?.code };
}

test("P2 (API): the admin lists every project, a member the public ones and his own, a guest his own, none the archived ones unless asked; a member joins a public project as a member at 65535, not before his own, and again changes nothing; a member cannot join a private project, nor a guest a public one, nor a guest the project he is a member of", async ({
  api,
  db,
}, testInfo) => {
  const adminEmail = emailFor(testInfo, "admin");
  const admin = (await createPAT(api, (await register(api, adminEmail)).access_token)).token;
  const memberEmail = emailFor(testInfo, "member");
  const member = (await createPAT(api, (await register(api, memberEmail)).access_token)).token;
  const guestEmail = emailFor(testInfo, "guest");
  const guest = (await createPAT(api, (await register(api, guestEmail)).access_token)).token;
  const slug = slugFor(testInfo);
  await createWorkspace(api, admin, { name: "Acme", slug });
  await inviteAndAccept(api, admin, slug, { email: memberEmail, token: member }, 15);
  await inviteAndAccept(api, admin, slug, { email: guestEmail, token: guest }, 5);
  const adminId = await accountId(api, admin);
  const memberId = await accountId(api, member);
  const guestId = await accountId(api, guest);
  // Each new project goes first in its creator's sidebar: the admin's Old, Docs, Secret, Web. Old is archived; the
  // guest is Docs' guest. The member made Notes, private, of which the admin is no member: the member's own, at 65535.
  const { web, secret, docs, notes } = await amidAnotherWorkspace(api, admin, testInfo, async () => {
    const made = {
      web: await createProject(api, admin, slug, { name: "Web", identifier: "WEB", network: 2 }),
      secret: await createProject(api, admin, slug, { name: "Secret", identifier: "SEC", network: 0 }),
      docs: await createProject(api, admin, slug, { name: "Docs", identifier: "DOCS", network: 2 }),
      notes: await createProject(api, member, slug, { name: "Notes", identifier: "NOTES", network: 0 }),
    };
    const old = await createProject(api, admin, slug, { name: "Old", identifier: "OLD", network: 2 });
    const archived = await api.POST("/api/v0/projects/{project_id}/archive", {
      params: { path: { project_id: old.id } },
      headers: bearer(admin),
    });
    expect(archived.response.status, "archive Old").toBe(200);
    await addProjectMembers(api, admin, made.docs.id, [{ member_id: guestId, role: 5 }]);
    return made;
  });

  // Each caller's places first, then the projects he has none in, by name.
  expect(await listed(api, admin, slug)).toEqual(["Docs", "Secret", "Web", "Notes"]);
  expect(await listed(api, member, slug)).toEqual(["Notes", "Docs", "Web"]);
  expect(await listed(api, guest, slug)).toEqual(["Docs"]);
  expect(await listed(api, admin, slug, true)).toEqual(["Old"]);
  expect(await listed(api, member, slug, true)).toEqual(["Old"]);
  expect(await listed(api, guest, slug, true)).toEqual([]);
  // The admin sees Notes as he sees every project, though he is no member of it.
  const seen = await api.GET("/api/v0/projects/{project_id}", {
    params: { path: { project_id: notes.id } },
    headers: bearer(admin),
  });
  expect({ status: seen.response.status, project: seen.data }).toMatchObject({
    status: 200,
    project: { id: notes.id, member_role: null, sort_order: null, member_ids: [memberId] },
  });

  // The member joins Web: a member's membership, with his display settings, both by him, at 65535 beside Notes: a
  // joiner's place is the default one, not before his other projects, which would be 55535 (M3 design 3.18).
  const joined = await join(api, member, web.id);
  expect(joined).toMatchObject({ status: 200, project: { id: web.id, member_role: 15, sort_order: 65535 } });
  expect(joined.project?.member_ids?.toSorted()).toEqual([adminId, memberId].toSorted());
  const membership = { role: 15, is_active: true, sort_order: 65535, by: memberEmail };
  await expectMember(db, web.id, memberEmail, membership);
  expect(await listed(api, member, slug)).toEqual(["Notes", "Web", "Docs"]);
  // Joining again changes nothing.
  expect(await join(api, member, web.id)).toMatchObject({
    status: 200,
    project: { member_role: 15, sort_order: 65535 },
  });
  await expectMember(db, web.id, memberEmail, membership);

  // Refused, nothing written: the member does not see Secret, the guest does not see Web; the guest sees Docs, as
  // its guest, and is refused as one.
  expect(await join(api, member, secret.id)).toEqual({ status: 404, code: "project.not_found" });
  await expectMember(db, secret.id, memberEmail, null);
  expect(await join(api, guest, web.id)).toEqual({ status: 404, code: "project.not_found" });
  await expectMember(db, web.id, guestEmail, null);
  expect(await join(api, guest, docs.id)).toEqual({ status: 403, code: "forbidden" });
  await expectMember(db, docs.id, guestEmail, { role: 5, is_active: true, sort_order: 65535, by: adminEmail });
});

/**
 * The reads of the project of projectId that nerve gives its members alone, which its pages make once the caller is one
 * (M3 design 7.1), as watchPage records them (none has a query).
 */
const membersReads = (projectId: string) => PROJECT_MEMBER.map((read) => valued(read, { [projectId]: "{project}" }));

test("P2 (page): a member opens a public project he is no member of by its address: the join screen, which reads the project alone; joined, its pages show and read the rest; a private project is not found, an archived one says so", async ({
  api,
  db,
  signedInPage,
}, testInfo) => {
  const adminEmail = emailFor(testInfo, "admin");
  const admin = (await registerOnboarded(api, adminEmail)).access_token;
  const memberEmail = emailFor(testInfo, "member");
  const member = await registerOnboarded(api, memberEmail);
  const slug = slugFor(testInfo);
  await createWorkspace(api, admin, { name: "Acme", slug });
  await inviteAndAccept(api, admin, slug, { email: memberEmail, token: member.access_token }, 15);
  const web = await createProject(api, admin, slug, { name: "Web", identifier: "WEB", network: 2 });
  const secret = await createProject(api, admin, slug, { name: "Secret", identifier: "SEC", network: 0 });
  const old = await createProject(api, admin, slug, { name: "Old", identifier: "OLD", network: 2 });
  await archiveProject(api, admin, old.id);

  const page = await signedInPage(member);
  const watch = await watchPage(page);
  await page.goto(`/${slug}/settings/projects/${web.id}`);
  const joinButton = page.getByRole("button", { name: "Join project" });
  await expect(joinButton).toBeVisible();
  // The page read the project, and none of what nerve gives its members alone.
  expect(watch.apiRequests).toContain(`GET /api/v0/projects/${web.id}`);
  expect(watch.apiRequests.filter((request) => membersReads(web.id).includes(request))).toEqual([]);

  // Joining: one request, the button busy until nerve answers; then the project's general page, which reads the rest.
  const { release } = await sentHeld(page, "POST", `/api/v0/projects/${web.id}/join`, () => joinButton.click());
  const joining = page.getByRole("button", { name: "Joining project" });
  expect(await enabledWithin(joining)).toBe(false);
  expect((await release()).status()).toBe(200);
  await expect(page.locator("#name")).toHaveValue("Web");
  expect(watch.apiRequests.filter((request) => request === `POST /api/v0/projects/${web.id}/join`)).toHaveLength(1);
  await expect.poll(() => membersReads(web.id).filter((read) => !watch.apiRequests.includes(read))).toEqual([]);
  await expectMember(db, web.id, memberEmail, { role: 15, is_active: true, sort_order: 65535, by: memberEmail });

  // A private project he does not see is not found; an archived one says so, and its button opens the archived ones.
  await page.goto(`/${slug}/projects/${secret.id}/issues`);
  await expect(page.getByText("Project not found")).toBeVisible();
  await page.goto(`/${slug}/projects/${old.id}/issues`);
  await expect(page.getByText("This project is archived")).toBeVisible();
  expect(watch.apiRequests).toContain(`GET /api/v0/projects/${old.id}`);
  expect(watch.apiRequests.filter((request) => membersReads(old.id).includes(request))).toEqual([]);
  await page.getByRole("button", { name: "Archived projects" }).click();
  await expect(page).toHaveURL(`/${slug}/projects/archives`);

  // The private project's 404, the browser's report of it
  expect([watch.cspViolations, watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([
    [],
    [`404 GET /api/v0/projects/${secret.id}`],
    [],
    [],
  ]);
  await expectQuietConsole(page, watch, {
    // the three loads: the settings page, the private project's, the archived one's
    warnings: [EMOJI_CHECK_WARNING, EMOJI_CHECK_WARNING, EMOJI_CHECK_WARNING],
    errors: ["Failed to load resource: the server responded with a status of 404 (Not Found)"],
  });
});

test("P2 (page): the projects page shows a member the public project to join and not the private one; its card copies its link, or says it could not, and joins it, the dialog held and busy until nerve answers, and open when nerve refuses", async ({
  api,
  db,
  signedInPage,
}, testInfo) => {
  const admin = (await registerOnboarded(api, emailFor(testInfo, "admin"))).access_token;
  const memberEmail = emailFor(testInfo, "member");
  const member = await registerOnboarded(api, memberEmail);
  const slug = slugFor(testInfo);
  await createWorkspace(api, admin, { name: "Acme", slug });
  await inviteAndAccept(api, admin, slug, { email: memberEmail, token: member.access_token }, 15);
  const web = await createProject(api, admin, slug, { name: "Web", identifier: "WEB", network: 2 });
  await createProject(api, admin, slug, { name: "Secret", identifier: "SEC", network: 0 });
  const gone = await createProject(api, admin, slug, { name: "Gone", identifier: "GONE", network: 2 });

  const page = await signedInPage(member);
  await page.context().grantPermissions(["clipboard-read", "clipboard-write"]);
  const watch = await watchPage(page);
  await page.goto(`/${slug}/projects`);
  const card = page.getByRole("link", { name: /Web/ });
  await expect(card).toBeVisible();
  await expect(page.getByRole("link", { name: /Secret/ })).toHaveCount(0);

  // The card copies the project's link; a browser that refuses the page the clipboard: the card says it could not.
  await card.getByRole("button", { name: "Copy link" }).click();
  await expect(page.getByText("Project link copied to clipboard")).toBeVisible();
  expect(await page.evaluate(() => navigator.clipboard.readText())).toBe(
    new URL(`/${slug}/projects/${web.id}/issues`, page.url()).toString()
  );
  await refuseClipboardWrites(page);
  await card.getByRole("button", { name: "Copy link" }).click();
  await expect(page.getByText("Something went wrong. Please try again.")).toBeVisible();

  // A join nerve refuses: the admin deletes Gone while its dialog is open. The page says nerve's reason, and the
  // dialog stays open, Join enabled again: it closes of itself only once joined. Cancel closes it.
  await page.getByRole("link", { name: /Gone/ }).getByRole("button", { name: "Join", exact: true }).click();
  const deleted = await api.DELETE("/api/v0/projects/{project_id}", {
    params: { path: { project_id: gone.id } },
    headers: bearer(admin),
  });
  expect(deleted.response.status).toBe(204);
  const joinGone = page.getByRole("dialog").getByRole("button", { name: "Join Project" });
  expect((await answerTo(page, "POST", `/api/v0/projects/${gone.id}/join`, () => joinGone.click())).status()).toBe(404);
  await expect(page.getByText("The project does not exist, or you cannot see it.")).toBeVisible();
  expect(await closedWithin(page)).toBe(false);
  await expect(joinGone).toBeEnabled();
  await page.getByRole("dialog").getByRole("button", { name: "Cancel" }).click();
  await expect(page.locator('[role="dialog"][aria-modal="true"]')).toHaveCount(0);

  // Joining Web from its card: the dialog cannot be dismissed while nerve has not answered, its button busy, and
  // sends one join.
  await card.getByRole("button", { name: "Join", exact: true }).click();
  const { release } = await sentHeld(page, "POST", `/api/v0/projects/${web.id}/join`, () =>
    page.getByRole("button", { name: "Join Project" }).click()
  );
  expect(await enabledWithin(page.getByRole("dialog").getByRole("button", { name: "Joining..." }))).toBe(false);
  await expect(page.getByRole("dialog").getByRole("button", { name: "Cancel" })).toBeDisabled();
  expect(await closedByEscape(page)).toBe(false);
  expect((await release()).status()).toBe(200);
  // joined, the page opens the project's work items
  await expect(page).toHaveURL(`/${slug}/projects/${web.id}/issues`);
  expect(watch.apiRequests.filter((request) => request === `POST /api/v0/projects/${web.id}/join`)).toHaveLength(1);
  await expectMember(db, web.id, memberEmail, { role: 15, is_active: true, sort_order: 65535, by: memberEmail });
  // The project's crumb in the header is one Tab stop, the button of its list (M3 design 7.7): its title, which opens
  // the project's work items by a click, is not another. Tab goes on to the next crumb.
  await page.getByRole("button", { name: "Web", exact: true }).first().focus();
  await page.keyboard.press("Tab");
  await expect(page.getByRole("link", { name: "Work Items", exact: true })).toBeFocused();

  // The refused join's 404; the work items' page asks Plane's address of the filters, M4's (P8b spec §5)
  const filters = `GET /api/workspaces/${slug}/projects/${web.id}/user-properties/`;
  await expect.poll(() => watch.apiFailures).toEqual([`404 POST /api/v0/projects/${gone.id}/join`, `404 ${filters}`]);
  expect([watch.cspViolations, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [filters], []]);
  await expectQuietConsole(page, watch, {
    warnings: [EMOJI_CHECK_WARNING],
    errors: [
      "Failed to load resource: the server responded with a status of 404 (Not Found)",
      "Failed to load resource: the server responded with a status of 404 (Not Found)",
    ],
  });
});
