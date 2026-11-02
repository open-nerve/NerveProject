/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */

import { lstatSync } from "node:fs";
import { join } from "node:path";
import type { Plugin } from "vite";
import { createTransferProxy } from "./transfer-proxy";

// This only assigns a request to Go, without changing it. Go owns rejection and
// signature verification. Decode percent octets for the ASCII root even if a
// later octet is malformed UTF-8; such a URL must not fall through to the SPA.
function touchesFileTree(url: string): boolean {
  const raw = url.split("?", 1)[0];
  if (!raw.startsWith("/")) return false;
  const path = raw.replace(/%([\da-f]{2})/gi, (_, hex: string) => String.fromCharCode(Number.parseInt(hex, 16)));
  const stack: string[] = [];
  for (const segment of path.split("/")) {
    if (segment === "" || segment === ".") continue;
    if (segment === "..") stack.pop();
    else {
      stack.push(segment);
      if (stack[0] === "static") return true;
    }
  }
  return false;
}

function assertNoStatic(directory: string) {
  if (lstatSync(join(directory, "static"), { throwIfNoEntry: false })) {
    throw new Error("/static belongs to local file transfer, not public/build assets");
  }
}

function checkBundle(bundle: Record<string, { fileName: string }>) {
  if (Object.values(bundle).some(({ fileName }) => fileName === "static" || fileName.startsWith("static/"))) {
    throw new Error("/static belongs to local file transfer, not bundle output");
  }
}

export function localTransfer(target: string): Plugin {
  let publicDir: string | false = false;
  let transfer: ReturnType<typeof createTransferProxy> | undefined;
  return {
    name: "nerve:local-transfer",
    applyToEnvironment: (environment) => environment.name === "client",
    configResolved(config) {
      publicDir = config.publicDir;
    },
    buildStart() {
      if (publicDir) assertNoStatic(publicDir);
    },
    generateBundle: {
      order: "post",
      handler(_options, bundle) {
        checkBundle(bundle);
      },
    },
    writeBundle: {
      order: "post",
      handler(options, bundle) {
        // Also catches output from a later generateBundle hook and copied public
        // files. A failed build cannot publish either kind of collision.
        checkBundle(bundle);
        if (options.dir) assertNoStatic(options.dir);
      },
    },
    configureServer(server) {
      if (publicDir) assertNoStatic(publicDir);
      const proxy = createTransferProxy(target, (message) => server.config.logger.error(message));
      transfer = proxy;
      server.middlewares.use((request, response, next) => {
        if (touchesFileTree(request.url ?? "")) proxy.forward(request, response);
        else next();
      });
    },
    closeBundle() {
      transfer?.close();
    },
  };
}
