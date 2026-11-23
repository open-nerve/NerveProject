import type { Page } from "@playwright/test";

import { archiveProject } from "../../fixtures/api";
import { contentOf, intakeRowOf } from "../../fixtures/assert/intake";
import { expectQuietConsole, watchPage } from "../../fixtures/browser";
import { failedLoads, recordApi, toast } from "../../fixtures/cycle-pages";
import { intakeIssueUpdated, submitIntakeIssue, triageIntakeIssue } from "../../fixtures/intake";
import {
  commentField,
  expectBlocksIdentified,
  expectHeaderOf,
  identifierOf,
  intakeDescription,
  intakeDetail,
  intakeEditPath,
  intakeEntry,
  intakeListReads,
  intakePath,
  intakeRow,
  intakeWorld,
  rowName,
  selecting,
  shownRows,
  shownStatus,
  writesSeen,
  wrote,
  type ApiWrite,
} from "../../fixtures/intake-pages";
import { createIssue, type Issue } from "../../fixtures/issues";
import { expect, test } from "../../fixtures/test";

// H11's page version, the text of a work item of a project's intake and who may change it (M7 design 2, 3.6; P5a
// spec 2.4, 2.7, 2.9, 2.12, R3, R15, R16, R17, R20, R27): its admin changes its name, its description and its
// properties, and restores a version of its description; a member who submitted it changes them too, and deletes it; a
// guest who submitted it changes its name and its description alone, neither reads its description's versions nor
// deletes it; a member who did not submit it, and anyone once it is triaged, reads it alone. While the work item is in
// the intake, each change goes to the intake's (PATCH /api/v0/intake-issues/{id}), the save that gives the blocks of an
// editable description their ids too; comments and reactions are the work item's own, anyone's who reads it. A
// duplicate names the work item it duplicates; an archived project's intake shows the archived page. None of the
// intake's work items is among the project's work items. Its images and attachments are P5b's; the API versions are
// h9-intake-list.spec.ts's change and deletion and h11-intake-assets.spec.ts.

/**
 * Checks that the intake page's detail shows the work item issue, named name, for reading alone: its name as text,
 * its description not editable, its properties' controls disabled, and no menu, so neither a triage nor its deletion.
 */
async function expectReadOnly(page: Page, issue: Issue, name = issue.name): Promise<void> {
  const detail = intakeDetail(page);
  await expect(detail.getByText(name, { exact: true })).toBeVisible();
  await expect(detail.getByRole("textbox", { name: "Work item title", exact: true })).toHaveCount(0);
  await expect(intakeDescription(page, issue)).toHaveAttribute("contenteditable", "false");
  await expect(detail.getByRole("button", { name: /^Priority: / })).toBeDisabled();
  await expect(detail.getByRole("button", { name: "Add assignees", exact: true })).toBeDisabled();
  await expect(detail.getByRole("button", { name: /^(Accept|Decline|Options)$/ })).toHaveCount(0);
}

/** Reacts to the work item issue from the intake page's detail with a thumbs up, by the picker: wrote's. */
async function thumbsUp(page: Page, writes: readonly ApiWrite[], issue: Issue) {
  await intakeDetail(page).getByRole("button", { name: "Add reaction", exact: true }).click();
  await page.getByRole("searchbox", { name: "Search", exact: true }).fill("thumbs up");
  return wrote(page, writes, "POST", `/api/v0/issues/${issue.id}/reactions`, () =>
    page.getByRole("gridcell", { name: "Thumbs up", exact: true }).click()
  );
}

