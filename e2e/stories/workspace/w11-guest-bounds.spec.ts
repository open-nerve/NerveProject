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
import { workspaceTables } from "../../fixtures/assert/workspace";
import { accountId, bearer, emailFor, newAccount } from "../../fixtures/auth";
import { EMOJI_CHECK_WARNING, expectQuietConsole, watchPage } from "../../fixtures/browser";
import { registerOnboarded } from "../../fixtures/settings-pages";
import { expect, test } from "../../fixtures/test";

// W11, a guest's bounds (M3 design 2, 9.2): the permission matrix's backend
// test holds every cell; this story samples four of them through the API, and
// its page version what a guest's pages show him.

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
        ["workspaces", ...workspaceTables].map(async (table) => [
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

test("W11 (page): a guest signs in: his sidebar lists Web, the project he is a guest of, not Secret; the workspace's members page and Web's state settings tell him he may not see them; no page he opens shows an address, his own neither", async ({
  api,
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
    const html = await page.content();
    expect([html.includes(adminEmail), html.includes(gusEmail)], `the addresses on ${path}`).toEqual([false, false]);
  };
  await opened(`/${slug}`, "Web");
  const joined = page.getByRole("complementary", { name: "Main sidebar" }).locator('[id^="sidebar-"][id$="-JOINED"]');
  await expect(joined).toHaveText([/Web/]);
  const denied = "Oops! You are not authorized to view this page";
  await opened(`/${slug}/settings/members`, denied);
  await opened(`/${slug}/settings/projects/${web.id}/states`, denied);
  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], []]);
  await expectQuietConsole(page, watch, { warnings: Array(3).fill(EMOJI_CHECK_WARNING) });
});
