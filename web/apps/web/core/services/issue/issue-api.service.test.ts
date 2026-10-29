/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { afterEach, beforeEach, expect, expectTypeOf, it, vi } from "vitest";
import type { IssueCreate, IssueUpdate } from "@nerve/api-client";
import { ApiError } from "@/lib/api-error";
import { FakeNerve, json, noContent } from "@/lib/auth/fake-nerve";
import { settle, until } from "@/lib/auth/fake-time";
import { SessionChangedError } from "@/lib/auth/token-manager";
import { issueDetailOf, requestedIssue } from "@/store/issue/fake-issues";
import { IssueApiService } from "./issue-api.service";

beforeEach(() => vi.useFakeTimers());
afterEach(() => vi.useRealTimers());

const create: IssueCreate = {
  name: "New issue",
  description_html: "<p>New</p>",
  state_id: "started",
  priority: "high",
  assignee_ids: ["user"],
  label_ids: ["bug"],
  start_date: "2026-10-29",
  target_date: "2026-11-01",
  parent_id: "parent",
};
const update: IssueUpdate = {
  name: "Renamed",
  description_html: "",
  state_id: "done",
  priority: "none",
  assignee_ids: [],
  label_ids: [],
  start_date: null,
  target_date: null,
  parent_id: null,
  sort_order: 0,
};
const changes = [
  { method: "get", verb: "GET", path: "/api/v0/issues/one", body: undefined },
  { method: "create", verb: "POST", path: "/api/v0/projects/A/issues", body: create },
  { method: "update", verb: "PATCH", path: "/api/v0/issues/one", body: update },
  { method: "delete", verb: "DELETE", path: "/api/v0/issues/one", body: undefined },
  { method: "archive", verb: "POST", path: "/api/v0/issues/one/archive", body: undefined },
  { method: "unarchive", verb: "POST", path: "/api/v0/issues/one/unarchive", body: undefined },
] as const;

async function invoke(service: IssueApiService, method: (typeof changes)[number]["method"]) {
  if (method === "create") return service.create("A", create);
  if (method === "update") return service.update("one", update);
  return service[method]("one");
}

it("accepts generated closed create/update bodies, never partial entities", () => {
  expectTypeOf<Parameters<IssueApiService["create"]>[1]>().toEqualTypeOf<IssueCreate>();
  expectTypeOf<Parameters<IssueApiService["update"]>[1]>().toEqualTypeOf<IssueUpdate>();
  expectTypeOf<Extract<keyof IssueCreate, "id" | "project_id" | "updated_at" | "created_by_id">>().toBeNever();
  expectTypeOf<Extract<keyof IssueUpdate, "id" | "project_id" | "updated_at" | "created_by_id">>().toBeNever();
  expectTypeOf<IssueUpdate["skip_activity"]>().toEqualTypeOf<boolean | undefined>();
  expectTypeOf<{}>().not.toExtend<IssueCreate>();
});

it.each(changes)(
  "$method sends exactly its method, path and body, returning the server answer",
  async ({ method, verb, path, body }) => {
    const nerve = new FakeNerve();
    const service = new IssueApiService(nerve.client());
    const { call, result } = await requestedIssue(nerve, () => invoke(service, method), [verb, path]);
    expect(call.queryEntries).toEqual([]);
    expect(call.body).toEqual(body);
    const detail = { ...issueDetailOf("one"), name: "Server normalized", updated_at: "2026-10-29T10:00:00.123456Z" };
    call.answer(method === "delete" ? noContent() : json(method === "create" ? 201 : 200, detail));
    await until(() => result.settled, "confirmed issue");
    expect(result.error).toBeUndefined();
    expect(result.value).toEqual(method === "delete" ? undefined : detail);
    expect(nerve.calls).toHaveLength(1);
  }
);

it("sends skip_activity as a boolean only alongside the explicit HTML update", async () => {
  const nerve = new FakeNerve();
  const service = new IssueApiService(nerve.client());
  const { call, result } = await requestedIssue(
    nerve,
    () => service.update("one", { description_html: "", skip_activity: true }),
    ["PATCH", "/api/v0/issues/one"]
  );
  expect(call.body).toEqual({ description_html: "", skip_activity: true });
  call.answer(json(200, { ...issueDetailOf("one"), description_html: "" }));
  await until(() => result.settled, "HTML update");
  expect(result.value?.description_html).toBe("");
  expect(nerve.calls).toHaveLength(1);
});

it.each(changes)(
  "$method returns the original Problem and session rejection without a follow-up",
  async ({ method, verb, path }) => {
    const nerve = new FakeNerve();
    const api = nerve.client();
    const service = new IssueApiService(api);
    const request = () => invoke(service, method);
    const { call, result } = await requestedIssue(nerve, request, [verb, path]);
    const failure = { status: 403, code: "forbidden", title: "Forbidden", detail: "No permission" };
    call.answer(json(403, failure));
    await until(() => result.settled, "refused issue");
    expect(result.value).toBeUndefined();
    expect(result.error).toBeInstanceOf(ApiError);
    expect((result.error as ApiError).problem).toEqual(failure);
    expect((result.error as ApiError).status).toBe(403);
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