test("H11 (page): alice, Web's admin, changes the name, the description and the priority of the work item bob submitted to its intake, each by the intake's change, and restores bob's version of its description; she comments on it and reacts to it as to any work item, and is offered its deletion; the list of Web's work items she left does not list it", async ({
  api,
  browserName,
  db,
  signedInPage,
}, testInfo) => {
  const { alice, bob, slug, web } = await intakeWorld(api, testInfo);
  const fix = await createIssue(api, alice.token, web.id, { name: "Fix the login" });
  const crash = await submitIntakeIssue(api, bob.token, web.id, {
    name: "Crash on save",
    description_html: "<p>It crashes</p>",
  });
  // alice's version of its description, beside bob's
  const edited = await intakeIssueUpdated(api, alice.token, crash.id, { description_html: "<p>It crashes often</p>" });
  expect(edited.status, "alice's version").toBe(200);

  // Web's work items, then its intake by the sidebar: the description, editable, gives its blocks their ids through the
  // intake before anything else changes.
  const page = await signedInPage(alice.tokens);
  const watch = await watchPage(page);
  const writes = await writesSeen(page, browserName);
  const reads = recordApi(page);
  await page.goto(`/${slug}/projects/${web.id}/issues`);
  await expect(page.locator(`#issue-${fix.id}`)).toBeVisible();
  await expectBlocksIdentified(page, writes, crash, "It crashes often", () =>
    intakeEntry(page, slug, web.id, 1).click()
  );
  await expect(page).toHaveURL(selecting(crash));

  // Its name, its description and its priority: each a change of the intake's, carrying itself alone.
  const detail = intakeDetail(page);
  const edit = intakeEditPath(crash);
  const title = detail.getByRole("textbox", { name: "Work item title", exact: true });
  expect(await wrote(page, writes, "PATCH", edit, () => title.fill("Crash in Firefox"))).toEqual([
    { name: "Crash in Firefox" },
    200,
  ]);
  await expect.poll(() => shownRows(page)).toEqual([`${identifierOf(crash)} Crash in Firefox`]);
  const description = intakeDescription(page, crash);
  const described = await wrote(page, writes, "PATCH", edit, async () => {
    await description.click();
    await description.press("End");
    await page.keyboard.insertText(" on save");
  });
  expect(described).toEqual([{ description_html: expect.stringMatching(/>It crashes often on save<\/p>$/) }, 200]);
  const prioritized = await wrote(page, writes, "PATCH", edit, async () => {
    await detail.getByRole("button", { name: "Priority: None", exact: true }).click();
    await page.getByRole("option", { name: "High", exact: true }).click();
  });
  expect(prioritized).toEqual([{ priority: "high" }, 200]);

  // bob's version, restored by the intake's change as he wrote it; its blocks then given ids.
  await detail.getByRole("button", { name: /^Last edited by Alice/ }).click();
  await expect(page.getByRole("menuitem")).toHaveText([/Alice/, /Bob/]);
  await page.getByRole("menuitem", { name: /Bob/ }).click();
  const history = page.getByRole("dialog", { name: "Description history", exact: true });
  await expect(history.getByText("It crashes", { exact: true })).toBeVisible();
  const restore = history.getByRole("button", { name: "Restore", exact: true });
  expect(await wrote(page, writes, "PATCH", edit, () => restore.click())).toEqual([
    { description_html: "<p>It crashes</p>" },
    200,
  ]);
  await expectBlocksIdentified(page, writes, crash, "It crashes", async () => undefined);
  await expect(history).toHaveCount(0);
  await expect(description).toHaveText("It crashes");

  // A comment and a reaction, the work item's own; its deletion offered in the menu.
  await commentField(page, crash).fill("Seen it too");
  const comment = detail.getByRole("button", { name: "Comment", exact: true });
  expect(await wrote(page, writes, "POST", `/api/v0/issues/${crash.id}/comments`, () => comment.click())).toEqual([
    { comment_html: expect.stringMatching(/>Seen it too<\/p>$/), attachment_ids: [] },
    201,
  ]);
  await expect(detail.getByText("Seen it too", { exact: true })).toBeVisible();
  expect(await thumbsUp(page, writes, crash)).toEqual([{ reaction: "128077-65039" }, 201]);
  await expect(detail.getByRole("button", { name: "👍️ 1", exact: true })).toBeVisible();
  await detail.getByRole("button", { name: "Options", exact: true }).click();
  await expect(page.getByRole("menuitem")).toHaveText(["Snooze", "Mark as duplicate", "Delete"]);
  await page.keyboard.press("Escape");
  expect(writes.map(({ write }) => write)).toEqual([
    ...Array<string>(6).fill(`PATCH ${edit}`),
    `POST /api/v0/issues/${crash.id}/comments`,
    `POST /api/v0/issues/${crash.id}/reactions`,
  ]);
  expect(await contentOf(db, crash.id)).toEqual({
    name: "Crash in Firefox",
    description_html: expect.stringMatching(/>It crashes<\/p>$/),
    priority: "high",
    written_by: alice.id,
    deleted: false,
  });
  expect(await intakeRowOf(db, crash.id)).toMatchObject({ status: "pending" });

  // Back on Web's work items by the sidebar, the detail's reads answered: the list held lists Fix the login alone.
  await expect.poll(() => reads().pending).toBe(0);
  await page
    .getByRole("complementary", { name: "Main sidebar", exact: true })
    .getByRole("link", { name: "Work items", exact: true })
    .click();
  await expect(page.locator(`#issue-${fix.id}`)).toBeVisible();
  await expect
    .poll(() => page.locator('[id^="issue-"]').evaluateAll((rows) => rows.map((row) => row.id)))
    .toEqual([`issue-${fix.id}`]);

  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], []]);
  await expectQuietConsole(page, watch);
});

