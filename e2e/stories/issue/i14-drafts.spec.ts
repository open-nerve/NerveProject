import { c12Evidence } from "../../fixtures/c12-evidence";
import type { Page, TestInfo } from "@playwright/test";
import { randomUUID } from "node:crypto";

import {
  addProjectMembers,
  archiveProject,
  createProject,
  createWorkspace,
  inviteAndAccept,
  projectMemberWrites,
  slugFor,
  type Api,
} from "../../fixtures/api";
import { createdFor, expectIssueCreated } from "../../fixtures/assert/issue";
import { expectInitialDescription } from "../../fixtures/assert/issue-collaboration";
import { bearer, emailFor, newAccount, password } from "../../fixtures/auth";
import { signInPath, submitSignIn } from "../../fixtures/auth-pages";
import { watchPage } from "../../fixtures/browser";
import { focusInDialog, modalDialog, pageBehindHidden } from "../../fixtures/dialogs";
import { expectListPageQuiet, issueListProject } from "../../fixtures/issue-list-page";
import { answerTo, registerOnboarded, sentHeld } from "../../fixtures/settings-pages";
import {
  createDraft,
  createIssue,
  draftDeleted,
  draftPublished,
  draftRead,
  draftsListed,
  draftUpdated,
  issueCreated,
  issueRead,
  type IssueCreate,
} from "../../fixtures/issues";
import { createState } from "../../fixtures/states";
import { expect, test } from "../../fixtures/test";

// I14 uses only HTTP for its preconditions. Database reads check exact publication audit stamps.
async function draftWorld(api: Api, testInfo: TestInfo) {
  const admin = await newAccount(api, testInfo, "admin");
  const member = await newAccount(api, testInfo, "member");
  const other = await newAccount(api, testInfo, "other");
  const guest = await newAccount(api, testInfo, "guest");
  const slug = slugFor(testInfo);
  await createWorkspace(api, admin.token, { name: "Acme", slug });
  await Promise.all([member, other, guest].map((who) => inviteAndAccept(api, admin.token, slug, who, 15)));
  const project = await createProject(api, admin.token, slug, { name: "Web", identifier: "WEB", network: 0 });
  await addProjectMembers(api, admin.token, project.id, [
    { member_id: member.id, role: 15 },
    { member_id: other.id, role: 15 },
    { member_id: guest.id, role: 5 },
  ]);
  return { admin, member, other, guest, slug, project };
}

