/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import type { Locator, Page, TestInfo } from "@playwright/test";

import { addProjectMembers, createProject, type Api } from "../fixtures/api";
import { cyclesPath, turnCycles } from "../fixtures/cycle-pages";
import { listCycles } from "../fixtures/cycles";
import { modalDialog } from "../fixtures/dialogs";
import { modulesPath, noModulesYet, turnModules } from "../fixtures/module-pages";
import { listModules } from "../fixtures/modules";
import { cycleListWorld } from "../fixtures/planning-list-pages";
import { answerTo } from "../fixtures/settings-pages";
import { expect, test } from "../fixtures/test";

// G1's and G9's acceptance check of the project of a new cycle's and a new module's dialog (M6 P8 spec 2.11; M2
// handoff 5(d); cycles/form.tsx, modules/form.tsx): its project dropdown, opened by the keyboard, its search focused,
// offers the projects whose cycles (modules) the caller may write, the dialog's first; searched and picked by the keys,
// the dialog's project is the one picked, and the cycle (module) is made there, none in the project of the page it was
// opened on, in each engine. The world is made through the API.

/**
 * mem's world (cycleListWorld): Web, whose cycles and modules are on; Ops, the admin's other project, where mem is a
 * member and both are on; Infra, where mem is a member too and both are off, a new project's.
 */
async function projectsWorld(api: Api, testInfo: TestInfo) {
  const w = await cycleListWorld(api, testInfo);
  await turnModules(api, w.token, w.web, true);
  const project = async (name: string, identifier: string) => {
    const made = await createProject(api, w.token, w.slug, { name, identifier });
    await addProjectMembers(api, w.token, made.id, [{ member_id: w.mem.id, role: 15 }]);
    return made;
  };
  const ops = await project("Ops", "OPS");
  await turnCycles(api, w.token, ops, true);
  await turnModules(api, w.token, ops, true);
  const infra = await project("Infra", "INF");
  return { ...w, ops, infra };
}

/**
 * In dialog, opens its project's dropdown by the keyboard, checks the projects it offers, searches Ops and picks it by
 * the keys: the dropdown's button, which then names Ops and has the focus back.
 */
async function pickOps(page: Page, dialog: Locator): Promise<void> {
  const project = (name: string) => dialog.getByRole("button", { name: `Project: ${name}`, exact: true });
  await expect(project("Web")).toBeVisible();
  await project("Web").focus();
  await page.keyboard.press("Enter");
  const search = page.getByRole("combobox", { name: "Search" });
  await expect(search).toBeFocused();
  const options = page.getByRole("listbox").getByRole("option");
  await expect(options).toHaveText(["Web", "Ops"]);
  await page.keyboard.type("Op");
  await expect(options).toHaveText(["Ops"]);
  await page.keyboard.press("ArrowDown");
  await expect(options.first()).toHaveAttribute("data-focus", "");
  await page.keyboard.press("Enter");
  await expect(page.getByRole("listbox")).toHaveCount(0);
  await expect(project("Ops")).toBeFocused();
}

test("G1 (acceptance): mem's new cycle dialog on Web's cycles list: its project, opened by the keyboard with its search focused, offers Web and Ops, not Infra, whose cycles are off; Op typed and Ops picked by the keys, the dialog's project is Ops, and its cycle Held is made in Ops, none in Web", async ({
  api,
  signedInPage,
}, testInfo) => {
  const w = await projectsWorld(api, testInfo);
  const page = await signedInPage(w.mem.tokens);
  await page.goto(cyclesPath(w.slug, w.web.id));
  await expect(page.getByText("Group and timebox your work in Cycles.", { exact: true })).toBeVisible();
  await page.getByRole("button", { name: "Add cycle", exact: true }).click();
  const dialog = modalDialog(page);
  await expect(dialog).toHaveAccessibleName("Create cycle");

  await pickOps(page, dialog);
  await dialog.getByRole("textbox", { name: "Title", exact: true }).fill("Held");
  const made = await answerTo(page, "POST", `/api/v0/projects/${w.ops.id}/cycles`, () =>
    dialog.getByRole("button", { name: "Create cycle", exact: true }).click()
  );
  expect(made.status()).toBe(201);
  await expect(dialog).toHaveCount(0);
  expect((await listCycles(api, w.token, w.ops.id)).map((cycle) => cycle.name)).toEqual(["Held"]);
  expect(await listCycles(api, w.token, w.web.id)).toEqual([]);
});

test("G9 (acceptance): mem's new module dialog on Web's modules list: its project, opened by the keyboard with its search focused, offers Web and Ops, not Infra, whose modules are off; Op typed and Ops picked by the keys, the dialog's project is Ops, and its module Held is made in Ops, none in Web", async ({
  api,
  signedInPage,
}, testInfo) => {
  const w = await projectsWorld(api, testInfo);
  const page = await signedInPage(w.mem.tokens);
  await page.goto(modulesPath(w.slug, w.web.id));
  await expect(noModulesYet(page)).toBeVisible();
  await page.getByRole("button", { name: "Add Module", exact: true }).click();
  const dialog = modalDialog(page);
  await expect(dialog).toHaveAccessibleName("Create Module");

  await pickOps(page, dialog);
  await dialog.getByRole("textbox", { name: "Title", exact: true }).fill("Held");
  const made = await answerTo(page, "POST", `/api/v0/projects/${w.ops.id}/modules`, () =>
    dialog.getByRole("button", { name: "Create Module", exact: true }).click()
  );
  expect(made.status()).toBe(201);
  await expect(dialog).toHaveCount(0);
  expect((await listModules(api, w.token, w.ops.id)).map((module) => module.name)).toEqual(["Held"]);
  expect(await listModules(api, w.token, w.web.id)).toEqual([]);
});
