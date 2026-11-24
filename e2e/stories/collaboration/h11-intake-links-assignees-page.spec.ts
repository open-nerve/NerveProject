/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import type { Locator, Page, Response } from "@playwright/test";

import { intakeLinksOf, intakeRowOf, submittedOf } from "../../fixtures/assert/intake";
import { expectQuietConsole, watchPage } from "../../fixtures/browser";
import { dismissToasts, recordApi, seenBy, toast } from "../../fixtures/cycle-pages";
import { submitIntakeIssue, triageIntakeIssue } from "../../fixtures/intake";
import {
  expectHeaderOf,
  intakeDescription,
  intakeDetail,
  intakeEditPath,
  intakePath,
  intakeWorld,
  selecting,
  writesSeen,
  wrote,
} from "../../fixtures/intake-pages";
import type { Issue } from "../../fixtures/issues";
import { completedLinkCrawl, createIssueLink, listIssueLinks } from "../../fixtures/links";
import { expectPopupBesideButton, holdAnswer } from "../../fixtures/settings-pages";
import { expect, test } from "../../fixtures/test";

// H11's page version, the links and the assignees of a work item of a project's intake (M7 design 2, 3.6; P5b spec
// 2.3, 2.6, 2.8, R10, R19, R21). Its links are the work item's own, the contract's issue.update: an admin or a member
// adds, changes and deletes them whatever the work item's status, a declined one's too, which the rest of its detail
// shows him for reading alone; a guest reads them, offered nothing to change. Its assignees' dropdown on the detail
// opens by the keyboard beside its button, its search focused, and Escape gives the focus back to the button; two of
// its members picked one after the other, the first one's change still unanswered, are both assigned (Q34: a change of
// the set is worked out in its turn from the set then held, v0 design 7.7). The addresses the links lead to are
// link-local: nerve's crawl refuses them, so nothing leaves the machine.

/**
 * The link of the work item that region shows to url, titled title: its anchor, named by its title (the activity's
 * anchors to the same address are named "link").
 */
function linked(region: Locator, url: string, title: string): Locator {
  return region.getByRole("link", { name: title, exact: true }).and(region.locator(`a[href="${url}"]`));
}

/**
 * Resolves with the next read of the links of the work item issue that page gets in which the link of url is, and each
 * link has what nerve's crawl wrote of it: until then the detail reads them again each second.
 */
function crawledRead(page: Page, issue: Issue, url: string): Promise<Response> {
  const path = `/api/v0/issues/${issue.id}/links`;
  return page.waitForResponse(
    async (response) => {
      if (response.request().method() !== "GET" || new URL(response.url()).pathname !== path) return false;
      const { data } = (await response.json()) as { data: { url: string; metadata: object }[] };
      return data.some((link) => link.url === url) && data.every((link) => Object.keys(link.metadata).length > 0);
    },
    { timeout: 15_000 }
  );
}

