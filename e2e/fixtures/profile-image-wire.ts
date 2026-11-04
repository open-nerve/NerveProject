import { createServer, request as forward, type ClientRequest } from "node:http";
import { promisify } from "node:util";
import type { PageFileStoryWorld } from "./file-story";
import { test as base } from "./test";

interface ImageWireRequest {
  method: string;
  path: string;
  headers: {
    authorization?: string;
    cookie?: string;
    referer?: string;
    "content-type"?: string;
    "content-length"?: string;
  };
  body: Buffer;
}
export interface ImageWire {
  baseURL: string;
  requests: ImageWireRequest[];
}

/** Observe browser bytes before forwarding unchanged to real nerve; WebKit's debug API omits Cookie/File data. */
async function observeImageWire(upstreamURL: string): Promise<ImageWire & { close(): Promise<void> }> {
  const requests: ImageWireRequest[] = [];
  const pending = new Set<ClientRequest>();
  const server = createServer((incoming, outgoing) => {
    const url = new URL(incoming.url ?? "/", upstreamURL);
    const wanted = incoming.method === "PUT" || url.searchParams.has("image-cookie-probe");
    if (wanted) {
      const chunks: Buffer[] = [];
      incoming.on("data", (chunk: Buffer) => chunks.push(chunk));
      incoming.on("end", () =>
        requests.push({
          method: incoming.method ?? "",
          path: url.pathname,
          headers: {
            authorization: incoming.headers.authorization,
            cookie: incoming.headers.cookie,
            referer: incoming.headers.referer,
            "content-type": incoming.headers["content-type"],
            "content-length": incoming.headers["content-length"],
          },
          body: Buffer.concat(chunks),
        })
      );
    }
    const upstream = forward(url, { method: incoming.method, headers: incoming.headers }, (response) => {
      outgoing.writeHead(response.statusCode ?? 502, response.headers);
      response.pipe(outgoing);
    });
    pending.add(upstream);
    upstream.once("close", () => pending.delete(upstream));
    upstream.once("error", () => {
      if (!outgoing.destroyed) outgoing.destroy();
    });
    incoming.once("aborted", () => upstream.destroy());
    outgoing.once("close", () => {
      if (!outgoing.writableFinished) upstream.destroy();
    });
    incoming.pipe(upstream);
  });
  let closing: Promise<void> | undefined;
  const close = () =>
    (closing ??= (async () => {
      for (const request of pending) request.destroy();
      if (!server.listening) return;
      const stopped = promisify(server.close.bind(server))();
      server.closeAllConnections();
      await stopped;
    })());
  try {
    await new Promise<void>((resolve, reject) => {
      server.once("error", reject);
      server.listen(0, "127.0.0.1", () => {
        server.off("error", reject);
        resolve();
      });
    });
    const address = server.address();
    if (!address || typeof address === "string") throw new Error("Image wire has no TCP port");
    return { baseURL: `http://127.0.0.1:${address.port}`, requests, close };
  } catch (error) {
    await close();
    throw error;
  }
}

/** Only P6 local image stories opt into the real receiving-side observation. */
export const test = base.extend<{ wire: ImageWire; imageWorld: PageFileStoryWorld }>({
  wire: async ({ nerve }, use) => {
    const wire = await observeImageWire(nerve.baseURL);
    try {
      await use(wire);
    } finally {
      await wire.close();
    }
  },
  baseURL: async ({ wire }, use) => {
    await use(wire.baseURL);
  },
  imageWorld: async ({ api, db, nerve, page, wire }, use) => {
    await use({ api, db, nerve, page, wire, baseURL: wire.baseURL, uploadStatus: 204 });
  },
});
