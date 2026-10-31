import {
  addProjectMembers,
  changeProject,
  changeRole,
  createProject,
  createWorkspace,
  inviteAndAccept,
  projectMembershipOf,
  projectMemberWrites,
  slugFor,
  readProjectPreferences,
} from "../../fixtures/api";
import { expectIssuePage } from "../../fixtures/assert/issue";
import { newAccount, accountId, emailFor } from "../../fixtures/auth";
import { commentPage, createIssueComment } from "../../fixtures/comments";
import { createIssue, issueCreated, issueRead, listIssues } from "../../fixtures/issues";
import {
  issueListProject,
  openIssueList,
  saveListPreferences,
  expectListPageQuiet,
} from "../../fixtures/issue-list-page";
import { answerTo, registerOnboarded } from "../../fixtures/settings-pages";
import { expect, test } from "../../fixtures/test";

test("I20 (API): a member creates his issue before becoming a guest; own-only hides another detail and list row, creation is forbidden but his own comment is accepted", async ({
  api,
  db,
}, info) => {
  const admin = await newAccount(api, info, "admin");
  const guest = await newAccount(api, info, "guest");
  const slug = slugFor(info);
  await createWorkspace(api, admin.token, { name: "Acme", slug });
  await inviteAndAccept(api, admin.token, slug, guest, 15);
  const project = await createProject(api, admin.token, slug, { name: "Web", identifier: "WEB" });
  expect((await changeProject(api, admin.token, project.id, { guest_view_all_features: false })).status).toBe(200);
  await addProjectMembers(api, admin.token, project.id, [{ member_id: guest.id, role: 15 }]);
  const own = await createIssue(api, guest.token, project.id, { name: "My issue" });
  const other = await createIssue(api, admin.token, project.id, { name: "Another issue" });
  expect(
    await projectMemberWrites(api, project.id).change(
      admin.token,
      await projectMembershipOf(api, admin.token, project.id, guest.id),
      5
    )
  ).toEqual({ status: 200 });
  await changeRole(api, admin.token, slug, guest.id, 5);
  expect(await issueRead(api, guest.token, other.id)).toEqual({ status: 404, code: "issue.not_found" });
  expect(await issueCreated(api, guest.token, project.id, { name: "Forbidden" })).toEqual({
    status: 403,
    code: "forbidden",
  });
  expectIssuePage(await listIssues(api, guest.token, project.id), [own.id]);
  const comment = await createIssueComment(api, guest.token, own.id, "<p>My guest comment</p>");
  expect(comment).toMatchObject({ actor_id: guest.id, issue_id: own.id, comment_html: "<p>My guest comment</p>" });
  expect((await commentPage(api, guest.token, own.id)).data).toEqual([comment]);
  expect(
    await db.query(
      "SELECT actor_id::text, issue_id::text, comment_html FROM issue_comments WHERE id=$1 AND deleted_at IS NULL",
      [comment.id]
    )
  ).toEqual([{ actor_id: guest.id, issue_id: own.id, comment_html: "<p>My guest comment</p>" }]);
  expect((await changeProject(api, admin.token, project.id, { guest_view_all_features: true })).status).toBe(200);
  expectIssuePage(await listIssues(api, guest.token, project.id), [other.id, own.id]);
  expect(await issueRead(api, guest.token, other.id)).toMatchObject({ status: 200, issue: { id: other.id } });
  expect(await issueCreated(api, guest.token, project.id, { name: "Still forbidden" })).toEqual({
    status: 403,
    code: "forbidden",
  });
});

