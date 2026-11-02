/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */

import { mkdtemp, mkdir, realpath, rm, writeFile } from "node:fs/promises";
import { createServer as createHTTPServer, request, type RequestListener, type Server } from "node:http";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { createLogger, createServer, type Plugin } from "vite";
import { onTestFinished } from "vitest";
import { emojibase } from "./emojibase";
import { localTransfer } from "./local-transfer";

export async function listen(server: Server): Promise<string> {
  await new Promise<void>((resolve, reject) => {
    server.once("error", reject);
    server.listen(0, "127.0.0.1", () => {
      server.removeListener("error", reject);
      resolve();
    });
  });
  onTestFinished(async () => {
    server.closeAllConnections();
    if (server.listening) await new Promise<void>((resolve) => server.close(() => resolve()));
  });
  const address = server.address();
  if (!address || typeof address === "string") throw new Error("fixture has no TCP address");
  return `http://127.0.0.1:${address.port}`;
}

export async function upstream(handler: RequestListener) {
  const server = createHTTPServer(handler);
  return { server, url: await listen(server) };
}

export async function buildRoot() {
  const root = await realpath(await mkdtemp(join(tmpdir(), "nerve-local-transfer-")));
  onTestFinished(() => rm(root, { recursive: true, force: true }));
  await writeFile(join(root, "index.html"), "<html><body>ordinary workspace</body></html>");
  return root;
}

export async function devServer(target: string, plugins: Plugin[] = []) {
  const root = await buildRoot();
  await mkdir(join(root, "public/assets"), { recursive: true });
  await writeFile(join(root, "public/assets/app-hash.js"), "export {};\n");
  const logs: string[] = [];
  const logger = createLogger();
  for (const level of ["info", "warn", "warnOnce", "error"] as const) {
    logger[level] = (message, options) =>
      logs.push(`${message} ${options && "error" in options ? (options.error?.stack ?? "") : ""}`);
  }
  const vite = await createServer({
    root,
    configFile: false,
    customLogger: logger,
    plugins: [localTransfer(target), emojibase(), ...plugins],
    optimizeDeps: { noDiscovery: true, include: [] },
    server: { middlewareMode: true, watch: null, hmr: false, preTransformRequests: false },
  });
  onTestFinished(() => vite.close());
  // Vite treats port=0 as its default. A real HTTP listener around its supported
  // middleware mode gives each parallel test an OS-assigned, owned port.
  const url = await listen(createHTTPServer(vite.middlewares));
  return { vite, url, logs };
}

// Node's path option preserves dot segments and query bytes, unlike URL/fetch.
export function reply(base: string, path: string, method = "GET", body?: Buffer, headers: Record<string, string> = {}) {
  return new Promise<{ status: number; headers: import("node:http").IncomingHttpHeaders; body: Buffer }>(
    (resolve, reject) => {
      const req = request(base, { path, method, headers, agent: false }, (res) => {
        const chunks: Buffer[] = [];
        res.on("data", (chunk: Buffer) => chunks.push(chunk));
        res.on("error", reject);
        res.on("end", () =>
          resolve({ status: res.statusCode ?? 0, headers: res.headers, body: Buffer.concat(chunks) })
        );
      });
      req.on("error", reject);
      req.end(body);
    }
  );
}

export function signal<T = void>() {
  let resolve!: (value: T) => void;
  const promise = new Promise<T>((done) => {
    resolve = done;
  });
  return { promise, resolve };
}
