import type { Page } from "@playwright/test";

import { expectModuleLinks, moduleLinkMoments } from "../../fixtures/assert/planning";
import { expectQuietConsole, watchPage } from "../../fixtures/browser";
import { dismissToasts, failedLoads, seenBy, toast } from "../../fixtures/cycle-pages";
import { announced, closedByEscape, closedWithin, modalDialog, sentWithin } from "../../fixtures/dialogs";
import { startLinkPage } from "../../fixtures/link-page";
import { modulePagesWorld, modulePath } from "../../fixtures/module-pages";
import {
  createModule,
  createModuleLink,
  expectModuleLinkCrawled,
  moduleCalls,
  moduleLinkCrawls,
} from "../../fixtures/modules";
import { sentHeld, sentTo } from "../../fixtures/settings-pages";
import { expect, test } from "../../fixtures/test";

// G10's page version (M6 P6a spec 2.8, R10, R11): a module's links in its sidebar, on its page, by its member in his
// browser, by Web's guest, and once the module is archived. The page a link leads to is the story's own, on this
// machine (startLinkPage), so nerve's crawl reads it and nothing leaves the machine. The API version is
// g10-module-links.spec.ts; both assert the same rows.

/** The addresses the links of the module page shows lead to, in their order. */
function linkedAddresses(page: Page): Promise<string[]> {
  return page
    .getByRole("listitem")
    .locator('a[target="_blank"]')
    .evaluateAll((links) => links.map((link) => link.getAttribute("href") ?? ""));
}

