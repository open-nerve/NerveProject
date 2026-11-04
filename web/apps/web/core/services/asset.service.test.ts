/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type {
  Asset,
  Content,
  DraftAssetPreparationPage,
  DuplicateAsset,
  NewAsset,
  PreparedAsset,
  Problem,
} from "@nerve/api-client";
import { ApiError } from "@/lib/api-error";
import { FakeNerve, json, noContent } from "@/lib/auth/fake-nerve";
import { settle, track, until } from "@/lib/auth/fake-time";
import { SessionChangedError } from "@/lib/auth/token-manager";
import { AssetService } from "./asset.service";

beforeEach(() => vi.useFakeTimers());
afterEach(() => vi.useRealTimers());

const input: NewAsset = {
  purpose: "USER_AVATAR",
  scope: { kind: "user" },
  name: "头像.png",
  size: 9,
  content_type: "image/png",
};
const asset: Asset = {
  ...input,
  id: "a-19",
  state: "uploaded",
  is_bound: false,
  width: 3,
  height: 2,
  created_at: "2026-11-04T00:00:00Z",
  updated_at: "2026-11-04T00:00:01Z",
};
const prepared: PreparedAsset = {
  asset: { ...asset, state: "pending", width: null, height: null },
  upload: {
    method: "PUT",
    url: "/static/file?sig=abc",
    headers: [{ name: "Content-Type", value: "image/png" }],
    expires_at: "2026-11-04T00:10:00Z",
  },
};
type Answer = Asset | Content | DraftAssetPreparationPage | PreparedAsset | void;
const copy: DuplicateAsset = {
  purpose: "ISSUE_DESCRIPTION",
  scope: { kind: "issue", id: "target-8" },
  name: "copy.png",
};
const content: Content = { url: "/assets/storage/local/a?signature=secret", expires_at: "2026-11-05T01:00:00Z" };
const draftPage: DraftAssetPreparationPage = {
  data: [{ ...asset, purpose: "DRAFT_ISSUE_DESCRIPTION", scope: { kind: "draft", id: "draft-5" } }],
  next_cursor: "opaque-next",
};
const operations: {
  name: string;
  method: "POST" | "GET" | "DELETE";
  path: string;
  body?: NewAsset | DuplicateAsset | Record<string, never>;
  query?: [string, string][];
  answer: Answer;
  run: (service: AssetService, signal?: AbortSignal) => Promise<Answer>;
}[] = [
  {
    name: "prepare",
    method: "POST",
    path: "/api/v0/assets",
    body: input,
    answer: prepared,
    run: (s: AssetService, signal?: AbortSignal) => s.prepare(input, signal),
  },
  {
    name: "complete",
    method: "POST",
    path: "/api/v0/assets/a-19/complete",
    body: {},
    answer: asset,
    run: (s: AssetService, signal?: AbortSignal) => s.complete("a-19", signal),
  },
  {
    name: "read",
    method: "GET",
    path: "/api/v0/assets/a-19",
    answer: asset,
    run: (s, signal) => s.read("a-19", signal),
  },
  {
    name: "remove",
    method: "DELETE",
    path: "/api/v0/assets/a-19",
    answer: undefined,
    run: (s, signal) => s.remove("a-19", signal),
  },
  {
    name: "duplicate source directly without a target preflight",
    method: "POST",
    path: "/api/v0/assets/a-19/duplicate",
    body: copy,
    answer: { ...asset, ...copy, id: "copy-20" },
    run: (s, signal) => s.duplicate("a-19", copy, signal),
  },
  {
    name: "content without a download query",
    method: "GET",
    path: "/api/v0/assets/a-19/content",
    answer: content,
    run: (s, signal) => s.content("a-19", signal),
  },
  {
    name: "first preparation page",
    method: "GET",
    path: "/api/v0/drafts/draft-5/asset-preparations",
    query: [["limit", "50"]],
    answer: draftPage,
    run: (s, signal) => s.listDraftPreparations("draft-5", undefined, signal),
  },
  {
    name: "next preparation page",
    method: "GET",
    path: "/api/v0/drafts/draft-5/asset-preparations",
    query: [
      ["limit", "50"],
      ["cursor", "opaque+cursor/=="],
    ],
    answer: { data: [], next_cursor: null },
    run: (s, signal) => s.listDraftPreparations("draft-5", "opaque+cursor/==", signal),
  },
];

it.each(operations)(
  "$name sends the native JSON request and returns the generated response",
  async ({ method, path, body, query, answer, run }) => {
    const nerve = new FakeNerve();
    nerve.respond = (call) => call.answer(answer === undefined ? noContent() : json(200, answer));
    const result = await settle(run(new AssetService(nerve.client())), "asset response");
    expect(nerve.calls).toHaveLength(1);
    expect(nerve.calls[0]).toMatchObject({ method, path, body, queryEntries: query ?? [] });
    expect(result.error).toBeUndefined();
    expect(result.value).toEqual(answer);
  }
);

it.each(operations)("$name preserves API problem fields", async ({ run }) => {
  const nerve = new FakeNerve();
  const problem: Problem = {
    status: 409,
    code: "asset.object_missing",
    title: "Conflict",
    detail: "Missing",
    errors: [{ field: "name", code: "invalid_format", message: "Invalid name" }],
  };
  nerve.respond = (call) => call.answer(json(409, problem));
  const { error } = await settle(run(new AssetService(nerve.client())), "asset problem");
  expect(error).toBeInstanceOf(ApiError);
  expect(error).toMatchObject({ status: 409, problem });
});

it.each(operations)("$name propagates parent cancellation to its request", async ({ run }) => {
  const nerve = new FakeNerve();
  const parent = new AbortController();
  const result = track(run(new AssetService(nerve.client()), parent.signal));
  await until(() => nerve.calls.length > 0 || result.settled, "asset request");
  expect(nerve.calls).toHaveLength(1);
  parent.abort();
  await until(() => result.settled, "asset abort");
  expect(nerve.calls[0]?.aborted()).toBe(true);
  expect(result.error).toMatchObject({ name: "AbortError" });
});

it.each(operations)("$name preserves middleware errors without wrapping", async ({ run }) => {
  const nerve = new FakeNerve();
  const api = nerve.client();
  const original = new SessionChangedError();
  api.use({
    onRequest() {
      throw original;
    },
  });
  expect((await settle(run(new AssetService(api)), "retired session")).error).toBe(original);
  expect(nerve.calls).toHaveLength(0);
});
