import type { Locator, Page } from "@playwright/test";

import {
  addProjectMembers,
  changeRole,
  createProject,
  createWorkspace,
  inviteAndAccept,
  slugFor,
} from "../../fixtures/api";
import { expectStates, statesOfANewProject, type StateRow } from "../../fixtures/assert/project";
import { accountId, bearer, emailFor, login } from "../../fixtures/auth";
import { expectQuietConsole, watchPage } from "../../fixtures/browser";
import {
  announced,
  closedByEscape,
  closedWithin,
  enabledWithin,
  modalDialog,
  pageBehindHidden,
  sentWithin,
} from "../../fixtures/dialogs";
import { dragOnto, tabCycle } from "../../fixtures/project-pages";
import { registerOnboarded, sentHeld, sentTo } from "../../fixtures/settings-pages";
import { createState, stateIds } from "../../fixtures/states";
import { expect, test } from "../../fixtures/test";
import { anotherBrowser } from "../../fixtures/workspace-pages";

// P6's page version (M3 design 2, 3.17, 7.6): a project's states on its settings page, as its admin changes them, and
// as its member and a workspace admin who is its member see them. The API version is p6-states.spec.ts.

/** The row of the state of name on page's state settings. */
function stateRow(page: Page, name: string): Locator {
  return page.locator("div.group", { has: page.getByRole("heading", { name, exact: true, level: 6 }) });
}

/** The heading of the group named name on page's state settings, which opens and closes its list. */
function groupHeading(page: Page, name: string): Locator {
  return page.getByRole("button", { name, exact: true });
}

/** The button beside the heading of the group named name that adds a state to it, named by the group. */
function addTo(page: Page, name: string): Locator {
  return page.getByRole("button", { name: `Add a state to ${name}`, exact: true });
}

/** The form page shows to create or change a state: one at a time. */
function stateForm(page: Page): Locator {
  return page.locator("div.rounded-sm.p-3", { has: page.locator("#name") });
}

/** rows with the state of name changed as to says. */
function changed(rows: StateRow[], name: string, to: Partial<StateRow>): StateRow[] {
  return rows.map((s) => (s.name === name ? { ...s, ...to } : s));
}