test("H11 (page): bob, a member of Web, changes the name and the priority of the work item he submitted to its intake, by the intake's change, its deletion alone in its menu; Gus's idea, not his, he reads alone, its description's versions too, which he may not restore, and comments on; he deletes his own: it leaves the intake, the list focused and its first row selected", async ({
  api,
  browserName,
  db,
  signedInPage,
}, testInfo) => {
  const { bob, gus, slug, web } = await intakeWorld(api, testInfo);
  const idea = await submitIntakeIssue(api, gus.token, web.id, {
    name: "Gus's idea",
    description_html: "<p>Make it blue</p>",
  });
  const dark = await submitIntakeIssue(api, bob.token, web.id, {
    name: "Dark mode",
    description_html: "<p>Too bright</p>",
  });

  // From the closed tab to the open one: his own selected, its description, editable, giving its blocks their ids.
  const page = await signedInPage(bob.tokens);
  const watch = await watchPage(page);
  const writes = await writesSeen(page, browserName);
  await page.goto(`${intakePath(slug, web.id)}?currentTab=closed`);
  await expect(page.getByText("No request closed yet", { exact: true })).toBeVisible();
  const open = page.getByRole("tab", { name: "Open", exact: true });
  await expectBlocksIdentified(page, writes, dark, "Too bright", () => open.click());
  await expect.poll(() => shownRows(page)).toEqual([dark, idea].map(rowName));
  await expect(page).toHaveURL(selecting(dark));

  // His own: its name and its priority his to change, through the intake; its deletion alone in its menu.
  const detail = intakeDetail(page);
  const edit = intakeEditPath(dark);
  const title = detail.getByRole("textbox", { name: "Work item title", exact: true });
  expect(await wrote(page, writes, "PATCH", edit, () => title.fill("Dark mode, please"))).toEqual([
    { name: "Dark mode, please" },
    200,
  ]);
  const prioritized = await wrote(page, writes, "PATCH", edit, async () => {
    await detail.getByRole("button", { name: "Priority: None", exact: true }).click();
    await page.getByRole("option", { name: "Low", exact: true }).click();
  });
  expect(prioritized).toEqual([{ priority: "low" }, 200]);
  await detail.getByRole("button", { name: "Options", exact: true }).click();
  await expect(page.getByRole("menuitem")).toHaveText(["Delete"]);
  await page.keyboard.press("Escape");

  // Gus's idea: his to read alone, its versions too, which he may not restore; his comment the work item's own.
  await intakeRow(page, idea).click();
  await expectHeaderOf(page, idea);
  await expectReadOnly(page, idea);
  await detail.getByRole("button", { name: /^Last edited by Gus/ }).click();
  await page.getByRole("menuitem", { name: /Gus/ }).click();
  const history = page.getByRole("dialog", { name: "Description history", exact: true });
  await expect(history.getByText("Make it blue", { exact: true })).toBeVisible();
  await expect(history.getByRole("button", { name: "Restore", exact: true })).toHaveCount(0);
  await page.keyboard.press("Escape");
  await expect(history).toHaveCount(0);
  await commentField(page, idea).fill("Blue is good");
  const comment = detail.getByRole("button", { name: "Comment", exact: true });
  expect(await wrote(page, writes, "POST", `/api/v0/issues/${idea.id}/comments`, () => comment.click())).toEqual([
    { comment_html: expect.stringMatching(/>Blue is good<\/p>$/), attachment_ids: [] },
    201,
  ]);
  await expect(detail.getByText("Blue is good", { exact: true })).toBeVisible();

  // His own deleted, by the intake's deletion: gone from the list, the list focused, Gus's idea selected in its place.
  await intakeRow(page, { ...dark, name: "Dark mode, please" }).click();
  await expectHeaderOf(page, dark);
  await detail.getByRole("button", { name: "Options", exact: true }).click();
  await page.getByRole("menuitem", { name: "Delete", exact: true }).click();
  const deleting = page.getByRole("dialog", { name: "Delete work item", exact: true });
  await expect(deleting).toContainText("The work item itself is deleted, not only its place in the intake.");
  const confirm = deleting.getByRole("button", { name: "Delete", exact: true });
  expect(await wrote(page, writes, "DELETE", edit, () => confirm.click())).toEqual([null, 204]);
  await expect(page.getByRole("region", { name: "Intake work items", exact: true })).toBeFocused();
  await expect(page).toHaveURL(selecting(idea));
  await expect(deleting).toHaveCount(0);
  await expect(toast(page, "Work item deleted successfully")).toBeVisible();
  await expect.poll(() => shownRows(page)).toEqual([rowName(idea)]);
  expect(writes.map(({ write }) => write)).toEqual([
    ...Array<string>(3).fill(`PATCH ${edit}`),
    `POST /api/v0/issues/${idea.id}/comments`,
    `DELETE ${edit}`,
  ]);
  expect(await contentOf(db, dark.id)).toEqual({
    name: "Dark mode, please",
    description_html: expect.stringMatching(/>Too bright<\/p>$/),
    priority: "low",
    written_by: bob.id,
    deleted: true,
  });

  // The deletion's read again, in its turn, finds the work item gone (nerve's 404), as the deletion means.
  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([
    [`404 GET /api/v0/issues/${dark.id}`],
    [],
    [],
  ]);
  await expectQuietConsole(page, watch, { errors: failedLoads(browserName, 404) });
});

