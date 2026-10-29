/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { afterEach, beforeEach, expect, expectTypeOf, it, vi } from "vitest";
import type { IssueListPage, operations } from "@nerve/api-client";
import { ApiError } from "@/lib/api-error";
import { FakeNerve, json } from "@/lib/auth/fake-nerve";
import { settle, until } from "@/lib/auth/fake-time";
import { SessionChangedError } from "@/lib/auth/token-manager";
import { issueOf, issuePageOf, requestedIssue } from "@/store/issue/fake-issues";
import { IssueListService, type MemberIssueQuery, type ProjectIssueQuery } from "./issue-list.service";

beforeEach(() => vi.useFakeTimers());
afterEach(() => vi.useRealTimers());

it("uses generated operation queries, with required member relation and no project filter on a project list", () => {
  expectTypeOf<ProjectIssueQuery>().toEqualTypeOf<NonNullable<operations["listIssues"]["parameters"]["query"]>>();
  expectTypeOf<MemberIssueQuery>().toEqualTypeOf<operations["listMemberIssues"]["parameters"]["query"]>();
  expectTypeOf<"project_id">().not.toExtend<keyof ProjectIssueQuery>();
  expectTypeOf<{}>().not.toExtend<MemberIssueQuery>();
  expectTypeOf<Extract<keyof MemberIssueQuery, "state_id" | "assignee_id" | "sub_group_by" | "archived">>().toBeNever();
});

it("sends project CSV alternatives and repeated date AND conditions without changing its query or page", async () => {
  const nerve = new FakeNerve();
  const service = new IssueListService(nerve.client());
  const query: ProjectIssueQuery = {
    state_id: ["state-a", "state-b"],
    state_group: ["started", "completed"],
    priority: ["high", "none"],
    label_id: ["label", "none"],
    assignee_id: ["me", "none"],
    created_by_id: ["actor", "me"],
    mention_id: ["me", "actor"],
    subscriber_id: ["actor", "me"],
    start_date: ["2026-10-01..", "..2026-10-31"],
    target_date: ["2026-10-15..", "..2026-11-01"],
    sub_issue: "false",
    archived: "true",
    group_by: "state_id",
    sub_group_by: "priority",
    group_key: "state-a",
    sub_group_key: "high",
    order_by: "-updated_at",
    limit: 2,
    cursor: "opaque+/=",
  };
  const original = structuredClone(query);
  const page = issuePageOf([issueOf("one")], "opaque-next");
  const { call, result } = await requestedIssue(nerve, () => service.listProject("A", query), [
    "GET",
    "/api/v0/projects/A/issues",
  ]);
  expect(call.body).toBeUndefined();
  expect(call.queryEntries).toEqual([
    ["state_id", "state-a,state-b"],
    ["state_group", "started,completed"],
    ["priority", "high,none"],
    ["label_id", "label,none"],
    ["assignee_id", "me,none"],
    ["created_by_id", "actor,me"],
    ["mention_id", "me,actor"],
    ["subscriber_id", "actor,me"],
    ["start_date", "2026-10-01.."],
    ["start_date", "..2026-10-31"],
    ["target_date", "2026-10-15.."],
    ["target_date", "..2026-11-01"],
    ["sub_issue", "false"],
    ["archived", "true"],
    ["group_by", "state_id"],
    ["sub_group_by", "priority"],
    ["group_key", "state-a"],
    ["sub_group_key", "high"],
    ["order_by", "-updated_at"],
    ["limit", "2"],
    ["cursor", "opaque+/="],
  ]);
  call.answer(json(200, page));
  await until(() => result.settled, "project page");
  expect(result.value).toEqual(page);
  expect(query).toEqual(original);
  expect(nerve.calls).toHaveLength(1);
});

it("sends the member's required relation and narrower filters as one member request", async () => {
  const nerve = new FakeNerve();
  const service = new IssueListService(nerve.client());
  const { call, result } = await requestedIssue(
    nerve,
    () =>
      service.listMember("acme", "user", {
        relation: "subscribed",
        project_id: ["A", "B"],
        state_group: ["started", "unstarted"],
        priority: ["high", "none"],
        label_id: ["bug", "none"],
        start_date: ["2026-10-01..", "..2026-10-31"],
        target_date: ["2026-10-15..", "..2026-11-01"],
        sub_issue: "true",
        group_by: "project_id",
        group_key: "A",
        order_by: "sort_order",
        limit: 1,
        cursor: "next+/=",
      }),
    ["GET", "/api/v0/workspaces/acme/members/user/issues"]
  );
  expect(call.body).toBeUndefined();
  expect(call.queryEntries).toEqual([
    ["relation", "subscribed"],
    ["project_id", "A,B"],
    ["state_group", "started,unstarted"],
    ["priority", "high,none"],
    ["label_id", "bug,none"],
    ["start_date", "2026-10-01.."],
    ["start_date", "..2026-10-31"],
    ["target_date", "2026-10-15.."],
    ["target_date", "..2026-11-01"],
    ["sub_issue", "true"],
    ["group_by", "project_id"],
    ["group_key", "A"],
    ["order_by", "sort_order"],
    ["limit", "1"],
    ["cursor", "next+/="],
  ]);
  const page: IssueListPage = {
    total: 2,
    groups: [{ key: "A", total: 2, data: [issueOf("one")], next_cursor: "opaque" }],
  };
  call.answer(json(200, page));
  await until(() => result.settled, "member page");
  expect(result.value).toEqual(page);
  expect(nerve.calls).toHaveLength(1);
});

it.each([
  { name: "empty", page: issuePageOf([]) },
  {
    name: "lane",
    page: {
      total: 1,
      groups: [
        { key: "state", total: 1, subgroups: [{ key: "none", total: 1, data: [issueOf("one")], next_cursor: null }] },
      ],
      subgroup_totals: [{ key: "none", total: 1 }],
    },
  },
] satisfies { name: string; page: IssueListPage }[])(
  "preserves $name page branches without response conversion",
  async ({ page }) => {
    const nerve = new FakeNerve();
    const service = new IssueListService(nerve.client());
    const { call, result } = await requestedIssue(nerve, () => service.listProject("A", {}), [
      "GET",
      "/api/v0/projects/A/issues",
    ]);
    expect(call.queryEntries).toEqual([]);
    call.answer(json(200, page));
    await until(() => result.settled, "page branches");
    expect(result.value).toEqual(page);
    expect(nerve.calls).toHaveLength(1);
  }
);

it.each(["project", "member"] as const)(
  "returns the original Problem and session rejection for %s without a follow-up",
  async (scope) => {
    const nerve = new FakeNerve();
    const api = nerve.client();
    const service = new IssueListService(api);
    const request = () =>
      scope === "project" ? service.listProject("A", {}) : service.listMember("acme", "user", { relation: "assigned" });
    const path = scope === "project" ? "/api/v0/projects/A/issues" : "/api/v0/workspaces/acme/members/user/issues";
    const failure = { status: 422, code: "validation_failed", title: "Invalid", detail: "Invalid filter" };
    const { call, result } = await requestedIssue(nerve, request, ["GET", path]);
    call.answer(json(422, failure));
    await until(() => result.settled, "problem");
    expect(result.error).toBeInstanceOf(ApiError);
    expect((result.error as ApiError).problem).toEqual(failure);
    expect((result.error as ApiError).status).toBe(422);
    const replaced = new SessionChangedError();
    api.use({
      onRequest: () => {
        throw replaced;
      },
    });
    expect((await settle(request(), "session rejection")).error).toBe(replaced);
    expect(nerve.calls).toHaveLength(1);
  }
);
