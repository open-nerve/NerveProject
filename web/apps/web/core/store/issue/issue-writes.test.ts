/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { issueDetailOf, issueOf, issueStoreTab, requestedIssue } from "./fake-issues";
import { json } from "@/lib/auth/fake-nerve";
import { IssueListBinding } from "./helpers/issue-list-binding";
import { loadProjects, projectOf } from "@/store/project/fake-projects";
vi.stubGlobal("localStorage", { getItem: () => null, setItem: () => {}, removeItem: () => {} });
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: {}, publicClient: {} }));
vi.mock("@/lib/store-context", () => ({ rootStore: undefined }));
let tab: Awaited<ReturnType<typeof issueStoreTab>>;
const version = (n: number) => `2026-10-28T00:00:00.00000000${n}Z`;
beforeEach(async () => {
  vi.useFakeTimers();
  tab = await issueStoreTab();
});
afterEach(() => {
  tab.app.dispose();
  vi.useRealTimers();
});

// The write unit takes the scope its caller gives (P7a spec 2.1): it never works one out from the held copy.
it("confirms a write into the lists covering the scope it is given, whatever project the held copy names", async () => {
  await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [
    tab.project,
    projectOf("B", tab.workspace.id, { id: "B" }),
  ]);
  const given = new IssueListBinding(tab.app.issue, {
    scope: "project",
    workspaceSlug: "ws",
    workspaceID: tab.workspace.id,
    projectID: "B",
    query: { group_by: "state_id", order_by: "sort_order", limit: 10 },
  });
  const listed = { ...issueOf("x", "B"), state_id: "A", sort_order: 5, updated_at: version(1) };
  const read = await requestedIssue(tab.nerve, () => given.fetchFirst(), ["GET", "/api/v0/projects/B/issues"]);
  read.call.answer(
    json(200, {
      total: 1,
      groups: [
        { key: "A", total: 1, data: [listed], next_cursor: null },
        { key: "B", total: 0, data: [], next_cursor: null },
      ],
    })
  );
  await read.promise;
  tab.app.issue.acceptListIssues([{ ...listed, project_id: "A", updated_at: version(2) }], tab.app.issue.captureRead());
  expect(tab.app.issue.issues.getIssueById("x")?.project_id).toBe("A");
  const writing = await requestedIssue(
    tab.nerve,
    () => tab.app.issue.writes.updateIssue("x", () => ({ state_id: "B" }), undefined, tab.app.issue.scope("ws", "B")),
    ["PATCH", "/api/v0/issues/x"]
  );
  writing.call.answer(json(200, { ...issueDetailOf("x", "B"), state_id: "B", sort_order: 5, updated_at: version(3) }));
  await writing.promise;
  await vi.advanceTimersByTimeAsync(1);
  expect(given.group("A")?.rows.map((row) => row.id)).toEqual([]);
  expect(given.group("B")?.rows.map((row) => row.id)).toEqual(["x"]);
});
