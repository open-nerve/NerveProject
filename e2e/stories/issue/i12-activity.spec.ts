import { c12Evidence } from "../../fixtures/c12-evidence";
import type { components } from "@nerve/api-client";

import { addProjectMembers, createProject, createWorkspace, inviteAndAccept, slugFor } from "../../fixtures/api";
import { byId } from "../../fixtures/assert/issue";
import { accountId, emailFor, newAccount } from "../../fixtures/auth";
import { commentDeleted, commentPage, commentUpdated, createIssueComment } from "../../fixtures/comments";
import { openIssueDetail } from "../../fixtures/issue-detail-page";
import { expectListPageQuiet, issueListProject } from "../../fixtures/issue-list-page";
import { activityPage, createIssue, createIssueRelations, updateIssue } from "../../fixtures/issues";
import { instantKey } from "../../fixtures/instant";
import { answerTo, registerOnboarded } from "../../fixtures/settings-pages";
import { expect, test } from "../../fixtures/test";

type Activity = components["schemas"]["IssueActivity"];
type Comment = components["schemas"]["IssueComment"];
type TimelineRow = Activity | Comment;
const timelineOrder = (a: TimelineRow, b: TimelineRow) =>
  instantKey(a.created_at).localeCompare(instantKey(b.created_at)) || byId(a, b);

/** Exhausts multiple real pages and retains the request cursor that produced the last page. */
async function exhaust<T>(read: (cursor?: string) => Promise<{ data: T[]; next_cursor: string | null }>) {
  const rows: T[] = [];
  let pages = 0;
  const visit = async (cursor?: string): Promise<{ rows: T[]; cursor: string | undefined; pages: number }> => {
    const page = await read(cursor);
    rows.push(...page.data);
    pages++;
    if (!page.next_cursor) return { rows, cursor, pages };
    expect(pages, "bounded pagination").toBeLessThan(30);
    return visit(page.next_cursor);
  };
  return visit();
}

test("I12 (API): comments and activities traverse multiple pages then incremental reads replace a merged description by id and reorder the joint timeline without loss or duplication", async ({
  api,
}, testInfo) => {
  const admin = await newAccount(api, testInfo, "admin");
  const slug = slugFor(testInfo);
  await createWorkspace(api, admin.token, { name: "Acme", slug });
  const project = await createProject(api, admin.token, slug, { name: "Web", identifier: "WEB" });
  const issue = await createIssue(api, admin.token, project.id, { name: "Timeline" });
  // Each title write follows its comment so both independent resources have distinguishable pages.
  await [0, 1, 2, 3].reduce(async (before, n) => {
    await before;
    await createIssueComment(api, admin.token, issue.id, `<p>Comment ${n}</p>`);
    await updateIssue(api, admin.token, issue.id, { name: `Timeline ${n}` });
  }, Promise.resolve());
  await updateIssue(api, admin.token, issue.id, { description_html: "<p>Draft</p>" });
  const comments = await exhaust((cursor) => commentPage(api, admin.token, issue.id, cursor));
  const activities = await exhaust((cursor) => activityPage(api, admin.token, issue.id, cursor));
  expect(comments.pages).toBeGreaterThan(1);
  expect(activities.pages).toBeGreaterThan(1);
  expect(comments.cursor).toEqual(expect.any(String));
  expect(activities.cursor).toEqual(expect.any(String));
  const description = activities.rows.find((row) => row.field === "description");
  if (!description) throw new Error("no description activity");
  expect(description).not.toHaveProperty("comment");
  expect(activities.rows.map((row) => row.id)).toEqual(
    (await activityPage(api, admin.token, issue.id, undefined, 100)).data.map((row) => row.id)
  );
  const inserted = await createIssueComment(api, admin.token, issue.id, "<p>Between saves</p>");
  expect(timelineOrder(description, inserted)).toBeLessThan(0);
  await updateIssue(api, admin.token, issue.id, { description_html: "<p>Final</p>" });
  const newComments = await exhaust((cursor) => commentPage(api, admin.token, issue.id, cursor));
  const newActivities = await exhaust((cursor) => activityPage(api, admin.token, issue.id, cursor));
  // Incremental reads restart from each final page's request position, never from a null final response cursor.
  const commentDelta = await commentPage(api, admin.token, issue.id, comments.cursor, 100);
  const activityDelta = await activityPage(api, admin.token, issue.id, activities.cursor, 100);
  const moved = activityDelta.data.find((row) => row.id === description.id);
  expect(moved).toMatchObject({ id: description.id, old_value: issue.description_html, new_value: "<p>Final</p>" });
  if (!moved) throw new Error("incremental activity omitted merged description");
  expect(timelineOrder(inserted, moved)).toBeLessThan(0);
  expect(instantKey(moved.created_at) > instantKey(description.created_at)).toBe(true);
  const byIdentity = new Map<string, TimelineRow>();
  for (const row of [...comments.rows, ...activities.rows, ...commentDelta.data, ...activityDelta.data])
    byIdentity.set(row.id, row);
  const merged = [...byIdentity.values()].toSorted(timelineOrder);
  const expected = [...newComments.rows, ...newActivities.rows].toSorted(timelineOrder);
  const expectedOrderedIds = expected.map((row) => row.id);
  expect(merged.map((row) => row.id)).toEqual(expectedOrderedIds);
  expect(new Set(merged.map((row) => row.id)).size).toBe(merged.length);
  expect(merged).toEqual(expected);
  expect(merged.findIndex((row) => row.id === moved.id)).toBeGreaterThan(
    merged.findIndex((row) => row.id === inserted.id)
  );
});

