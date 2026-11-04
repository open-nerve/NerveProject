/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { PreparedAsset } from "@nerve/api-client";
import { track } from "@/lib/auth/fake-time";
import { putAsset, UploadFailure } from "./asset-upload";

const now = Date.parse("2026-11-04T00:00:00Z");
const file = new File([new Uint8Array([0, 255, 3, 8])], "private-name.png", { type: "image/png" });
const grant = (patch: Partial<PreparedAsset["upload"]> = {}): PreparedAsset["upload"] => ({
  method: "PUT",
  url: "/static/objects/a?token=private%2Fsignature+v&x=1&x=2",
  headers: [{ name: "Content-Type", value: "image/png" }],
  expires_at: new Date(now + 60_000).toISOString(),
  ...patch,
});
const parent = () => new AbortController();
beforeEach(() => {
  vi.useFakeTimers();
  vi.setSystemTime(now);
});
afterEach(() => {
  vi.unstubAllGlobals();
  vi.restoreAllMocks();
  vi.useRealTimers();
});

function wire() {
  let answer!: (response: Response) => void;
  let fail!: (error: unknown) => void;
  let request: { url: RequestInfo | URL; init?: RequestInit } | undefined;
  vi.stubGlobal("fetch", (url: RequestInfo | URL, init?: RequestInit) => {
    request = { url, init };
    return new Promise<Response>((resolve, reject) => {
      answer = resolve;
      fail = reject;
    });
  });
  return {
    get request() {
      return request;
    },
    answer: (response: Response) => answer(response),
    fail: (error: unknown) => fail(error),
  };
}

it.each([
  "/static/objects/a?token=a%2Fb+c&same=1&same=2",
  "https://nerve.test/static/a?token=a%2fb",
  "https://s3.test/bucket/key?X-Signature=a%2Fb%20c&same=1&same=2",
  "http://127.0.0.1:9999/bucket/file?token=a+b",
])("uploads original File and untouched signed URL without session credentials: %s", async (url) => {
  const network = wire();
  const result = putAsset(file, grant({ url }), parent().signal);
  expect(network.request?.url).toBe(url);
  const init = network.request?.init;
  expect(init).toMatchObject({ method: "PUT", credentials: "omit", redirect: "error", referrerPolicy: "no-referrer" });
  expect(init?.body).toBe(file);
  expect([...new Headers(init?.headers)]).toEqual([["content-type", "image/png"]]);
  network.answer(new Response(null, { status: 204 }));
  await expect(result).resolves.toBeUndefined();
  expect(vi.getTimerCount()).toBe(0);
});

it.each([200, 201, 204, 299])("accepts %i and releases the unused response body", async (status) => {
  let cancelled = false;
  vi.stubGlobal(
    "fetch",
    async () =>
      new Response(
        status === 204
          ? null
          : new ReadableStream({
              cancel() {
                cancelled = true;
              },
            }),
        { status }
      )
  );
  await expect(putAsset(file, grant(), parent().signal)).resolves.toBeUndefined();
  if (status !== 204) expect(cancelled).toBe(true);
  expect(vi.getTimerCount()).toBe(0);
});

it.each([403, 500, 308])("reports only HTTP %i, never XML or capability details", async (status) => {
  vi.stubGlobal("fetch", async () => new Response("<Error>private-signature private-name.png</Error>", { status }));
  const error = await putAsset(file, grant(), parent().signal).catch((failure: unknown) => failure);
  expect(error).toBeInstanceOf(UploadFailure);
  expect(error).toMatchObject({ kind: "http", status });
  expect(String(error)).not.toMatch(/private|signature|<Error>|static/);
  expect(error).not.toHaveProperty("cause");
  expect(vi.getTimerCount()).toBe(0);
});

it("redacts fetch errors (including rejected redirects)", async () => {
  vi.stubGlobal("fetch", async () => {
    throw new TypeError("private-name.png at /static/?token=secret");
  });
  const error = await putAsset(file, grant(), parent().signal).catch((failure: unknown) => failure);
  expect(error).toMatchObject({ kind: "network" });
  expect(String(error)).not.toMatch(/private|static|secret/);
  expect(vi.getTimerCount()).toBe(0);
});

