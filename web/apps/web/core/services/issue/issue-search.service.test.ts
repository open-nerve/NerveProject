/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { FakeNerve, json, problem } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { SessionChangedError } from "@/lib/auth/token-manager";
import { issueOf, requestedIssue } from "@/store/issue/fake-issues";
import { IssueSearchService, type IssueSearchQuery } from "./issue-search.service";

beforeEach(() => vi.useFakeTimers());
afterEach(() => vi.useRealTimers());
const cases: { query: IssueSearchQuery; entries: [string, string][] }[] = [
  {
    query: { project_id: "A", target_date: ["none"] },
    entries: [
      ["project_id", "A"],
      ["target_date", "none"],
    ],
  },
  {
    query: { project_id: "A", for: "sub_issue", issue_id: "parent", q: "标题" },
    entries: [
      ["project_id", "A"],
      ["for", "sub_issue"],
      ["issue_id", "parent"],
      ["q", "标题"],
    ],
  },
  {
    query: { project_id: "A", for: "relation", issue_id: "source" },
    entries: [
      ["project_id", "A"],
      ["for", "relation"],
      ["issue_id", "source"],
    ],
  },
  {
    query: { for: "relation", issue_id: "source", limit: 100 },
    entries: [
      ["for", "relation"],
      ["issue_id", "source"],
      ["limit", "100"],
    ],
  },
];
it.each(cases)("usesNativeSearchQueriesForAllThreeConsumers: $query", async ({ query, entries }) => {
  const nerve = new FakeNerve();
  const service = new IssueSearchService(nerve.client());
  const { call, promise } = await requestedIssue(nerve, () => service.search("ws", query), [
    "GET",
    "/api/v0/workspaces/ws/issue-search",
  ]);
  expect(call.queryEntries).toEqual(entries);
  expect(call.body).toBeUndefined();
  const answer = { data: [issueOf("native")] };
  call.answer(json(200, answer));
  expect(await promise).toEqual(answer);
});
it.each([403, 422])("preserves search refusal %s", async (status) => {
  const nerve = new FakeNerve();
  const { call, result } = await requestedIssue(nerve, () => new IssueSearchService(nerve.client()).search("ws", {}), [
    "GET",
    "/api/v0/workspaces/ws/issue-search",
  ]);
  call.answer(problem(status, "original.code"));
  await until(() => result.settled, "search refusal");
  expect(result.error).toMatchObject({ status, problem: { code: "original.code" } });
  expect(result.value).toBeUndefined();
});
it("preserves session retirement without sending a search", async () => {
  const nerve = new FakeNerve();
  const api = nerve.client();
  FakeNerve.replaceSession(api);
  await expect(new IssueSearchService(api).search("ws", {})).rejects.toBeInstanceOf(SessionChangedError);
  expect(nerve.calls).toHaveLength(0);
});
