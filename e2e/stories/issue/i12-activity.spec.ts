import type { components } from "@nerve/api-client";

import { addProjectMembers, createProject, createWorkspace, inviteAndAccept, slugFor } from "../../fixtures/api";
import { byId } from "../../fixtures/assert/issue";
import { newAccount } from "../../fixtures/auth";
import { commentPage, createIssueComment } from "../../fixtures/comments";
import { activityPage, createIssue, createIssueRelations, updateIssue } from "../../fixtures/issues";
import { instantKey } from "../../fixtures/instant";
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