it.each([
  "//evil.test/file",
  "relative/file",
  "ftp://s3.test/file",
  "data:text/plain,file",
  "javascript:alert(1)",
  "https://user:password@s3.test/file",
  "https://@s3.test/file",
  "https://s3.test/file#part",
  "https://s3.test/file#",
  "/static/file#",
  "https://s3.test/\nfile",
  " /static/file",
  "https:\\s3.test\\file",
  "/\\evil.test/file",
  "https:///s3.test/file",
])("rejects malformed grant address before any request: %s", async (url) => {
  const network = wire();
  await expect(putAsset(file, grant({ url }), parent().signal)).rejects.toMatchObject({ kind: "invalid-grant" });
  expect(network.request).toBeUndefined();
  expect(vi.getTimerCount()).toBe(0);
});

it.each(
  [
    [],
    [{ name: "Authorization", value: "Bearer secret" }],
    [
      { name: "Content-Type", value: "image/png" },
      { name: "content-type", value: "image/png" },
    ],
    [
      { name: "Content-Type", value: "image/png" },
      { name: "Cookie", value: "session=secret" },
    ],
    [{ name: "Content-Type", value: "image/png\r\nCookie: secret" }],
    [{ name: "Content-Type", value: "" }],
  ].map((headers) => ({ headers }))
)("rejects missing, extra, duplicate or invalid headers", async ({ headers }) => {
  const network = wire();
  await expect(putAsset(file, grant({ headers }), parent().signal)).rejects.toMatchObject({ kind: "invalid-grant" });
  expect(network.request).toBeUndefined();
});

it("rejects a wrong method and invalid expiry without sending a request", async () => {
  const network = wire();
  await expect(putAsset(file, grant({ method: "POST" as "PUT" }), parent().signal)).rejects.toMatchObject({
    kind: "invalid-grant",
  });
  await expect(putAsset(file, grant({ expires_at: "unknown" }), parent().signal)).rejects.toMatchObject({
    kind: "invalid-grant",
  });
  await expect(
    putAsset(file, grant({ expires_at: new Date(now).toISOString() }), parent().signal)
  ).rejects.toMatchObject({ kind: "expired" });
  expect(network.request).toBeUndefined();
});

it("does not stop an active upload when the start authorization expires", async () => {
  const network = wire();
  const result = track(putAsset(file, grant({ expires_at: new Date(now + 1).toISOString() }), parent().signal));
  await vi.advanceTimersByTimeAsync(60_000);
  expect(result.settled).toBe(false);
  expect(network.request?.init?.signal?.aborted).toBe(false);
  network.answer(new Response(null, { status: 204 }));
  await vi.advanceTimersByTimeAsync(0);
  expect(result.settled).toBe(true);
  expect(result.error).toBeUndefined();
});

it.each(["parent", "timeout"])(
  "actively aborts a stuck fetch, preserving the first %s terminal cause",
  async (first) => {
    const network = wire();
    const controller = parent();
    const removed = vi.spyOn(controller.signal, "removeEventListener");
    const result = track(putAsset(file, grant(), controller.signal));
    if (first === "parent") controller.abort("private reason");
    await vi.advanceTimersByTimeAsync(15 * 60_000);
    if (first === "timeout") controller.abort();
    expect(network.request?.init?.signal?.aborted).toBe(true);
    expect(result.error).toMatchObject({ kind: first === "parent" ? "cancelled" : "timeout" });
    expect(removed).toHaveBeenCalledWith("abort", expect.any(Function));
    expect(vi.getTimerCount()).toBe(0);
    network.answer(new Response(null, { status: 204 }));
    await vi.advanceTimersByTimeAsync(0);
    expect(result.error).toMatchObject({ kind: first === "parent" ? "cancelled" : "timeout" });
  }
);

it("pre-aborted input makes no request or timer", async () => {
  const network = wire();
  const controller = parent();
  controller.abort();
  await expect(putAsset(file, grant(), controller.signal)).rejects.toMatchObject({ kind: "cancelled" });
  expect(network.request).toBeUndefined();
  expect(vi.getTimerCount()).toBe(0);
});

it("success removes the parent listener, so a later abort cannot change the result", async () => {
  const controller = parent();
  const removed = vi.spyOn(controller.signal, "removeEventListener");
  const network = wire();
  const result = putAsset(file, grant(), controller.signal);
  expect(network.request).toBeDefined();
  network.answer(new Response(null, { status: 204 }));
  await result;
  controller.abort();
  expect(removed).toHaveBeenCalledWith("abort", expect.any(Function));
  expect(network.request?.init?.signal?.aborted).toBe(false);
  expect(vi.getTimerCount()).toBe(0);
});