test("P6 (page): the admin creates Review in the started group, its form gone through by Tab and held until nerve answers; a name taken is said under the name; he recolours Review, drags it before In Progress and makes it the default, Backlog the default no longer; the default and a group's only state offer no deletion; a deletion nerve refuses leaves its dialog open, one under way cannot be dismissed, and In Progress is deleted; its member sees the states and can change none, a workspace admin who is its member can", async ({
  api,
  baseURL,
  browser,
  db,
  signedInPage,
}, testInfo) => {
  const adminEmail = emailFor(testInfo, "admin");
  const memEmail = emailFor(testInfo, "mem");
  const wandaEmail = emailFor(testInfo, "wanda");
  const [admin, mem, wanda] = await Promise.all([
    registerOnboarded(api, adminEmail),
    registerOnboarded(api, memEmail),
    registerOnboarded(api, wandaEmail),
  ]);
  const slug = slugFor(testInfo);
  await createWorkspace(api, admin.access_token, { name: "Acme", slug });
  // mem and wanda members of acme and of Web; wanda then acme's admin, Web's member still
  await inviteAndAccept(api, admin.access_token, slug, { email: memEmail, token: mem.access_token }, 15);
  await inviteAndAccept(api, admin.access_token, slug, { email: wandaEmail, token: wanda.access_token }, 15);
  const web = await createProject(api, admin.access_token, slug, { name: "Web", identifier: "WEB" });
  const wandaId = await accountId(api, wanda.access_token);
  await addProjectMembers(api, admin.access_token, web.id, [
    { member_id: await accountId(api, mem.access_token), role: 15 },
    { member_id: wandaId, role: 15 },
  ]);
  await changeRole(api, admin.access_token, slug, wandaId, 20);
  const listed = `/api/v0/projects/${web.id}/states`;
  const statesPage = `/${slug}/settings/projects/${web.id}/states`;
  const inProgressId = (await stateIds(api, admin.access_token, web.id))["In Progress"];
  if (!inProgressId) throw new Error("Web was made without In Progress");
  let rows = statesOfANewProject(adminEmail);

  const page = await signedInPage(admin);
  const watch = await watchPage(page);
  await page.goto(statesPage);
  await expect(stateRow(page, "In Progress")).toBeVisible();
  await Promise.all(
    ["Backlog", "Unstarted", "Started", "Completed", "Canceled"].map((name) =>
      expect(groupHeading(page, name), name).toHaveAttribute("aria-expanded", "true")
    )
  );

  // A group's heading closes its list and opens it again from the keyboard: Enter closes the completed group's, Done's
  // row gone; Space opens it, the row back.
  const completed = groupHeading(page, "Completed");
  await completed.focus();
  await page.keyboard.press("Enter");
  await expect(completed).toHaveAttribute("aria-expanded", "false");
  await expect(stateRow(page, "Done")).toHaveCount(0);
  await page.keyboard.press("Space");
  await expect(completed).toHaveAttribute("aria-expanded", "true");
  await expect(stateRow(page, "Done")).toBeVisible();

  // Review, in the started group: Tab goes through the form in its order from its colour's button, named and marked
  // as Shift+Tab reaches it; the form waits for nerve, its buttons disabled, Cancel too; it closes once nerve has made
  // the state, last of its project's.
  await addTo(page, "Started").click();
  await expect(page.locator("#name")).toBeFocused();
  await page.locator("#name").fill("Review");
  const colour = stateForm(page).getByRole("button", { name: "Color", exact: true });
  await page.keyboard.press("Shift+Tab");
  await expect(colour).toBeFocused();
  await expect(colour).not.toHaveCSS("outline-style", "none");
  expect(await tabCycle(page, stateForm(page), 4)).toEqual(["name", "description", "Create", "Cancel"]);
  const creation = await sentHeld(page, "POST", listed, () =>
    stateForm(page).getByRole("button", { name: "Create" }).click()
  );
  expect(creation.body).toEqual({ name: "Review", description: "", color: "#f59e0b", group: "started" });
  expect(await enabledWithin(stateForm(page).getByRole("button", { name: "Creating" }))).toBe(false);
  expect(await enabledWithin(stateForm(page).getByRole("button", { name: "Cancel" }))).toBe(false);
  expect((await creation.release()).status()).toBe(201);
  const reviewId = (await stateIds(api, admin.access_token, web.id)).Review;
  if (!reviewId) throw new Error("nerve made no Review");
  await expect(stateRow(page, "Review")).toBeVisible();
  await expect(stateForm(page)).toHaveCount(0);
  rows = [
    ...rows,
    {
      name: "Review",
      color: "#f59e0b",
      group: "started",
      sequence: 70000,
      default: false,
      deleted: false,
      by: adminEmail,
    },
  ];
  await expectStates(db, web.id, rows);

  // Todo again, in the unstarted group: said under the name, the form open with what it holds.
  await addTo(page, "Unstarted").click();
  await page.locator("#name").fill("Todo");
  const taken = await sentTo(page, "POST", listed, () =>
    stateForm(page).getByRole("button", { name: "Create" }).click()
  );
  expect(taken.answer.status()).toBe(409);
  await expect(stateForm(page).getByText("A state of this project already has this name.")).toBeVisible();
  await expect(page.locator("#name")).toHaveValue("Todo");
  // the name has the focus, and is read invalid, its message its description
  await expect(page.locator("#name")).toBeFocused();
  await expect(page.locator("#name")).toHaveAttribute("aria-invalid", "true");
  await expect(page.locator("#name")).toHaveAccessibleDescription("A state of this project already has this name.");
  await stateForm(page).getByRole("button", { name: "Cancel" }).click();

  // Review recoloured, its row's Edit reached by Tab, its row's controls shown while the focus is in it as while the
  // pointer is over it (unseen while neither is), each named by Review, Edit and Delete each marked as Tab reaches it:
  // the fields its form edits.
  const review = stateRow(page, "Review");
  const reviewEdit = review.getByRole("button", { name: "Edit Review", exact: true });
  const reviewDelete = review.getByRole("button", { name: "Delete Review", exact: true });
  await page.mouse.move(0, 0);
  await expect(reviewEdit.locator("xpath=../..")).toHaveCSS("opacity", "0");
  await review.getByRole("button", { name: "Mark as default: Review", exact: true }).focus();
  await page.keyboard.press("Tab");
  await expect(reviewEdit).toBeFocused();
  await expect(reviewEdit.locator("xpath=../..")).toHaveCSS("opacity", "1");
  await expect(reviewEdit).not.toHaveCSS("outline-style", "none");
  await page.keyboard.press("Tab");
  await expect(reviewDelete).toBeFocused();
  await expect(reviewDelete).not.toHaveCSS("outline-style", "none");
  await page.keyboard.press("Shift+Tab");
  await expect(reviewEdit).toBeFocused();
  await page.keyboard.press("Enter");
  await colour.click();
  // its hex field shows where the keyboard is (react-color draws it with no outline of its own)
  const hex = page.locator('input[id^="rc-editable-input"]');
  await hex.focus();
  await expect(hex).not.toHaveCSS("outline-style", "none");
  await page.locator('[title="#0693E3"]').click();
  const recoloured = await sentTo(page, "PATCH", `/api/v0/states/${reviewId}`, () =>
    stateForm(page).getByRole("button", { name: "Update" }).click()
  );
  expect([recoloured.answer.status(), recoloured.body]).toEqual([
    200,
    { name: "Review", color: "#0693e3", description: "" },
  ]);
  rows = changed(rows, "Review", { color: "#0693e3" });

  // Review dragged onto In Progress's top: before it, the first of the started group, a step before In Progress.
  const moved = await sentTo(page, "PATCH", `/api/v0/states/${reviewId}`, () =>
    dragOnto(page, review, review, stateRow(page, "In Progress"), { x: 60, y: 4 })
  );
  expect([moved.answer.status(), moved.body]).toEqual([200, { group: "started", sequence: 20000 }]);
  const order = ["Backlog", "Todo", "Review", "In Progress", "Done", "Cancelled"];
  await expect(page.locator("div.group h6")).toHaveText(order);
  rows = changed(rows, "Review", { sequence: 20000 });

  // Its member drags no state, though the started group holds two: In Progress dragged onto Review's top on his page
  // sends nothing, and the order stays. He signs in anew, each browser a session of its own: nerve rotates a session's
  // refresh token, so a second browser given the same tokens would be signed out once the first had refreshed.
  const member = await anotherBrowser(browser, baseURL ?? "", await login(api, memEmail));
  await member.page.goto(statesPage);
  const memberInProgress = stateRow(member.page, "In Progress");
  expect(
    await sentWithin(member.page, "PATCH", `/api/v0/states/${inProgressId}`, () =>
      dragOnto(member.page, memberInProgress, memberInProgress, stateRow(member.page, "Review"), { x: 60, y: 4 })
    )
  ).toBe(false);
  await expect(member.page.locator("div.group h6")).toHaveText(order);
  await member.close();

  // Review the default, its button busy until nerve answers; Backlog the default no longer; neither offers its
  // deletion, nor Todo, its group's only state.
  const marked = `/api/v0/states/${reviewId}/mark-default`;
  const markReview = () => review.getByRole("button", { name: "Mark as default" }).click();
  const marking = await sentHeld(page, "POST", marked, markReview);
  expect(await enabledWithin(review.getByRole("button", { name: "Marking as default" }))).toBe(false);
  expect((await marking.release()).status()).toBe(204);
  await expect(review.getByRole("button", { name: "Default", exact: true })).toBeDisabled();
  await expect(stateRow(page, "Backlog").getByRole("button", { name: "Mark as default" })).toBeEnabled();
  await Promise.all(
    ["Review", "Backlog", "Todo"].map((name) =>
      expect(stateRow(page, name).getByRole("button", { name: "Delete" }), name).toBeDisabled()
    )
  );
  rows = changed(changed(rows, "Backlog", { default: false }), "Review", { default: true });
  await expectStates(db, web.id, rows);

  // In Progress made the default meanwhile, elsewhere: nerve refuses its deletion, said in a toast read out beside the
  // dialog, which stays open,
  // Tab going round its two buttons and the page behind it hidden; Cancel closes it, and the page and the focus are
  // back.
  const elsewhere = await api.POST("/api/v0/states/{state_id}/mark-default", {
    params: { path: { state_id: inProgressId } },
    headers: bearer(admin.access_token),
  });
  expect(elsewhere.response.status).toBe(204);
  const inProgress = stateRow(page, "In Progress");
  const deleted = `/api/v0/states/${inProgressId}`;
  const dialog = page.getByRole("dialog");
  await inProgress.getByRole("button", { name: "Delete" }).click();
  // the dialog is named by its title (ModalTitle)
  await expect(modalDialog(page)).toHaveAccessibleName("Delete state");
  await expect(modalDialog(page).getByText(/^Are you sure you want to delete state In Progress\?/)).toBeVisible();
  const refused = await sentTo(page, "DELETE", deleted, () => dialog.getByRole("button", { name: "Delete" }).click());
  expect(refused.answer.status()).toBe(409);
  const refusal = page.getByText("The default state cannot be deleted. Make another state the default first.");
  await expect(refusal).toBeVisible();
  expect(await announced(refusal)).toBe(true);
  expect(await closedWithin(page)).toBe(false);
  await dialog.getByRole("button", { name: "Cancel" }).focus();
  expect(await tabCycle(page, modalDialog(page), 4)).toEqual(["Delete", "Cancel"]);
  expect(await pageBehindHidden(page)).toBe(true);
  await dialog.getByRole("button", { name: "Cancel" }).click();
  expect(await closedWithin(page)).toBe(true);
  expect(await pageBehindHidden(page)).toBe(false);
  await expect(inProgress.getByRole("button", { name: "Delete" })).toBeFocused();

  // Reloaded, the page shows In Progress the default, as nerve holds it; Review made the default again. In Progress
  // deleted, its dialog held while nerve answers: Escape does not dismiss it, Cancel is disabled; it closes once nerve
  // has, Review left in the started group.
  await page.reload();
  await expect(inProgress.getByRole("button", { name: "Default", exact: true })).toBeDisabled();
  expect((await sentTo(page, "POST", marked, markReview)).answer.status()).toBe(204);
  await inProgress.getByRole("button", { name: "Delete" }).click();
  const deletion = await sentHeld(page, "DELETE", deleted, () =>
    dialog.getByRole("button", { name: "Delete" }).click()
  );
  expect(await closedByEscape(page)).toBe(false);
  expect(await enabledWithin(dialog.getByRole("button", { name: "Cancel" }))).toBe(false);
  expect((await deletion.release()).status()).toBe(204);
  expect(await closedWithin(page)).toBe(true);
  await expect(inProgress).toHaveCount(0);
  rows = changed(rows, "In Progress", { deleted: true });
  await expectStates(db, web.id, rows);
  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([
    [`409 POST ${listed}`, `409 DELETE ${deleted}`],
    [],
    [],
  ]);
  await expectQuietConsole(page, watch, {
    errors: Array(2).fill("Failed to load resource: the server responded with a status of 409 (Conflict)"),
  });

  // A name nerve took through its API, longer than the page let one type (nerve takes up to 255 characters): one
  // character deleted, then Enter and Tab, it is not read invalid. Its rules are the form's and nerve's alone: an
  // input's own maximum, which Base UI takes on Enter, would read it invalid with no message.
  const docs = await createProject(api, admin.access_token, slug, { name: "Docs", identifier: "DOCS" });
  const longName = `Review ${"x".repeat(113)}`;
  await createState(api, admin.access_token, docs.id, { name: longName, color: "#f59e0b", group: "started" });
  await page.goto(`/${slug}/settings/projects/${docs.id}/states`);
  await page.getByRole("button", { name: `Edit ${longName}`, exact: true }).click();
  const nameField = page.locator("#name");
  await expect(nameField).toHaveValue(longName);
  await nameField.press("End");
  await nameField.press("Backspace");
  await nameField.press("Enter");
  await nameField.press("Tab");
  await expect(nameField).not.toHaveAttribute("aria-invalid");

  // Its member sees the states and no control that changes them; a workspace admin who is its member has them all.
  const seenBy = async (who: string, tokens: typeof mem, controls: boolean) => {
    const other = await anotherBrowser(browser, baseURL ?? "", tokens);
    const otherWatch = await watchPage(other.page);
    await other.page.goto(statesPage);
    await expect(stateRow(other.page, "Review"), who).toBeVisible();
    await expect(addTo(other.page, "Started"), who).toBeEnabled({ enabled: controls });
    await expect(other.page.getByRole("button", { name: /^Mark as default: / }), who).toHaveCount(controls ? 4 : 0);
    await expect(other.page.getByRole("button", { name: /^Edit / }), who).toHaveCount(controls ? 5 : 0);
    expect([otherWatch.apiFailures, otherWatch.pageErrors], who).toEqual([[], []]);
    await expectQuietConsole(other.page, otherWatch);
    await other.close();
  };
  await seenBy("its member", mem, false);
  await seenBy("the workspace admin who is its member", wanda, true);
});
