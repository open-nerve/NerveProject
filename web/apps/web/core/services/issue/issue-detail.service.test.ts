/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { afterEach, beforeEach, expect, expectTypeOf, it, vi } from "vitest";
import type {
  IssueCollection,
  IssueLink,
  IssueLinkCreate,
  IssueLinkPatch,
  IssueRelationCollection,
  IssueRelationsCreate,
  Problem,
  SubIssuesAdd,
} from "@nerve/api-client";
import { ApiError } from "@/lib/api-error";
import { FakeNerve, json, noContent, type Endpoint } from "@/lib/auth/fake-nerve";
import { settle, until } from "@/lib/auth/fake-time";
import { SessionChangedError } from "@/lib/auth/token-manager";
import { issueOf, requestedIssue } from "@/store/issue/fake-issues";
import { IssueDetailService } from "./issue-detail.service";

beforeEach(() => vi.useFakeTimers());
afterEach(() => vi.useRealTimers());

const children: SubIssuesAdd = { sub_issue_ids: ["child-z", "child-a"] };
const relations: IssueRelationsCreate = { relation_type: "blocked_by", issue_ids: ["target-z", "target-a"] };
const create: IssueLinkCreate = { url: "https://example.com/path", title: null };
const patch: IssueLinkPatch = { title: "" };
const link: IssueLink = {
  id: "link-72",
  issue_id: "source-11",
  url: "https://example.com/normalized",
  title: null,
  metadata: { title: null, favicon: null, favicon_url: null, url: null, error: "blocked" },
  created_by_id: null,
  created_at: "2026-10-30T00:00:00.123456Z",
  updated_at: "2026-10-30T00:00:00.123456Z",
};
const childAnswer: IssueCollection = { data: [issueOf("child-z"), issueOf("child-a")] };
const relationAnswer: IssueRelationCollection = {
  data: [{ id: "relation-93", relation_type: "blocked_by", issue: issueOf("target-24"), created_at: link.created_at }],
};
const operations: {
  name: string;
  endpoint: Endpoint;
  run: (service: IssueDetailService) => Promise<unknown>;
  body?: unknown;
  answer?: unknown;
  status: number;
}[] = [
  {
    name: "listSubIssues",
    endpoint: ["GET", "/api/v0/issues/source-11/sub-issues"],
    run: (s) => s.listSubIssues("source-11"),
    answer: childAnswer,
    status: 200,
  },
  {
    name: "addSubIssues",
    endpoint: ["POST", "/api/v0/issues/source-11/sub-issues"],
    run: (s) => s.addSubIssues("source-11", children),
    body: children,
    answer: childAnswer,
    status: 200,
  },
  {
    name: "listRelations",
    endpoint: ["GET", "/api/v0/issues/source-11/relations"],
    run: (s) => s.listRelations("source-11"),
    answer: relationAnswer,
    status: 200,
  },
  {
    name: "createRelations",
    endpoint: ["POST", "/api/v0/issues/source-11/relations"],
    run: (s) => s.createRelations("source-11", relations),
    body: relations,
    answer: relationAnswer,
    status: 201,
  },
  {
    name: "deleteRelation",
    endpoint: ["DELETE", "/api/v0/issue-relations/relation-93"],
    run: (s) => s.deleteRelation("relation-93"),
    status: 204,
  },
  {
    name: "listLinks",
    endpoint: ["GET", "/api/v0/issues/source-11/links"],
    run: (s) => s.listLinks("source-11"),
    answer: { data: [link] },
    status: 200,
  },
  {
    name: "createLink",
    endpoint: ["POST", "/api/v0/issues/source-11/links"],
    run: (s) => s.createLink("source-11", create),
    body: create,
    answer: link,
    status: 201,
  },
  {
    name: "updateLink",
    endpoint: ["PATCH", "/api/v0/issue-links/link-72"],
    run: (s) => s.updateLink("link-72", patch),
    body: patch,
    answer: { ...link, title: "" },
    status: 200,
  },
  {
    name: "deleteLink",
    endpoint: ["DELETE", "/api/v0/issue-links/link-72"],
    run: (s) => s.deleteLink("link-72"),
    status: 204,
  },
];

it("accepts the generated closed request bodies", () => {
  expectTypeOf<Parameters<IssueDetailService["addSubIssues"]>[1]>().toEqualTypeOf<SubIssuesAdd>();
  expectTypeOf<Parameters<IssueDetailService["createRelations"]>[1]>().toEqualTypeOf<IssueRelationsCreate>();
  expectTypeOf<Parameters<IssueDetailService["createLink"]>[1]>().toEqualTypeOf<IssueLinkCreate>();
  expectTypeOf<Parameters<IssueDetailService["updateLink"]>[1]>().toEqualTypeOf<IssueLinkPatch>();
});

it.each(operations)("usesNativeDetailRoutesAndBodies: $name", async ({ run, endpoint, body, answer, status }) => {
  const nerve = new FakeNerve();
  const service = new IssueDetailService(nerve.client());
  const { call, result } = await requestedIssue(nerve, () => run(service), endpoint);
  expect(call.body).toEqual(body);
  expect(call.queryEntries).toEqual([]);
  call.answer(status === 204 ? noContent() : json(status, answer));
  await until(() => result.settled, "native detail response");
  expect(result.error).toBeUndefined();
  expect(result.value).toEqual(answer);
  expect(nerve.calls).toHaveLength(1);
});

it.each(
  operations.flatMap((operation) =>
    [422, 403, 404, 409].map((status) => ({
      name: operation.name,
      run: operation.run,
      endpoint: operation.endpoint,
      refusal: status,
    }))
  )
)("doesNotTransformRefusals: $name / $refusal", async ({ run, endpoint, refusal }) => {
  const nerve = new FakeNerve();
  const service = new IssueDetailService(nerve.client());
  const { call, result } = await requestedIssue(nerve, () => run(service), endpoint);
  const problem: Problem = {
    status: refusal,
    code: "original.code",
    title: "Original",
    detail: "Original detail",
    errors: [{ field: "url", code: "invalid_format", message: "Invalid URL" }],
  };
  call.answer(json(refusal, problem));
  await until(() => result.settled, "refused detail operation");
  expect(result.value).toBeUndefined();
  expect(result.error).toBeInstanceOf(ApiError);
  expect((result.error as ApiError).status).toBe(refusal);
  expect((result.error as ApiError).problem).toEqual(problem);
  expect(nerve.calls).toHaveLength(1);
});

it.each(operations)("doesNotTransformRefusals: $name / session replacement", async ({ run }) => {
  const nerve = new FakeNerve();
  const api = nerve.client();
  const error = new SessionChangedError();
  api.use({
    onRequest() {
      throw error;
    },
  });
  expect((await settle(run(new IssueDetailService(api)), "replaced session")).error).toBe(error);
  expect(nerve.calls).toHaveLength(0);
});
