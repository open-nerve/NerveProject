import { createProject, createWorkspace, slugFor } from "../../fixtures/api";
import { expectModules } from "../../fixtures/assert/planning";
import { emailFor } from "../../fixtures/auth";
import { expectQuietConsole, watchPage } from "../../fixtures/browser";
import { toast } from "../../fixtures/cycle-pages";
import { closedWithin, composeWithEnter, composingKey, modalDialog, sentWithin } from "../../fixtures/dialogs";
import { madeModule, modulesPath, turnModules } from "../../fixtures/module-pages";
import { registerOnboarded, sentTo } from "../../fixtures/settings-pages";
import { expect, test } from "../../fixtures/test";

// G9's page version, its new module's name composed by an input method (M4 handoff M6-04; M6 P6a fix F2), as G1's page
// version composes a new cycle's (g1-create-cycle-page.spec.ts). The rest of G9's page version is
// g9-create-module-page.spec.ts; the API version is g9-create-update-module.spec.ts.

test("G9 (page): an Enter that ends an input method's composition in the name of the admin's new module is the input method's: nothing is sent and the dialog stays, holding the name composed; the Enter after the composition sends it, and the dialog closes once nerve has made the module", async ({
  api,
  browserName,
  db,
  signedInPage,
}, testInfo) => {
  const email = emailFor(testInfo, "admin");
  const tokens = await registerOnboarded(api, email);
  const token = tokens.access_token;
  const slug = slugFor(testInfo);
  await createWorkspace(api, token, { name: "Acme", slug });
  const web = await createProject(api, token, slug, { name: "Web", identifier: "WEB" });
  await turnModules(api, token, web, true);
  const listed = `/api/v0/projects/${web.id}/modules`;

  const page = await signedInPage(tokens);
  const watch = await watchPage(page);
  await page.goto(modulesPath(slug, web.id));
  await page.getByRole("button", { name: "Set your first module", exact: true }).click();
  const dialog = modalDialog(page);
  await expect(dialog).toHaveAccessibleName("Create Module");
  const name = dialog.getByRole("textbox", { name: "Title", exact: true });
  await expect(name).toBeFocused();

  // Chromium composes 模块 in the name through its own input method, an Enter ending the composition as an input method
  // gives it, and nothing is sent; Playwright drives no input method in Firefox or WebKit, where 模块 is filled in.
  if (browserName === "chromium") {
    expect(await sentWithin(page, "POST", listed, () => composeWithEnter(page, "模块"))).toBe(false);
  } else {
    await name.fill("模块");
  }
  // In each engine, what an Enter that ends a composition gives the page, in Chromium's and Firefox's order and in
  // Safari's (composingKey): nothing is sent, and the dialog stays, holding 模块.
  expect(await sentWithin(page, "POST", listed, () => composingKey(name, "Enter"))).toBe(false);
  expect(await closedWithin(page)).toBe(false);
  await expect(name).toHaveValue("模块");
  await expectModules(db, web.id, []);

  // The Enter after the composition sends the name, with the status the dialog holds; once nerve has made the module,
  // the dialog closes and says so.
  const made = await sentTo(page, "POST", listed, () => page.keyboard.press("Enter"));
  expect([made.answer.status(), made.body]).toEqual([201, { name: "模块", status: "backlog" }]);
  expect(await closedWithin(page)).toBe(true);
  await expect(toast(page, "Module created successfully.")).toBeVisible();
  await expectModules(db, web.id, [madeModule("模块", "backlog", email)]);
  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], []]);
  await expectQuietConsole(page, watch);
});
