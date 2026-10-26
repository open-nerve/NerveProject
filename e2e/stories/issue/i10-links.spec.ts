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
