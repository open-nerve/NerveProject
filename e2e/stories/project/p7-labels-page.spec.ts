import type { Locator, Page } from "@playwright/test";

import {
  addProjectMembers,
  changeRole,
  createLabel,
  createProject,
  createWorkspace,
  inviteAndAccept,
  slugFor,
  type Api,
  type Label,
} from "../../fixtures/api";
import { expectLabels } from "../../fixtures/assert/project";
import { accountId, bearer, emailFor } from "../../fixtures/auth";
import { EMOJI_CHECK_WARNING, expectQuietConsole, watchPage } from "../../fixtures/browser";
import {
  announced,
  closedByEscape,
  closedWithin,
  enabledWithin,
  modalDialog,
  pageBehindHidden,
} from "../../fixtures/dialogs";
import { dragOnto, labelRow, tabCycle } from "../../fixtures/project-pages";
import { registerOnboarded, sentHeld, sentTo } from "../../fixtures/settings-pages";
import { expect, test } from "../../fixtures/test";
import { anotherBrowser } from "../../fixtures/workspace-pages";

// P7's page version (M3 design 2, 3.16, 7.6): a project's labels on its settings page, as its admin changes them, and
// as its member and a workspace admin who is its member see them. The API version is p7-labels.spec.ts.

/**
 * The label of name on page as a place to drop another on, which tells where by the point: its row, the labels under
 * it and the lines that show where a label dropped goes; its top quarter puts the label dropped before it, its middle
 * under it. That is the nearest div around its row with no class.
 */
function dropZone(page: Page, name: string): Locator {
  return labelRow(page, name).locator("xpath=ancestor::div[not(@class)][1]");
}

/** Drags the label of name on page by its grip onto the label of onto, at its point at (its middle when left out). */
function dragLabel(page: Page, name: string, onto: string, at?: { x: number; y: number }): Promise<void> {
  const row = labelRow(page, name);
  return dragOnto(page, row, row.locator("button.cursor-grab"), dropZone(page, onto), at);
}

/** The inline form page shows to create or change a label: one at a time; the page scrolls to it. */
function labelForm(page: Page): Locator {
  return page.locator("div.scroll-m-8", { has: page.locator("#labelName") });
}

/** The labels of the project of projectId, as the caller of token lists them: the label of a name. */
async function labelsOf(api: Api, token: string, projectId: string): Promise<(name: string) => Label> {
  const { data } = await api.GET("/api/v0/projects/{project_id}/labels", {
    params: { path: { project_id: projectId } },
    headers: bearer(token),
  });
  const labels = data?.data ?? [];
  return (name) => {
    const label = labels.find((l) => l.name === name);
    if (!label) throw new Error(`${projectId} has no label ${name}`);
    return label;
  };
}

