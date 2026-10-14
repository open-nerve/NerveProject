import { createProject, createWorkspace, slugFor } from "../../fixtures/api";
import { emailFor } from "../../fixtures/auth";
import { EMOJI_CHECK_WARNING, expectQuietConsole, watchPage } from "../../fixtures/browser";
import { chooseByKeyboard } from "../../fixtures/menus";
import { registerOnboarded } from "../../fixtures/settings-pages";
import { expect, test } from "../../fixtures/test";

// S8, the sidebar by the keyboard (M3 closeout, P11's keyboard list): what the keyboard and assistive technology
// reach in the workspace's sidebar is what it shows.

test("S8 (page): the sidebar's copy for peeking holds no control while it is hidden, and the sidebar none while it is collapsed, its toggle handing the focus to the header's and back; the workspace menu's heading is one Tab stop, named by its words, that says whether the menu shows and opens and closes it by the keyboard, its chevron neither read nor a Tab stop; the projects list's heading is one Tab stop, named by its words, that says whether the list shows and opens and closes it by the keyboard, its chevron neither read nor a Tab stop; the new project's button and a project's menu each show and are marked as Tab reaches them; a project's row is one Tab stop, its link, and its menu another, each named by the project, the menu worked by the keyboard; its chevron and grip are none", async ({
  api,
  signedInPage,
}, testInfo) => {
  const tokens = await registerOnboarded(api, emailFor(testInfo));
  const slug = slugFor(testInfo);
  await createWorkspace(api, tokens.access_token, { name: "Acme", slug });
  const web = await createProject(api, tokens.access_token, slug, { name: "Web", identifier: "WEB" });
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

  // The workspace menu's heading is named by its words and says whether the menu shows; Enter there closes the menu
  // and opens it again. Its chevron, unseen until the pointer is over the heading, does what the heading does:
  // assistive technology reads the heading alone, and Tab goes from it to the menu's first page.
  const workspaceMenu = sidebar.getByRole("button", { name: "Workspace", exact: true });
  const firstPage = sidebar.getByRole("link", { name: "Projects", exact: true });
  await expect(workspaceMenu).toHaveAttribute("aria-expanded", "true");
  await workspaceMenu.focus();
  await page.keyboard.press("Enter");
  await expect(workspaceMenu).toHaveAttribute("aria-expanded", "false");
  await expect(firstPage).toHaveCount(0);
  await page.keyboard.press("Enter");
  await expect(workspaceMenu).toHaveAttribute("aria-expanded", "true");
  await expect(workspaceMenu).toBeFocused();
  await expect(workspaceMenu.locator("xpath=..").getByRole("button")).toHaveCount(1);
  await page.keyboard.press("Tab");
  await expect(firstPage).toBeFocused();

  // The projects list's heading is named by its words and says whether the list shows; Enter there closes the list
  // and opens it again. Its chevron, which does what the heading does, is neither read nor a Tab stop: the heading's
  // row reads the heading and the new project's button alone, and Tab goes from that button to the list's first
  // project.
  const heading = sidebar.getByRole("button", { name: "Projects", exact: true });
  const row = sidebar.getByRole("link", { name: "Web", exact: true });
  await expect(heading).toHaveAttribute("aria-expanded", "true");
  await heading.focus();
  await page.keyboard.press("Enter");
  await expect(heading).toHaveAttribute("aria-expanded", "false");
  await expect(row).toHaveCount(0);
  await page.keyboard.press("Enter");
  await expect(heading).toHaveAttribute("aria-expanded", "true");
  await expect(heading).toBeFocused();
  await expect(heading.locator("xpath=..").getByRole("button")).toHaveCount(2);

  // The new project's button, unseen until the pointer is over the list's heading, shows and is marked as Tab reaches
  // it from the heading.
  const create = sidebar.getByRole("button", { name: "Create new project" });
  await page.keyboard.press("Tab");
  await expect(create).toBeFocused();
  await expect(create).toHaveCSS("opacity", "1");
  await expect(create).not.toHaveCSS("outline-style", "none");
  await page.keyboard.press("Tab");
  await expect(row).toBeFocused();

  // Web's row is one Tab stop, its link, named by Web, which opens and closes its pages; its menu, named by Web too,
  // the next, shown and marked as Tab reaches it. Its chevron, which does what the link does, and its grip, which the
  // pointer drags, are none, though the pointer over the row shows them.
  const menu = sidebar.getByRole("button", { name: "Web quick actions", exact: true });
  await expect(menu).toHaveCount(1);
  await expect(row).toHaveAttribute("aria-expanded", "false");
  await expect(sidebar.getByRole("button", { name: "Web", exact: true })).toHaveCount(0);
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

  // Collapsed by its toggle and the keyboard, the sidebar holds no control the keyboard reaches, and the focus goes on
  // to the toggle of the page's header, outside either sidebar, seen while the sidebar is collapsed; Enter there opens
  // the sidebar again, the focus going back to its toggle.
  const toggle = sidebar.getByRole("button", { name: "Toggle sidebar" });
  await toggle.focus();
  await page.keyboard.press("Enter");
  await expect.poll(inert).toEqual([true, true]);
  await expect(
    page.locator("xpath=//button[@aria-label='Toggle sidebar'][not(ancestor::*[@role='complementary'])]")
  ).toBeFocused();
  await page.keyboard.press("Enter");
  await expect.poll(inert).toEqual([false, true]);
  await expect(toggle).toBeFocused();

  // Web's menu by the keyboard alone: Enter opens it, and the arrow keys and Enter choose its Settings.
  await row.focus();
  await page.keyboard.press("Tab");
  await chooseByKeyboard(page, menu, "Settings");
  await expect(page).toHaveURL(`/${slug}/settings/projects/${web.id}`);
  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], []]);
  await expectQuietConsole(page, watch, { warnings: [EMOJI_CHECK_WARNING] });
});
