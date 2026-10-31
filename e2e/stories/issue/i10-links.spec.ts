import type { TestInfo } from "@playwright/test";

import {
  addProjectMembers,
  createProject,
  createWorkspace,
  inviteAndAccept,
  slugFor,
  type Api,
} from "../../fixtures/api";
import {
  createdFor,
  expectIssueUpdated,
  linkCreatedRow,
  linkDeletedRow,
  linkUpdatedRow,
  madeBy,
} from "../../fixtures/assert/issue";
import { newAccount } from "../../fixtures/auth";
import { createIssue } from "../../fixtures/issues";
import { startLinkPage, type LinkPage } from "../../fixtures/link-page";
import {
  completedLinkCrawl,
  createIssueLink,
  issueLinkCreated,
  issueLinkDeleted,
  linkCrawlJobs,
  listIssueLinks,
  updateIssueLink,
} from "../../fixtures/links";
import { expect, test } from "../../fixtures/test";
import { issueListProject, expectListPageQuiet } from "../../fixtures/issue-list-page";
import { openIssueDetail } from "../../fixtures/issue-detail-page";
import { answerTo, bodiesSentTo, sentHeld } from "../../fixtures/settings-pages";
import { recordFocuses } from "../../fixtures/browser";

test("I10 (page): URL refusals are described before focus; a confirmed link first shows its address, then real worker metadata without external image requests; title and URL edits, a duplicate refusal and deletion retain exact activity", async ({
  api,
  db,
  page,
  browserName,
}, info) => {
  const seed = await issueListProject(api, info);
  const issue = await createIssue(api, seed.member.token, seed.project.id, { name: "Read the release guide" });
  const source = await startLinkPage();
  try {
    const watch = await openIssueDetail(page, seed, issue);
    const path = `/api/v0/issues/${issue.id}/links`;
    const bodies = await bodiesSentTo(page, "POST", path);
    await page.getByRole("button", { name: "Add link", exact: true }).click();
    const form = page.getByRole("dialog", { name: "Add link", exact: true });
    const url = form.getByRole("textbox", { name: "URL", exact: true });
    const title = form.getByRole("textbox", { name: "Display title", exact: false });
    const submit = form.getByRole("button", { name: "Add Link", exact: true });
    await url.fill("  ");
    await title.focus();
    const blankFocus = await recordFocuses(page);
    await submit.click();
    await expect(url).toBeFocused();
    expect((await blankFocus()).at(-1)).toEqual({ id: "url", invalid: "true", description: "URL is invalid" });
    expect(bodies).toEqual([]);
    await url.fill("https://bad.test/\u0000");
    await title.focus();
    const refusedFocus = await recordFocuses(page);
    expect((await answerTo(page, "POST", path, () => submit.click())).status()).toBe(422);
    await expect(url).toBeFocused();
    expect((await refusedFocus()).at(-1)).toEqual({ id: "url", invalid: "true", description: "Not allowed" });
    await url.fill(`  ${source.url.slice("http://".length)}  `);
    const held = await sentHeld(page, "POST", path, () => submit.dblclick());
    try {
      expect(held.body).toEqual({ url: source.url.slice("http://".length), title: null });
      await expect
        .poll(() =>
          db.query("SELECT url, metadata FROM issue_links WHERE issue_id=$1 AND deleted_at IS NULL", [issue.id])
        )
        .toEqual([{ url: source.url, metadata: {} }]);
      await page.keyboard.press("Escape");
      await expect(form).toHaveCount(1);
    } finally {
      expect((await held.release()).status()).toBe(201);
    }
    await expect(form).toHaveCount(0);
    expect(watch.apiRequests.filter((request) => request === `POST ${path}`)).toHaveLength(2);
    const linksButton = page.getByRole("button", { name: "Links 1", exact: true });
    await expect(linksButton).toHaveAttribute("aria-expanded", "true");
    const panel = linksButton.locator("../..");
    const anchor = panel.locator(`a[href="${source.url}"]`);
    await expect(anchor).toHaveText(source.url);
    const stored = await listIssueLinks(api, seed.member.token, issue.id);
    const link = stored[0];
    if (!link) throw new Error("the confirmed link is missing");
    expect(stored).toHaveLength(1);
    expect(link.metadata).toEqual({});
    const jobs = await linkCrawlJobs(db, link.id, source.url);
    expect(jobs).toHaveLength(1);
    expect(jobs[0]?.state).not.toBe("completed");
    source.release();
    const firstJob = await completedLinkCrawl(db, link.id, source.url);
    await expect(anchor).toContainText("Release & notes", { timeout: 10000 });
    const row = anchor.locator("../..");
    await expect(row.getByRole("img", { name: "favicon" })).toHaveAttribute("src", /^data:image\/png;base64,/);
    expect(watch.requestUrls.some((requestURL) => requestURL.startsWith(new URL(source.url).origin))).toBe(false);
    const edit = async () => {
      await row.locator('button[aria-haspopup="menu"]').click();
      await page.getByRole("menuitem", { name: "Edit", exact: true }).click();
      return page.getByRole("dialog", { name: "Update link", exact: true });
    };
    let editing = await edit();
    await editing.getByRole("textbox", { name: "Display title", exact: false }).fill("My guide");
    expect(
      (
        await answerTo(page, "PATCH", `/api/v0/issue-links/${link.id}`, () =>
          editing.getByRole("button", { name: "Update Link", exact: true }).click()
        )
      ).status()
    ).toBe(200);
    await expect(editing).toHaveCount(0);
    await expect(anchor).toHaveText("My guide Release & notes");
    expect(await linkCrawlJobs(db, link.id)).toEqual([firstJob]);
    editing = await edit();
    const nextURL = new URL("/next", source.url).href;
    await editing.getByRole("textbox", { name: "URL", exact: true }).fill(nextURL);
    expect(
      (
        await answerTo(page, "PATCH", `/api/v0/issue-links/${link.id}`, () =>
          editing.getByRole("button", { name: "Update Link", exact: true }).click()
        )
      ).status()
    ).toBe(200);
    await expect(editing).toHaveCount(0);
    const moved = panel.locator(`a[href="${nextURL}"]`);
    await expect(moved).toHaveText("My guide Updated release guide", { timeout: 10000 });
    const nextJob = await completedLinkCrawl(db, link.id, nextURL);
    expect(nextJob.id).not.toBe(firstJob.id);
    await expect(anchor).toHaveCount(0);
    await page.getByRole("button", { name: "Add link", exact: true }).click();
    await form.getByRole("textbox", { name: "URL", exact: true }).fill(nextURL);
    await form.getByRole("textbox", { name: "Display title", exact: false }).focus();
    const duplicateFocus = await recordFocuses(page);
    expect((await answerTo(page, "POST", path, () => submit.click())).status()).toBe(409);
    await expect(url).toBeFocused();
    expect((await duplicateFocus()).at(-1)).toEqual({
      id: "url",
      invalid: "true",
      description: "This work item already has this link.",
    });
    await expect(form.getByText("This work item already has this link.", { exact: true })).toHaveCount(1);
    await page.keyboard.press("Escape");
    await expect(form).toHaveCount(0);
    await moved.locator("../..").locator('button[aria-haspopup="menu"]').click();
    expect(
      (
        await answerTo(page, "DELETE", `/api/v0/issue-links/${link.id}`, () =>
          page.getByRole("menuitem", { name: "Delete", exact: true }).click()
        )
      ).status()
    ).toBe(204);
    await expect(moved).toHaveCount(0);
    expect(await listIssueLinks(api, seed.member.token, issue.id)).toEqual([]);
    expect(await linkCrawlJobs(db, link.id)).toEqual([firstJob, nextJob]);
    expect(
      await db.query(
        "SELECT deleted_at = updated_at AS at_write, updated_by_id = $2 AS by_writer FROM issue_links WHERE id=$1",
        [link.id, seed.member.id]
      )
    ).toEqual([{ at_write: true, by_writer: true }]);
    await expectIssueUpdated(db, issue.id, {
      ...madeBy(seed.member.email, issue.name),
      by: seed.member.email,
      wrote: 1,
      activity: [
        ...createdFor(),
        linkCreatedRow(source.url),
        linkUpdatedRow(source.url, nextURL),
        linkDeletedRow(nextURL),
      ],
    });
    await expectListPageQuiet(page, watch, {
      failures: [`422 POST ${path}`, `409 POST ${path}`],
      errors:
        browserName === "chromium"
          ? [
              "Failed to load resource: the server responded with a status of 422 (Unprocessable Entity)",
              "Failed to load resource: the server responded with a status of 409 (Conflict)",
            ]
          : [],
    });
  } finally {
    await source.close();
  }
});