test("P7 (page): the admin creates Bug, its form gone through by Tab and held until nerve answers, and Feature; bug is said taken under the name; he drags UI under Bug, which sends its parent alone, renames Bug Defect and drags Feature first; Feature put under Defect meanwhile, a third level the page asks for is said refused; its member sees the labels and can change none, a group's neither, a workspace admin who is its member can; a deletion of Docs, deleted meanwhile, is refused and leaves its dialog open; he takes UI out of Defect's group, which sends no parent; Defect is deleted, its dialog held, with Feature under it", async ({
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
  // UI and Docs at the top, at 65535 and 75535, without a colour
  const ui = await createLabel(api, admin.access_token, web.id, { name: "UI" });
  const docs = await createLabel(api, admin.access_token, web.id, { name: "Docs" });
  const listed = `/api/v0/projects/${web.id}/labels`;

  const page = await signedInPage(admin);
  const watch = await watchPage(page);
  await page.goto(`/${slug}/settings/projects/${web.id}/labels`);
  await expect(page.locator("h6")).toHaveText(["UI", "Docs"]);

  // The sidebar's way back to the project, an icon button of the app's (propel's), named and marked as Shift+Tab
  // reaches it from the first of the sidebar's links, its ring set off from it as on each of the app's buttons.
  const back = page.getByRole("button", { name: "Go back", exact: true });
  await page.getByRole("link", { name: "General", exact: true }).focus();
  await page.keyboard.press("Shift+Tab");
  await expect(back).toBeFocused();
  await expect(back).not.toHaveCSS("outline-style", "none");
  await expect(back).toHaveCSS("outline-offset", "2px");

  // Bug: the form opens on the name (its effect, which may run once the form shows), and Tab goes through it in its
  // order from its colour's button, named and marked as Shift+Tab reaches it, to Add, a button of the app's (propel's)
  // marked too, its ring set off from its fill; the form waits for nerve, its buttons disabled, Cancel too; it closes
  // once nerve has made the label, at the top, last of the project's. Feature after it.
  await page.getByRole("button", { name: "Add label" }).click();
  await expect(page.locator("#labelName")).toBeFocused();
  await page.locator("#labelName").fill("Bug");
  const colour = labelForm(page).getByRole("button", { name: "Color", exact: true });
  const add = labelForm(page).getByRole("button", { name: "Add" });
  await page.keyboard.press("Shift+Tab");
  await expect(colour).toBeFocused();
  await expect(colour).not.toHaveCSS("outline-style", "none");
  expect(await tabCycle(page, labelForm(page), 3)).toEqual(["labelName", "Cancel", "Add"]);
  await expect(add).toBeFocused();
  await expect(add).not.toHaveCSS("outline-style", "none");
  await expect(add).toHaveCSS("outline-offset", "2px");
  const creation = await sentHeld(page, "POST", listed, () => add.click());
  expect(creation.body).toEqual({ name: "Bug", color: expect.stringMatching(/^#/) });
  expect(await enabledWithin(labelForm(page).getByRole("button", { name: "Adding" }))).toBe(false);
  expect(await enabledWithin(labelForm(page).getByRole("button", { name: "Cancel" }))).toBe(false);
  expect((await creation.release()).status()).toBe(201);
  await expect(page.locator("h6")).toHaveText(["UI", "Docs", "Bug"]);
  await expect(labelForm(page)).toHaveCount(0);
  await page.getByRole("button", { name: "Add label" }).click();
  await page.locator("#labelName").fill("Feature");
  const created = await sentTo(page, "POST", listed, () =>
    labelForm(page).getByRole("button", { name: "Add" }).click()
  );
  expect(created.answer.status()).toBe(201);

  // bug: said under the name (the line after the form), the form open with what it holds.
  await page.getByRole("button", { name: "Add label" }).click();
  await page.locator("#labelName").fill("bug");
  const taken = await sentTo(page, "POST", listed, () => labelForm(page).getByRole("button", { name: "Add" }).click());
  expect(taken.answer.status()).toBe(409);
  await expect(labelForm(page).locator("xpath=following-sibling::p[1]")).toHaveText(
    "A label of this project already has this name, in this case or another."
  );
  await expect(page.locator("#labelName")).toHaveValue("bug");
  await labelForm(page).getByRole("button", { name: "Cancel" }).click();
  const named = await labelsOf(api, admin.access_token, web.id);
  const [bug, feature] = [named("Bug"), named("Feature")];

  // UI dragged onto Bug's middle: under it, which has no label under it, so the page sends UI's parent alone.
  const under = await sentTo(page, "PATCH", `/api/v0/labels/${ui.id}`, () => dragLabel(page, "UI", "Bug"));
  expect([under.answer.status(), under.body]).toEqual([200, { parent_id: bug.id }]);
  await expect(page.locator("h6")).toHaveText(["Docs", "Bug", "UI", "Feature"]);

  // Bug renamed Defect, from its menu as a group: the fields its form edits. Its disclosure is named by it, and says
  // the labels under it are shown.
  await labelRow(page, "Bug").getByRole("button", { name: "Options" }).click();
  await page.getByRole("menuitem", { name: "Edit label" }).click();
  await page.locator("#labelName").fill("Defect");
  const renamed = await sentTo(page, "PATCH", `/api/v0/labels/${bug.id}`, () =>
    labelForm(page).getByRole("button", { name: "Update" }).click()
  );
  expect([renamed.answer.status(), renamed.body]).toEqual([200, { name: "Defect", color: bug.color }]);
  await expect(page.locator("h6")).toHaveText(["Docs", "Defect", "UI", "Feature"]);
  await expect(page.getByRole("button", { name: "Defect", exact: true, expanded: true })).toBeVisible();

  // Feature dragged onto Docs's top: first of the labels at the top, a step before Docs.
  const first = await sentTo(page, "PATCH", `/api/v0/labels/${feature.id}`, () =>
    dragLabel(page, "Feature", "Docs", { x: 60, y: 2 })
  );
  expect([first.answer.status(), first.body]).toEqual([200, { parent_id: null, sort_order: 65535 }]);
  await expect(page.locator("h6")).toHaveText(["Feature", "Docs", "Defect", "UI"]);

  // Feature put under Defect meanwhile, elsewhere: Docs dropped onto it would make a third level, which nerve refuses
  // and the page says; Docs stays where it was.
  const elsewhere = await api.PATCH("/api/v0/labels/{label_id}", {
    params: { path: { label_id: feature.id } },
    body: { parent_id: bug.id },
    headers: bearer(admin.access_token),
  });
  expect(elsewhere.response.status).toBe(200);
  const third = await sentTo(page, "PATCH", `/api/v0/labels/${docs.id}`, () => dragLabel(page, "Docs", "Feature"));
  expect([third.answer.status(), third.body]).toEqual([422, { parent_id: feature.id }]);
  await expect(
    page.getByText(
      "A label can go only under a label at the top of its project, and a label with labels under it stays at the " +
        "top: labels have two levels."
    )
  ).toBeVisible();
  await expect(page.locator("h6")).toHaveText(["Feature", "Docs", "Defect", "UI"]);

  // Docs's controls, unseen while neither the pointer nor the focus is in its row (a click elsewhere hides them), are
  // shown while Tab goes through them, as while the pointer is over it, each marked as Tab reaches it. Tab comes to its
  // menu from Feature's delete button, past its grip, which only a pointer drags.
  const docsRow = labelRow(page, "Docs");
  const docsControls = docsRow.locator(":scope > div.absolute");
  const docsOptions = docsRow.getByRole("button", { name: "Options" });
  const docsDelete = docsRow.getByRole("button", { name: "Delete" });
  await docsRow.locator("h6").click();
  await page.mouse.move(0, 0);
  await expect(docsControls).toHaveCSS("opacity", "0");
  await labelRow(page, "Feature").getByRole("button", { name: "Delete" }).focus();
  await page.keyboard.press("Tab");
  await expect(docsOptions).toBeFocused();
  await expect(docsOptions).not.toHaveCSS("outline-style", "none");
  await page.keyboard.press("Tab");
  await expect(docsDelete).toBeFocused();
  await expect(docsDelete).not.toHaveCSS("outline-style", "none");
  await expect(docsControls).toHaveCSS("opacity", "1");

  // Its member sees the labels, Defect's group among them, and no control that changes them, no grip, no label to drag;
  // a workspace admin who is its member has them all: a grip, a menu and a drag each, a delete button each but the
  // group.
  const shown = ["Docs", "Defect", "UI", "Feature"];
  const seenBy = async (who: string, tokens: typeof mem, controls: boolean) => {
    const other = await anotherBrowser(browser, baseURL ?? "", tokens);
    const otherWatch = await watchPage(other.page);
    await other.page.goto(`/${slug}/settings/projects/${web.id}/labels`);
    await expect(other.page.locator("h6"), who).toHaveText(shown);
    await expect(other.page.getByRole("button", { name: "Add label" }), who).toHaveCount(controls ? 1 : 0);
    await expect(other.page.locator("button.cursor-grab"), who).toHaveCount(controls ? 4 : 0);
    await expect(other.page.locator('[draggable="true"]:has(h6)'), who).toHaveCount(controls ? 4 : 0);
    await expect(other.page.getByRole("button", { name: "Options" }), who).toHaveCount(controls ? 4 : 0);
    await expect(other.page.getByRole("button", { name: "Delete" }), who).toHaveCount(controls ? 3 : 0);
    expect([otherWatch.apiFailures, otherWatch.pageErrors], who).toEqual([[], []]);
    await expectQuietConsole(other.page, otherWatch, { warnings: [EMOJI_CHECK_WARNING] });
    await other.close();
  };
  await seenBy("its member", mem, false);
  await seenBy("the workspace admin who is its member", wanda, true);

  // Docs deleted meanwhile, elsewhere: nerve refuses its deletion here, said in a toast read out beside the dialog,
  // which stays open, Tab going
  // round its two buttons and the page behind it hidden; Cancel closes it, and the page and the focus are back.
  const gone = await api.DELETE("/api/v0/labels/{label_id}", {
    params: { path: { label_id: docs.id } },
    headers: bearer(admin.access_token),
  });
  expect(gone.response.status).toBe(204);
  const dialog = page.getByRole("dialog");
  await docsDelete.click();
  const refused = await sentTo(page, "DELETE", `/api/v0/labels/${docs.id}`, () =>
    dialog.getByRole("button", { name: "Delete" }).click()
  );
  expect(refused.answer.status()).toBe(404);
  const refusal = page.getByText("The label does not exist, or you cannot see its project.");
  await expect(refusal).toBeVisible();
  expect(await announced(refusal)).toBe(true);
  expect(await closedWithin(page)).toBe(false);
  await dialog.getByRole("button", { name: "Cancel" }).focus();
  expect(await tabCycle(page, modalDialog(page), 4)).toEqual(["Delete", "Cancel"]);
  expect(await pageBehindHidden(page)).toBe(true);
  await dialog.getByRole("button", { name: "Cancel" }).click();
  expect(await closedWithin(page)).toBe(true);
  expect(await pageBehindHidden(page)).toBe(false);
  await expect(docsDelete).toBeFocused();

  // Reloaded, the page shows Docs gone, and Feature under Defect, after UI, of the same place and made before it. UI
  // taken out of Defect's group from its menu: its parent alone, none; at the top, first by its place.
  await page.reload();
  await expect(page.locator("h6")).toHaveText(["Defect", "UI", "Feature"]);
  await labelRow(page, "UI").getByRole("button", { name: "Options" }).click();
  const removed = await sentTo(page, "PATCH", `/api/v0/labels/${ui.id}`, () =>
    page.getByRole("menuitem", { name: "Remove from group" }).click()
  );
  expect([removed.answer.status(), removed.body]).toEqual([200, { parent_id: null }]);
  await expect(page.locator("h6")).toHaveText(["UI", "Defect", "Feature"]);

  // Defect deleted from its menu, its dialog held while nerve answers: Escape does not dismiss it, Cancel is disabled;
  // it closes once nerve has deleted Defect with the label under it.
  await labelRow(page, "Defect").getByRole("button", { name: "Options" }).click();
  await page.getByRole("menuitem", { name: "Delete label" }).click();
  const deletion = await sentHeld(page, "DELETE", `/api/v0/labels/${bug.id}`, () =>
    dialog.getByRole("button", { name: "Delete" }).click()
  );
  expect(await closedByEscape(page)).toBe(false);
  expect(await enabledWithin(dialog.getByRole("button", { name: "Cancel" }))).toBe(false);
  expect((await deletion.release()).status()).toBe(204);
  expect(await closedWithin(page)).toBe(true);
  await expect(page.locator("h6")).toHaveText(["UI"]);
  await expectLabels(db, web.id, [
    { name: "Feature", color: feature.color, parent: "Defect", sort_order: 65535, deleted: true, by: adminEmail },
    { name: "UI", color: "", parent: null, sort_order: 65535, deleted: false, by: adminEmail },
    { name: "Docs", color: "", parent: null, sort_order: 75535, deleted: true, by: adminEmail },
    { name: "Defect", color: bug.color, parent: null, sort_order: 85535, deleted: true, by: adminEmail },
  ]);
  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([
    [`409 POST ${listed}`, `422 PATCH /api/v0/labels/${docs.id}`, `404 DELETE /api/v0/labels/${docs.id}`],
    [],
    [],
  ]);
  await expectQuietConsole(page, watch, {
    warnings: [EMOJI_CHECK_WARNING, EMOJI_CHECK_WARNING],
    errors: [
      "Failed to load resource: the server responded with a status of 409 (Conflict)",
      "Failed to load resource: the server responded with a status of 422 (Unprocessable Entity)",
      "Failed to load resource: the server responded with a status of 404 (Not Found)",
    ],
  });
});