for (const workspaceAdmin of [false, true]) {
  test(`I20 (page): project guest with workspace admin=${workspaceAdmin} follows own-only and the returned grant, with hidden and absent details indistinguishable`, async ({
    api,
    db,
    page,
    browserName,
  }, info) => {
    const seed = await issueListProject(api, info);
    const email = emailFor(info, "page-guest");
    const tokens = await registerOnboarded(api, email);
    const guest = { email, token: tokens.access_token, id: await accountId(api, tokens.access_token) };
    await inviteAndAccept(api, seed.member.token, seed.slug, guest, 15);
    await addProjectMembers(api, seed.member.token, seed.project.id, [{ member_id: guest.id, role: 15 }]);
    const own = await createIssue(api, guest.token, seed.project.id, {
      name: "Guest-owned detail",
      description_html: '<p class="editor-paragraph-block" data-id="own-description">Own description</p>',
    });
    const other = await createIssue(api, seed.member.token, seed.project.id, { name: "Hidden member detail" });
    expect(
      (await changeProject(api, seed.member.token, seed.project.id, { guest_view_all_features: false })).status
    ).toBe(200);
    expect(
      await projectMemberWrites(api, seed.project.id).change(
        seed.member.token,
        await projectMembershipOf(api, seed.member.token, seed.project.id, guest.id),
        5
      )
    ).toEqual({ status: 200 });
    // A workspace admin can only be added to a project as an admin. Promote after the project's guest role is set,
    // as the real workspace membership operation preserves an existing project's role on promotion.
    await changeRole(api, seed.member.token, seed.slug, guest.id, workspaceAdmin ? 20 : 5);
    const guestSeed = { ...seed, member: guest };
    const preferences = await readProjectPreferences(api, guest.token, seed.project.id);
    await saveListPreferences(api, guestSeed, {
      filters: {},
      display_filters: { ...preferences.display_filters, layout: "list" },
    });
    const listRead = page.waitForResponse(
      (response) => new URL(response.url()).pathname === `/api/v0/projects/${seed.project.id}/issues`
    );
    const { watch } = await openIssueList(page, guestSeed);
    expect(await (await listRead).json()).toMatchObject({ data: [{ id: own.id }], total: 1 });
    await expect(page.locator(`#issue-${own.id}`)).toBeVisible();
    await expect(page.locator(`#issue-${other.id}`)).toHaveCount(0);
    if (!workspaceAdmin) await expect(page.getByRole("button", { name: "Add work item", exact: true })).toHaveCount(0);
    await page.goto(`/${seed.slug}/browse/WEB-${own.sequence_id}`);
    await expect(page.getByText("Own description", { exact: true })).toBeVisible();
    if (workspaceAdmin)
      await expect(page.getByRole("textbox", { name: "Work item title", exact: true })).toHaveValue(own.name);
    else {
      await expect(page.getByText(own.name, { exact: true })).toBeVisible();
      await expect(page.getByRole("textbox", { name: "Work item title", exact: true })).toHaveCount(0);
      await page.getByRole("button", { name: `Options: ${own.name}`, exact: true }).click();
      await expect(page.getByRole("menuitem", { name: "Delete", exact: true })).toHaveCount(0);
      await page.keyboard.press("Escape");
    }
    const rejected: { body: unknown; contentType: string | undefined; alert: string | null }[] = [];
    const failures: string[] = [];
    await [other.sequence_id, 999999].reduce(async (previous, sequence) => {
      await previous;
      const path = `/api/v0/workspaces/${seed.slug}/issues/WEB-${sequence}`;
      const response = await answerTo(page, "GET", path, () =>
        page.goto(`/${seed.slug}/browse/WEB-${sequence}`).then(() => undefined)
      );
      expect(response.status()).toBe(404);
      const alert = page.getByRole("alert");
      await expect(alert).toContainText("The work item does not exist, or you cannot see it.");
      await expect(page.getByText(other.name, { exact: true })).toHaveCount(0);
      rejected.push({
        body: await response.json(),
        contentType: response.headers()["content-type"],
        alert: await alert.textContent(),
      });
      failures.push(`404 GET ${path}`);
    }, Promise.resolve());
    expect(rejected[0]).toEqual(rejected[1]);
    expect(
      (await changeProject(api, seed.member.token, seed.project.id, { guest_view_all_features: true })).status
    ).toBe(200);
    await page.goto(`/${seed.slug}/browse/WEB-${other.sequence_id}`);
    if (workspaceAdmin)
      await expect(page.getByRole("textbox", { name: "Work item title", exact: true })).toHaveValue(other.name);
    else {
      await expect(page.getByText(other.name, { exact: true })).toBeVisible();
      await expect(page.getByRole("textbox", { name: "Work item title", exact: true })).toHaveCount(0);
    }
    expect(await db.query("SELECT name, updated_by_id::text FROM issues WHERE id=$1", [own.id])).toEqual([
      { name: own.name, updated_by_id: guest.id },
    ]);
    expect(
      watch.apiRequests.filter((request) =>
        /^(POST|PATCH|DELETE) \/api\/v0\/(issues|projects\/[^/]+\/issues)/.test(request)
      )
    ).toEqual([]);
    await expectListPageQuiet(page, watch, {
      failures,
      errors:
        browserName === "chromium" || browserName === "webkit"
          ? failures.map(() => "Failed to load resource: the server responded with a status of 404 (Not Found)")
          : [],
    });
  });
}
