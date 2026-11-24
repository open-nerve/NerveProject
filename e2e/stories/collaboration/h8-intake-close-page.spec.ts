import type { Page, TestInfo } from "@playwright/test";

import { changeProject } from "../../fixtures/api";
import { expectQuietConsole, watchPage } from "../../fixtures/browser";
import { dismissToasts, failedLoads, recordApi, tabKey, tabStops, toast } from "../../fixtures/cycle-pages";
import { closedByBackdropClick, closedByEscape, sentWithin } from "../../fixtures/dialogs";
import { holdEditorImplementation } from "../../fixtures/editor-chunks";
import {
  EMPTY_INTAKE,
  addWorkItem,
  createMore,
  createModal,
  descriptionEditor,
  discardConfirmation,
  focusInModal,
  modalButton,
  nameField,
  noteFocus,
  openCreateModal,
  submitPath,
} from "../../fixtures/intake-create";
import { focusedName } from "../../fixtures/intake-focus";
import { intakePath, intakeWorld } from "../../fixtures/intake-pages";
import { answerTo, expectPopupBesideButton, holdAnswer } from "../../fixtures/settings-pages";
import { expect, test } from "../../fixtures/test";

// H8's page version, closing the create modal of a project's intake (M7 design 2, 3.5; P5b spec 2.1, 2.8, R4, R7,
// R21, R28): one rule for Escape, the backdrop and Discard. A form with nothing typed closes, its description's editor
// built or not; one typed in asks first (the discard confirmation, whose Escape closes it alone); while the editor's
// chunk is on its way, nothing is submitted. While a submission is out the modal does not close, and the focus stays
// in it, through Escape, the backdrop and nerve's refusal. Escape in the assignees' list and the due date's calendar
// closes them alone; the assignees open beside their button by the keyboard; Tab goes through the description both
// ways. The first story holds the editor's chunk from before the page loads, Web's intake empty (re-95 I2): a row
// would be selected, and its detail would load the same chunk first, which would then hold nothing; the second has
// the editor built. The submissions are h8-intake-submit-page.spec.ts's.

/** Checks that the description editor's chunk, held, was asked for and that the modal shows no editor: it is not built. */
async function expectEditorHeld(page: Page, held: readonly string[]): Promise<void> {
  // held never empties once asked: the hold's evidence is this together with no [contenteditable="true"] below
  await expect.poll(() => held.length, { message: "the editor's chunk asked for, and held" }).toBeGreaterThan(0);
  await expect(descriptionEditor(page)).toHaveCount(0);
}

/**
 * Opens the modal, types a name, and checks that Escape, the backdrop and Discard each ask first, check (the editor
 * held, or built) holding before each: the confirmation is named by its title, and closing it keeps the form, the
 * focus landing in the modal once the confirmation has left (on the dialog for Escape, Headless UI having blurred the
 * name, and for the backdrop, pressed outside; back on Discard for Discard). Escape's confirmation is cancelled; the
 * backdrop's is closed by its Escape, which it alone claims; Discard's is cancelled, then confirmed: both close, the
 * focus back on Add work item, and the modal opened again is empty.
 */
async function expectEachAsks(page: Page, testInfo: TestInfo, check: () => Promise<void>): Promise<void> {
  const name = nameField(page);
  const confirmation = discardConfirmation(page);
  const discard = modalButton(page, "Discard");
  const asked = async () => expect(confirmation).toHaveAccessibleName("Discard this work item?");
  const left = async (step: string) => {
    await expect(confirmation).toHaveCount(0);
    await expect(createModal(page)).toHaveCount(1);
    await expect(name).toHaveValue("Crash on save");
    // the focus passes through the body while the confirmation leaves (its 200 ms transition), then lands
    await expect.poll(() => focusInModal(page), { message: `the focus in the modal after ${step}` }).toBe(true);
    await noteFocus(page, testInfo, step);
  };
  await openCreateModal(page);
  await name.fill("Crash on save");
  await check();
  await page.keyboard.press("Escape");
  await asked();
  await confirmation.getByRole("button", { name: "Cancel", exact: true }).click();
  await left("the confirmation's Cancel, Escape having opened it");
  await check();
  await page.mouse.click(1, 1);
  await asked();
  await page.keyboard.press("Escape");
  await left("the confirmation's Escape, the backdrop having opened it");
  await check();
  await discard.click();
  await asked();
  await confirmation.getByRole("button", { name: "Cancel", exact: true }).click();
  await left("the confirmation's Cancel, Discard having opened it");
  await expect(discard).toBeFocused();
  await check();
  await discard.click();
  await confirmation.getByRole("button", { name: "Discard", exact: true }).click();
  await expect(createModal(page)).toHaveCount(0);
  await expect(confirmation).toHaveCount(0);
  await expect(addWorkItem(page)).toBeFocused();
  await openCreateModal(page);
  await expect(name).toHaveValue("");
}

