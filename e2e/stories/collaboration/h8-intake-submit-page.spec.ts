import type { Page } from "@playwright/test";

import { changeProject, createLabel } from "../../fixtures/api";
import { contentOf, intakeNamesOf, intakeRowOf, submittedOf } from "../../fixtures/assert/intake";
import { expectQuietConsole, watchPage } from "../../fixtures/browser";
import { dismissToasts, failedLoads, recordApi, toast } from "../../fixtures/cycle-pages";
import {
  EMPTY_INTAKE,
  createMore,
  createModal,
  descriptionEditor,
  modalButton,
  nameField,
  openCreateModal,
  pickDueDay,
  submissions,
  submitPath,
  submitted,
} from "../../fixtures/intake-create";
import {
  browserDay,
  expectHeaderOf,
  intakeEntry,
  intakePath,
  intakeWorld,
  rowName,
  selecting,
  shownRows,
  writesSeen,
} from "../../fixtures/intake-pages";
import type { Issue } from "../../fixtures/issues";
import { answerTo } from "../../fixtures/settings-pages";
import { anotherBrowser } from "../../fixtures/workspace-pages";
import { expect, test } from "../../fixtures/test";

// H8's page version, submitting a work item to a project's intake through its create modal (M7 design 2, 3.5; P5b
// spec 2.1, 2.8, R5): its admin, a member and a guest each submit one, the body exactly what the form holds, an intake
// row pending of the submitter's; the page then selects it, and its list and the sidebar's count read it. The name is
// refused under its field, too long as one types, blank once a submission is: nothing is sent, the name has the focus.
// With Create more on, a submission empties the form and keeps it open. With the intake turned off meanwhile, nerve's
// refusal is said in a toast, the form kept. The API version is h8-intake-submit.spec.ts; closing the modal is
// h8-intake-close-page.spec.ts's.

/** The text of the toast of a submission nerve made. */
const CREATED = "Work item created successfully";

/**
 * The description of the editor holding text as one paragraph, as the modal sends it: the paragraph with the id the
 * editor gave its block.
 */
function paragraph(text: string) {
  return expect.stringMatching(new RegExp(`^<p class="editor-paragraph-block" data-id="[0-9a-f-]{36}">${text}</p>$`));
}

/** The description the modal sends for an editor it never heard from (none typed): an empty paragraph. */
const NO_DESCRIPTION = "<p></p>";

/**
 * Checks that page, once the modal it submitted issue with has closed, selects issue (its address and its detail's
 * header), says so in a toast, and that its list and the sidebar's count read it: rows, the names of the list's rows in
 * its order, and pending, the count.
 */
async function expectSelected(
  page: Page,
  slug: string,
  projectId: string,
  issue: Issue,
  rows: Issue[],
  pending: number
) {
  await expect(createModal(page)).toHaveCount(0);
  await expect(page).toHaveURL(selecting(issue));
  await expectHeaderOf(page, issue);
  await expect(toast(page, CREATED)).toBeVisible();
  await expect.poll(() => shownRows(page)).toEqual(rows.map(rowName));
  await expect(intakeEntry(page, slug, projectId, pending)).toBeVisible();
}

/** The live intake row of a work item just submitted by the account of submitterId: pending, his. */
function pendingOf(submitterId: string) {
  return { status: "pending", snoozed_till: null, duplicate_to_id: null, written_by: submitterId };
}

