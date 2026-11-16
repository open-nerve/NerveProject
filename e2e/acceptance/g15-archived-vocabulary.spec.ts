/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import type { Locator, Page } from "@playwright/test";

import { dayFromToday } from "../fixtures/cycle-pages";
import { issueArchived } from "../fixtures/issues";
import { displayPanel } from "../fixtures/issue-filters-page";
import { turnModules } from "../fixtures/module-pages";
import { createModule } from "../fixtures/modules";
import { cycleListWorld, listQueries, shownItems } from "../fixtures/planning-list-pages";
import { openCommands, selectCommand } from "../fixtures/power-k";
import { answerTo, moveWithinApp } from "../fixtures/settings-pages";
import { expect, test } from "../fixtures/test";

// G15's acceptance check of a project's archive of work items by its cycles and modules (M6 P8 spec 2.4, 2.11; P7b
// R8): reached through the project's pages, which have read its cycles and its modules, the archive's filter offers the
// cycle and the module, and its display groups by either; filtered by Sprint and grouped by module, nerve is asked for
// exactly that (the archived work items of Sprint, grouped by module) and the page shows its groups; with the project's
// cycles and modules turned off, the archive offers neither the module's filter nor the grouping by cycle, the two not
// chosen, while the cycle's condition and the grouping by module, chosen, stay. The admin turns the switches in the
// project's settings, as a person does; the world is made through the API.

/** The names of the options the archive's Add filter offers, which Escape then closes. */
async function filtersOffered(page: Page): Promise<string[]> {
  await page.getByRole("button", { name: "Add filter", exact: true }).click();
  const options = page.getByRole("option");
  await expect(options.first()).toBeVisible();
  const names = await options.allTextContents();
  await page.keyboard.press("Escape");
  await expect(options).toHaveCount(0);
  return names;
}

/** The display panel's grouping, opened: its section's options, each a button pressed or not. */
async function groupings(page: Page): Promise<Locator> {
  const groupBy = (await displayPanel(page)).getByRole("button", { name: "Group by", exact: true });
  if ((await groupBy.getAttribute("aria-expanded")) !== "true") await groupBy.click();
  return groupBy.locator("../..").locator("button[aria-pressed]");
}

/** The list's groups, in order, each by its header's name with the names of the work items of its rows, sorted. */
function groupsShown(page: Page): Promise<[string, string[]][]> {
  return page.locator('[class~="group/list-header"] > button[aria-expanded]').evaluateAll((headers) =>
    headers.map((header): [string, string[]] => [
      header.getAttribute("aria-label") ?? "",
      Array.from(
        header.parentElement?.parentElement?.parentElement?.querySelectorAll('button[aria-label^="Options: "]') ?? []
      )
        .map((row) => row.getAttribute("aria-label")?.slice("Options: ".length) ?? "")
        .toSorted(),
    ])
  );
}

