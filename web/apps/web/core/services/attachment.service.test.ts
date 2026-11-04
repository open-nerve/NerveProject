/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { AttachmentPage, Problem } from "@nerve/api-client";
import { ApiError } from "@/lib/api-error";
import { FakeNerve, json } from "@/lib/auth/fake-nerve";
import { settle, track, until } from "@/lib/auth/fake-time";
import { SessionChangedError } from "@/lib/auth/token-manager";
import { AttachmentService } from "./attachment.service";

beforeEach(() => vi.useFakeTimers());
afterEach(() => vi.useRealTimers());

const page: AttachmentPage = {
  data: [
    {
      id: "file-8",
      name: "附件.pdf",
      size: 7,
      content_type: "application/pdf",
      created_at: "2026-11-05T00:00:00Z",
      created_by_id: null,
    },
  ],
  next_cursor: "opaque-next",
};
const hosts = [
  { target: { kind: "issue", id: "issue-3" }, path: "/api/v0/issues/issue-3/attachments" },
  { target: { kind: "comment", id: "comment-7" }, path: "/api/v0/issue-comments/comment-7/attachments" },
  { target: { kind: "draft", id: "draft-2" }, path: "/api/v0/drafts/draft-2/attachments" },
] as const;

it.each(hosts)("reads $target.kind pages with the server cursor and default limit", async ({ target, path }) => {
  const nerve = new FakeNerve();
  nerve.respond = (call) => call.answer(json(200, page));
  const service = new AttachmentService(nerve.client());
  expect((await settle(service.list(target), "first page")).value).toEqual(page);
  expect((await settle(service.list(target, "opaque+cursor/=="), "next page")).value).toEqual(page);
  expect(
    nerve.calls.map(({ method, path: actual, body, queryEntries }) => ({ method, path: actual, body, queryEntries }))
  ).toEqual([
    { method: "GET", path, body: undefined, queryEntries: [["limit", "50"]] },
    {
      method: "GET",
      path,
      body: undefined,
      queryEntries: [
        ["limit", "50"],
        ["cursor", "opaque+cursor/=="],
      ],
    },
  ]);
});

it.each(hosts)("keeps $target.kind refusal instead of replacing it with an empty page", async ({ target }) => {
  const nerve = new FakeNerve();
  const problem: Problem = { status: 404, code: "not_found", title: "Not Found", detail: "Not found" };
  nerve.respond = (call) => call.answer(json(404, problem));
  const result = await settle(new AttachmentService(nerve.client()).list(target), "refusal");
  expect(result.error).toBeInstanceOf(ApiError);
  expect(result.error).toMatchObject({ status: 404, problem });
  expect(result.value).toBeUndefined();
});

it.each(hosts)("cancels an outstanding $target.kind page", async ({ target }) => {
  const nerve = new FakeNerve();
  const parent = new AbortController();
  const result = track(new AttachmentService(nerve.client()).list(target, undefined, parent.signal));
  await until(() => nerve.calls.length > 0 || result.settled, "page request");
  expect(nerve.calls).toHaveLength(1);
  parent.abort();
  await until(() => result.settled, "page abort");
  expect(nerve.calls[0]?.aborted()).toBe(true);
  expect(result.error).toMatchObject({ name: "AbortError" });
});

it("preserves a retired session error without wrapping it", async () => {
  const nerve = new FakeNerve();
  const api = nerve.client();
  const original = new SessionChangedError();
  api.use({
    onRequest() {
      throw original;
    },
  });
  expect((await settle(new AttachmentService(api).list(hosts[0].target), "retired session")).error).toBe(original);
  expect(nerve.calls).toHaveLength(0);
});
