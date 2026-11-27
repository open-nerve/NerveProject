/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// A workspace's page holds its preferences by a page id of its own (M7/P6b spec 2.4, R2, R3): a saved view's is the
// view's id; a default view's is its key in its workspace, since the four keys repeat in every workspace and the
// preference services take an id alone. A default view's own term, `me` in it, is fixed in its query, and its key is
// left out of the words its page filters by (§2.5, R4).
import { describe, expect, it } from "vitest";
import { defaultViewOfPage, ownTerm, workspacePageId, workspacePageWords } from "./page-id";

const SAVED = "019a0000-0000-7000-8000-0000000000f1";
const ELEVEN = [
  "priority",
  "state_group",
  "state_id",
  "label_id",
  "assignee_id",
  "created_by_id",
  "mention_id",
  "subscriber_id",
  "project_id",
  "start_date",
  "target_date",
];

describe("a workspace's page id", () => {
  it("is a default view's key in its workspace, and a saved view's id", () => {
    expect(workspacePageId("acme", "assigned")).toBe(JSON.stringify(["acme", "assigned"]));
    expect(workspacePageId("beta", "assigned")).not.toBe(workspacePageId("acme", "assigned"));
    expect(workspacePageId("acme", "all-issues")).not.toBe(workspacePageId("acme", "assigned"));
    expect(workspacePageId("acme", SAVED)).toBe(SAVED);
  });

  it("names its default view back, and none for a saved view or anything else", () => {
    for (const key of ["all-issues", "assigned", "created", "subscribed"] as const)
      expect(defaultViewOfPage(workspacePageId("acme", key))).toBe(key);
    expect(defaultViewOfPage(SAVED)).toBeUndefined();
    for (const page of ['["acme","mine"]', '["acme"]', '[1,"assigned"]', '["acme","assigned","x"]', "[", "{}"])
      expect(defaultViewOfPage(page)).toBeUndefined();
  });
});

describe.each([
  { key: "all-issues", own: {}, words: ELEVEN },
  { key: "assigned", own: { assignee_id: ["me"] }, words: ELEVEN.filter((word) => word !== "assignee_id") },
  { key: "created", own: { created_by_id: ["me"] }, words: ELEVEN.filter((word) => word !== "created_by_id") },
  { key: "subscribed", own: { subscriber_id: ["me"] }, words: ELEVEN.filter((word) => word !== "subscriber_id") },
  { key: SAVED, own: {}, words: ELEVEN },
])("the page of $key", ({ key, own, words }) => {
  it("has its own term fixed in its query", () => {
    expect(ownTerm(key)).toEqual(own);
  });
  it("filters by the contract's words, its own key not among them", () => {
    expect(workspacePageWords(key)).toEqual(words);
  });
});