test("I14 (API): a member saves missing-name input, receives create's 400 on publish, replaces the payload and publishes its description and parent at one moment; the consumed draft disappears and cannot publish twice", async ({
  api,
  db,
}, testInfo) => {
  const { member, slug, project } = await draftWorld(api, testInfo);
  // Retaining this invalid priority after PATCH would expose an incorrect payload merge.
  const incomplete = { description_html: "<p>Unfinished</p>", priority: "not-a-priority" };
  const draft = await createDraft(api, member.token, slug, { project_id: project.id, payload: incomplete });
  expect(draft).toMatchObject({ project_id: project.id, payload: incomplete });
  expect(await draftRead(api, member.token, draft.id)).toEqual({ status: 200, draft });
  const refused = await draftPublished(api, member.token, draft.id);
  expect(refused).toMatchObject({ status: 400, code: "bad_request" });
  // Only this deliberately invalid call narrows around IssueCreate's required name.
  expect(refused).toEqual(await issueCreated(api, member.token, project.id, incomplete as IssueCreate));
  expect(await draftRead(api, member.token, draft.id)).toEqual({ status: 200, draft });

  const parent = await createIssue(api, member.token, project.id, { name: "Release" });
  const description = "<p>Ship the saved plan</p>";
  const payload = { name: "Ship it", description_html: description, parent_id: parent.id };
  expect(await draftUpdated(api, member.token, draft.id, { payload })).toMatchObject({
    status: 200,
    draft: { id: draft.id, project_id: project.id, payload },
  });
  const saved = await draftRead(api, member.token, draft.id);
  expect(saved).toMatchObject({ status: 200 });
  if (!("draft" in saved)) throw new Error("the patched draft is missing");
  expect(saved.draft.payload).toEqual(payload);

  const published = await draftPublished(api, member.token, draft.id);
  expect(published.status, JSON.stringify(published)).toBe(201);
  if (!("issue" in published)) throw new Error("publish answered without the issue");
  const issue = published.issue;
  expect(issue).toMatchObject({
    name: "Ship it",
    description_html: description,
    project_id: project.id,
    parent_id: parent.id,
    sub_issues_count: 0,
  });
  expect(await issueRead(api, member.token, parent.id)).toMatchObject({
    status: 200,
    issue: { sub_issues_count: 1 },
  });
  await expectIssueCreated(db, issue.id, {
    sequence_id: 2,
    name: "Ship it",
    state: "Backlog",
    priority: "none",
    start_date: null,
    target_date: null,
    sort_order: 75535,
    parent: "Release",
    completed: false,
    by: member.email,
    assignees: [],
    labels: [],
    subscribers: [member.email],
    activity: createdFor(),
  });
  await expectInitialDescription(db, issue.id, description, []);
  expect(await draftsListed(api, member.token, slug)).toEqual({
    status: 200,
    page: { data: [], next_cursor: null, total: 0 },
  });
  const missing = { status: 404, code: "issue.draft_not_found" };
  expect(await draftRead(api, member.token, draft.id)).toEqual(missing);
  expect(await draftPublished(api, member.token, draft.id)).toEqual(missing);
  // PostgreSQL compares microseconds directly; JavaScript Date would truncate them.
  expect(
    await db.query(
      `SELECT d.deleted_at = i.created_at AND d.updated_at = i.created_at AS same_moment,
              d.updated_by_id = i.created_by_id AND i.updated_by_id = i.created_by_id
                AND i.created_by_id = $3::uuid AS same_actor
         FROM draft_issues d JOIN issues i ON i.id = $2 WHERE d.id = $1`,
      [draft.id, issue.id, member.id]
    )
  ).toEqual([{ same_moment: true, same_actor: true }]);
  expect(await db.query(`SELECT count(*)::int AS count FROM issues WHERE project_id = $1`, [project.id])).toEqual([
    { count: 2 },
  ]);
});

test("I14 (API): a deleted saved state is revalidated (422), another member or admin sees no personal draft (404), a project guest cannot publish (403), and discarding creates no issue", async ({
  api,
  db,
}, testInfo) => {
  const { admin, member, other, guest, slug, project } = await draftWorld(api, testInfo);
  const state = await createState(api, admin.token, project.id, { name: "Review", color: "#0EA5E9", group: "started" });
  const draft = await createDraft(api, member.token, slug, {
    project_id: project.id,
    payload: { name: "Review release", state_id: state.id },
  });
  await Promise.all(
    [other, admin].map(async (who) => {
      const missing = { status: 404, code: "issue.draft_not_found" };
      expect(await draftRead(api, who.token, draft.id)).toEqual(missing);
      expect(await draftUpdated(api, who.token, draft.id, { payload: {} })).toEqual(missing);
      expect(await draftDeleted(api, who.token, draft.id)).toEqual(missing);
      expect(await draftPublished(api, who.token, draft.id)).toEqual(missing);
      expect(await draftsListed(api, who.token, slug)).toMatchObject({ status: 200, page: { data: [], total: 0 } });
    })
  );
  const removed = await api.DELETE("/api/v0/states/{state_id}", {
    params: { path: { state_id: state.id } },
    headers: bearer(admin.token),
  });
  expect(removed.response.status).toBe(204);
  expect(await draftPublished(api, member.token, draft.id)).toEqual({
    status: 422,
    code: "validation_failed",
    errors: [{ field: "state_id", code: "not_allowed" }],
  });
  expect(await draftRead(api, member.token, draft.id)).toEqual({ status: 200, draft });
  const guestDraft = await createDraft(api, guest.token, slug, { project_id: project.id, payload: { name: "Guest" } });
  expect(await draftPublished(api, guest.token, guestDraft.id)).toEqual({ status: 403, code: "forbidden" });
  expect(await draftRead(api, guest.token, guestDraft.id)).toEqual({ status: 200, draft: guestDraft });
  expect(await draftDeleted(api, member.token, draft.id)).toEqual({ status: 204 });
  expect(await draftRead(api, member.token, draft.id)).toEqual({ status: 404, code: "issue.draft_not_found" });
  expect(await draftsListed(api, member.token, slug)).toMatchObject({ status: 200, page: { data: [], total: 0 } });
  expect(await db.query(`SELECT count(*)::int AS count FROM issues WHERE project_id = $1`, [project.id])).toEqual([
    { count: 0 },
  ]);
});