test("G15 (acceptance): Web's archive, reached from its work items: its filter offers Cycle and Module and its display groups by Cycle and Modules; Sprint's archived work items grouped by module are what nerve is asked and the page shows, Old A in Alpha and Old C in Beta; with Web's cycles and modules turned off, the archive offers neither the module's filter nor the grouping by cycle, while Sprint's condition and the grouping by module stay", async ({
  api,
  signedInPage,
}, testInfo) => {
  const w = await cycleListWorld(api, testInfo);
  const { token, web } = w;
  await turnModules(api, token, web, true);
  // Sprint in progress and Next ahead; Alpha and Beta. Old A to Old D, done and archived: A in Sprint and Alpha, B in
  // Next and Beta, C in Sprint and Beta, D in neither. Live is Web's work item that is not archived.
  const next = await w.cycle("Next", { start_date: dayFromToday(20), end_date: dayFromToday(30) });
  const sprint = await w.cycle("Sprint", { start_date: dayFromToday(-3), end_date: dayFromToday(10) });
  const beta = await createModule(api, token, web.id, { name: "Beta" });
  const alpha = await createModule(api, token, web.id, { name: "Alpha" });
  const done = w.states.Done ?? "";
  const archived = async (name: string, cycle: string | null, modules: string[]) => {
    const item = await w.item(name, { state_id: done, ...(cycle ? { cycle_id: cycle } : {}), module_ids: modules });
    expect((await issueArchived(api, token, item.id)).status, `${name} archived`).toBe(200);
  };
  await archived("Old A", sprint.id, [alpha.id]);
  await archived("Old B", next.id, [beta.id]);
  await archived("Old C", sprint.id, [beta.id]);
  await archived("Old D", null, []);
  await w.item("Live");

  // The admin's archive of Web, from its work items by the palette's Go to project archives.
  const page = await signedInPage(w.admin.tokens);
  const queries = listQueries(page, web.id);
  await page.goto(w.projectPath);
  await expect.poll(() => shownItems(page)).toEqual(["Live"]);
  await openCommands(page);
  await selectCommand(page, /^Go to project archives/);
  const archive = `/${w.slug}/projects/${web.id}/archives/issues`;
  await expect(page).toHaveURL(archive);
  await expect.poll(() => shownItems(page)).toEqual(["Old A", "Old B", "Old C", "Old D"]);

  // Its filter offers the cycle and the module, and its display groups by either.
  expect(await filtersOffered(page)).toEqual(expect.arrayContaining(["Cycle", "Module"]));
  await expect(await groupings(page)).toContainText(["Cycle", "Modules"]);
  await page.keyboard.press("Escape");

  // Filtered by Sprint: nerve is asked for Sprint's archived work items, Old A and Old C.
  await page.getByRole("button", { name: "Add filter", exact: true }).click();
  await page.getByRole("option", { name: "Cycle", exact: true }).click();
  await page.getByRole("option", { name: "Sprint", exact: true }).click();
  await page.keyboard.press("Escape");
  await expect.poll(() => shownItems(page)).toEqual(["Old A", "Old C"]);
  // Grouped by module: nerve is asked for them grouped by module, which the groups show.
  await (await groupings(page)).filter({ hasText: /^Modules$/ }).click();
  await page.keyboard.press("Escape");
  const asked = () =>
    queries
      .filter((query) => query.get("archived") === "true")
      .map((query) => [query.get("cycle_id"), query.get("group_by")]);
  await expect.poll(asked).toContainEqual([sprint.id, "module_id"]);
  await expect
    .poll(() => groupsShown(page))
    .toEqual([
      ["Alpha", ["Old A"]],
      ["Beta", ["Old C"]],
    ]);

  // The admin turns Web's cycles and modules off in its settings, and comes back to its archive.
  const turnOff = async (feature: "cycles" | "modules", title: string) => {
    await moveWithinApp(page, `/${w.slug}/settings/projects/${web.id}/features/${feature}`);
    const toggle = page.getByRole("switch", { name: title, exact: true });
    await expect(toggle).toBeChecked();
    expect((await answerTo(page, "PATCH", `/api/v0/projects/${web.id}`, () => toggle.click())).status()).toBe(200);
    await expect(toggle).not.toBeChecked();
  };
  await turnOff("cycles", "Enable cycles");
  await turnOff("modules", "Enable modules");
  await moveWithinApp(page, archive);
  await expect.poll(() => shownItems(page)).toEqual(["Old A", "Old C"]);

  // The two not chosen are not offered: the module's filter, the grouping by cycle. The two chosen stay: Sprint's
  // condition, removable, and the grouping by module, pressed.
  const offered = await filtersOffered(page);
  expect(offered).not.toContain("Cycle");
  expect(offered).not.toContain("Module");
  const shown = await groupings(page);
  await expect(shown.filter({ hasText: /^Cycle$/ })).toHaveCount(0);
  await expect(shown.filter({ hasText: /^Modules$/ })).toHaveAttribute("aria-pressed", "true");
  await page.keyboard.press("Escape");
  await expect(page.getByRole("button", { name: "Cycle", exact: true })).toBeVisible();
  expect(asked().at(-1)).toEqual([sprint.id, "module_id"]);
  await page.getByRole("button", { name: "Remove filter", exact: true }).click();
  await expect.poll(() => shownItems(page)).toEqual(["Old A", "Old B", "Old C", "Old D"]);
  await expect.poll(() => asked().at(-1)).toEqual([null, "module_id"]);
});