test("H11 (page): bob, a member of Web who did not submit it, adds a link to the work item gus submitted to its intake, declined, changes its title and deletes alice's, each by the work item's own links (issue.update, whatever its status) while the rest of its detail is his to read alone, nerve keeping the links as he wrote them; gus, the guest who submitted it, sees its link, offered neither an Add link button nor its options", async ({
  api,
  baseURL,
  browser,
  browserName,
  db,
  signedInPage,
}, testInfo) => {
  const { alice, bob, gus, slug, web } = await intakeWorld(api, testInfo);
  const login = await submitIntakeIssue(api, gus.token, web.id, { name: "Login breaks" });
  const notesURL = "http://169.254.169.254/h11/notes";
  const notes = await createIssueLink(api, alice.token, login.id, { url: notesURL, title: "Notes" });
  await completedLinkCrawl(db, notes.id, notesURL);
  await triageIntakeIssue(api, alice.token, login.id, { status: "declined" });
  const address = `${intakePath(slug, web.id)}?currentTab=closed&inboxIssueId=${login.id}`;

  // The declined work item, selected: bob reads its description and its properties alone; alice's link listed.
  const page = await signedInPage(bob.tokens);
  const watch = await watchPage(page);
  const writes = await writesSeen(page, browserName);
  const reads = recordApi(page);
  await page.goto(address);
  await expect(page).toHaveURL(selecting(login));
  await expectHeaderOf(page, login);
  const detail = intakeDetail(page);
  await expect(intakeDescription(page, login)).toHaveAttribute("contenteditable", "false");
  await expect(detail.getByRole("button", { name: "Add assignees", exact: true })).toBeDisabled();
  await expect(linked(detail, notesURL, "Notes")).toBeVisible();
  await expect(detail.getByRole("button", { name: "Links 1", exact: true })).toBeVisible();
  await expect.poll(() => reads().pending).toBe(0);

  // His link, added by the named button: listed by its title once nerve has it, its crawl's refusal read in turn.
  const specURL = "http://169.254.169.254/h11/spec";
  await detail.getByRole("button", { name: "Add link", exact: true }).click();
  const adding = page.getByRole("dialog", { name: "Add link", exact: true });
  await adding.getByRole("textbox", { name: "URL", exact: true }).fill(specURL);
  await adding.getByRole("textbox", { name: /^Display title/ }).fill("Spec");
  const crawled = crawledRead(page, login, specURL);
  const linksPath = `/api/v0/issues/${login.id}/links`;
  const add = adding.getByRole("button", { name: "Add Link", exact: true });
  expect(await wrote(page, writes, "POST", linksPath, () => add.click())).toEqual([
    { url: specURL, title: "Spec" },
    201,
  ]);
  await expect(adding).toHaveCount(0);
  await expect(toast(page, "The link has been successfully created")).toBeVisible();
  await expect(linked(detail, specURL, "Spec")).toBeVisible();
  await expect(detail.getByRole("button", { name: "Links 2", exact: true })).toBeVisible();
  const spec = (await listIssueLinks(api, bob.token, login.id)).find((link) => link.url === specURL);
  expect(spec, "bob's link as nerve lists it").toBeDefined();
  await completedLinkCrawl(db, spec!.id, specURL);
  await crawled;
  await expect.poll(() => reads().pending).toBe(0);
  await dismissToasts(page);

  // Its title changed by its options' Edit, the address kept.
  await detail.getByRole("button", { name: "Spec options", exact: true }).click();
  await page.getByRole("menuitem", { name: "Edit", exact: true }).click();
  const updating = page.getByRole("dialog", { name: "Update link", exact: true });
  await expect(updating.getByRole("textbox", { name: "URL", exact: true })).toHaveValue(specURL);
  await updating.getByRole("textbox", { name: /^Display title/ }).fill("Login spec");
  const update = updating.getByRole("button", { name: "Update Link", exact: true });
  expect(await wrote(page, writes, "PATCH", `/api/v0/issue-links/${spec!.id}`, () => update.click())).toEqual([
    { url: specURL, title: "Login spec" },
    200,
  ]);
  await expect(updating).toHaveCount(0);
  await expect(toast(page, "The link has been successfully updated")).toBeVisible();
  await expect(linked(detail, specURL, "Login spec")).toBeVisible();
  await expect.poll(() => reads().pending).toBe(0);
  await dismissToasts(page);

  // Alice's deleted by its options' Delete: gone once nerve has deleted it. (M4's removal says nothing once done: the
  // detail's read again withdraws the list whose scope it reports through.)
  await detail.getByRole("button", { name: "Notes options", exact: true }).click();
  const remove = page.getByRole("menuitem", { name: "Delete", exact: true });
  expect(await wrote(page, writes, "DELETE", `/api/v0/issue-links/${notes.id}`, () => remove.click())).toEqual([
    null,
    204,
  ]);
  await expect(detail.getByRole("link", { name: "Notes", exact: true })).toHaveCount(0);
  await expect(detail.getByRole("button", { name: "Links 1", exact: true })).toBeVisible();
  await expect(linked(detail, specURL, "Login spec")).toBeVisible();
  await expect.poll(() => reads().pending).toBe(0);
  expect(writes.map(({ write }) => write)).toEqual([
    `POST ${linksPath}`,
    `PATCH /api/v0/issue-links/${spec!.id}`,
    `DELETE /api/v0/issue-links/${notes.id}`,
  ]);
  expect(await intakeLinksOf(db, login.id)).toEqual([
    { url: notesURL, title: "Notes", adder: alice.id, by: bob.id, deleted: true },
    { url: specURL, title: "Login spec", adder: bob.id, by: bob.id, deleted: false },
  ]);
  expect((await intakeRowOf(db, login.id)).status, "still declined").toBe("declined");

  // gus sees bob's link on the work item he submitted, offered neither an Add link button nor its options.
  await seenBy(browser, baseURL ?? "", api, gus.email, address, async (other) => {
    await expectHeaderOf(other, login);
    const shown = intakeDetail(other);
    await expect(linked(shown, specURL, "Login spec")).toBeVisible();
    await expect(shown.getByRole("button", { name: "Links 1", exact: true })).toBeVisible();
    await expect(shown.getByRole("button", { name: "Copy link: Login spec", exact: true })).toBeVisible();
    await expect(shown.getByRole("button", { name: "Add link", exact: true })).toHaveCount(0);
    await expect(shown.getByRole("button", { name: "Login spec options", exact: true })).toBeDisabled();
  });

  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], []]);
  await expectQuietConsole(page, watch);
});

