import {
  createProject,
  createWorkspace,
  inviteAndAccept,
  slugFor,
  type Api,
  type ProjectCreate,
} from "../../fixtures/api";
import { countProjects, expectProjectCreated } from "../../fixtures/assert/project";
import { accountId, bearer, createPAT, emailFor, register } from "../../fixtures/auth";
import { EMOJI_CHECK_WARNING, expectQuietConsole, requestsElsewhere, watchPage } from "../../fixtures/browser";
import { registerOnboarded } from "../../fixtures/settings-pages";
import { expect, test } from "../../fixtures/test";

// P1, create a project (M3 design 2, 3.17, 3.18), and its page (P10).

/** Where nerve serves the emoji picker's data (M3 design 7.7). */
const EMOJIBASE = "/assets/emojibase/15.3.2/en";

/** The answer of GET /api/v0/workspaces/{slug}/project-identifiers/{identifier}. */
async function availability(api: Api, token: string, slug: string, identifier: string): Promise<unknown> {
  const { data, error, response } = await api.GET("/api/v0/workspaces/{slug}/project-identifiers/{identifier}", {
    params: { path: { slug, identifier } },
    headers: bearer(token),
  });
  expect(response.status, `check ${identifier}: ${JSON.stringify(error)}`).toBe(200);
  return data;
}

/** The answer of GET /api/v0/projects/{project_id}: its status, and the project or the problem's code. */
async function read(
  api: Api,
  token: string,
  id: string
): Promise<{ status: number; project?: unknown; code?: string }> {
  const { data, error, response } = await api.GET("/api/v0/projects/{project_id}", {
    params: { path: { project_id: id } },
    headers: bearer(token),
  });
  return data ? { status: response.status, project: data } : { status: response.status, code: error?.code };
}

/** The answer to a createProject whose field the rules do not allow. */
function notAllowed(field: string) {
  return { status: 422, code: "validation_failed", errors: [{ field, code: "not_allowed" }] };
}