/** A regular project member owns the work item; the workspace admin only prepares membership. */
async function linkIssue(api: Api, info: TestInfo) {
  const admin = await newAccount(api, info, "admin");
  const writer = await newAccount(api, info, "writer");
  const slug = slugFor(info);
  await createWorkspace(api, admin.token, { name: "Link stories", slug });
  await inviteAndAccept(api, admin.token, slug, writer, 15);
  const project = await createProject(api, admin.token, slug, { name: "Web", identifier: "WEB" });
  await addProjectMembers(api, admin.token, project.id, [{ member_id: writer.id, role: 15 }]);
  const issue = await createIssue(api, writer.token, project.id, { name: "Read the release guide" });
  return { writer, issue };
}

test("I10 (API): a member adds a bare local URL, receives pending metadata, crawls its title and embedded icon, changes title without a job, changes URL with a fresh crawl, rejects a duplicate and deletes with activity", async ({
  api,
  db,
}, info) => {
  const { writer, issue } = await linkIssue(api, info);
  const page: LinkPage = await startLinkPage();
  try {
    const link = await createIssueLink(api, writer.token, issue.id, { url: page.url.slice("http://".length) });
    expect(link).toMatchObject({
      issue_id: issue.id,
      url: page.url,
      title: null,
      metadata: {},
      created_by_id: writer.id,
    });
    const stored = () =>
      db.query(
        `SELECT l.*, created_at::text AS creation_stamp, updated_at::text AS update_stamp,
                deleted_at::text AS deletion_stamp FROM issue_links l WHERE id = $1`,
        [link.id]
      );
    const pending = await stored();
    expect(pending).toEqual([
      expect.objectContaining({ id: link.id, issue_id: issue.id, url: page.url, title: null, metadata: {} }),
    ]);
    const queued = await linkCrawlJobs(db, link.id, page.url);
    expect(queued).toEqual([
      expect.objectContaining({ kind: "issue.crawl_link", args: { link_id: link.id, url: page.url } }),
    ]);
    expect(queued[0]!.state).not.toBe("completed");

    page.release();
    const firstJob = await completedLinkCrawl(db, link.id, page.url);
    expect(firstJob.id).toBe(queued[0]!.id);
    const metadata = {
      title: "Release & notes",
      favicon:
        "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+jf1sAAAAASUVORK5CYII=",
      url: page.url,
      favicon_url: new URL("/icon.png", page.url).href,
    };
    expect(await listIssueLinks(api, writer.token, issue.id)).toEqual([{ ...link, metadata }]);
    // Crawl writes only metadata, retaining every audit field and its full timestamp precision.
    expect(await stored()).toEqual([{ ...pending[0]!, metadata }]);

    const beforeTitle = await linkCrawlJobs(db, link.id);
    const titled = await updateIssueLink(api, writer.token, link.id, { title: "My release guide" });
    expect(titled).toMatchObject({ id: link.id, url: page.url, title: "My release guide", metadata });
    expect(await linkCrawlJobs(db, link.id)).toEqual(beforeTitle);
    expect(await stored()).toEqual([expect.objectContaining({ title: "My release guide", metadata })]);

    const nextURL = new URL("/next", page.url).href;
    const moved = await updateIssueLink(api, writer.token, link.id, { url: nextURL });
    expect(moved).toMatchObject({ id: link.id, url: nextURL, title: "My release guide", metadata: {} });
    const nextJob = await completedLinkCrawl(db, link.id, nextURL);
    expect(nextJob.id).not.toBe(firstJob.id);
    const nextMetadata = { ...metadata, title: "Updated release guide", url: nextURL };
    expect(await listIssueLinks(api, writer.token, issue.id)).toEqual([{ ...moved, metadata: nextMetadata }]);
    expect(await stored()).toEqual([expect.objectContaining({ url: nextURL, metadata: nextMetadata })]);
    const crawled = await linkCrawlJobs(db, link.id);
    expect(crawled).toEqual([firstJob, nextJob]);

    const beforeDuplicate = await stored();
    expect(await issueLinkCreated(api, writer.token, issue.id, { url: nextURL })).toEqual({
      status: 409,
      code: "issue.link_exists",
    });
    expect(await stored()).toEqual(beforeDuplicate);
    expect(await linkCrawlJobs(db, link.id)).toEqual(crawled);

    expect(await issueLinkDeleted(api, writer.token, link.id)).toEqual({ status: 204 });
    expect(await listIssueLinks(api, writer.token, issue.id)).toEqual([]);
    expect(
      await db.query(
        `SELECT l.deleted_at = l.updated_at AND l.updated_at = i.updated_at AS at_write,
                l.updated_by_id = i.updated_by_id AND l.updated_by_id = $2 AS by_writer
           FROM issue_links l JOIN issues i ON i.id = l.issue_id WHERE l.id = $1`,
        [link.id, writer.id]
      )
    ).toEqual([{ at_write: true, by_writer: true }]);
    await expectIssueUpdated(db, issue.id, {
      ...madeBy(writer.email, issue.name),
      by: writer.email,
      wrote: 1,
      activity: [...createdFor(), linkCreatedRow(page.url), linkUpdatedRow(page.url, nextURL), linkDeletedRow(nextURL)],
    });
    expect(await linkCrawlJobs(db, link.id)).toEqual(crawled);
  } finally {
    await page.close();
  }
});

test("I10 (API): the metadata endpoint reports a blocked link-local address", async ({ api }, info) => {
  const { writer, issue } = await linkIssue(api, info);
  const link = await createIssueLink(api, writer.token, issue.id, { url: "http://169.254.169.254/i10" });
  // The network's zero-connection property belongs to outbound's Go tests.
  await expect
    .poll(
      async () => (await listIssueLinks(api, writer.token, issue.id)).find((row) => row.id === link.id)?.metadata.error,
      {
        timeout: 10_000,
        intervals: [20, 50, 100],
      }
    )
    .toBe("blocked");
});
