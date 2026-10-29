/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { Issue, IssueDetail } from "@nerve/api-client";

import { issueOf, issueStoreTab } from "./fake-issues";
vi.stubGlobal("localStorage", { getItem: () => null, setItem: () => {}, removeItem: () => {} });
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: {}, publicClient: {} }));
vi.mock("@/lib/store-context", () => ({}));
let tab: Awaited<ReturnType<typeof issueStoreTab>>;
beforeEach(async () => {
  vi.useFakeTimers();
  tab = await issueStoreTab();
});
afterEach(() => {
  tab.app.dispose();
  vi.useRealTimers();
});
const accept = (issues: (Issue | IssueDetail)[]) => tab.app.issue.acceptListIssues(issues, tab.app.issue.captureRead());

it("keeps_detail_when_a_list_row_arrives", () => {
  const store = tab.app.issue.issues;
  const row: Issue = issueOf("R1");
  const detail: IssueDetail = { ...row, name: "detail name", description_html: "<p>kept</p>" };
  accept([detail]);
  accept([{ ...row, name: "latest list name", priority: "high" }]);
  expect(store.getIssueById(row.id)?.name).toBe("latest list name");
  expect(store.getIssueById(row.id)?.priority).toBe("high");
  expect(store.getIssueDetailById(row.id)?.description_html).toBe("<p>kept</p>");

  const reverse = store;
  accept([{ ...row, id: "reverse" }]);
  accept([{ ...detail, id: "reverse" }]);
  expect(reverse.getIssueDetailById("reverse")?.description_html).toBe("<p>kept</p>");
  expect(reverse.getIssueById("reverse")?.name).toBe("detail name");
});

it("does_not_claim_list_rows_are_details", () => {
  const store = tab.app.issue.issues;
  const row: Issue = issueOf("R1");
  accept([row]);
  expect(store.getIssueDetailById("missing")).toBeUndefined();
  expect(store.getIssueDetailById(row.id)).toBeUndefined();
  const emptyDetail: IssueDetail = { ...row, description_html: "" };
  accept([emptyDetail]);
  expect(store.getIssueDetailById(row.id)).toEqual(emptyDetail);
});

it("keeps_nullable_actor_and_hidden_parent", () => {
  const store = tab.app.issue.issues;
  const row: Issue = { ...issueOf("R1"), created_by_id: null, updated_by_id: null, parent_id: null };
  accept([row]);
  const saved = store.getIssueById(row.id);
  expect(saved?.created_by_id).toBeNull();
  expect(saved?.updated_by_id).toBeNull();
  expect(saved?.parent_id).toBeNull();
  expect(saved).not.toHaveProperty("created_by");
  expect(saved).not.toHaveProperty("updated_by");
  expect(store.getIssueDetailById(row.id)).toBeUndefined();
});
