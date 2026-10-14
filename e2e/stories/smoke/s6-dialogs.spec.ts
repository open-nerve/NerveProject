import { createWorkspace, slugFor } from "../../fixtures/api";
import { createPAT, emailFor, shownNameOf } from "../../fixtures/auth";
import { expectQuietConsole, EMOJI_CHECK_WARNING, watchPage } from "../../fixtures/browser";
import {
  closedByComposingEscape,
  closedByEscape,
  closedWithin,
  composingKey,
  hiddenWithin,
  modalDialog,
  pageBehindHidden,
} from "../../fixtures/dialogs";
import { tabCycle } from "../../fixtures/project-pages";
import { registerOnboarded } from "../../fixtures/settings-pages";
import { expect, test } from "../../fixtures/test";

// S6, the modal dialogs the workspace's pages open that no other story does (M3 closeout: every dialog is ModalCore,
// Headless UI's Dialog, which makes the page behind it inert and hidden while it is open): the command palette, its
// list of keyboard shortcuts, and a dialog opened over another, the settings dialog's revoke dialog.

test("S6 (page): the command palette and its list of keyboard shortcuts are modal dialogs, the palette's search keeping an Escape and a Backspace that end a composition: while each is open the page behind it is inert and hidden and Tab stays in it; Escape closes each and gives the page back; the palette's command that copies the page's address copies it and says so", async ({
  api,
  signedInPage,
}, testInfo) => {
  const tokens = await registerOnboarded(api, emailFor(testInfo));
  const slug = slugFor(testInfo);
  await createWorkspace(api, tokens.access_token, { name: "Acme", slug });
  const page = await signedInPage(tokens);
  const watch = await watchPage(page);
  await page.goto(`/${slug}/projects`);
  await expect(page.getByRole("button", { name: "Add Project", exact: true })).toBeVisible();

  // The palette: its search has the focus, and Tab goes round what it holds. An Escape that ends an input method's
  // composition in the search is the method's: the palette, which an Escape in its empty search closes, stays open.
  await page.keyboard.press("ControlOrMeta+k");
  await expect(modalDialog(page).getByRole("combobox")).toBeFocused();
  expect(await closedByComposingEscape(modalDialog(page).getByRole("combobox"))).toBe(false);
  expect(await pageBehindHidden(page)).toBe(true);
  const stops = await tabCycle(page, modalDialog(page));
  expect(stops.filter((stop) => stop.startsWith("outside: "))).toEqual([]);
  // A page of the palette (the first day of the week): a Backspace in its empty search goes back, but one that ends a
  // composition there is the method's, and the page stays.
  const paletteSearch = modalDialog(page).getByRole("combobox");
  await paletteSearch.fill("Change first day of week");
  await page.keyboard.press("Enter");
  const monday = modalDialog(page).getByRole("option", { name: "Monday", exact: true });
  await expect(monday).toBeVisible();
  const left = hiddenWithin(monday);
  await composingKey(paletteSearch, "Backspace");
  expect(await left).toBe(false);
  await page.keyboard.press("Backspace");
  await expect(monday).toHaveCount(0);
  expect(await closedByEscape(page)).toBe(true);
  expect(await pageBehindHidden(page)).toBe(false);

  // The shortcuts, named by their title alone: from its search Tab goes to its list, which scrolls, then to its close
  // button, named, and round; no stop outside it.
  await page.keyboard.press("ControlOrMeta+/");
  await expect(modalDialog(page)).toHaveAccessibleName("Keyboard shortcuts");
  const search = page.getByRole("textbox", { name: "Search for shortcuts" });
  await expect(search).toBeFocused();
  expect(await pageBehindHidden(page)).toBe(true);
  const shortcutStops = await tabCycle(page, modalDialog(page), 4);
  expect(shortcutStops).toHaveLength(3);
  expect(shortcutStops[0]).toMatch(/^Work item actions/);
  expect(shortcutStops.slice(1)).toEqual(["Close", "Search for shortcuts"]);
  expect(await closedByEscape(page)).toBe(true);
  expect(await pageBehindHidden(page)).toBe(false);

  // The palette's command that copies the page's address.
  await page.context().grantPermissions(["clipboard-read", "clipboard-write"]);
  await page.keyboard.press("ControlOrMeta+k");
  await page.keyboard.type("Copy current page URL");
  await page.keyboard.press("Enter");
  await expect(page.getByText("Current page URL copied to clipboard.")).toBeVisible();
  expect(await page.evaluate(() => navigator.clipboard.readText())).toBe(page.url());
  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], []]);
  await expectQuietConsole(page, watch, { warnings: [EMOJI_CHECK_WARNING] });
});

test("S6 (page): a dialog opened over another, the revoke dialog over the settings dialog of the user menu: the settings dialog is inert and hidden behind it; Escape closes the top one alone, the focus back where it was, then its tooltip, then the other", async ({
  api,
  signedInPage,
}, testInfo) => {
  const email = emailFor(testInfo);
  const tokens = await registerOnboarded(api, email);
  const slug = slugFor(testInfo);
  await createWorkspace(api, tokens.access_token, { name: "Acme", slug });
  await createPAT(api, tokens.access_token, { label: "deploy" });
  const page = await signedInPage(tokens);
  const watch = await watchPage(page);
  await page.goto(`/${slug}/projects`);

  await page
    .getByRole("button", { name: shownNameOf(email) })
    .first()
    .click();
  await page.getByRole("menuitem", { name: "Settings" }).click();
  await modalDialog(page).getByRole("button", { name: "Personal Access Tokens" }).click();
  const revoke = modalDialog(page).getByRole("button", { name: "Revoke token" });
  await revoke.click();
  const dialogs = modalDialog(page);
  await expect(dialogs).toHaveCount(2);
  await expect(dialogs.last().getByRole("button", { name: "Delete" })).toBeVisible();
  // The settings dialog beneath, as the page behind both: inert and hidden.
  expect(
    await dialogs
      .first()
      .evaluate((settings) => settings.closest("[inert]") !== null && settings.closest('[aria-hidden="true"]') !== null)
  ).toBe(true);
  expect(await pageBehindHidden(page)).toBe(true);
  // Escape closes the revoke dialog alone; the settings dialog is back, the button that opened it focused.
  await page.keyboard.press("Escape");
  await expect(dialogs).toHaveCount(1);
  expect(
    await dialogs.evaluate(
      (settings) => settings.closest("[inert]") === null && settings.closest('[aria-hidden="true"]') === null
    )
  ).toBe(true);
  await expect(revoke).toBeFocused();
  expect(await pageBehindHidden(page)).toBe(true);
  // The focus is the keyboard's, so the button's tooltip shows: Escape closes it first, then the settings dialog.
  const tooltip = page.getByText("Revoke token", { exact: true });
  await expect(tooltip).toBeVisible();
  await page.keyboard.press("Escape");
  await expect(tooltip).toHaveCount(0);
  expect(await closedWithin(page)).toBe(false);
  expect(await closedByEscape(page)).toBe(true);
  expect(await pageBehindHidden(page)).toBe(false);
  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], []]);
  await expectQuietConsole(page, watch, { warnings: [EMOJI_CHECK_WARNING] });
});