test("P1 (API): a member creates a project with the admin its lead, both its admins, first in their sidebars, with its six states; a taken identifier or name, a name with a forbidden character, a guest, or a lead who is a guest or no member adds nothing", async ({
  api,
  db,
}, testInfo) => {
  const adminEmail = emailFor(testInfo, "admin");
  const admin = (await createPAT(api, (await register(api, adminEmail)).access_token)).token;
  const memberEmail = emailFor(testInfo, "member");
  const member = (await createPAT(api, (await register(api, memberEmail)).access_token)).token;
  const guestEmail = emailFor(testInfo, "guest");
  const guest = (await createPAT(api, (await register(api, guestEmail)).access_token)).token;
  const stranger = (await register(api, emailFor(testInfo, "stranger"))).access_token;
  const slug = slugFor(testInfo);
  await createWorkspace(api, admin, { name: "Acme", slug, timezone: "Asia/Shanghai" });
  // The stranger admins a workspace of his own: his role there makes him no lead of Acme's, and each workspace
  // answers for its own identifiers.
  const strangers = slugFor(testInfo, "stranger");
  await createWorkspace(api, stranger, { name: "Stranger", slug: strangers });
  await inviteAndAccept(api, admin, slug, { email: memberEmail, token: member }, 15);
  await inviteAndAccept(api, admin, slug, { email: guestEmail, token: guest }, 5);
  const [adminId, memberId, guestId, strangerId] = await Promise.all(
    [admin, member, guest, stranger].map((token) => accountId(api, token))
  );
  expect(await availability(api, member, slug, "web")).toEqual({ available: true });
  expect(await availability(api, member, slug, "WE-B")).toEqual({ available: false });

  const logo = { in_use: "emoji", emoji: { value: "128640" } } as const;
  const created = await createProject(api, member, slug, {
    name: "Web",
    identifier: "web",
    description: "The site",
    network: 2,
    project_lead_id: adminId,
    logo_props: logo,
  });

  const web = {
    name: "Web",
    identifier: "WEB",
    description: "The site",
    network: 2,
    timezone: "Asia/Shanghai",
    logo_props: logo,
  };
  expect(created).toMatchObject({
    ...web,
    project_lead_id: adminId,
    archived_at: null,
    member_role: 20,
    sort_order: 65535,
    member_ids: [memberId, adminId],
  });
  const members = [
    { email: memberEmail, sort_order: 65535 },
    { email: adminEmail, sort_order: 65535 },
  ];
  expect(await expectProjectCreated(db, slug, web, memberEmail, adminEmail, members)).toBe(created.id);
  expect(await availability(api, admin, slug, "Web")).toEqual({ available: false });
  expect(await availability(api, member, slug, "ops")).toEqual({ available: true });
  expect(await availability(api, stranger, strangers, "web")).toEqual({ available: true });

  const before = await countProjects(db);
  // Each refused, all at once: none writes.
  const refusals: { token: string; body: ProjectCreate; want: { status: number; code: string } }[] = [
    {
      token: member,
      body: { name: "Web 2", identifier: "Web" },
      want: { status: 409, code: "project.identifier_taken" },
    },
    { token: member, body: { name: "Web", identifier: "WEB2" }, want: { status: 409, code: "project.name_taken" } },
    { token: member, body: { name: "Web-2", identifier: "WEB2" }, want: notAllowed("name") },
    { token: member, body: { name: "Web.2", identifier: "WEB2" }, want: notAllowed("name") },
    { token: guest, body: { name: "Mine", identifier: "MINE" }, want: { status: 403, code: "forbidden" } },
    {
      token: member,
      body: { name: "Ops", identifier: "OPS", project_lead_id: guestId },
      want: notAllowed("project_lead_id"),
    },
    {
      token: member,
      body: { name: "Ops", identifier: "OPS", project_lead_id: strangerId },
      want: notAllowed("project_lead_id"),
    },
  ];
  const answers = await Promise.all(
    refusals.map(async ({ token, body }) => {
      const { error, response } = await api.POST("/api/v0/workspaces/{slug}/projects", {
        params: { path: { slug } },
        body,
        headers: bearer(token),
      });
      return {
        status: response.status,
        code: error?.code,
        errors: error?.errors?.map((e) => ({ field: e.field, code: e.code })),
      };
    })
  );
  expect(answers, "the refusals").toEqual(refusals.map((r) => r.want));
  expect(await countProjects(db)).toEqual(before);
  // No refusal changed the project created first.
  expect(await expectProjectCreated(db, slug, web, memberEmail, adminEmail, members)).toBe(created.id);

  // A new project goes first in the sidebar of each of its admins (M3 design 3.18), each sidebar by its account's own
  // places. Ops, which the admin creates, goes before Web in the admin's sidebar. Docs, which the member creates with
  // the admin its lead, goes before Web in the member's sidebar and before Ops in the admin's sidebar.
  const ops = await createProject(api, admin, slug, { name: "Ops", identifier: "ops", network: 0 });
  expect(ops).toMatchObject({
    identifier: "OPS",
    network: 0,
    member_role: 20,
    sort_order: 55535,
    member_ids: [adminId],
  });
  // Ops, given no lead, is led by no one.
  const opsRow = {
    name: "Ops",
    identifier: "OPS",
    description: "",
    network: 0,
    timezone: "Asia/Shanghai",
    logo_props: {},
  };
  expect(
    await expectProjectCreated(db, slug, opsRow, adminEmail, null, [{ email: adminEmail, sort_order: 55535 }])
  ).toBe(ops.id);
  const docs = await createProject(api, member, slug, { name: "Docs", identifier: "docs", project_lead_id: adminId });
  expect(docs).toMatchObject({
    identifier: "DOCS",
    member_role: 20,
    sort_order: 55535,
    member_ids: [memberId, adminId],
  });
  const docsRow = {
    name: "Docs",
    identifier: "DOCS",
    description: "",
    network: 2,
    timezone: "Asia/Shanghai",
    logo_props: {},
  };
  expect(
    await expectProjectCreated(db, slug, docsRow, memberEmail, adminEmail, [
      { email: memberEmail, sort_order: 55535 },
      { email: adminEmail, sort_order: 45535 },
    ])
  ).toBe(docs.id);

  // An older project reads as its reader sees it (M3 design 3.19): the admin's private Ops, between his Web and his
  // Docs in his sidebar; Docs, which the member created with the admin its lead; the member does not see Ops.
  expect(await read(api, admin, ops.id)).toMatchObject({
    status: 200,
    project: { identifier: "OPS", network: 0, member_role: 20, sort_order: 55535, member_ids: [adminId] },
  });
  expect(await read(api, admin, docs.id)).toMatchObject({
    status: 200,
    project: { identifier: "DOCS", member_role: 20, sort_order: 45535, member_ids: [memberId, adminId] },
  });
  expect(await read(api, member, ops.id)).toEqual({ status: 404, code: "project.not_found" });
});

test("P1 (page): a new project's icon picker shows the emoji nerve serves itself, nothing asked of another address, nothing blocked", async ({
  api,
  signedInPage,
}, testInfo) => {
  const member = await registerOnboarded(api, emailFor(testInfo, "member"));
  const slug = slugFor(testInfo);
  await createWorkspace(api, member.access_token, { name: "Acme", slug });
  const page = await signedInPage(member);
  const watch = await watchPage(page);

  await page.goto(`/${slug}/projects`);
  await page.getByRole("button", { name: "Add Project", exact: true }).click();
  const data = page.waitForResponse((answer) => new URL(answer.url()).pathname === `${EMOJIBASE}/data.json`);
  const messages = page.waitForResponse((answer) => new URL(answer.url()).pathname === `${EMOJIBASE}/messages.json`);
  await page.getByRole("button", { name: "Project icon" }).click();
  expect([(await data).status(), (await messages).status()]).toEqual([200, 200]);
  await page.getByRole("searchbox").fill("rocket");
  await page.getByRole("gridcell", { name: "Rocket" }).click();
  await expect(page.getByRole("button", { name: "Project icon" })).toContainText("🚀");

  // The picker asks nerve, which answers its HEAD too (frimousse compares the files' ETags before it reads its cache).
  expect((await page.request.head(`${EMOJIBASE}/data.json`)).status()).toBe(200);
  expect(requestsElsewhere(page, watch)).toEqual([]);
  expect([watch.cspViolations, watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], [], []]);
  await expectQuietConsole(page, watch, { warnings: [EMOJI_CHECK_WARNING] });
});