// In WebKit, any page.route active while the editor's chunk is imported (a dynamic import) makes WebKit leave the
// chunks it preloaded for it unused and fetch them again, and it warns of each about 3 s later: a story that holds the
// chunk and lasts longer than that cannot keep a quiet console there. This one runs in Chromium and Firefox; the close
// rule with the editor built is the next one's, in WebKit too.
test("H8 (page): with the description editor's chunk held from before the page loads, alice's empty form closes by Escape and by Discard; typed in, Escape, the backdrop and Discard each ask first, the confirmation's Cancel and Escape keeping the form with the focus in it, Discard's on Discard, its Discard closing both, the focus back on Add work item and the form empty again; a submission is refused with the editor not ready, sending nothing; the chunk let through, the empty form closes by the backdrop", async ({
  api,
  browserName,
  db,
  page,
  signedInPage,
}, testInfo) => {
  test.skip(
    browserName === "webkit",
    "In WebKit, any page.route active while the editor's chunk is dynamically imported makes WebKit skip its preloads and fetch them again; the warnings come about 3 s later, so a held-chunk story longer than that cannot keep a quiet console"
  );
  const { held, release } = await holdEditorImplementation(page);
  try {
    const { alice, slug, web } = await intakeWorld(api, testInfo);
    await signedInPage(alice.tokens);
    const watch = await watchPage(page);
    await page.goto(intakePath(slug, web.id));
    await expect(page.getByText(EMPTY_INTAKE, { exact: true })).toBeVisible();
    const name = nameField(page);

    // Empty, the editor not built: Escape closes, and so does Discard; the focus is back on the header's button.
    await openCreateModal(page);
    await expectEditorHeld(page, held);
    expect(await closedByEscape(page)).toBe(true);
    await expect(addWorkItem(page)).toBeFocused();
    await openCreateModal(page);
    await expectEditorHeld(page, held);
    await modalButton(page, "Discard").click();
    await expect(createModal(page)).toHaveCount(0);
    await expect(addWorkItem(page)).toBeFocused();

    // Typed in, each asks first (expectEachAsks), the editor still not built.
    await expectEachAsks(page, testInfo, () => expectEditorHeld(page, held));

    // Named, the editor not built: the submission is refused, said so, and nothing is sent.
    await name.fill("Crash on save");
    await expectEditorHeld(page, held);
    const path = submitPath(web.id);
    expect(await sentWithin(page, "POST", path, () => modalButton(page, "Create work item").click())).toBe(false);
    await expect(toast(page, "Editor is not ready to discard changes")).toBeVisible();
    await expect(createModal(page)).toHaveCount(1);
    await dismissToasts(page);

    // The chunk let through, the editor built: the form emptied again (the editor's own empty paragraph is no change)
    // closes by the backdrop.
    await name.fill("");
    release();
    await expect(descriptionEditor(page)).toHaveCount(1);
    expect(await closedByBackdropClick(page)).toBe(true);
    await expect(addWorkItem(page)).toBeFocused();
    expect(await db.query("SELECT id FROM issues WHERE project_id = $1", [web.id])).toEqual([]);

    expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], []]);
    await expectQuietConsole(page, watch);
  } finally {
    release();
  }
});

