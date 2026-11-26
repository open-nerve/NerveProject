import type { Page } from "@playwright/test";

import { expectQuietConsole, watchPage } from "../../fixtures/browser";
import { recordApi, seenBy } from "../../fixtures/cycle-pages";
import { closedByEscape, modalDialog, sentWithin } from "../../fixtures/dialogs";
import { createIssue } from "../../fixtures/issues";
import { shownItems } from "../../fixtures/planning-list-pages";
import { openCommands, selectCommand } from "../../fixtures/power-k";
import { answerTo, holdAnswer, moveWithinApp } from "../../fixtures/settings-pages";
import { expect, test } from "../../fixtures/test";
import {
  listedViews,
  pageMain,
  viewPath,
  viewReads,
  viewWorld,
  viewsCrumb,
  viewsPath,
  viewsRead,
} from "../../fixtures/view-pages";

// H20's page version for the command palette's views (M7 design 2, H20; P6a spec 2.8, 2.11; R13, R14, R20): its
// "Open a project view" lists the route's project's views by the views list's own read (one key: open beside the list,
// the two make one read) and opens the one chosen; its search, at a project's scope alone, names the project's views
// whose names hold the query, filtered on the page, reading the project's views only while the page holds none of them;
// its New view opens the view's dialog, for the project's admins and members: a guest lists his views, never New view.
// The views pages' late answers and keys are h20-view-pages.spec.ts's.

/** The options of the palette's search group of name ("Views"), each as the palette shows it. */
const groupOptions = (page: Page, name: string) => page.getByRole("group", { name, exact: true }).getByRole("option");

test("H20 (page): alice's palette on Ops' work items reads Ops' views once, its search for tri naming OPS Triage among Views, none at the workspace's level, and opens it; opened again it reads none; on Web's views, their read held, Open a project view waits for that one read, then lists Web's views and opens Roadmap; New view opens Create View", async ({
  api,
  signedInPage,
}, testInfo) => {
  const w = await viewWorld(api, testInfo);
  const ops = await w.project("Ops", "OPS", true);
  await createIssue(api, w.alice.token, ops.id, { name: "Deploy" });
  const triage = await w.view(w.alice, { name: "Triage" }, ops.id);
  await w.view(w.alice, { name: "Board" }, ops.id);
  const roadmap = await w.view(w.alice, { name: "Roadmap" });
  await w.view(w.bob, { name: "Bugs" });
  const webViews = `/api/v0/projects/${w.web.id}/views`;

  const page = await signedInPage(w.alice.tokens);
  const watch = await watchPage(page);
  const recorded = recordApi(page);
  const settled = () => expect.poll(() => recorded().pending, "the reads out answered").toBe(0);
  // The page's clock, flowing as the real one, lets the story pass SWR's 2-second deduplication window at once: a read
  // of Ops' views the palette made on opening again would not be hidden by it.
  await page.clock.install();
  await page.goto(`/${w.slug}/projects/${ops.id}/issues`);
  await expect.poll(() => shownItems(page)).toEqual(["Deploy"]);
  await settled();
  expect(viewReads(recorded().requests), "the project's shell reads no views").toEqual([]);

  // Opened at Ops' scope, the page holding none of its views: one read of them. tri names Triage among Views, after
  // Ops' identifier; at the workspace's level, no Views; back at Ops', picked, Triage's page.
  await openCommands(page);
  await expect.poll(() => viewReads(recorded().requests)).toEqual([viewsRead(ops.id)]);
  await page.getByRole("combobox").fill("tri");
  await expect(groupOptions(page, "Views")).toHaveText(["OPS Triage"]);
  const workspaceLevel = page.getByRole("switch", { name: "Workspace level", exact: true });
  await workspaceLevel.click();
  await expect(groupOptions(page, "Views")).toHaveCount(0);
  await workspaceLevel.click();
  await expect(groupOptions(page, "Views")).toHaveText(["OPS Triage"]);
  await settled();
  expect(viewReads(recorded().requests), "what the search read of views").toEqual([viewsRead(ops.id)]);
  await selectCommand(page, /^OPS ?Triage$/);
  await expect(page).toHaveURL(viewPath(w.slug, ops.id, triage.id));
  await expect(pageMain(page).getByRole("button", { name: "Triage", exact: true })).toBeVisible();

  // Opened again, past SWR's window: the page holds Ops' views now, and reads none of them; tri names Triage still.
  await settled();
  await page.clock.fastForward(3_000);
  await openCommands(page);
  await page.getByRole("combobox").fill("tri");
  await expect(groupOptions(page, "Views")).toHaveText(["OPS Triage"]);
  await settled();
  expect(recorded().requests.filter((request) => request === viewsRead(ops.id))).toHaveLength(1);
  await page.keyboard.press("Escape");
  await page.keyboard.press("Escape");
  await expect(modalDialog(page)).toHaveCount(0);

  // Ops' views, then its Open a project view, which lists them: the list's code and the menu's loaded before any route
  // is (WebKit's preloads).
  await viewsCrumb(page, w.slug, ops.id).click();
  await expect.poll(() => listedViews(page)).toEqual(["Board", "Triage"]);
  await openCommands(page);
  await selectCommand(page, /^Open a project view/);
  await expect(page.getByRole("option")).toHaveText(["Board", "Triage"]);
  await page.keyboard.press("Escape");
  await expect(modalDialog(page)).toHaveCount(0);
  await settled();

  // Web's views, their read held: Open a project view waits for it, sending no read of its own; let through, the
  // menu lists Web's views, the page too, and Roadmap picked opens its page.
  const release = await holdAnswer(page, "GET", webViews);
  const held = recordApi(page);
  await moveWithinApp(page, viewsPath(w.slug, w.web.id));
  await expect.poll(() => held().requests).toContain(viewsRead(w.web.id));
  await openCommands(page);
  expect(await sentWithin(page, "GET", webViews, () => selectCommand(page, /^Open a project view/))).toBe(false);
  await expect(modalDialog(page).getByRole("status")).toBeVisible();
  expect(viewReads(held().requests)).toEqual([viewsRead(w.web.id)]);
  expect((await answerTo(page, "GET", webViews, release)).status()).toBe(200);
  await expect(page.getByRole("option")).toHaveText(["Bugs", "Roadmap"]);
  expect(await listedViews(page)).toEqual(["Bugs", "Roadmap"]);
  await selectCommand(page, /^Roadmap$/);
  await expect(page).toHaveURL(viewPath(w.slug, w.web.id, roadmap.id));
  await expect(pageMain(page).getByRole("button", { name: "Roadmap", exact: true })).toBeVisible();
  await settled();

  // New view: the palette gone, the view's dialog, Create View, its name empty and focused; Escape closes it.
  await openCommands(page);
  await selectCommand(page, /^New view/);
  const dialog = page.getByRole("dialog", { name: "Create View", exact: true });
  await expect(dialog).toHaveCount(1);
  await expect(modalDialog(page)).toHaveCount(1);
  const name = dialog.getByRole("textbox", { name: "Title", exact: true });
  await expect(name).toHaveValue("");
  await expect(name).toBeFocused();
  expect(await closedByEscape(page)).toBe(true);

  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], []]);
  await expectQuietConsole(page, watch);
});