test("I14 (API): archived and left private projects keep drafts in the personal list while publish applies current permissions before archive validation", async ({
  api,
}, testInfo) => {
  const { admin, member, guest, slug, project } = await draftWorld(api, testInfo);
  const draft = await createDraft(api, member.token, slug, { project_id: project.id, payload: { name: "Keep me" } });
  const guestDraft = await createDraft(api, guest.token, slug, { project_id: project.id, payload: { name: "Guest" } });
  await archiveProject(api, admin.token, project.id);
  expect(await draftsListed(api, member.token, slug)).toEqual({
    status: 200,
    page: { data: [draft], next_cursor: null, total: 1 },
  });
  expect(await draftPublished(api, member.token, draft.id)).toEqual({ status: 409, code: "project.archived" });
  expect(await draftPublished(api, guest.token, guestDraft.id)).toEqual({ status: 403, code: "forbidden" });
  // Leaving is allowed even after archive; the workspace membership still owns the draft.
  expect((await projectMemberWrites(api, project.id).leave(member.token)).status).toBe(204);
  expect(await draftsListed(api, member.token, slug)).toEqual({
    status: 200,
    page: { data: [draft], next_cursor: null, total: 1 },
  });
  expect(await draftRead(api, member.token, draft.id)).toEqual({ status: 200, draft });
  expect(await draftPublished(api, member.token, draft.id)).toEqual({ status: 404, code: "project.not_found" });
});

async function openDrafts(page: Page, slug: string, email: string) {
  const watch = await watchPage(page);
  await page.goto(signInPath(`/${slug}/drafts`));
  expect(await submitSignIn(page, email, password)).toBe(200);
  await expect(page).toHaveURL(`/${slug}/drafts`);
  await expect(page.getByRole("button", { name: "Draft a work item", exact: true })).toBeVisible();
  return watch;
}

