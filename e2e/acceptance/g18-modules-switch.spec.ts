/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import type { Page } from "@playwright/test";

import { changeProjectPreferences, readProjectPreferences } from "../fixtures/api";
import { seenBy } from "../fixtures/cycle-pages";
import { modulePath, modulesPath, turnModules } from "../fixtures/module-pages";
import { createModule, createModuleLink, listModuleLinks, listModules } from "../fixtures/modules";
import { cycleListWorld, listQueries, shownItems } from "../fixtures/planning-list-pages";
import { openCommands, selectCommand } from "../fixtures/power-k";
import { answerTo, moveWithinApp } from "../fixtures/settings-pages";
import { expect, test } from "../fixtures/test";

// G18's acceptance check of a project's modules turned off and on again in its settings (M6 design 3.6; P8 spec 2.11;
// P7b R8): off, the project's sidebar and the command palette give no way to its modules, a module's page read by its
// address still shows the module, its sidebar and its link, and the project's work items list shows no modules column
// and offers no module filter, the condition already applied staying, removable, with nerve's answer; on again, the
// way back, the column, the condition's name and the module's page with its work item and its link are all there. The
// admin turns the switch as a person does, in his browser; the world is made through the API.

/**
 * The ways page gives to the project's modules, whose list is at path: whether the sidebar's project links to it (the
 * app renders the sidebar's projects more than once, the other copies off screen or of no size), and the palette's
 * commands to them.
 */
async function waysToModules(page: Page, path: string): Promise<{ sidebar: boolean; palette: string[] }> {
  const sidebar =
    (await page
      .getByRole("link", { name: "Modules", exact: true })
      .and(page.locator(`a[href="${path}"]`))
      .count()) > 0;
  await openCommands(page);
  const palette = await page.getByRole("option", { name: /^(Open a module|Go to modules)/ }).allTextContents();
  await page.keyboard.press("Escape");
  await expect(page.getByRole("combobox")).toHaveCount(0);
  return { sidebar, palette: palette.map((text) => text.replace(/[A-Z]then[A-Z]$/, "")) };
}

test("G18 (acceptance): the admin turns Web's modules off: its sidebar and the palette give no way to them, Alpha's page read by its address still shows Alpha and its link Notes, and Web's list shows no modules column and offers no module filter, Alpha's condition staying with Plan; turned on again, the way back, the column, the condition's Alpha and Alpha's page with Plan and Notes are all there", async ({
  api,
  baseURL,
  browser,
  signedInPage,
}, testInfo) => {
  const w = await cycleListWorld(api, testInfo);
  const { admin, token, web } = w;
  await turnModules(api, token, web, true);
  // Alpha, with its link Notes, holds Plan; Ship is in no module. The admin's list of Web shows the modules column.
  const alpha = await createModule(api, token, web.id, { name: "Alpha" });
  await createModuleLink(api, token, alpha.id, { url: "http://169.254.169.254/g18/notes", title: "Notes" });
  await w.item("Plan", { module_ids: [alpha.id] });
  await w.item("Ship");
  const saved = await readProjectPreferences(api, token, web.id);
  const shown = await changeProjectPreferences(api, token, web.id, {
    display_properties: { ...saved.display_properties, modules: true },
  });
  expect(shown.status, "the admin's list of Web shows the modules column").toBe(200);
  const [list, settings] = [modulesPath(w.slug, web.id), `/${w.slug}/settings/projects/${web.id}/features/modules`];

  const page = await signedInPage(admin.tokens);
  const queries = listQueries(page, web.id);
  await page.goto(w.projectPath);
  await expect.poll(() => shownItems(page)).toEqual(["Plan", "Ship"]);
  const column = page.getByRole("button", { name: /^Modules: Alpha , Plan$/ });
  await expect(column).toBeVisible();
  expect(await waysToModules(page, list)).toEqual({ sidebar: true, palette: ["Open a module", "Go to modules"] });
  // Filtered by Alpha: Plan alone.
  await page.getByRole("button", { name: "Add filter", exact: true }).click();
  await page.getByRole("option", { name: "Module", exact: true }).click();
  await page.getByRole("option", { name: "Alpha", exact: true }).click();
  await page.keyboard.press("Escape");
  await expect.poll(() => shownItems(page)).toEqual(["Plan"]);
  const condition = page.getByRole("button", { name: "Remove filter", exact: true }).locator("..");
  await expect(condition.getByRole("button", { name: "Alpha", exact: true })).toBeVisible();

  // The switch, in Web's settings, turned off or on by the admin, who comes back to Web's list.
  const turn = async (on: boolean) => {
    await moveWithinApp(page, settings);
    const toggle = page.getByRole("switch", { name: "Enable modules", exact: true });
    await expect(toggle).toBeChecked({ checked: !on });
    expect((await answerTo(page, "PATCH", `/api/v0/projects/${web.id}`, () => toggle.click())).status()).toBe(200);
    await expect(toggle).toBeChecked({ checked: on });
    await moveWithinApp(page, w.projectPath);
    await expect(page.getByRole("button", { name: "Add filter", exact: true })).toBeVisible();
  };

  // Off: no way to the modules; Plan's row has no modules column; no module filter is offered, Alpha's condition stays,
  // removable, and nerve's answer lists Plan alone.
  await turn(false);
  await expect.poll(() => shownItems(page)).toEqual(["Plan"]);
  await expect(column).toHaveCount(0);
  expect(await waysToModules(page, list)).toEqual({ sidebar: false, palette: [] });
  await page.getByRole("button", { name: "Add filter", exact: true }).click();
  await expect(page.getByRole("option", { name: "State", exact: true })).toBeVisible();
  await expect(page.getByRole("option", { name: "Module", exact: true })).toHaveCount(0);
  await page.keyboard.press("Escape");
  await expect(condition.getByRole("button", { name: "Module", exact: true })).toBeVisible();
  await expect(condition.getByRole("button", { name: "Remove filter", exact: true })).toBeEnabled();
  expect(queries.at(-1)?.get("module_id")).toBe(alpha.id);

  // Alpha's page, read by its address by mem in a browser of his own: Alpha, its sidebar and its link, its work items
  // the switch's.
  await seenBy(browser, baseURL ?? "", api, w.mem.email, modulePath(w.slug, web.id, alpha.id), async (other) => {
    await expect(other.getByRole("heading", { name: "Alpha", exact: true })).toBeVisible();
    await expect(other.getByRole("link", { name: "Notes", exact: true })).toBeVisible();
    await expect(other.getByRole("heading", { name: "Modules are not enabled for the project." })).toBeVisible();
  });

  // On again: the way back, the column, the condition's Alpha; Alpha's page, by the palette, lists Plan with Notes.
  await turn(true);
  await expect.poll(() => shownItems(page)).toEqual(["Plan"]);
  await expect(column).toBeVisible();
  await expect(condition.getByRole("button", { name: "Alpha", exact: true })).toBeVisible();
  expect(await waysToModules(page, list)).toEqual({ sidebar: true, palette: ["Open a module", "Go to modules"] });
  await openCommands(page);
  await selectCommand(page, /^Open a module/);
  await selectCommand(page, /^Alpha$/);
  await expect(page).toHaveURL(modulePath(w.slug, web.id, alpha.id));
  await expect.poll(() => shownItems(page)).toEqual(["Plan"]);
  await expect(page.getByRole("link", { name: "Notes", exact: true })).toBeVisible();
  expect((await listModules(api, token, web.id)).map((module) => module.name)).toEqual(["Alpha"]);
  expect((await listModuleLinks(api, token, alpha.id)).map((link) => link.title)).toEqual(["Notes"]);
});
