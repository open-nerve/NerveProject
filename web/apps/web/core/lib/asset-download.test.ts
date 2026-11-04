// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { Asset, Content } from "@nerve/api-client";
import { settle, track, until } from "./auth/fake-time";
import { downloadAsset } from "./asset-download";

const now = Date.parse("2026-11-05T00:00:00Z");
const asset: Asset = {
  id: "file-5",
  purpose: "ISSUE_ATTACHMENT",
  scope: { kind: "issue", id: "issue-9" },
  name: "原始文件.bin",
  size: 4,
  content_type: "application/octet-stream",
  state: "uploaded",
  is_bound: true,
  width: null,
  height: null,
  created_at: "2026-11-05T00:00:00Z",
  updated_at: "2026-11-05T00:00:00Z",
};
const content: Content = {
  url: "/assets/storage/local/file?signature=a%2Fb+c&x=1&x=2",
  expires_at: new Date(now + 60_000).toISOString(),
};
const signal = () => new AbortController().signal;
let saved: Blob | undefined;
let clicked: { href: string; name: string; attached: boolean }[];
let revoked: string[];

beforeEach(() => {
  vi.useFakeTimers();
  vi.setSystemTime(now);
  saved = undefined;
  clicked = [];
  revoked = [];
  // jsdom cannot perform a native download. Observe the boundary's Blob, link and cleanup instead.
  vi.spyOn(URL, "createObjectURL").mockImplementation((value) => {
    if (value instanceof Blob) saved = value;
    return "blob:nerve/download";
  });
  vi.spyOn(URL, "revokeObjectURL").mockImplementation((url) => {
    revoked.push(url);
  });
  vi.spyOn(HTMLAnchorElement.prototype, "click").mockImplementation(function (this: HTMLAnchorElement) {
    clicked.push({ href: this.href, name: this.download, attached: this.isConnected });
  });
});
afterEach(() => {
  document.body.replaceChildren();
  vi.unstubAllGlobals();
  vi.restoreAllMocks();
  vi.useRealTimers();
});

function wire() {
  let answer!: (response: Response) => void;
  let request: { url: RequestInfo | URL; init?: RequestInit } | undefined;
  vi.stubGlobal("fetch", (url: RequestInfo | URL, init?: RequestInit) => {
    request = { url, init };
    return new Promise<Response>((resolve) => {
      answer = resolve;
    });
  });
  return {
    get request() {
      return request;
    },
    answer: (response: Response) => answer(response),
  };
}

function responseBytes(chunks: number[][]) {
  return new Response(
    new ReadableStream<Uint8Array>({
      start(controller) {
        chunks.forEach((bytes) => controller.enqueue(new Uint8Array(bytes)));
        controller.close();
      },
    }),
    { headers: { "Content-Length": "999999999" } }
  );
}

function bytesOf(blob: Blob): Promise<number[]> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.addEventListener(
      "load",
      () => {
        if (reader.result instanceof ArrayBuffer) resolve([...new Uint8Array(reader.result)]);
        else reject(new Error("Expected binary download"));
      },
      { once: true }
    );
    reader.addEventListener("error", () => reject(reader.error), { once: true });
    reader.readAsArrayBuffer(blob);
  });
}