test("I14 (page): personal unassigned drafts paginate with server totals, preserve unresolved fields through real editing, and delete once from a keyboard or pointer menu", async ({
  api,
  page,
  db,
}, info) => {
  const email = emailFor(info, "unassigned-drafts");
  const { access_token: token } = await registerOnboarded(api, email);
  const slug = slugFor(info);
  await createWorkspace(api, token, { name: "No projects", slug });
  await Promise.all(
    Array.from({ length: 50 }, (_, index) => createDraft(api, token, slug, { payload: { name: `Saved ${index}` } }))
  );
  const payload = {
    name: "Unassigned original",
    description_html: "<p>Keep unfinished content</p>",
    priority: "future-invalid-priority",
    state_id: randomUUID(),
    parent_id: randomUUID(),
    assignee_ids: [randomUUID()],
    label_ids: [randomUUID()],
    start_date: "2026-10-01",
    target_date: "2026-10-30",
  };
  const draft = await createDraft(api, token, slug, { payload });
  const watch = await openDrafts(page, slug, email);
  await expect(page.locator('[id^="issue-"]')).toHaveCount(50);
  await expect(page.getByText("51", { exact: true })).toBeVisible();
  await page.getByRole("button", { name: "Load more", exact: false }).click();
  await expect(page.locator('[id^="issue-"]')).toHaveCount(51);
  await c12Evidence(page, "B18", "unassigned-drafts-51-rows");
  await expect(page.getByRole("button", { name: "Load more", exact: false })).toHaveCount(0);
  const row = page.locator(`[id="issue-${draft.id}"]`);
  const trigger = row.locator('button[aria-haspopup="menu"]:visible').last();
  await expect(trigger).toHaveAccessibleName("Unassigned original options");
  await trigger.focus();
  await trigger.click({ button: "right" });
  await expect(page.getByRole("menu")).toHaveCount(1);
  await page.keyboard.press("Escape");
  await expect(trigger).toBeFocused();
  await expect(page.getByRole("menu")).toHaveCount(0);
  expect(
    await page
      .locator('[data-context-menu="true"]')
      .evaluateAll((menus) => menus.every((menu) => !!menu.closest('[inert][aria-hidden="true"]')))
  ).toBe(true);
  await trigger.press("Shift+F10");
  await page.getByRole("menuitem", { name: "Edit", exact: true }).press("Enter");
  const modal = modalDialog(page);
  await expect(modal.getByRole("textbox", { name: "Title", exact: true })).toHaveValue(payload.name);
  await c12Evidence(page, "B18", "unassigned-draft-edit");
  expect(await pageBehindHidden(page)).toBe(true);
  await expect(modal.locator('[contenteditable="true"]')).toContainText("Keep unfinished content");
  await modal.getByRole("textbox", { name: "Title", exact: true }).fill("Unassigned edited");
  const save = await sentHeld(page, "PATCH", `/api/v0/drafts/${draft.id}`, () =>
    modal.getByRole("button", { name: "Update", exact: true }).dblclick()
  );
  expect(save.body).toMatchObject({
    project_id: null,
    payload: {
      ...payload,
      name: "Unassigned edited",
      description_html: expect.stringContaining("Keep unfinished content"),
    },
  });
  await page.keyboard.press("Escape");
  await expect(modal).toHaveCount(1);
  expect(await focusInDialog(page)).toBe(true);
  expect((await save.release()).status()).toBe(200);
  await expect(modal).toHaveCount(0);
  await row.getByText("Unassigned edited", { exact: true }).dblclick();
  await expect(modal.getByRole("textbox", { name: "Title", exact: true })).toHaveValue("Unassigned edited");
  await expect(modal.locator('[contenteditable="true"]')).toContainText("Keep unfinished content");
  await page.keyboard.press("Escape");
  await expect(modal).toHaveCount(0);
  await trigger.press("Enter");
  await page.getByRole("menuitem", { name: "Delete", exact: true }).press("Enter");
  await expect(modal.getByRole("heading", { name: "Delete draft", exact: true })).toBeVisible();
  await c12Evidence(page, "B18", "draft-delete-confirmation");
  const deletion = await sentHeld(page, "DELETE", `/api/v0/drafts/${draft.id}`, () =>
    modal.getByRole("button", { name: "Delete", exact: true }).dblclick()
  );
  await page.keyboard.press("Escape");
  await expect(modal).toHaveCount(1);
  expect((await deletion.release()).status()).toBe(204);
  await expect(row).toHaveCount(0);
  await expect(page.getByText("50", { exact: true })).toBeVisible();
  expect(watch.apiRequests.filter((request) => request === `PATCH /api/v0/drafts/${draft.id}`)).toHaveLength(1);
  expect(watch.apiRequests.filter((request) => request === `DELETE /api/v0/drafts/${draft.id}`)).toHaveLength(1);
  expect(watch.apiRequests.some((request) => /issue.*preferences|display-properties/.test(request))).toBe(false);
  expect(
    await db.query("SELECT payload, project_id, deleted_at IS NOT NULL AS deleted FROM draft_issues WHERE id=$1", [
      draft.id,
    ])
  ).toEqual([
    {
      payload: {
        ...payload,
        name: "Unassigned edited",
        description_html: expect.stringContaining("Keep unfinished content"),
      },
      project_id: null,
      deleted: true,
    },
  ]);
  await expectListPageQuiet(page, watch);
});

