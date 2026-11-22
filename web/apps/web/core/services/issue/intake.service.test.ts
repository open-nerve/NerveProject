/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// The intake's operations (P5a spec 2.1), against a fake nerve: one method per operation, nerve's answer as it is.
import { afterEach, beforeEach, expect, expectTypeOf, it, vi } from "vitest";
import type { operations } from "@nerve/api-client";
import { ApiError } from "@/lib/api-error";
import { FakeNerve, json, problem } from "@/lib/auth/fake-nerve";
import { settle } from "@/lib/auth/fake-time";
import { countsOf, intakeIssueOf, pageOf } from "@/store/issue/intake/fake-intake";
import { type IntakeListQuery, IntakeService } from "./intake.service";

beforeEach(() => vi.useFakeTimers());
afterEach(() => vi.useRealTimers());

it("counts the pending work items of a workspace's intakes by its slug and gives nerve's answer as it is", async () => {
  const nerve = new FakeNerve();
  const counts = countsOf({ "p-web": 3, "p-api": 0 });
  nerve.respond = (call) => call.answer(json(200, counts));
  const service = new IntakeService(nerve.client());
  expect((await settle(service.counts("acme"), "the counts")).value).toEqual(counts);
  expect(nerve.calls.map(({ method, path, query, body }) => ({ method, path, query, body }))).toEqual([
    { method: "GET", path: "/api/v0/workspaces/acme/intake-counts", query: {}, body: undefined },
  ]);
});

it("hands nerve's refusal of the counts to the caller", async () => {
  const nerve = new FakeNerve();
  nerve.respond = (call) => call.answer(problem(404, "workspace.not_found"));
  const service = new IntakeService(nerve.client());
  const read = await settle(service.counts("gone"), "the counts");
  expect(read.error).toBeInstanceOf(ApiError);
  expect(read.error).toMatchObject({ status: 404, problem: { code: "workspace.not_found" } });
});

it("lists a project's intake by its id, the query as it is: arrays as one CSV parameter, limit a number", async () => {
  expectTypeOf<IntakeListQuery>().toEqualTypeOf<NonNullable<operations["listIntakeIssues"]["parameters"]["query"]>>();
  expectTypeOf<IntakeListQuery["limit"]>().toEqualTypeOf<number | undefined>();
  const nerve = new FakeNerve();
  const page = pageOf([intakeIssueOf("one")], "c-2", 3);
  nerve.respond = (call) => call.answer(json(200, page));
  const service = new IntakeService(nerve.client());
  const query: IntakeListQuery = {
    status: ["pending", "snoozed"],
    priority: ["high", "urgent"],
    assignee_id: ["me", "none"],
    created_by_id: ["me", "u-ann"],
    label_id: ["l-bug", "none"],
    created_at: "2026-11-01..2026-11-07",
    updated_at: "2026-11-08..",
    order_by: "-updated_at",
    limit: 50,
    cursor: "opaque+/=",
  };
  expect((await settle(service.list("p-web", query), "the list")).value).toEqual(page);
  expect(nerve.calls.map(({ method, path, queryEntries, body }) => ({ method, path, queryEntries, body }))).toEqual([
    {
      method: "GET",
      path: "/api/v0/projects/p-web/intake-issues",
      queryEntries: [
        ["status", "pending,snoozed"],
        ["priority", "high,urgent"],
        ["assignee_id", "me,none"],
        ["created_by_id", "me,u-ann"],
        ["label_id", "l-bug,none"],
        ["created_at", "2026-11-01..2026-11-07"],
        ["updated_at", "2026-11-08.."],
        ["order_by", "-updated_at"],
        ["limit", "50"],
        ["cursor", "opaque+/="],
      ],
      body: undefined,
    },
  ]);
});

it("hands nerve's refusal of a list to the caller", async () => {
  const nerve = new FakeNerve();
  nerve.respond = (call) => call.answer(problem(422, "not_allowed"));
  const service = new IntakeService(nerve.client());
  const read = await settle(service.list("p-web", { status: ["pending"], limit: 50 }), "the list");
  expect(read.error).toBeInstanceOf(ApiError);
  expect(read.error).toMatchObject({ status: 422, problem: { code: "not_allowed" } });
});