test("H8 (page): alice, Web's admin, is refused a name too long as she types it, an empty one and one of spaces as she submits, nothing sent and the name focused; she submits Crash on save with a description, a priority, an assignee, a label and tomorrow as its due date, the body exactly the form's, her pending intake row and the work item made, the page selecting it, the list and the sidebar's count reading it; with Create more on, her next submission empties the form and keeps it open", async ({
  api,
  browserName,
  db,
  signedInPage,
}, testInfo) => {
  const { alice, bob, slug, web } = await intakeWorld(api, testInfo);
  const bug = await createLabel(api, alice.token, web.id, { name: "Bug" });
  const page = await signedInPage(alice.tokens);
  const watch = await watchPage(page);
  const writes = await writesSeen(page, browserName);
  await page.goto(intakePath(slug, web.id));
  await expect(page.getByText(EMPTY_INTAKE, { exact: true })).toBeVisible();
  await openCreateModal(page);
  const name = nameField(page);
  const description = descriptionEditor(page);
  const create = modalButton(page, "Create work item");
  await expect(description).toHaveCount(1);

  // Too long: said under the name as she types it; submitted by the pointer, refused, the name focused.
  await name.fill("x".repeat(256));
  await expect(name).toHaveAttribute("aria-invalid", "true");
  await expect(name).toHaveAccessibleDescription("Too long");
  await create.click();
  await expect(name).toBeFocused();
  // Blank: said only once a submission is refused; empty by the pointer, spaces from the button by the keyboard.
  await name.fill("");
  await expect(name).not.toHaveAttribute("aria-invalid", "true");
  await create.click();
  await expect(name).toBeFocused();
  await expect(name).toHaveAttribute("aria-invalid", "true");
  await expect(name).toHaveAccessibleDescription("Title is required");
  await name.fill("x");
  await expect(name).not.toHaveAttribute("aria-invalid", "true");
  await name.fill("   ");
  await expect(name).not.toHaveAttribute("aria-invalid", "true");
  await create.focus();
  await page.keyboard.press("Enter");
  await expect(name).toBeFocused();
  await expect(name).toHaveAttribute("aria-invalid", "true");
  await expect(name).toHaveAccessibleDescription("Title is required");
  expect(submissions(writes, web.id), "nothing sent for a refused name").toEqual([]);

  // Crash on save, with all the form holds; its due date tomorrow, as the browser counts days.
  await name.fill("Crash on save");
  await description.fill("It crashes");
  await modalButton(page, "Priority: None").click();
  await page.getByRole("option", { name: "High", exact: true }).click();
  await modalButton(page, "Add assignees").click();
  await page.getByRole("option", { name: "Bob Bob", exact: true }).click();
  await page.keyboard.press("Escape");
  await modalButton(page, "Labels: Crash on save").click();
  await page.getByRole("option", { name: "Bug", exact: true }).click();
  await page.keyboard.press("Escape");
  const tomorrow = await browserDay(page, 1);
  await pickDueDay(page, tomorrow);
  const crash = await submitted(page, writes, web.id);
  expect(crash.body).toEqual({
    name: "Crash on save",
    description_html: paragraph("It crashes"),
    priority: "high",
    label_ids: [bug.id],
    assignee_ids: [bob.id],
    target_date: tomorrow,
  });
  await expectSelected(page, slug, web.id, crash.issue, [crash.issue], 1);
  expect(await intakeRowOf(db, crash.issue.id)).toEqual(pendingOf(alice.id));
  expect(await contentOf(db, crash.issue.id)).toEqual({
    name: "Crash on save",
    description_html: expect.stringMatching(/>It crashes<\/p>$/),
    priority: "high",
    written_by: alice.id,
    deleted: false,
  });
  expect(await submittedOf(db, crash.issue.id)).toEqual({
    submitter: alice.id,
    target_date: tomorrow,
    assignee_ids: [bob.id],
    label_ids: [bug.id],
  });

  // Create more: a switch, on and off and on; the next submission empties the form, which stays open with it on.
  await dismissToasts(page);
  await openCreateModal(page);
  const more = createMore(page);
  await more.click();
  await expect(more).toBeChecked();
  await more.click();
  await expect(more).not.toBeChecked();
  await more.click();
  await expect(more).toBeChecked();
  await name.fill("Login is slow");
  await description.fill("Takes ten seconds");
  const login = await submitted(page, writes, web.id);
  expect(login.body).toEqual({
    name: "Login is slow",
    description_html: paragraph("Takes ten seconds"),
    priority: "none",
    label_ids: [],
    assignee_ids: [],
  });
  await expect(toast(page, CREATED)).toBeVisible();
  await expect(createModal(page)).toHaveCount(1);
  await expect(name).toHaveValue("");
  await expect(description).toHaveText("");
  await expect(more).toBeChecked();
  expect(await intakeRowOf(db, login.issue.id)).toEqual(pendingOf(alice.id));
  // Emptied, it closes by Escape; the page still selects Crash on save, its list and count read Login is slow too.
  await page.keyboard.press("Escape");
  await expect(createModal(page)).toHaveCount(0);
  await expect(page).toHaveURL(selecting(crash.issue));
  await expect.poll(() => shownRows(page)).toEqual([login.issue, crash.issue].map(rowName));
  await expect(intakeEntry(page, slug, web.id, 2)).toBeVisible();
  expect(
    writes.map(({ write }) => write),
    "her writes: the two submissions alone"
  ).toEqual([`POST ${submitPath(web.id)}`, `POST ${submitPath(web.id)}`]);

  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], []]);
  await expectQuietConsole(page, watch);
});