test("I14 (page): publish saves latest form first, retains it on refusal, and confirmed publication retries only its failed list read", async ({
  api,
  page,
  db,
  browserName,
}, info) => {
  const seed = await issueListProject(api, info);
  const draft = await createDraft(api, seed.member.token, seed.slug, {
    project_id: seed.project.id,
    payload: { name: "Publish original", description_html: "<p>Published content</p>" },
  });
  const path = `/api/v0/drafts/${draft.id}`;
  const publishPath = `${path}/publish`;
  const listPath = `/api/v0/workspaces/${seed.slug}/drafts`;
  const watch = await openDrafts(page, seed.slug, seed.member.email);
  const row = page.locator(`[id="issue-${draft.id}"]`);
  await row.locator('button[aria-haspopup="menu"]:visible').last().press("Enter");
  await page.getByRole("menuitem", { name: "Move to project", exact: true }).press("Enter");
  const modal = modalDialog(page);
  await expect(modal.locator('[contenteditable="true"]')).toContainText("Published content");
  await modal.getByRole("textbox", { name: "Title", exact: true }).fill("Latest published");
  let refusePublish = true;
  await page.route(
    (url) => url.pathname === publishPath,
    async (route) => {
      if (!refusePublish) return route.fallback();
      refusePublish = false;
      await route.fulfill({
        status: 409,
        contentType: "application/problem+json",
        body: JSON.stringify({ status: 409, title: "Conflict", code: "issue.draft_changed" }),
      });
    }
  );
  const publish = modal.getByRole("button", { name: "Add to project", exact: true });
  const refused = page.waitForResponse((response) => new URL(response.url()).pathname === publishPath);
  const saved = await sentHeld(page, "PATCH", path, () => publish.dblclick());
  expect(saved.body).toMatchObject({ project_id: seed.project.id, payload: { name: "Latest published" } });
  expect(watch.apiRequests.filter((request) => request === `POST ${publishPath}`)).toHaveLength(0);
  expect((await saved.release()).status()).toBe(200);
  expect((await refused).status()).toBe(409);
  await expect(publish).toBeEnabled();
  await expect(modal.getByRole("textbox", { name: "Title", exact: true })).toHaveValue("Latest published");
  await c12Evidence(page, "B18", "publication-conflict-keeps-input");
  expect(await draftRead(api, seed.member.token, draft.id)).toMatchObject({
    status: 200,
    draft: { payload: { name: "Latest published" } },
  });
  let failList = true;
  await page.route(
    (url) => url.pathname === listPath,
    async (route) => {
      if (route.request().method() !== "GET" || !failList) return route.fallback();
      failList = false;
      await route.fulfill({
        status: 500,
        contentType: "application/problem+json",
        body: JSON.stringify({ status: 500, title: "Unavailable", code: "unavailable" }),
      });
    }
  );
  const sent = await sentHeld(page, "POST", publishPath, () => publish.dblclick());
  expect(sent.body).toBeNull();
  await page.keyboard.press("Escape");
  await expect(modal).toHaveCount(1);
  const confirmation = await sent.release();
  expect(confirmation.status()).toBe(201);
  const published = await confirmation.json();
  await expect(modal).toHaveCount(0);
  await expect(row).toHaveCount(0);
  await expect(page.getByRole("alert")).toBeVisible();
  await c12Evidence(page, "B18", "published-list-get-refusal");
  expect(
    (
      await answerTo(page, "GET", listPath, () =>
        page.getByRole("alert").getByRole("button", { name: "Retry", exact: true }).click()
      )
    ).status()
  ).toBe(200);
  await expect(page.getByRole("alert")).toHaveCount(0);
  await expect(page.getByText("0", { exact: true })).toBeVisible();
  expect(watch.apiRequests.filter((request) => request === `POST ${publishPath}`)).toHaveLength(2);
  expect(watch.apiRequests.filter((request) => request === `PATCH ${path}`)).toHaveLength(2);
  expect(await db.query("SELECT name, project_id FROM issues WHERE id=$1", [published.id])).toEqual([
    { name: "Latest published", project_id: seed.project.id },
  ]);
  expect(await db.query("SELECT count(*)::int AS count FROM issues WHERE project_id=$1", [seed.project.id])).toEqual([
    { count: 1 },
  ]);
  await expectListPageQuiet(page, watch, {
    failures: [`409 POST ${publishPath}`, `500 GET ${listPath}`],
    errors:
      browserName === "chromium" || browserName === "webkit"
        ? [
            "Failed to load resource: the server responded with a status of 409 (Conflict)",
            "Failed to load resource: the server responded with a status of 500 (Internal Server Error)",
          ]
        : [],
  });
});

