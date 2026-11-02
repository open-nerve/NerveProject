/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */

import { request, type IncomingHttpHeaders, type ServerResponse } from "node:http";
import { describe, expect, it, onTestFinished } from "vitest";
import { devServer, reply, signal, upstream } from "./transfer-proxy.fixture";

describe("local transfer streams and owns its connections", () => {
  it("streams an upload before its end and preserves signed method, query, bytes and end-to-end headers", async () => {
    const arrived = signal();
    const received = signal<{ body: Buffer; headers: IncomingHttpHeaders; method?: string; url?: string }>();
    const backend = await upstream((req, res) => {
      const chunks: Buffer[] = [];
      req.on("data", (chunk: Buffer) => {
        chunks.push(chunk);
        arrived.resolve();
      });
      req.on("end", () => {
        received.resolve({ body: Buffer.concat(chunks), headers: req.headers, method: req.method, url: req.url });
        res.writeHead(204, { "Cache-Control": "no-store" }).end();
      });
    });
    const dev = await devServer(backend.url);
    const body = Buffer.from([0, 1, 255, 13, 10, 128]);
    const done = signal<number>();
    const client = request(
      dev.url,
      {
        path: "/static/local/key?tag=exact-tag&payload=exact-payload",
        method: "PUT",
        agent: false,
        headers: {
          "Content-Type": "image/png",
          "Content-Length": body.length,
          Connection: "close, X-Hop",
          "X-Hop": "remove-me",
          "X-Request-ID": "keep-me",
        },
      },
      (res) => {
        res.resume();
        res.on("end", () => done.resolve(res.statusCode ?? 0));
      }
    );
    onTestFinished(() => {
      client.destroy();
    });
    client.on("error", () => {});
    client.write(body.subarray(0, 2));
    await arrived.promise;
    client.end(body.subarray(2));
    expect(await done.promise).toBe(204);
    const got = await received.promise;
    expect(got.body).toEqual(body);
    expect([got.method, got.url]).toEqual(["PUT", "/static/local/key?tag=exact-tag&payload=exact-payload"]);
    expect(got.headers).toMatchObject({
      "content-type": "image/png",
      "content-length": "6",
      "x-request-id": "keep-me",
      host: new URL(backend.url).host,
    });
    expect(got.headers["x-hop"]).toBeUndefined();
    expect(got.headers.authorization).toBeUndefined();
    expect(got.headers.cookie).toBeUndefined();
  });

  it("preserves Range, conditional and HEAD semantics and private response metadata", async () => {
    const seen: IncomingHttpHeaders[] = [];
    const backend = await upstream((req, res) => {
      seen.push(req.headers);
      const headers = {
        "Content-Type": "image/png",
        "Content-Length": "3",
        "Cache-Control": "private, max-age=42",
        ETag: '"version"',
        "X-Content-Type-Options": "nosniff",
        "Content-Disposition": "attachment; filename=test.png",
        Connection: "close, X-Hop",
        "X-Hop": "remove-me",
      };
      if (req.headers["if-none-match"]) res.writeHead(304, headers).end();
      else if (req.method === "HEAD") res.writeHead(200, headers).end();
      else res.writeHead(206, { ...headers, "Content-Range": "bytes 0-2/10" }).end(Buffer.from([0, 255, 10]));
    });
    const dev = await devServer(backend.url);
    const path = "/static/local/key?payload=fake&tag=fake";
    const get = await reply(dev.url, path, "GET", undefined, { Range: "bytes=0-2", "If-Range": '"version"' });
    expect([get.status, get.body]).toEqual([206, Buffer.from([0, 255, 10])]);
    expect(get.headers).toMatchObject({
      "content-range": "bytes 0-2/10",
      "cache-control": "private, max-age=42",
      etag: '"version"',
      "x-content-type-options": "nosniff",
      "content-disposition": "attachment; filename=test.png",
    });
    expect(get.headers["x-hop"]).toBeUndefined();
    expect(seen[0]).toMatchObject({ range: "bytes=0-2", "if-range": '"version"' });
    const head = await reply(dev.url, path, "HEAD");
    expect([head.status, head.body.length, head.headers["content-length"]]).toEqual([200, 0, "3"]);
    const unchanged = await reply(dev.url, path, "GET", undefined, { "If-None-Match": '"version"' });
    expect([unchanged.status, unchanged.body.length]).toEqual([304, 0]);
    expect(seen[2]["if-none-match"]).toBe('"version"');
  });

  it("streams a download before the upstream finishes and truncates an upstream failure without an error tail", async () => {
    const started = signal<ServerResponse>();
    const backend = await upstream((_req, res) => {
      started.resolve(res);
      res.writeHead(200, {
        "Content-Type": "application/octet-stream",
        "Content-Length": "1024",
        "Cache-Control": "private, max-age=3",
      });
      res.write("original bytes");
    });
    const dev = await devServer(backend.url);
    const first = signal();
    const failed = signal();
    const chunks: Buffer[] = [];
    let completed = false;
    const client = request(
      dev.url,
      { path: "/static/local/key?payload=private-marker&tag=tag-marker", agent: false },
      (res) => {
        expect([res.statusCode, res.headers["cache-control"]]).toEqual([200, "private, max-age=3"]);
        res.on("data", (chunk: Buffer) => {
          chunks.push(chunk);
          first.resolve();
        });
        res.on("end", () => {
          completed = true;
        });
        res.on("error", () => failed.resolve());
      }
    );
    onTestFinished(() => {
      client.destroy();
    });
    client.on("error", () => failed.resolve());
    client.end();
    await first.promise;
    (await started.promise).destroy();
    await failed.promise;
    expect(completed).toBe(false);
    expect(Buffer.concat(chunks).toString()).toBe("original bytes");
    expect(dev.logs.join("\n")).toContain("local file transfer");
    expect(dev.logs.join("\n")).not.toMatch(/private-marker|tag-marker|payload=|tag=/);
  });

  for (const end of ["client cancellation", "Vite shutdown"] as const) {
    it(`releases the upstream of a partial upload on ${end}`, async () => {
      const started = signal();
      const closed = signal();
      const backend = await upstream((req) => {
        req.on("data", () => started.resolve());
        req.on("aborted", () => closed.resolve());
        req.on("error", () => {});
      });
      const dev = await devServer(backend.url);
      const client = request(dev.url, {
        path: "/static/local/key?payload=fake&tag=fake",
        method: "PUT",
        headers: { "Content-Length": "10000" },
        agent: false,
      });
      client.on("error", () => {});
      onTestFinished(() => {
        client.destroy();
      });
      client.write("partial upload");
      await started.promise;
      if (end === "client cancellation") client.destroy();
      else await dev.vite.close();
      await closed.promise;
      expect(dev.logs).toEqual([]);
    });
  }

  it("releases the upstream when the reader cancels a download", async () => {
    const closed = signal();
    const backend = await upstream((_req, res) => {
      res.on("close", () => closed.resolve());
      res.writeHead(200, { "Content-Length": "10000" });
      res.write("partial download");
    });
    const dev = await devServer(backend.url);
    const client = request(dev.url, { path: "/static/local/key", agent: false }, (res) => {
      res.on("error", () => {});
      res.once("data", () => res.destroy());
    });
    client.on("error", () => {});
    onTestFinished(() => {
      client.destroy();
    });
    client.end();
    await closed.promise;
    expect(dev.logs).toEqual([]);
  });
});
