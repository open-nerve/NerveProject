/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */

import { request as httpRequest, type IncomingHttpHeaders, type IncomingMessage, type ServerResponse } from "node:http";
import { request as httpsRequest } from "node:https";

// Wider than Go's maximum 60-second transfer idle budget. Go owns the business
// deadline; this only bounds dead development connections on either side.
const IDLE_TIMEOUT_MS = 120_000;

function endToEnd(headers: IncomingHttpHeaders): IncomingHttpHeaders {
  const hop = new Set([
    "connection",
    "keep-alive",
    "proxy-authenticate",
    "proxy-authorization",
    "te",
    "trailer",
    "transfer-encoding",
    "upgrade",
  ]);
  for (const name of (headers.connection ?? "").split(",")) hop.add(name.trim().toLowerCase());
  return Object.fromEntries(Object.entries(headers).filter(([name]) => !hop.has(name)));
}

function unavailable(response: ServerResponse) {
  response.writeHead(502, {
    "Cache-Control": "no-store",
    "Content-Type": "text/plain; charset=utf-8",
    Connection: "close",
  });
  response.end("Local file transfer unavailable\n");
}

/** Owns raw byte streams. It neither validates grants nor handles JSON APIs. */
export function createTransferProxy(target: string, logFailure: (message: string) => void) {
  const origin = new URL(target);
  if (
    !["http:", "https:"].includes(origin.protocol) ||
    origin.username ||
    origin.password ||
    origin.pathname !== "/" ||
    origin.search ||
    origin.hash
  ) {
    throw new Error("local transfer target must be an HTTP origin");
  }
  const request = origin.protocol === "https:" ? httpsRequest : httpRequest;
  const active = new Set<() => void>();
  let closed = false;

  return {
    forward(input: IncomingMessage, output: ServerResponse) {
      if (closed) {
        unavailable(output);
        return;
      }
      let ended = false;
      let upstreamResponse: IncomingMessage | undefined;
      const upstream = request(origin, {
        method: input.method,
        path: input.url,
        headers: { ...endToEnd(input.headers), host: origin.host },
        agent: false,
      });

      const release = () => {
        if (ended) return;
        ended = true;
        active.delete(stop);
        input.unpipe(upstream);
        upstreamResponse?.unpipe(output);
        upstreamResponse?.destroy();
        upstream.destroy();
      };
      const stop = () => {
        release();
        output.destroy();
      };
      const fail = (category: "upstream" | "timeout") => {
        if (ended) return;
        release();
        // Never forward arbitrary Error.message/stack: it may contain a signed
        // URL. The fixed category still distinguishes outages from idle peers.
        logFailure(`local file transfer failed (${category})`);
        if (output.headersSent) output.destroy();
        else unavailable(output);
      };

      active.add(stop);
      input.once("aborted", stop);
      input.on("error", stop);
      output.on("error", stop);
      output.once("close", release);
      output.once("finish", release);
      upstream.on("error", () => fail("upstream"));
      upstream.setTimeout(IDLE_TIMEOUT_MS, () => fail("timeout"));
      output.setTimeout(IDLE_TIMEOUT_MS, () => fail("timeout"));
      upstream.on("response", (response) => {
        if (ended) {
          response.destroy();
          return;
        }
        upstreamResponse = response;
        response.on("error", () => fail("upstream"));
        response.once("aborted", () => fail("upstream"));
        const headers = endToEnd(response.headers);
        // An early refusal must not leave an unfinished upload on a reusable
        // client socket after its upstream has been released.
        if (!input.complete) headers.connection = "close";
        output.writeHead(response.statusCode ?? 502, headers);
        response.pipe(output);
      });
      input.pipe(upstream);
    },
    close() {
      closed = true;
      for (const stop of active) stop();
    },
  };
}