it.each([content.url, "https://s3.test/bucket/key?signature=a%2fb%20c&same=1&same=2", "http://127.0.0.1:9000/a?x=a+b"])(
  "downloads exact streamed bytes using the untouched capability without business credentials: %s",
  async (url) => {
    const network = wire();
    const parent = new AbortController();
    const removed = vi.spyOn(parent.signal, "removeEventListener");
    const result = track(downloadAsset({ asset, content: { ...content, url } }, parent.signal));
    expect(network.request?.url).toBe(url);
    expect(network.request?.init).toMatchObject({
      method: "GET",
      credentials: "omit",
      redirect: "error",
      referrerPolicy: "no-referrer",
    });
    expect([...new Headers(network.request?.init?.headers)]).toEqual([]);
    expect(network.request?.init?.body).toBeUndefined();
    network.answer(
      responseBytes([
        [0, 255],
        [3, 8],
      ])
    );
    await until(() => result.settled, "download");
    expect(result.error).toBeUndefined();
    expect(clicked).toEqual([{ href: "blob:nerve/download", name: "原始文件.bin", attached: true }]);
    expect(saved).toBeInstanceOf(Blob);
    expect(saved?.type).toBe("application/octet-stream");
    expect((await settle(bytesOf(saved!), "downloaded Blob bytes")).value).toEqual([0, 255, 3, 8]);
    expect(revoked).toEqual(["blob:nerve/download"]);
    expect(document.querySelector("a")).toBeNull();
    expect(removed).toHaveBeenCalledWith("abort", expect.any(Function));
    expect(vi.getTimerCount()).toBe(0);
  }
);

it.each([
  "//evil.test/a",
  "a/file",
  "https://user:pass@s3.test/a",
  "https://@s3.test/a",
  "https:///s3.test/a",
  "/\\evil.test/a",
  "/assets/a#",
  "javascript:alert(1)",
  "https://s3.test/a\n",
  "https://s3.test/a b",
])("rejects an invalid capability before fetching: %s", async (url) => {
  const network = wire();
  const result = await settle(downloadAsset({ asset, content: { ...content, url } }, signal()), "invalid capability");
  expect(result.error).toMatchObject({ kind: "invalid-grant" });
  expect(network.request).toBeUndefined();
  expect(vi.getTimerCount()).toBe(0);
});

it.each([
  { expires_at: "unknown", kind: "invalid-grant" },
  { expires_at: new Date(now).toISOString(), kind: "expired" },
])("rejects $kind expiry before fetching", async ({ expires_at, kind }) => {
  const network = wire();
  expect(
    (await settle(downloadAsset({ asset, content: { ...content, expires_at } }, signal()), "expiry")).error
  ).toMatchObject({ kind });
  expect(network.request).toBeUndefined();
});

it.each([0, -1, 1.5, NaN, Infinity, 104857601])(
  "rejects invalid size %s without allocating or fetching",
  async (size) => {
    const network = wire();
    expect((await settle(downloadAsset({ asset: { ...asset, size }, content }, signal()), "size")).error).toMatchObject(
      { kind: "invalid-size" }
    );
    expect(network.request).toBeUndefined();
    expect(saved).toBeUndefined();
  }
);

it("rejects a short body despite an advertised Content-Length", async () => {
  vi.stubGlobal("fetch", async () => responseBytes([[0, 255]]));
  expect((await settle(downloadAsset({ asset, content }, signal()), "short read")).error).toMatchObject({
    kind: "size-mismatch",
  });
  expect(clicked).toEqual([]);
  expect(saved).toBeUndefined();
  expect(vi.getTimerCount()).toBe(0);
});

it("cancels an oversized stream immediately instead of buffering the rest", async () => {
  let cancelled = false;
  vi.stubGlobal(
    "fetch",
    async () =>
      new Response(
        new ReadableStream({
          start(controller) {
            controller.enqueue(new Uint8Array([0, 1, 2, 3, 4]));
          },
          cancel() {
            cancelled = true;
          },
        })
      )
  );
  expect((await settle(downloadAsset({ asset, content }, signal()), "long read")).error).toMatchObject({
    kind: "size-mismatch",
  });
  expect(cancelled).toBe(true);
  expect(saved).toBeUndefined();
  expect(vi.getTimerCount()).toBe(0);
});

