import { execFile } from "node:child_process";
import { promisify } from "node:util";
import { createServer } from "node:http";
import { mkdtemp, readFile, rm } from "node:fs/promises";
import { tmpdir } from "node:os";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { test as base } from "./test";

/** Serve a separately built production-control fixture; no test-only route enters the application. */
export const test = base.extend<object, { filterBrowserURL: string }>({
  filterBrowserURL: [
    // oxlint-disable-next-line no-empty-pattern -- Playwright reads fixture dependencies from this pattern
    async ({}, use) => {
      const directory = await mkdtemp(path.join(tmpdir(), "nerve-filter-browser-"));
      const web = fileURLToPath(new URL("../../web/apps/web/", import.meta.url));
      const server = createServer(async (request, response) => {
        const pathname = new URL(request.url ?? "/", "http://localhost").pathname;
        const file = path.resolve(directory, `.${pathname === "/" ? "/index.html" : pathname}`);
        if (!file.startsWith(`${directory}${path.sep}`)) {
          response.writeHead(403).end();
          return;
        }
        try {
          const content = await readFile(file);
          const types: Record<string, string> = {
            ".html": "text/html",
            ".js": "text/javascript",
            ".css": "text/css",
            ".woff2": "font/woff2",
          };
          response
            .writeHead(200, { "Content-Type": types[path.extname(file)] ?? "application/octet-stream" })
            .end(content);
        } catch {
          response.writeHead(404).end();
        }
      });
      try {
        await promisify(execFile)(process.execPath, ["scripts/filter-browser/build.mjs", directory], {
          cwd: web,
          maxBuffer: 4 * 1024 * 1024,
        });
        await new Promise<void>((resolve, reject) => {
          server.once("error", reject);
          server.listen(0, "127.0.0.1", resolve);
        });
        const address = server.address();
        if (!address || typeof address === "string") throw new Error("Browser fixture did not bind a TCP port");
        await use(`http://127.0.0.1:${address.port}`);
      } finally {
        server.closeAllConnections();
        if (server.listening)
          await new Promise<void>((resolve, reject) =>
            server.close((error) => {
              if (error) {
                reject(error);
                return;
              }
              resolve();
            })
          );
        await rm(directory, { recursive: true, force: true });
      }
    },
    { scope: "worker", timeout: 120_000 },
  ],
});
