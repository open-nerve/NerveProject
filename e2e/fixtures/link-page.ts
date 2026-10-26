import { createServer } from "node:http";
import { promisify } from "node:util";

export interface LinkPage {
  url: string;
  release(): void;
  close(): Promise<void>;
}

/** A story-owned local page; cancellation also settles an unreleased first request. */
export async function startLinkPage(): Promise<LinkPage> {
  const cancellation = new AbortController();
  let release!: () => void;
  const firstPage = new Promise<void>((resolve) => {
    release = resolve;
  });
  cancellation.signal.addEventListener("abort", release, { once: true });
  const server = createServer(async (request, response) => {
    if (request.url === "/page") await firstPage;
    if (cancellation.signal.aborted || response.destroyed) return;
    if (request.url === "/icon.png") {
      response.writeHead(200, { "Content-Type": "image/png" });
      response.end(
        Buffer.from(
          "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+jf1sAAAAASUVORK5CYII=",
          "base64"
        )
      );
    } else if (request.url === "/page" || request.url === "/next") {
      const title = request.url === "/page" ? "Release &amp; notes" : "Updated release guide";
      response.writeHead(200, { "Content-Type": "text/html; charset=utf-8" });
      response.end(`<html><head><title>${title}</title><link rel="icon" href="/icon.png"></head></html>`);
    } else {
      response.writeHead(404).end();
    }
  });
  let closing: Promise<void> | undefined;
  const close = () => {
    cancellation.abort();
    closing ??= (async () => {
      if (!server.listening) return;
      const stopped = promisify(server.close.bind(server))();
      server.closeAllConnections();
      await stopped;
    })();
    return closing;
  };
  try {
    await new Promise<void>((resolve, reject) => {
      server.once("error", reject);
      server.listen(0, "127.0.0.1", () => {
        server.off("error", reject);
        resolve();
      });
    });
    const address = server.address();
    if (!address || typeof address === "string") throw new Error("link page has no TCP address");
    return { url: `http://127.0.0.1:${address.port}/page`, release, close };
  } catch (error) {
    await close();
    throw error;
  }
}