it.each([403, 500, 308])("discards failed HTTP %i bodies without leaking their content", async (status) => {
  let cancelled = false;
  vi.stubGlobal(
    "fetch",
    async () =>
      new Response(
        new ReadableStream({
          cancel() {
            cancelled = true;
          },
        }),
        { status }
      )
  );
  const result = await settle(downloadAsset({ asset, content }, signal()), "HTTP failure");
  expect(result.error).toMatchObject({ kind: "http", status });
  expect(cancelled).toBe(true);
  expect(saved).toBeUndefined();
  expect(vi.getTimerCount()).toBe(0);
});

it.each(["fetch", "stream"])("redacts a %s failure and cleans the download", async (where) => {
  vi.stubGlobal("fetch", async () => {
    if (where === "fetch") throw new TypeError("private-signature file-name");
    return new Response(
      new ReadableStream({
        start(controller) {
          controller.error(new Error("private-signature file-name"));
        },
      })
    );
  });
  const { error } = await settle(downloadAsset({ asset, content }, signal()), "network failure");
  expect(error).toMatchObject({ kind: "network" });
  expect(String(error)).not.toMatch(/private|signature|file-name/);
  expect(error).not.toHaveProperty("cause");
  expect(saved).toBeUndefined();
  expect(vi.getTimerCount()).toBe(0);
});

it.each(["cancelled", "timeout"])("settles a stuck fetch on %s and discards its late body", async (kind) => {
  const network = wire();
  const parent = new AbortController();
  const result = track(downloadAsset({ asset, content }, parent.signal));
  if (kind === "cancelled") parent.abort("private reason");
  await vi.advanceTimersByTimeAsync(kind === "timeout" ? 15 * 60_000 : 0);
  expect(result.settled).toBe(true);
  expect(result.error).toMatchObject({ kind });
  expect(network.request?.init?.signal?.aborted).toBe(true);
  let cancelled = false;
  network.answer(
    new Response(
      new ReadableStream({
        cancel() {
          cancelled = true;
        },
      })
    )
  );
  await vi.advanceTimersByTimeAsync(0);
  expect(cancelled).toBe(true);
  expect(saved).toBeUndefined();
  expect(vi.getTimerCount()).toBe(0);
});

it.each(["cancelled", "timeout"])(
  "interrupts a stalled reader on %s without downloading partial data",
  async (kind) => {
    let reading = false;
    let cancelled = false;
    vi.stubGlobal(
      "fetch",
      async () =>
        new Response(
          new ReadableStream({
            pull() {
              reading = true;
            },
            cancel() {
              cancelled = true;
            },
          })
        )
    );
    const parent = new AbortController();
    const result = track(downloadAsset({ asset, content }, parent.signal));
    await until(() => reading, "active reader");
    if (kind === "cancelled") parent.abort();
    await vi.advanceTimersByTimeAsync(kind === "timeout" ? 15 * 60_000 : 0);
    expect(result.error).toMatchObject({ kind });
    expect(cancelled).toBe(true);
    expect(clicked).toEqual([]);
    expect(vi.getTimerCount()).toBe(0);
  }
);

it("does not start a pre-cancelled download", async () => {
  const network = wire();
  const parent = new AbortController();
  parent.abort();
  expect((await settle(downloadAsset({ asset, content }, parent.signal), "cancelled start")).error).toMatchObject({
    kind: "cancelled",
  });
  expect(network.request).toBeUndefined();
  expect(vi.getTimerCount()).toBe(0);
});

it("releases the object URL and element even if the browser refuses the download action", async () => {
  vi.stubGlobal("fetch", async () => responseBytes([[0, 255, 3, 8]]));
  vi.spyOn(HTMLAnchorElement.prototype, "click").mockImplementation(() => {
    throw new Error("private filename");
  });
  const { error } = await settle(downloadAsset({ asset, content }, signal()), "download action");
  expect(error).toMatchObject({ kind: "download" });
  expect(String(error)).not.toContain("private");
  expect(revoked).toEqual(["blob:nerve/download"]);
  expect(document.querySelector("a")).toBeNull();
  expect(vi.getTimerCount()).toBe(0);
});