test("G10 (page): on Docs's page, its member's sidebar lists its Notes link; an address left empty is said under it and sent nowhere; his link of a bare address and an empty title waits for nerve, is listed first and shows its address until nerve has read its page, then that page's title and its icon kept inline, the browser asking nothing of the page itself; the same address again is refused, the dialog keeping it and saying why; its title changed shows instead, the page not read anew; Notes deleted goes once nerve has deleted it; Web's guest sees the links and copies one, and may add, change or delete none; once Docs is archived, its member may neither", async ({
  api,
  baseURL,
  browser,
  browserName,
  db,
  signedInPage,
}, testInfo) => {
  // mem's page, then two other browsers, each signed in anew
  test.setTimeout(90_000);
  const { admin, mem, gus, slug, web, token } = await modulePagesWorld(api, testInfo);
  const docs = await createModule(api, token, web.id, { name: "Docs" });
  // The admin's Notes link, link-local: its crawl is blocked, and it shows its title.
  const notesURL = "http://169.254.169.254/g10/notes";
  const notes = await createModuleLink(api, token, docs.id, { url: notesURL, title: "Notes" });
  await expectModuleLinkCrawled(db, notes.id, [notesURL]);
  const source = await startLinkPage();
  try {
    const page = await signedInPage(mem.tokens);
    const watch = await watchPage(page);
    const path = modulePath(slug, web.id, docs.id);
    await page.goto(path);
    const notesLink = page.locator(`a[href="${notesURL}"]`);
    await expect(notesLink).toHaveText("Notes");

    // The link's dialog, named by its title, says an address left empty under it, which takes the focus; nothing is
    // sent.
    await page.getByRole("button", { name: "Add link", exact: true }).click();
    const dialog = modalDialog(page);
    await expect(dialog).toHaveAccessibleName("Add link");
    const url = dialog.getByRole("textbox", { name: "URL", exact: true });
    const add = dialog.getByRole("button", { name: "Add link", exact: true });
    const linksPath = `/api/v0/modules/${docs.id}/links`;
    expect(await sentWithin(page, "POST", linksPath, () => add.click())).toBe(false);
    await expect(url).toHaveAccessibleDescription("This field is required");
    await expect(url).toBeFocused();

    // mem's link to the story's page, by its address without its scheme and no title: sent as typed, the title empty,
    // and the dialog waits for nerve. Added, it is listed first, by the address nerve gave it its scheme, while nerve
    // reads its page in the background.
    const bare = source.url.slice("http://".length);
    await url.fill(bare);
    const adding = await sentHeld(page, "POST", linksPath, () => add.click());
    expect(adding.body).toEqual({ url: bare, title: null });
    expect(await closedByEscape(page)).toBe(false);
    const added = await adding.release();
    expect(added.status()).toBe(201);
    const guide: string = (await added.json()).id;
    expect(await closedWithin(page)).toBe(true);
    await expect(toast(page, "The link has been successfully created")).toBeVisible();
    const anchor = page.locator(`a[href="${source.url}"]`);
    await expect(anchor).toHaveText(source.url);
    expect(await linkedAddresses(page)).toEqual([source.url, notesURL]);

    // Once nerve has read the page, the link shows the page's title and its icon, which nerve keeps inline: the browser
    // asks nothing of the page itself.
    source.release();
    await expectModuleLinkCrawled(db, guide, [source.url]);
    await expect(anchor).toHaveText("Release & notes", { timeout: 10_000 });
    const row = page.getByRole("listitem").filter({ has: anchor });
    await expect(row.locator('img[src^="data:image/png;base64,"]')).toBeVisible();
    expect(watch.requestUrls.filter((address) => address.startsWith(new URL(source.url).origin))).toEqual([]);

    // The same address again, with its scheme: nerve refuses it (409), said beside the dialog, which stays holding it.
    await dismissToasts(page);
    await page.getByRole("button", { name: "Add link", exact: true }).click();
    await expect(dialog).toHaveAccessibleName("Add link");
    await url.fill(source.url);
    const again = await sentTo(page, "POST", linksPath, () => add.click());
    expect([again.answer.status(), again.body]).toEqual([409, { url: source.url, title: null }]);
    const refusal = toast(page, "This module already has this link.");
    await expect(refusal).toBeVisible();
    expect(await announced(refusal)).toBe(true);
    expect(await closedWithin(page)).toBe(false);
    await expect(url).toHaveValue(source.url);
    await dialog.getByRole("button", { name: "Cancel", exact: true }).click();
    expect(await closedWithin(page)).toBe(true);

    // Its title changed to Guide by its dialog, which holds its address: Guide shows instead, and nerve does not read
    // the page again, its address the same.
    await dismissToasts(page);
    await row.getByRole("button", { name: "Edit Release & notes", exact: true }).click();
    await expect(dialog).toHaveAccessibleName("Update link");
    await expect(url).toHaveValue(source.url);
    await dialog.getByRole("textbox", { name: /^Display title/ }).fill("Guide");
    const titled = await sentTo(page, "PATCH", `/api/v0/module-links/${guide}`, () =>
      dialog.getByRole("button", { name: "Update link", exact: true }).click()
    );
    expect([titled.answer.status(), titled.body]).toEqual([200, { url: source.url, title: "Guide" }]);
    expect(await closedWithin(page)).toBe(true);
    await expect(toast(page, "The link has been successfully updated")).toBeVisible();
    await expect(anchor).toHaveText("Guide");
    expect(await moduleLinkCrawls(db, guide)).toEqual([{ url: source.url, state: "completed" }]);

    // Notes deleted, at once: it goes once nerve has deleted it, and the sidebar says so.
    await dismissToasts(page);
    const deleted = await sentTo(page, "DELETE", `/api/v0/module-links/${notes.id}`, () =>
      page.getByRole("button", { name: "Delete Notes", exact: true }).click()
    );
    expect(deleted.answer.status()).toBe(204);
    await expect(toast(page, "The link has been successfully removed")).toBeVisible();
    await expect(notesLink).toHaveCount(0);
    expect(await linkedAddresses(page)).toEqual([source.url]);
    const removed = (await moduleLinkMoments(db, docs.id))[notesURL] ?? "";
    await expectModuleLinks(db, docs.id, [
      { url: source.url, title: "Guide", adder: mem.email, by: mem.email, removed: null },
      { url: notesURL, title: "Notes", adder: admin.email, by: mem.email, removed },
    ]);

    // Web's guest sees Docs's link and copies its address, and may add, change or delete none. Chromium lets a page
    // write the clipboard once its browser allows it, and read it back; Firefox and WebKit let a click write it.
    const readOnly = (copies: boolean) => async (other: Page) => {
      await expect(other.locator(`a[href="${source.url}"]`)).toHaveText("Guide");
      await expect(other.getByRole("button", { name: "Add link", exact: true })).toHaveCount(0);
      await expect(other.getByRole("button", { name: "Edit Guide", exact: true })).toHaveCount(0);
      await expect(other.getByRole("button", { name: "Delete Guide", exact: true })).toHaveCount(0);
      const copy = other.getByRole("button", { name: "Copy link: Guide", exact: true });
      await expect(copy).toBeVisible();
      if (!copies) return;
      const chromium = browserName === "chromium";
      if (chromium) await other.context().grantPermissions(["clipboard-read", "clipboard-write"]);
      await copy.click();
      await expect(toast(other, "Link copied to clipboard")).toBeVisible();
      if (chromium) expect(await other.evaluate(() => navigator.clipboard.readText())).toBe(source.url);
    };
    await seenBy(browser, baseURL ?? "", api, gus.email, path, readOnly(true));

    // The admin completes Docs and archives it: its member sees its link as before, and may change it no more.
    const { update, archive } = moduleCalls(api);
    expect(await update(token, docs.id, { status: "completed" }), "the admin completes Docs").toEqual({ status: 200 });
    expect(await archive(token, docs.id), "the admin archives Docs").toEqual({ status: 200 });
    await seenBy(browser, baseURL ?? "", api, mem.email, path, readOnly(false));
    expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[`409 POST ${linksPath}`], [], []]);
    await expectQuietConsole(page, watch, { errors: failedLoads(browserName, 409) });
  } finally {
    await source.close();
  }
});
