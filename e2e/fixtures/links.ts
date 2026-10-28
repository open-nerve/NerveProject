import type { operations } from "@nerve/api-client";
import { expect } from "@playwright/test";

import type { Api } from "./api";
import { answer } from "./api";
import { bearer } from "./auth";
import type { Database } from "./db";

type LinkCreate = operations["createIssueLink"]["requestBody"]["content"]["application/json"];
type Link = operations["createIssueLink"]["responses"][201]["content"]["application/json"];
type LinkPatch = operations["updateIssueLink"]["requestBody"]["content"]["application/json"];

/** The generated create response, including refusal fields for duplicate stories. */
export async function issueLinkCreated(api: Api, token: string, issueId: string, body: LinkCreate) {
  const { data, error, response } = await api.POST("/api/v0/issues/{issue_id}/links", {
    params: { path: { issue_id: issueId } },
    body,
    headers: bearer(token),
  });
  return data ? { status: response.status, link: data } : answer(response, error);
}

/** Creates a link through the issue API for catalog-based deletion stories. */
export async function createIssueLink(api: Api, token: string, issueId: string, body: LinkCreate): Promise<Link> {
  const result = await issueLinkCreated(api, token, issueId, body);
  expect(result.status, `create a link on ${issueId}: ${JSON.stringify(result)}`).toBe(201);
  if (!("link" in result)) throw new Error(`createIssueLink on ${issueId} answered 201 without a link`);
  return result.link;
}

/** Reads the real collection, including its current crawler metadata. */
export async function listIssueLinks(api: Api, token: string, issueId: string): Promise<Link[]> {
  const { data, error, response } = await api.GET("/api/v0/issues/{issue_id}/links", {
    params: { path: { issue_id: issueId } },
    headers: bearer(token),
  });
  expect(response.status, JSON.stringify(error)).toBe(200);
  if (!data) throw new Error("listIssueLinks answered without a collection");
  return data.data;
}

/** Updates a link using the generated patch and response types. */
export async function updateIssueLink(api: Api, token: string, linkId: string, body: LinkPatch): Promise<Link> {
  const { data, error, response } = await api.PATCH("/api/v0/issue-links/{link_id}", {
    params: { path: { link_id: linkId } },
    body,
    headers: bearer(token),
  });
  expect(response.status, JSON.stringify(error)).toBe(200);
  if (!data) throw new Error("updateIssueLink answered without a link");
  return data;
}

/** Deletes the real link, retaining refusal fields. */
export async function issueLinkDeleted(api: Api, token: string, linkId: string) {
  const { error, response } = await api.DELETE("/api/v0/issue-links/{link_id}", {
    params: { path: { link_id: linkId } },
    headers: bearer(token),
  });
  return answer(response, error);
}

/** Stable identities and payloads of this link's tasks, never another link's historical completion. */
export function linkCrawlJobs(db: Database, linkId: string, url?: string) {
  return db.query<{ id: string; kind: string; args: { link_id: string; url: string }; state: string }>(
    `SELECT id::text, kind, args, state::text FROM river_job
      WHERE kind = 'issue.crawl_link' AND args->>'link_id' = $1
        AND ($2::text IS NULL OR args->>'url' = $2) ORDER BY river_job.id`,
    [linkId, url ?? null]
  );
}

/** Waits for exactly this link and URL's one job, with a finite deadline. */
export async function completedLinkCrawl(db: Database, linkId: string, url: string) {
  await expect
    .poll(async () => (await linkCrawlJobs(db, linkId, url)).map((job) => job.state), {
      timeout: 10_000,
      intervals: [20, 50, 100],
      message: `complete the crawl of link ${linkId} at ${url}`,
    })
    .toEqual(["completed"]);
  return (await linkCrawlJobs(db, linkId, url))[0]!;
}