test("I12 (API): hidden historical relation references are filtered before pagination and visible neighboring activity remains", async ({
  api,
}, testInfo) => {
  const admin = await newAccount(api, testInfo, "admin");
  const reader = await newAccount(api, testInfo, "reader");
  const slug = slugFor(testInfo);
  await createWorkspace(api, admin.token, { name: "Acme", slug });
  await inviteAndAccept(api, admin.token, slug, reader, 15);
  const visible = await createProject(api, admin.token, slug, { name: "Visible", identifier: "VIS", network: 0 });
  const hidden = await createProject(api, admin.token, slug, { name: "Hidden", identifier: "HID", network: 0 });
  await addProjectMembers(api, admin.token, visible.id, [{ member_id: reader.id, role: 15 }]);
  const issue = await createIssue(api, admin.token, visible.id, { name: "Visible item" });
  const secret = await createIssue(api, admin.token, hidden.id, { name: "Secret item" });
  await createIssueRelations(api, admin.token, issue.id, { relation_type: "relates_to", issue_ids: [secret.id] });
  await updateIssue(api, admin.token, issue.id, { priority: "high" });
  const adminRows = (await exhaust((cursor) => activityPage(api, admin.token, issue.id, cursor, 1))).rows;
  expect(adminRows.map((row) => row.field)).toEqual([null, "relates_to", "priority"]);
  const readerRows = (await exhaust((cursor) => activityPage(api, reader.token, issue.id, cursor, 1))).rows;
  expect(readerRows).toEqual(adminRows.filter((row) => row.field !== "relates_to"));
  expect(JSON.stringify(readerRows)).not.toContain(secret.id);
  expect(JSON.stringify(readerRows)).not.toContain("HID-");
});

