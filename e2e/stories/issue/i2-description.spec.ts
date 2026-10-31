import { addProjectMembers, createProject, createWorkspace, inviteAndAccept, slugFor } from "../../fixtures/api";
import { expectDescriptionActivity } from "../../fixtures/assert/issue-collaboration";
import { newAccount } from "../../fixtures/auth";
import {
  activityPage,
  createIssue,
  descriptionPage,
  descriptionVersion,
  updateIssue,
  type IssueUpdate,
} from "../../fixtures/issues";
import { instantKey } from "../../fixtures/instant";
import { expect, test } from "../../fixtures/test";
import type { Database } from "../../fixtures/db";
import { openIssueDetail } from "../../fixtures/issue-detail-page";
import { expectListPageQuiet, issueListProject } from "../../fixtures/issue-list-page";
import { answerTo, bodiesSentTo } from "../../fixtures/settings-pages";

/** The description's audit identity, read without losing the database's sub-millisecond precision. */
async function descriptionAuditRows(db: Database, id: string) {
  return Promise.all([
    db.query("SELECT description_stripped, updated_at::text, updated_by_id FROM issues WHERE id=$1", [id]),
    db.query(
      "SELECT id, description_html, last_saved_at::text, updated_at::text FROM issue_description_versions WHERE issue_id=$1 AND deleted_at IS NULL ORDER BY id",
      [id]
    ),
    db.query(
      "SELECT id, field, old_value, new_value, created_at::text FROM issue_activities WHERE issue_id=$1 AND deleted_at IS NULL ORDER BY id",
      [id]
    ),
  ]);
}

test("I2 (page): real description editing and undo-redo keep the visible value, a refusal retains it for explicit retry, and pure block-id migration preserves audit stamps", async ({
  api,
  db,
  page,
  browserName,
}, info) => {
  const seed = await issueListProject(api, info);
  const issue = await createIssue(api, seed.member.token, seed.project.id, {
    name: "Description lifecycle",
    description_html: "<p>First</p>",
  });
  const beforeMigration = await descriptionAuditRows(db, issue.id);
  const path = `/api/v0/issues/${issue.id}`;
  let refuse = false;
  await page.route(
    (url) => url.pathname === path,
    async (route) => {
      if (route.request().method() !== "PATCH" || !refuse) return route.fallback();
      refuse = false;
      await route.fulfill({
        status: 503,
        contentType: "application/problem+json",
        body: JSON.stringify({ type: "about:blank", title: "Service Unavailable", status: 503, code: "server_busy" }),
      });
    }
  );
  const bodies = await bodiesSentTo(page, "PATCH", path);
  const migrationAnswer = page.waitForResponse(
    (response) => response.request().method() === "PATCH" && new URL(response.url()).pathname === path
  );
  const watch = await openIssueDetail(page, seed, issue);
  const editor = page.locator(`[id="editor-container-${issue.id}"] [contenteditable="true"]`);
  await expect(editor).toHaveText("First");
  expect((await migrationAnswer).status()).toBe(200);
  expect(bodies).toHaveLength(1);
  const migration = bodies[0] as IssueUpdate;
  expect(Object.keys(migration).toSorted()).toEqual(["description_html", "skip_activity"]);
  expect(migration.skip_activity).toBe(true);
  expect(migration.description_html).toMatch(/^<p class="editor-paragraph-block" data-id="[^"]+">First<\/p>$/);
  expect(await db.query("SELECT description_html FROM issues WHERE id=$1", [issue.id])).toEqual([
    { description_html: migration.description_html },
  ]);
  expect(await descriptionAuditRows(db, issue.id)).toEqual(beforeMigration);

  expect(
    (
      await answerTo(page, "PATCH", path, async () => {
        await editor.click();
        await editor.press("End");
        await page.keyboard.insertText(" edited");
      })
    ).status()
  ).toBe(200);
  const first = bodies[1] as IssueUpdate;
  expect(Object.keys(first)).toEqual(["description_html"]);
  expect(first.description_html).toContain(">First edited</p>");
  await expect(editor).toHaveText("First edited");
  const beforeUndo = await descriptionAuditRows(db, issue.id);
  // Both real keyboard transactions happen inside the existing 1.5-second save window.
  const began = Date.now();
  await editor.press("ControlOrMeta+z");
  await expect(editor).toHaveText("First");
  await editor.press("ControlOrMeta+Shift+z");
  await expect(editor).toHaveText("First edited");
  expect(Date.now() - began).toBeLessThan(1500);
  await expect(
    page.waitForRequest((request) => request.method() === "PATCH" && new URL(request.url()).pathname === path, {
      timeout: 1800,
    })
  ).rejects.toThrow(/Timeout.*exceeded/);
  expect(bodies).toHaveLength(2);
  expect(await descriptionAuditRows(db, issue.id)).toEqual(beforeUndo);

  refuse = true;
  expect(
    (
      await answerTo(page, "PATCH", path, async () => {
        await editor.press("ControlOrMeta+A");
        await page.keyboard.insertText("Second");
        await page.keyboard.insertText(" final");
      })
    ).status()
  ).toBe(503);
  await expect(editor).toHaveText("Second final");
  await expect(page.getByRole("alert")).toContainText("The server is busy. Please try again later.");
  expect(await descriptionAuditRows(db, issue.id)).toEqual(beforeUndo);
  expect(
    (
      await answerTo(page, "PATCH", path, () => page.getByRole("button", { name: "Retry", exact: true }).click())
    ).status()
  ).toBe(200);
  await expect(page.getByRole("button", { name: "Retry", exact: true })).toHaveCount(0);
  expect(bodies).toHaveLength(4);
  const last = bodies[3] as IssueUpdate;
  expect(bodies[2]).toEqual(last);
  expect(Object.keys(last)).toEqual(["description_html"]);
  expect(last.description_html).toContain(">Second final</p>");
  await expectDescriptionActivity(db, issue.id, migration.description_html!, last.description_html!);
  expect(
    await db.query(
      `SELECT i.description_html, i.description_stripped, i.updated_at > i.created_at AS changed,
       (SELECT count(*)::int FROM issue_description_versions v WHERE v.issue_id=i.id AND v.deleted_at IS NULL) AS versions,
       EXISTS (SELECT 1 FROM issue_description_versions v WHERE v.issue_id=i.id AND v.deleted_at IS NULL
         AND v.description_html=i.description_html AND v.last_saved_at=i.updated_at) AS version_matches
     FROM issues i WHERE i.id=$1`,
      [issue.id]
    )
  ).toEqual([
    {
      description_html: last.description_html,
      description_stripped: "Second final",
      changed: true,
      versions: 1,
      version_matches: true,
    },
  ]);
  await expectListPageQuiet(page, watch, {
    failures: [`503 PATCH ${path}`],
    errors:
      browserName === "chromium"
        ? ["Failed to load resource: the server responded with a status of 503 (Service Unavailable)"]
        : [],
  });
});