test("H11 (page): gus, a guest of Web, changes the name and the description of the work item he submitted to its intake, each change carrying itself alone, by the intake's change; its properties he reads alone, its description's versions he is neither offered nor reads, nor is he offered its deletion; he reacts to it", async ({
  api,
  browserName,
  db,
  signedInPage,
}, testInfo) => {
  const { bob, gus, slug, web } = await intakeWorld(api, testInfo);
  await submitIntakeIssue(api, bob.token, web.id, { name: "Dark mode" });
  const idea = await submitIntakeIssue(api, gus.token, web.id, {
    name: "Gus's idea",
    description_html: "<p>Make it blue</p>",
  });

  // His own alone, from the closed tab to the open one: its description, editable, gives its blocks their ids.
  const page = await signedInPage(gus.tokens);
  const watch = await watchPage(page);
  const writes = await writesSeen(page, browserName);
  await page.goto(`${intakePath(slug, web.id)}?currentTab=closed`);
  await expect(page.getByText("No request closed yet", { exact: true })).toBeVisible();
  const open = page.getByRole("tab", { name: "Open", exact: true });
  await expectBlocksIdentified(page, writes, idea, "Make it blue", () => open.click());
  await expect.poll(() => shownRows(page)).toEqual([rowName(idea)]);
  await expect(page).toHaveURL(selecting(idea));

  // Its name, then its description: each change of the intake's carries itself alone.
  const detail = intakeDetail(page);
  const edit = intakeEditPath(idea);
  const title = detail.getByRole("textbox", { name: "Work item title", exact: true });
  expect(await wrote(page, writes, "PATCH", edit, () => title.fill("Gus's blue idea"))).toEqual([
    { name: "Gus's blue idea" },
    200,
  ]);
  const description = intakeDescription(page, idea);
  const described = await wrote(page, writes, "PATCH", edit, async () => {
    await description.click();
    await description.press("End");
    await page.keyboard.insertText(", please");
  });
  expect(described).toEqual([{ description_html: expect.stringMatching(/>Make it blue, please<\/p>$/) }, 200]);

  // Its properties disabled; no versions, no triage and no menu; a reaction his.
  await expect(detail.getByRole("button", { name: "Priority: None", exact: true })).toBeDisabled();
  await expect(detail.getByRole("button", { name: "Add assignees", exact: true })).toBeDisabled();
  await expect(detail.getByRole("button", { name: /^Last edited by/ })).toHaveCount(0);
  await expect(detail.getByRole("button", { name: /^(Accept|Decline|Options)$/ })).toHaveCount(0);
  expect(await thumbsUp(page, writes, idea)).toEqual([{ reaction: "128077-65039" }, 201]);
  await expect(detail.getByRole("button", { name: "👍️ 1", exact: true })).toBeVisible();
  expect(watch.apiRequests.filter((request) => request.includes("description-versions"))).toEqual([]);
  expect(writes.map(({ write }) => write)).toEqual([
    ...Array<string>(3).fill(`PATCH ${edit}`),
    `POST /api/v0/issues/${idea.id}/reactions`,
  ]);
  expect(await contentOf(db, idea.id)).toEqual({
    name: "Gus's blue idea",
    description_html: expect.stringMatching(/>Make it blue, please<\/p>$/),
    priority: "none",
    written_by: gus.id,
    deleted: false,
  });

  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], []]);
  await expectQuietConsole(page, watch);
});