test("H11 (page): alice, Web's admin, opens the assignees of a pending work item of its intake on its detail by the keyboard: Enter opens their list beside the button, its search focused, and Escape closes it, the focus back on the button; she picks Bob, and Odo while nerve's answer to Bob's change is held: the second change carries both, and nerve keeps both", async ({
  api,
  browserName,
  db,
  signedInPage,
}, testInfo) => {
  const { alice, bob, odo, slug, web } = await intakeWorld(api, testInfo);
  const crash = await submitIntakeIssue(api, bob.token, web.id, { name: "Crash on save" });

  // Crash on save selected, its description's editor built (a route set up after it holds none of WebKit's preloads).
  const page = await signedInPage(alice.tokens);
  const watch = await watchPage(page);
  const writes = await writesSeen(page, browserName);
  const reads = recordApi(page);
  await page.goto(`${intakePath(slug, web.id)}?inboxIssueId=${crash.id}`);
  await expect(page).toHaveURL(selecting(crash));
  await expectHeaderOf(page, crash);
  await expect(intakeDescription(page, crash)).toHaveAttribute("contenteditable", "true");
  await expect.poll(() => reads().pending).toBe(0);

  // By the keyboard (R21): Enter opens the list beside the button, its search focused; Escape back on the button.
  const assignees = intakeDetail(page).getByRole("button", { name: "Add assignees", exact: true });
  await assignees.focus();
  await page.keyboard.press("Enter");
  const list = page.getByRole("listbox");
  await expectPopupBesideButton(list, assignees);
  await expect(page.getByRole("combobox", { name: "Search", exact: true })).toBeFocused();
  await page.keyboard.press("Escape");
  await expect(list).toHaveCount(0);
  await expect(assignees).toBeFocused();
  expect(writes).toEqual([]);

  // Picked one after the other (Q34, T8c): Bob, the answer to his change held once nerve has applied it; Odo meanwhile,
  // his change waiting its turn behind Bob's. Let through, the second change carries both, and nerve keeps both.
  const edit = intakeEditPath(crash);
  const answers: number[] = [];
  page.on("response", (response) => {
    if (response.request().method() === "PATCH" && new URL(response.url()).pathname === edit)
      answers.push(response.status());
  });
  let applied = false;
  const release = await holdAnswer(page, "PATCH", edit, 1, () => {
    applied = true;
  });
  await assignees.click();
  await list.getByRole("option", { name: "Bob Bob", exact: true }).click();
  await expect.poll(() => applied).toBe(true);
  await list.getByRole("option", { name: "Odo Odo", exact: true }).click();
  expect(writes.map(({ write }) => write)).toEqual([`PATCH ${edit}`]);
  await release();
  await expect.poll(() => answers).toEqual([200, 200]);
  await expect.poll(() => reads().pending).toBe(0);
  expect(writes).toEqual([
    { write: `PATCH ${edit}`, body: { assignee_ids: [bob.id] } },
    { write: `PATCH ${edit}`, body: { assignee_ids: [bob.id, odo.id] } },
  ]);
  expect((await submittedOf(db, crash.id)).assignee_ids).toEqual([bob.id, odo.id].toSorted());

  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], []]);
  await expectQuietConsole(page, watch);
});
