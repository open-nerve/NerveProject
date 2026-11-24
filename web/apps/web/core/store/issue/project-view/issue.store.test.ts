/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// A project's view's page's list (P6a spec 2.5, R1): the project's list under the page's draft of the view, a list of
// its own beside the project page's; whether it is current is the project's and the view's draft's. The session's
// stores and HTTP are real: the issue root's view page reads the view through the session's view store.
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { View } from "@nerve/api-client";
import { json } from "@/lib/auth/fake-nerve";
import { viewOf } from "@/store/views/fake-views";
import { issueOf, issuePageOf, issueStoreTab, requestedIssue } from "../fake-issues";

vi.mock("@/lib/auth/api-client", () => ({ tokenManager: {}, publicClient: {} }));
const binding = vi.hoisted(() => ({ app: undefined as unknown }));
vi.mock("@/lib/store-context", () => ({
  get rootStore() {
    return binding.app;
  },
}));
let tab: Awaited<ReturnType<typeof issueStoreTab>>;
beforeEach(async () => {
  vi.stubGlobal("localStorage", { getItem: () => null, setItem: () => {}, removeItem: () => {} });
  vi.useFakeTimers();
  tab = await issueStoreTab();
  binding.app = tab.app;
});
afterEach(() => {
  tab.app.dispose();
  vi.useRealTimers();
  vi.unstubAllGlobals();
});

const LIST: [string, string] = ["GET", "/api/v0/projects/A/issues"];
const high = { and: [{ priority__in: "high" }] };

it("lists the project under the view's draft, in a list of its own beside the project page's", async () => {
  const { app, nerve } = tab;
  const issues = app.issue.projectViewIssues;
  const filter = app.issue.projectViewIssuesFilter;
  const view: View = viewOf(tab.project, "Urgent", "u-me", { filters: { priority: ["urgent"] } });
  const read = await requestedIssue(nerve, () => filter.fetchFilters("ws", view.id), [
    "GET",
    `/api/v0/views/${view.id}`,
  ]);
  read.call.answer(json(200, view));
  await read.promise;

  const first = await requestedIssue(
    nerve,
    () => issues.fetchIssues("ws", "A", view.id, "init-loader", { limit: 20 }),
    LIST
  );
  expect(first.call.query).toMatchObject({ priority: "urgent", order_by: "-created_at", limit: "20" });
  first.call.answer(json(200, issuePageOf([issueOf("x")])));
  await first.promise;
  expect(issues.getIssueIds()).toEqual(["x"]);
  expect(issues.isPreferenceQueryCurrent("A", view.id)).toBe(true);
  expect(issues.isPreferenceQueryCurrent("B", view.id), "another project's list").toBe(false);

  // the page changes its draft: its list is no longer the draft's, and is read again under the draft, not the view
  await filter.updateFilterExpression("ws", view.id, high);
  expect(issues.isPreferenceQueryCurrent("A", view.id)).toBe(false);
  const again = await requestedIssue(nerve, () => issues.fetchIssuesWithExistingPagination("ws", "A", view.id), LIST);
  expect(again.call.query).toMatchObject({ priority: "high", limit: "20" });
  again.call.answer(json(200, issuePageOf([issueOf("y")])));
  await again.promise;
  expect(issues.isPreferenceQueryCurrent("A", view.id)).toBe(true);
  expect(issues.getIssueIds()).toEqual(["y"]);

  // the project page's list is its own, under its own preferences, and leaves the view page's as it is
  const project = await requestedIssue(
    nerve,
    () => app.issue.projectIssues.fetchIssues("ws", "A", "init-loader", { limit: 20 }),
    LIST
  );
  expect(project.call.query.priority).toBeUndefined();
  project.call.answer(json(200, issuePageOf([issueOf("z")])));
  await project.promise;
  expect(app.issue.projectIssues.getIssueIds()).toEqual(["z"]);
  expect(issues.getIssueIds()).toEqual(["y"]);
  expect(app.issue.projectIssues.currentList).not.toBe(issues.currentList);
  expect(app.issue.projectIssuesFilter.filters.A).toBeUndefined();
  expect(nerve.calls.filter((call) => call.method !== "GET")).toEqual([]);
});