// These description cases extend I2 without adding a second activity expectation shape.
test("I2 (API): ordinary title/description writes audit in order, repeated saves merge and skip block-id saves preserve audit/version stamps", async ({
  api,
  db,
}, testInfo) => {
  const admin = await newAccount(api, testInfo, "admin");
  const ann = await newAccount(api, testInfo, "ann");
  const slug = slugFor(testInfo);
  await createWorkspace(api, admin.token, { name: "Acme", slug });
  await inviteAndAccept(api, admin.token, slug, ann, 15);
  const project = await createProject(api, admin.token, slug, { name: "Web", identifier: "WEB" });
  await addProjectMembers(api, admin.token, project.id, [{ member_id: ann.id, role: 15 }]);
  const issue = await createIssue(api, admin.token, project.id, { name: "Draft", description_html: "<p>First</p>" });
  const before = await descriptionPage(api, admin.token, issue.id);
  const renamed = await updateIssue(api, ann.token, issue.id, { name: "Final", description_html: "<p>Second</p>" });
  const firstSave = await descriptionPage(api, admin.token, issue.id);
  expect(firstSave.data).toHaveLength(2);
  expect(firstSave.data[0]?.id).not.toBe(before.data[0]?.id);
  expect((await activityPage(api, ann.token, issue.id, undefined, 100)).data.map((row) => row.field)).toEqual([
    null,
    "name",
    "description",
  ]);
  const changed = await updateIssue(api, ann.token, issue.id, { description_html: "<p>Third</p>" });
  await expectDescriptionActivity(db, issue.id, issue.description_html, changed.description_html);
  const merged = await descriptionPage(api, admin.token, issue.id);
  expect(merged.data.map((row) => row.id)).toEqual(firstSave.data.map((row) => row.id));
  expect(instantKey(changed.updated_at) > instantKey(renamed.updated_at)).toBe(true);
  const activity = await activityPage(api, ann.token, issue.id, undefined, 100);
  const version = merged.data[0];
  if (!version) throw new Error("no merged version");
  expect((await descriptionVersion(api, admin.token, version.id)).description_html).toBe("<p>Third</p>");
  const saved = await updateIssue(api, admin.token, issue.id, {
    description_html: '<p data-id="block-123">Third</p>',
    skip_activity: true,
  });
  expect(saved.description_html).toBe('<p data-id="block-123">Third</p>');
  expect(saved.updated_at).toBe(changed.updated_at);
  expect(saved.updated_by_id).toBe(changed.updated_by_id);
  expect(await activityPage(api, ann.token, issue.id, undefined, 100)).toEqual(activity);
  expect(await descriptionPage(api, admin.token, issue.id)).toEqual(merged);
  expect((await descriptionVersion(api, admin.token, version.id)).description_html).toBe("<p>Third</p>");
});

for (const property of ["href", "checked"] as const) {
  test(`I2 (API): ${property}-only content changes audit normally even with skip_activity`, async ({
    api,
    db,
  }, testInfo) => {
    const admin = await newAccount(api, testInfo, "admin");
    const slug = slugFor(testInfo);
    await createWorkspace(api, admin.token, { name: "Acme", slug });
    const project = await createProject(api, admin.token, slug, { name: "Web", identifier: "WEB" });
    const beforeHtml = '<p><a href="https://example.test/a">Link</a><input type="checkbox"></p>';
    const nextHtml =
      property === "href"
        ? beforeHtml.replace("/a", "/b")
        : beforeHtml.replace('type="checkbox"', 'type="checkbox" checked');
    const issue = await createIssue(api, admin.token, project.id, {
      name: "Content identity",
      description_html: beforeHtml,
    });
    const initial = await descriptionPage(api, admin.token, issue.id);
    const changed = await updateIssue(api, admin.token, issue.id, { description_html: nextHtml, skip_activity: true });
    expect(instantKey(changed.updated_at) > instantKey(issue.updated_at)).toBe(true);
    await expectDescriptionActivity(db, issue.id, issue.description_html, changed.description_html);
    const versions = await descriptionPage(api, admin.token, issue.id);
    expect(versions.data.map((row) => row.id)).toEqual(initial.data.map((row) => row.id));
    const latest = versions.data[0];
    if (!latest) throw new Error("no latest description version");
    expect(instantKey(latest.last_saved_at) > instantKey(initial.data[0]?.last_saved_at ?? "")).toBe(true);
    expect((await descriptionVersion(api, admin.token, latest.id)).description_html).toBe(changed.description_html);
  });
}
