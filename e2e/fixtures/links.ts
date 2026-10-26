import type { operations } from "@nerve/api-client";
import { expect } from "@playwright/test";

import type { Api } from "./api";
import { bearer } from "./auth";

type LinkCreate = operations["createIssueLink"]["requestBody"]["content"]["application/json"];
type Link = operations["createIssueLink"]["responses"][201]["content"]["application/json"];

/** Creates a link through the issue API for catalog-based deletion stories. */
export async function createIssueLink(api: Api, token: string, issueId: string, body: LinkCreate): Promise<Link> {
  const { data, error, response } = await api.POST("/api/v0/issues/{issue_id}/links", {
    params: { path: { issue_id: issueId } },
    body,
    headers: bearer(token),
  });
  expect(response.status, `create a link on ${issueId}: ${JSON.stringify(error)}`).toBe(201);
  if (!data) throw new Error(`createIssueLink on ${issueId} answered 201 without a link`);
  return data;
}