test("H20 (page): bob, Web's member, finds New view in his palette; gus, Web's guest, who sees his own, finds Open a project view and no New view, which lists his own view alone and opens it", async ({
  api,
  baseURL,
  browser,
  signedInPage,
}, testInfo) => {
  const w = await viewWorld(api, testInfo);
  const own = await w.view(w.gus, { name: "Gus's own" });
  await w.view(w.alice, { name: "Roadmap" });
  await w.guest();

  // bob, a member: New view.
  await seenBy(browser, baseURL ?? "", api, w.bob.email, viewsPath(w.slug, w.web.id), async (other) => {
    await expect.poll(() => listedViews(other)).toEqual(["Roadmap", "Gus's own"]);
    await openCommands(other);
    await expect(other.getByRole("option", { name: /^New view/ })).toBeVisible();
    await other.keyboard.press("Escape");
    await expect(modalDialog(other)).toHaveCount(0);
  });

  // gus, a guest: Open a project view, no New view; it lists his own alone, and opens it.
  const page = await signedInPage(w.gus.tokens);
  const watch = await watchPage(page);
  const recorded = recordApi(page);
  await page.goto(viewsPath(w.slug, w.web.id));
  await expect.poll(() => listedViews(page)).toEqual(["Gus's own"]);
  await expect.poll(() => recorded().pending).toBe(0);
  await openCommands(page);
  await expect(page.getByRole("option", { name: /^Open a project view/ })).toBeVisible();
  await expect(page.getByRole("option", { name: /^New view/ })).toHaveCount(0);
  await selectCommand(page, /^Open a project view/);
  await expect(page.getByRole("option")).toHaveText(["Gus's own"]);
  await selectCommand(page, /^Gus's own$/);
  await expect(page).toHaveURL(viewPath(w.slug, w.web.id, own.id));
  await expect(pageMain(page).getByRole("button", { name: "Gus's own", exact: true })).toBeVisible();
  await expect.poll(() => recorded().pending).toBe(0);

  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], []]);
  await expectQuietConsole(page, watch);
});
