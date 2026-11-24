/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { expect, type Locator, type Page, type TestInfo } from "@playwright/test";

import type { ApiWrite } from "./intake-pages";
import { focusedName } from "./intake-focus";
import type { Issue } from "./issues";
import { answerTo } from "./settings-pages";

// The create modal of a project's intake (M7 P5b spec 2.1, 2.8), as a person uses it from the intake page's header:
// its dialog, what a story finds in it by role and name, a due day picked in its calendar, and a submission as it
// leaves the page. The intake page itself, its rows, its writes seen and its world are intake-pages.ts's; the toasts,
// Tab stops and recorded requests cycle-pages.ts's; what has the focus intake-focus.ts's.

/** The text of the intake page's open tab while it lists nothing. */
export const EMPTY_INTAKE = "Log Intake requests";

/** The create modal, named by its title. */
export function createModal(page: Page): Locator {
  return page.getByRole("dialog", { name: "Create intake work item", exact: true });
}

/** The modal's name field, named Title. */
export function nameField(page: Page): Locator {
  return createModal(page).getByRole("textbox", { name: "Title", exact: true });
}

/** The modal's description: its editor, once built (the editor's chunk loaded). */
export function descriptionEditor(page: Page): Locator {
  return createModal(page).locator('[contenteditable="true"]');
}

/** The button of the modal named name ("Discard", "Create work item", "Add assignees", …). */
export function modalButton(page: Page, name: string): Locator {
  return createModal(page).getByRole("button", { name, exact: true });
}

/** The modal's "Create more" switch. */
export function createMore(page: Page): Locator {
  return createModal(page).getByRole("switch", { name: "Create more", exact: true });
}

/**
 * The confirmation the modal asks before it loses what was typed: the dialog that says so, found by its text so that
 * a story checks the name it is given (its title, through ModalTitle).
 */
export function discardConfirmation(page: Page): Locator {
  return page.getByRole("dialog").filter({ hasText: "What you entered is not saved." });
}

/** The intake page header's button that opens the modal. */
export function addWorkItem(page: Page): Locator {
  return page.getByRole("button", { name: "Add work item", exact: true });
}

/** Opens the modal by the pointer from the intake page's header, and waits for its name field. */
export async function openCreateModal(page: Page): Promise<void> {
  await addWorkItem(page).click();
  await expect(createModal(page)).toHaveCount(1);
  await expect(nameField(page)).toBeVisible();
}

/** The address of the submission of a work item to the intake of the project of projectId. */
export function submitPath(projectId: string): string {
  return `/api/v0/projects/${projectId}/intake-issues`;
}

/** The bodies of the submissions to the intake of the project of projectId among writes (writesSeen's), in order. */
export function submissions(writes: readonly ApiWrite[], projectId: string): unknown[] {
  return writes.filter(({ write }) => write === `POST ${submitPath(projectId)}`).map(({ body }) => body);
}

/**
 * Submits the modal by a press of its button and resolves, once nerve has answered it with the work item made (201),
 * with the body page sent, as writes (writesSeen's) saw it leave, and that work item.
 */
export async function submitted(
  page: Page,
  writes: readonly ApiWrite[],
  projectId: string
): Promise<{ body: unknown; issue: Issue }> {
  const before = submissions(writes, projectId).length;
  const answer = await answerTo(page, "POST", submitPath(projectId), () =>
    modalButton(page, "Create work item").click()
  );
  expect(answer.status(), "the submission's answer").toBe(201);
  return { body: submissions(writes, projectId)[before], issue: (await answer.json()) as Issue };
}

/**
 * Picks day, YYYY-MM-DD, as the modal's due date in the calendar its button opens (by the calendar's year and month);
 * the calendar closes as the day is picked.
 */
export async function pickDueDay(page: Page, day: string): Promise<void> {
  await modalButton(page, "Add due date").click();
  const calendar = page.getByRole("dialog", { name: "Add due date", exact: true });
  const [year = "", month = ""] = day.split("-");
  await calendar.getByRole("combobox", { name: "Choose the Year" }).selectOption(year);
  await calendar.getByRole("combobox", { name: "Choose the Month" }).selectOption(String(Number(month) - 1));
  await calendar.locator(`[data-day="${day}"] button`).click();
  await expect(calendar).toHaveCount(0);
}

/**
 * Resolves with whether the focus is within the create modal, on its dialog itself or on something in it, and so not
 * on the page's body, nor in the page behind it: the modal is found by its name, so that a confirmation over it, or
 * one leaving, does not count.
 */
export async function focusInModal(page: Page): Promise<boolean> {
  return createModal(page).evaluate((dialog) => dialog.contains(dialog.ownerDocument.activeElement));
}

/**
 * Records in testInfo's annotations, for the ledger, what has page's focus after step (focusedName): where it lands
 * differs between the engines, which a story checks only to be within the modal.
 */
export async function noteFocus(page: Page, testInfo: TestInfo, step: string): Promise<void> {
  testInfo.annotations.push({ type: "focus", description: `${step}: ${await focusedName(page)}` });
}