test("H8 (page): with the description's editor built, typed in, Escape, the backdrop and Discard each ask first, as with it held; a submission out holds the modal: Escape and the backdrop close nothing and Discard and Create more are disabled, the focus staying in the modal, and after nerve's refusal too; Escape in the assignees' list and in the due date's calendar closes them alone; the assignees open by the keyboard beside their button, the search focused, Escape back on the button; Tab goes from the name through the description and the properties to Create work item and back to the name, and Shift+Tab leaves the description for Continue writing", async ({
  api,
  browserName,
  signedInPage,
}, testInfo) => {
  const { alice, slug, web } = await intakeWorld(api, testInfo);
  const page = await signedInPage(alice.tokens);
  const watch = await watchPage(page);
  const reads = recordApi(page);
  await page.goto(intakePath(slug, web.id));
  await expect(page.getByText(EMPTY_INTAKE, { exact: true })).toBeVisible();
  const dialog = createModal(page);
  const name = nameField(page);
  const description = descriptionEditor(page);

  // Typed in, each asks first (expectEachAsks), the editor built.
  await expectEachAsks(page, testInfo, () => expect(description).toHaveCount(1));
  await expect(description).toHaveCount(1);
  await name.fill("Crash on save");

  // Web's intake turned off, the page's reads answered first, and the submission's answer held as it comes back.
  await expect.poll(() => reads().pending).toBe(0);
  expect((await changeProject(api, alice.token, web.id, { intake_view: false })).status).toBe(200);
  const path = submitPath(web.id);
  const answer = await holdAnswer(page, "POST", path);
  await modalButton(page, "Create work item").click();
  await expect(modalButton(page, "Creating")).toBeDisabled();
  await expect(modalButton(page, "Discard")).toBeDisabled();
  await expect(createMore(page)).toBeDisabled();
  await noteFocus(page, testInfo, "submitted by the pointer");
  const before = await focusedName(page);
  expect(await closedByEscape(page)).toBe(false);
  expect(await focusInModal(page)).toBe(true);
  expect(await focusedName(page), "Escape leaves the focus where it was").toBe(before);
  await noteFocus(page, testInfo, "Escape in flight");
  expect(await closedByBackdropClick(page)).toBe(false);
  expect(await focusInModal(page)).toBe(true);
  await noteFocus(page, testInfo, "backdrop in flight");
  expect((await answerTo(page, "POST", path, answer)).status()).toBe(409);
  const refusal = "Intake is turned off in this project. Turn it on to submit or triage work items.";
  await expect(toast(page, refusal)).toBeVisible();
  await expect(modalButton(page, "Create work item")).toBeEnabled();
  expect(await focusInModal(page)).toBe(true);
  await noteFocus(page, testInfo, "after the refusal");
  await expect(name).toHaveValue("Crash on save");
  await dismissToasts(page);

  // Nested: Escape closes the assignees' list alone, then the due date's calendar alone (R7).
  const assignees = modalButton(page, "Add assignees");
  await assignees.click();
  await expect(page.getByRole("listbox")).toBeVisible();
  await page.keyboard.press("Escape");
  await expect(page.getByRole("listbox")).toHaveCount(0);
  await modalButton(page, "Add due date").click();
  const calendar = page.getByRole("dialog", { name: "Add due date", exact: true });
  await expect(calendar.getByRole("grid")).toBeVisible();
  await page.keyboard.press("Escape");
  await expect(calendar).toHaveCount(0);
  await expect(dialog).toHaveCount(1);
  await expect(discardConfirmation(page)).toHaveCount(0);
  await expect(name).toHaveValue("Crash on save");

  // The assignees by the keyboard (R21): Enter opens the list beside the button, its search focused; Escape back.
  await assignees.focus();
  await page.keyboard.press("Enter");
  await expectPopupBesideButton(page.getByRole("listbox"), assignees);
  await expect(page.getByRole("combobox", { name: "Search", exact: true })).toBeFocused();
  await page.keyboard.press("Escape");
  await expect(page.getByRole("listbox")).toHaveCount(0);
  await expect(assignees).toBeFocused();
  await expect(dialog).toHaveCount(1);

  // The natural order, as found: the editor's container adds one stop before it (Continue writing), and the editor, a
  // textbox with no name, is told by its tag; the dialog keeps the focus, Tab after the last control back at the name.
  const key = tabKey(browserName);
  await name.focus();
  expect(await tabStops(page, dialog, key)).toEqual([
    "Continue writing",
    "DIV",
    "Priority: None",
    "Add assignees",
    "Labels: Crash on save",
    "Add due date",
    "Create more",
    "Discard",
    "Create work item",
    "Title",
  ]);
  await page.keyboard.press(key);
  await page.keyboard.press(key);
  await expect(description).toBeFocused();
  await page.keyboard.press(`Shift+${key}`);
  await expect(modalButton(page, "Continue writing")).toBeFocused();

  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[`409 POST ${path}`], [], []]);
  await expectQuietConsole(page, watch, { errors: failedLoads(browserName, 409) });
});