test("I12 (page): three real pages merge into one timeline, a confirmed description save reorders its existing activity, and reopening replaces remote comment edits and deletions", async ({
  api,
  page,
}, info) => {
  test.setTimeout(60_000);
  const seed = await issueListProject(api, info);
  const issue = await createIssue(api, seed.member.token, seed.project.id, { name: "Timeline" });
  const comments: Comment[] = [];
  // The public default is fifty rows. Neither the API pages nor their cursors are intercepted.
  await Array.from({ length: 101 }, (_, n) => n).reduce(async (previous, n) => {
    await previous;
    comments.push(await createIssueComment(api, seed.member.token, issue.id, `<p>Comment ${n}</p>`));
    await updateIssue(api, seed.member.token, issue.id, { name: `Timeline ${n}` });
  }, Promise.resolve());
  await updateIssue(api, seed.member.token, issue.id, {
    description_html: '<p class="editor-paragraph-block" data-id="timeline-description">First</p>',
  });
  const between = await createIssueComment(api, seed.member.token, issue.id, "<p>Between saves</p>");
  const activities = (await exhaust((cursor) => activityPage(api, seed.member.token, issue.id, cursor, 50))).rows;
  const original = activities.find((row) => row.field === "description");
  if (!original) throw new Error("missing initial description activity");
  const reads = { activities: [] as URLSearchParams[], comments: [] as URLSearchParams[] };
  page.on("request", (request) => {
    const url = new URL(request.url());
    if (request.method() !== "GET") return;
    for (const kind of ["activities", "comments"] as const)
      if (url.pathname === `/api/v0/issues/${issue.id}/${kind}`) reads[kind].push(url.searchParams);
  });
  const watch = await openIssueDetail(page, seed, issue);
  const timeline = page.locator(`[id="comment-${comments[0]!.id}"]`).locator("..");
  const editor = page.locator(`[id="editor-container-${issue.id}"] [contenteditable="true"]`);
  await expect(editor).toHaveText("First");
  await expect(timeline.locator("xpath=./div")).toHaveCount(205);
  await expect(timeline.getByText("Comment 100", { exact: true })).toBeVisible();
  await expect(timeline.getByText("Between saves", { exact: true })).toBeVisible();
  await c12Evidence(page, "B16", "timeline-three-real-pages");
  expect(
    await timeline.locator("xpath=./div[@id]").evaluateAll((rows) => rows.map((row) => row.id.replace(/^comment-/, "")))
  ).toEqual([...comments.map((comment) => comment.id), between.id]);
  const descriptionRow = timeline.locator("xpath=./div").filter({ hasText: "updated the description" });
  await expect(descriptionRow).toHaveCount(1);
  const relativeDescriptionPosition = () =>
    timeline.locator("xpath=./div").evaluateAll((rows) => {
      const text = rows.map((row) => row.textContent ?? "");
      return (
        text.findIndex((value) => value.includes("updated the description")) -
        text.findIndex((value) => value.includes("Between saves"))
      );
    });
  expect(await relativeDescriptionPosition()).toBeLessThan(0);
  for (const kind of ["activities", "comments"] as const) {
    expect(reads[kind].length).toBeGreaterThanOrEqual(3);
    expect(reads[kind][0]!.get("cursor")).toBeNull();
    expect(reads[kind].at(-1)!.get("cursor")).toEqual(expect.any(String));
  }
  const before = { activities: reads.activities.length, comments: reads.comments.length };
  expect(
    (
      await answerTo(page, "PATCH", `/api/v0/issues/${issue.id}`, async () => {
        await editor.click();
        await editor.press("ControlOrMeta+A");
        await page.keyboard.insertText("Final");
      })
    ).status()
  ).toBe(200);
  await expect.poll(relativeDescriptionPosition).toBeGreaterThan(0);
  await expect(timeline.locator("xpath=./div")).toHaveCount(205);
  await expect(descriptionRow).toHaveCount(1);
  await Promise.all(
    (["activities", "comments"] as const).map(async (kind) => {
      await expect.poll(() => reads[kind].length).toBeGreaterThan(before[kind]);
      expect(reads[kind][before[kind]]!.get("cursor")).toBe(reads[kind][before[kind] - 1]!.get("cursor"));
    })
  );
  const refreshed = (await exhaust((cursor) => activityPage(api, seed.member.token, issue.id, cursor, 50))).rows;
  expect(refreshed.filter((row) => row.field === "description")).toHaveLength(1);
  expect(refreshed.at(-1)).toMatchObject({ id: original.id, field: "description" });
  expect(instantKey(refreshed.at(-1)!.created_at) > instantKey(between.created_at)).toBe(true);

  expect((await commentUpdated(api, seed.member.token, comments[0]!.id, "<p>Remote first edit</p>")).status).toBe(200);
  expect((await commentDeleted(api, seed.member.token, comments[1]!.id)).status).toBe(204);
  // These old rows are outside the incremental window; reopening performs a complete read.
  await page.goto(seed.path);
  // Complete the intermediate page's real sign-in refresh before replacing its document again.
  await expect(page.locator(`#issue-${issue.id}`)).toBeVisible();
  await page.goto(`/${seed.slug}/browse/WEB-${issue.sequence_id}`);
  await expect(timeline.locator("xpath=./div")).toHaveCount(204);
  await expect(timeline.getByText("Remote first edit", { exact: true })).toBeVisible();
  await c12Evidence(page, "B16", "timeline-reopened-remote-edit");
  await expect(page.locator(`[id="comment-${comments[1]!.id}"]`)).toHaveCount(0);
  await expect(descriptionRow).toHaveCount(1);
  await expectListPageQuiet(page, watch);
});

test("I12 (page): hidden historical relation references stay absent while the neighboring visible activity renders", async ({
  api,
  page,
}, info) => {
  const seed = await issueListProject(api, info);
  const email = emailFor(info, "timeline-reader");
  const token = (await registerOnboarded(api, email)).access_token;
  const reader = { email, token, id: await accountId(api, token) };
  await inviteAndAccept(api, seed.member.token, seed.slug, reader, 15);
  await addProjectMembers(api, seed.member.token, seed.project.id, [{ member_id: reader.id, role: 15 }]);
  const hidden = await createProject(api, seed.member.token, seed.slug, {
    name: "Hidden",
    identifier: "HID",
    network: 0,
  });
  const issue = await createIssue(api, seed.member.token, seed.project.id, { name: "Visible timeline" });
  const secret = await createIssue(api, seed.member.token, hidden.id, { name: "Secret historical item" });
  await createIssueRelations(api, seed.member.token, issue.id, { relation_type: "relates_to", issue_ids: [secret.id] });
  await updateIssue(api, seed.member.token, issue.id, { name: "Visible neighbor" });
  const watch = await openIssueDetail(page, { ...seed, member: reader }, issue);
  await expect(page.getByText("set the name to Visible neighbor.", { exact: true })).toBeVisible();
  await expect(page.locator("body")).not.toContainText("HID-");
  await expect(page.locator("body")).not.toContainText("Secret historical item");
  await c12Evidence(page, "B16", "hidden-relation-absent");
  expect(
    (await exhaust((cursor) => activityPage(api, reader.token, issue.id, cursor, 1))).rows.map((row) => row.field)
  ).toEqual([null, "name"]);
  await expectListPageQuiet(page, watch);
});
