/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { expect, it, vi } from "vitest";
import type { Issue, IssueDetail } from "@nerve/api-client";
import { IssueStore } from "./issue.store";
import { issueOf } from "./fake-issues";
vi.mock("@/lib/store-context", () => ({
  rootStore: { projectRoot: { project: { getProjectIdentifierById: () => "P" } } },
}));

it("keeps_detail_when_a_list_row_arrives", () => {
  const store = new IssueStore();
  const row: Issue = issueOf("R1");
  const detail: IssueDetail = { ...row, name: "detail name", description_html: "<p>kept</p>" };
  store.addIssue([detail]);
  store.addIssue([{ ...row, name: "latest list name", priority: "high" }]);
  expect(store.getIssueById(row.id)?.name).toBe("latest list name");
  expect(store.getIssueById(row.id)?.priority).toBe("high");
  expect(store.getIssueDetailById(row.id)?.description_html).toBe("<p>kept</p>");

  const reverse = new IssueStore();
  reverse.addIssue([row]);
  reverse.addIssue([detail]);
  expect(reverse.getIssueDetailById(row.id)?.description_html).toBe("<p>kept</p>");
  expect(reverse.getIssueById(row.id)?.name).toBe("detail name");
});

it("does_not_claim_list_rows_are_details", () => {
  const store = new IssueStore();
  const row: Issue = issueOf("R1");
  store.addIssue([row]);
  expect(store.getIssueDetailById("missing")).toBeUndefined();
  expect(store.getIssueDetailById(row.id)).toBeUndefined();
  const emptyDetail: IssueDetail = { ...row, description_html: "" };
  store.addIssue([emptyDetail]);
  expect(store.getIssueDetailById(row.id)).toEqual(emptyDetail);
});

it("keeps_nullable_actor_and_hidden_parent", () => {
  const store = new IssueStore();
  const row: Issue = { ...issueOf("R1"), created_by_id: null, updated_by_id: null, parent_id: null };
  store.addIssue([row]);
  const saved = store.getIssueById(row.id);
  expect(saved?.created_by_id).toBeNull();
  expect(saved?.updated_by_id).toBeNull();
  expect(saved?.parent_id).toBeNull();
  expect(saved).not.toHaveProperty("created_by");
  expect(saved).not.toHaveProperty("updated_by");
  expect(store.getIssueDetailById(row.id)).toBeUndefined();
});