test("H11 (page): on Web's closed tab, alice, its admin, reads a declined work item alone, offered neither its change, nor a triage, nor its deletion, and reacts to it; a duplicate, read alone too, names the work item it duplicates and links to it; Web archived, its intake shows the archived page and reads no list", async ({
  api,
  browserName,
  db,
  signedInPage,
}, testInfo) => {
  const { alice, bob, slug, web } = await intakeWorld(api, testInfo);
  const fix = await createIssue(api, alice.token, web.id, { name: "Fix the login" });
  const again = await submitIntakeIssue(api, bob.token, web.id, {
    name: "The login fails",
    description_html: "<p>Cannot log in</p>",
  });
  const spam = await submitIntakeIssue(api, bob.token, web.id, { name: "Buy now", description_html: "<p>Cheap</p>" });
  await triageIntakeIssue(api, alice.token, again.id, { status: "duplicate", duplicate_to_id: fix.id });
  await triageIntakeIssue(api, alice.token, spam.id, { status: "declined" });

  // Declined: read alone; a reaction still hers.
  const page = await signedInPage(alice.tokens);
  const watch = await watchPage(page);
  const writes = await writesSeen(page, browserName);
  const reads = recordApi(page);
  await page.goto(`${intakePath(slug, web.id)}?currentTab=closed`);
  await expect.poll(() => shownRows(page)).toEqual([spam, again].map(rowName));
  await expect(page).toHaveURL(selecting(spam));
  await expectHeaderOf(page, spam);
  await expect(shownStatus(page, "Declined")).toBeVisible();
  await expectReadOnly(page, spam);
  const detail = intakeDetail(page);
  expect(await thumbsUp(page, writes, spam)).toEqual([{ reaction: "128077-65039" }, 201]);
  await expect(detail.getByRole("button", { name: "👍️ 1", exact: true })).toBeVisible();

  // The duplicate: read alone, naming Fix the login, linked to it.
  await intakeRow(page, again).click();
  await expectHeaderOf(page, again);
  await expect(shownStatus(page, "Duplicate")).toBeVisible();
  await expectReadOnly(page, again);
  await expect(detail.getByRole("link", { name: `${identifierOf(fix)} · Fix the login`, exact: true })).toHaveAttribute(
    "href",
    `/${slug}/browse/${identifierOf(fix)}`
  );
  expect(writes.map(({ write }) => write)).toEqual([`POST /api/v0/issues/${spam.id}/reactions`]);
  // The reaction is alice's write of the declined one; the duplicate's as bob left it.
  const kept = { priority: "none", deleted: false };
  expect(await Promise.all([spam, again].map((issue) => contentOf(db, issue.id)))).toEqual([
    { ...kept, name: "Buy now", description_html: "<p>Cheap</p>", written_by: alice.id },
    { ...kept, name: "The login fails", description_html: "<p>Cannot log in</p>", written_by: bob.id },
  ]);
  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], []]);
  await expectQuietConsole(page, watch);

  // Web archived, its intake opened anew once the page's reads are answered: the archived page, and no list read.
  await expect.poll(() => reads().pending).toBe(0);
  await archiveProject(api, alice.token, web.id);
  const opened = recordApi(page);
  await page.goto(intakePath(slug, web.id));
  await expect(page.getByText("This project is archived", { exact: true })).toBeVisible();
  await expect(intakeDetail(page)).toHaveCount(0);
  await expect.poll(() => opened().pending).toBe(0);
  expect(intakeListReads(opened().requests)).toEqual([]);
  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], []]);
});