test("I14 (page): a workspace guest opens the draft route without personal reads or creation controls", async ({
  api,
  page,
}, info) => {
  const seed = await issueListProject(api, info);
  const email = emailFor(info, "draft-guest");
  const { access_token: token } = await registerOnboarded(api, email);
  await inviteAndAccept(api, seed.member.token, seed.slug, { email, token }, 5);
  await createDraft(api, seed.member.token, seed.slug, { payload: { name: "Other personal secret" } });
  const watch = await openDrafts(page, seed.slug, email);
  await expect(page.getByRole("button", { name: "Draft a work item", exact: true })).toBeDisabled();
  await c12Evidence(page, "B18", "guest-draft-route");
  await expect(page.getByText("Other personal secret", { exact: true })).toHaveCount(0);
  expect(watch.apiRequests.filter((request) => request.includes("/drafts"))).toEqual([]);
  await expectListPageQuiet(page, watch);
});

test("I14 (page): making a copy preserves unresolved draft fields through the actual editor and leaves the original unchanged", async ({
  api,
  page,
  db,
}, info) => {
  const email = emailFor(info, "copy-draft");
  const { access_token: token } = await registerOnboarded(api, email);
  const slug = slugFor(info);
  await createWorkspace(api, token, { name: "Unassigned copies", slug });
  const payload = {
    name: "Original copy source",
    description_html: "<p>Preserve unfinished content</p>",
    priority: "future-invalid-priority",
    state_id: randomUUID(),
    parent_id: randomUUID(),
    assignee_ids: [randomUUID()],
    label_ids: [randomUUID()],
    start_date: "2026-10-01",
    target_date: "2026-10-30",
  };
  const original = await createDraft(api, token, slug, { payload });
  const watch = await openDrafts(page, slug, email);
  await page.locator(`[id="issue-${original.id}"] button[aria-haspopup="menu"]:visible`).last().click();
  await page.getByRole("menuitem", { name: "Make a copy", exact: true }).click();
  const modal = modalDialog(page);
  await expect(modal.locator('[contenteditable="true"]')).toContainText("Preserve unfinished content");
  await modal.getByRole("textbox", { name: "Title", exact: true }).fill("Copied draft");
  await c12Evidence(page, "B18", "unassigned-copy-preserves-content");
  const saved = await sentHeld(page, "POST", `/api/v0/workspaces/${slug}/drafts`, () =>
    modal.getByRole("button", { name: "Save to Drafts", exact: true }).dblclick()
  );
  expect(saved.body).toEqual({ project_id: null, payload: { ...payload, name: "Copied draft" } });
  const response = await saved.release();
  expect(response.status()).toBe(201);
  const copy = await response.json();
  await expect(modal).toHaveCount(0);
  await expect(page.locator(`[id="issue-${copy.id}"]`)).toContainText("Copied draft");
  expect(watch.apiRequests.filter((request) => request === `POST /api/v0/workspaces/${slug}/drafts`)).toHaveLength(1);
  expect(
    await db.query("SELECT id, payload, project_id FROM draft_issues WHERE workspace_id=$1 ORDER BY id", [
      original.workspace_id,
    ])
  ).toEqual(
    [
      { id: original.id, payload, project_id: null },
      { id: copy.id, payload: { ...payload, name: "Copied draft" }, project_id: null },
    ].toSorted((a, b) => a.id.localeCompare(b.id))
  );
  await expectListPageQuiet(page, watch);
});
