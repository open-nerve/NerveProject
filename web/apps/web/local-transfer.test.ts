/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */

import { mkdir, writeFile } from "node:fs/promises";
import { join } from "node:path";
import { build } from "vite";
import { describe, expect, it } from "vitest";
import { localTransfer } from "./local-transfer";
import { buildRoot, devServer, reply, upstream } from "./transfer-proxy.fixture";

describe("local file namespace in the real Vite server", () => {
  it("delegates only the file tree, including intermediate and encoded paths, without changing raw query bytes", async () => {
    const paths: string[] = [];
    const backend = await upstream((req, res) => {
      paths.push(req.url ?? "");
      res.writeHead(404, { "Cache-Control": "no-store" }).end("file response");
    });
    const dev = await devServer(backend.url);
    const query = "?tag=signature-marker&payload=payload-marker";
    const filePaths = [
      "/static",
      "/static/",
      "/static/local/key",
      "/static//local/key",
      "/%73tatic/local/key",
      "/elsewhere/../static/local/../../settings",
      "/elsewhere/%2e%2e//static/local/../../settings",
      "/elsewhere/../static%2flocal/../../settings",
      "/elsewhere/../%73tatic/local/%FF",
    ];
    await Promise.all(
      filePaths.map(async (path) => {
        const result = await reply(dev.url, path + query);
        expect([result.status, result.body.toString(), result.headers["cache-control"]]).toEqual([
          404,
          "file response",
          "no-store",
        ]);
        expect(paths).toContain(path + query);
      })
    );
    const count = paths.length;
    await Promise.all(
      ["/storage/projects", "/static-more/projects", "/assets/storage-more/projects"].map(async (path) => {
        expect((await reply(dev.url, path)).body.toString()).toContain("ordinary workspace");
      })
    );
    expect((await reply(dev.url, "/assets/app-hash.js")).body.toString()).toBe("export {};\n");
    const emoji = await reply(dev.url, "/assets/emojibase/15.3.2/en/messages.json");
    expect(JSON.parse(emoji.body.toString())).toHaveProperty("groups");
    expect((await reply(dev.url, "/assets/missing.js", "GET", undefined, { Accept: "text/javascript" })).status).toBe(
      404
    );
    expect(paths).toHaveLength(count);
  });

  it("returns a fixed no-store failure and logs no signed URL when the upstream is unavailable", async () => {
    const backend = await upstream((_req, res) => res.end());
    await new Promise<void>((resolve) => backend.server.close(() => resolve()));
    const dev = await devServer(backend.url);
    const result = await reply(dev.url, "/static/local/key?payload=secret-payload&tag=secret-tag");
    expect([result.status, result.headers["cache-control"]]).toEqual([502, "no-store"]);
    expect(result.body.toString()).toBe("Local file transfer unavailable\n");
    expect(dev.logs.join("\n")).toContain("local file transfer");
    expect(dev.logs.join("\n")).not.toMatch(/secret-|payload=|tag=|\/static\/local/);
  });
});

describe("file namespace is not build output", () => {
  it("rejects public files under static", async () => {
    const root = await buildRoot();
    await mkdir(join(root, "public/static"), { recursive: true });
    await writeFile(join(root, "public/static/object"), "wrong owner");
    await expect(
      build({ root, configFile: false, logLevel: "silent", plugins: [localTransfer("http://127.0.0.1:8080")] })
    ).rejects.toThrow("/static");
  });

  it.each([undefined, "post"] as const)(
    "rejects static bundle output from a later plugin (order=%s)",
    async (order) => {
      const root = await buildRoot();
      await expect(
        build({
          root,
          configFile: false,
          logLevel: "silent",
          plugins: [
            localTransfer("http://127.0.0.1:8080"),
            {
              name: "test-file-root-collision",
              generateBundle: {
                order,
                handler() {
                  this.emitFile({ type: "asset", fileName: "static/object", source: "wrong owner" });
                },
              },
            },
          ],
        })
      ).rejects.toThrow("/static");
    }
  );
});
