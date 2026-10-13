import { createProject, createWorkspace, slugFor } from "../../fixtures/api";
import { emailFor } from "../../fixtures/auth";
import { EMOJI_CHECK_WARNING, expectQuietConsole, watchPage } from "../../fixtures/browser";
import { registerOnboarded } from "../../fixtures/settings-pages";
import { expect, test } from "../../fixtures/test";

// S8, the sidebar by the keyboard (M3 closeout, P11's keyboard list): what the keyboard and assistive technology
// reach in the workspace's sidebar is what it shows.

test("S8 (page): the sidebar's copy for peeking holds no control while it is hidden, and the sidebar none while it is collapsed; the workspace menu's chevron is neither read nor a Tab stop; the new project's button and a project's menu each show and are marked as Tab reaches them, the menu one Tab stop; a project's row and its menu are named by its project, and its grip is none", async ({
  api,
  signedInPage,
}, testInfo) => {
  const tokens = await registerOnboarded(api, emailFor(testInfo));
  const slug = slugFor(testInfo);
  await createWorkspace(api, tokens.access_token, { name: "Acme", slug });
  await createProject(api, tokens.access_token, slug, { name: "Web", identifier: "WEB" });
  const page = await signedInPage(tokens);
  const watch = await watchPage(page);
  await page.goto(`/${slug}/projects`);
  const sidebar = page.getByRole("complementary", { name: "Main sidebar" });
  // whether each of the two Project navigation buttons, the sidebar's and its copy's for peeking, is inert: neither
  // the keyboard nor assistive technology reaches it (Playwright's roles do not leave inert elements out)
  const navigation = page.getByRole("button", { name: "Project navigation" });
  const inert = () => navigation.evaluateAll((buttons) => buttons.map((button) => button.closest("[inert]") !== null));

  // The sidebar's copy for peeking, unseen until the pointer asks for it, holds no control.
  await expect(navigation).toHaveCount(2);
  expect(await inert()).toEqual([false, true]);

  // The workspace menu's chevron, unseen until the pointer is over its heading, does what the heading's button does:
  // assistive technology reads that button alone, and Tab goes from it to the menu's first page.
  const workspaceMenu = sidebar.getByRole("button", { name: "Close workspace menu" });
  await expect(workspaceMenu.locator("xpath=..").getByRole("button")).toHaveCount(1);
  await workspaceMenu.focus();
  await page.keyboard.press("Tab");
  await expect(sidebar.getByRole("link", { name: "Projects", exact: true })).toBeFocused();

  // The new project's button, unseen until the pointer is over the list's heading, shows and is marked as Tab reaches
  // it from the heading.
  const create = sidebar.getByRole("button", { name: "Create new project" });
  await sidebar.getByRole("button", { name: "Close projects menu" }).first().focus();
  await page.keyboard.press("Tab");
  await expect(create).toBeFocused();
  await expect(create).toHaveCSS("opacity", "1");
  await expect(create).not.toHaveCSS("outline-style", "none");

  // Web's menu, named by Web as its row is: one Tab stop after the row, shown and marked as Tab reaches it; its
  // chevron, which does what the row does, and its grip, which the pointer drags, are none, though the pointer over
  // the row shows them.
  const menu = sidebar.getByRole("button", { name: "Web quick actions", exact: true });
  await expect(menu).toHaveCount(1);
  const row = sidebar.getByRole("button", { name: "Web", exact: true });
  await row.focus();
  await page.keyboard.press("Tab");
  await expect(menu).toBeFocused();
  await expect(menu.locator("xpath=ancestor::div[@data-main-menu][1]")).toHaveCSS("opacity", "1");
  await expect(menu).not.toHaveCSS("outline-style", "none");
  await row.hover();
  await page.keyboard.press("Tab");
  await expect(menu).not.toBeFocused();
  await expect(sidebar.getByRole("button", { name: "Open project menu" })).not.toBeFocused();
  await expect(sidebar.locator(".cursor-grab:is(button, a, [tabindex]):not([tabindex='-1'])")).toHaveCount(0);

  // Collapsed, the sidebar holds no control the keyboard reaches; opened again, it does.
  await sidebar.getByRole("button", { name: "Toggle sidebar" }).click();
  await expect.poll(inert).toEqual([true, true]);
  // the page's header has its own, outside either sidebar, seen while the sidebar is collapsed
  await page.locator("xpath=//button[@aria-label='Toggle sidebar'][not(ancestor::*[@role='complementary'])]").click();
  await expect.poll(inert).toEqual([false, true]);
  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], []]);
  await expectQuietConsole(page, watch, { warnings: [EMOJI_CHECK_WARNING] });
});