test("H8 (page): bob, a member, and gus, a guest, each submit a work item from his own browser, the body exactly the form's, his pending intake row, the page selecting it and the list and the sidebar's count reading what he sees; with Web's intake turned off while gus's modal is open, his next submission is refused with the intake's toast, the form kept", async ({
  api,
  baseURL,
  browser,
  browserName,
  db,
  signedInPage,
}, testInfo) => {
  const { alice, bob, gus, slug, web } = await intakeWorld(api, testInfo);

  // bob, a member: Login is slow, named alone.
  const page = await signedInPage(bob.tokens);
  const watch = await watchPage(page);
  const writes = await writesSeen(page, browserName);
  await page.goto(intakePath(slug, web.id));
  await expect(page.getByText(EMPTY_INTAKE, { exact: true })).toBeVisible();
  await openCreateModal(page);
  await expect(descriptionEditor(page)).toHaveCount(1);
  await nameField(page).fill("Login is slow");
  const login = await submitted(page, writes, web.id);
  expect(login.body).toEqual({
    name: "Login is slow",
    description_html: NO_DESCRIPTION,
    priority: "none",
    label_ids: [],
    assignee_ids: [],
  });
  await expectSelected(page, slug, web.id, login.issue, [login.issue], 1);
  expect(await intakeRowOf(db, login.issue.id)).toEqual(pendingOf(bob.id));
  expect(await submittedOf(db, login.issue.id)).toEqual({
    submitter: bob.id,
    target_date: null,
    assignee_ids: [],
    label_ids: [],
  });
  expect(
    writes.map(({ write }) => write),
    "his writes: the submission alone"
  ).toEqual([`POST ${submitPath(web.id)}`]);
  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], []]);
  await expectQuietConsole(page, watch);

  // gus, a guest, in his browser: Typo on the home page, low; he sees his own work items alone.
  const other = await anotherBrowser(browser, baseURL ?? "", gus.tokens);
  try {
    const guest = other.page;
    const seen = await watchPage(guest);
    const sent = await writesSeen(guest, browserName);
    const reads = recordApi(guest);
    await guest.goto(intakePath(slug, web.id));
    await expect(guest.getByText(EMPTY_INTAKE, { exact: true })).toBeVisible();
    await openCreateModal(guest);
    await expect(descriptionEditor(guest)).toHaveCount(1);
    await nameField(guest).fill("Typo on the home page");
    await modalButton(guest, "Priority: None").click();
    await guest.getByRole("option", { name: "Low", exact: true }).click();
    const typo = await submitted(guest, sent, web.id);
    expect(typo.body).toEqual({
      name: "Typo on the home page",
      description_html: NO_DESCRIPTION,
      priority: "low",
      label_ids: [],
      assignee_ids: [],
    });
    await expectSelected(guest, slug, web.id, typo.issue, [typo.issue], 1);
    expect(await intakeRowOf(db, typo.issue.id)).toEqual(pendingOf(gus.id));
    await dismissToasts(guest);

    // Web's intake turned off by alice while his modal is open, the page's reads answered first: nerve refuses his
    // next submission, the toast says why, and the form stays as he left it.
    await openCreateModal(guest);
    await expect(descriptionEditor(guest)).toHaveCount(1);
    await nameField(guest).fill("Dark mode");
    await expect.poll(() => reads().pending).toBe(0);
    expect((await changeProject(api, alice.token, web.id, { intake_view: false })).status).toBe(200);
    const path = submitPath(web.id);
    const before = submissions(sent, web.id).length;
    const refused = await answerTo(guest, "POST", path, () => modalButton(guest, "Create work item").click());
    expect(refused.status()).toBe(409);
    expect(submissions(sent, web.id).slice(before)).toEqual([
      { name: "Dark mode", description_html: NO_DESCRIPTION, priority: "none", label_ids: [], assignee_ids: [] },
    ]);
    await expect(
      toast(guest, "Intake is turned off in this project. Turn it on to submit or triage work items.")
    ).toBeVisible();
    await expect(createModal(guest)).toHaveCount(1);
    await expect(nameField(guest)).toHaveValue("Dark mode");
    expect(await intakeNamesOf(db, web.id)).toEqual(["Login is slow", "Typo on the home page"]);
    expect(
      sent.map(({ write }) => write),
      "his writes: the two submissions alone"
    ).toEqual([`POST ${path}`, `POST ${path}`]);
    expect([seen.apiFailures, seen.oldApiRequests, seen.pageErrors]).toEqual([[`409 POST ${path}`], [], []]);
    await expectQuietConsole(guest, seen, { errors: failedLoads(browserName, 409) });
  } finally {
    await other.close();
  }
});
