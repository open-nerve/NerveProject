/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { readFileSync } from "node:fs";
import { createRequire } from "node:module";
import type { Plugin } from "vite";

const require = createRequire(import.meta.url);

/** The emoji picker's data: its emoji, and the names of their groups, in the one locale it reads (frimousse's en). */
const FILES = ["en/data.json", "en/messages.json"];

/** The version of emojibase-data the web app has: its package.json pins it (M3 design 7.7). */
function installedVersion(): string {
  const manifest: unknown = JSON.parse(readFileSync(require.resolve("emojibase-data/package.json"), "utf8"));
  if (
    typeof manifest !== "object" ||
    manifest === null ||
    !("version" in manifest) ||
    typeof manifest.version !== "string"
  ) {
    throw new Error("emojibase-data's package.json names no version");
  }
  return manifest.version;
}

/**
 * Where nerve serves the emoji picker's data: among the build's assets, at a path that names the version, which the
 * picker reads (propel's EMOJIBASE_URL, frimousse's emojibaseUrl) instead of the CDN of frimousse's default, which the
 * page's Content-Security-Policy blocks (connect-src 'self'; M2-closeout §10).
 */
export const emojibaseUrl = `/assets/emojibase/${installedVersion()}`;

/** The picker's data files, each as the build writes it: its name under the client's build, and what it holds. */
export function emojibaseFiles(): { fileName: string; source: string }[] {
  return FILES.map((file) => ({
    fileName: `${emojibaseUrl.slice(1)}/${file}`,
    source: readFileSync(require.resolve(`emojibase-data/${file}`), "utf8"),
  }));
}

/**
 * Serves the emoji picker's data from nerve: the client's build writes emojibaseFiles among its assets, which nerve
 * embeds and serves as it does every asset (webui: cached for good, a new version having a new path); the development
 * server answers the same paths.
 */
export function emojibase(): Plugin {
  return {
    name: "nerve:emojibase",
    applyToEnvironment: (environment) => environment.name === "client",
    generateBundle() {
      for (const { fileName, source } of emojibaseFiles()) {
        this.emitFile({ type: "asset", fileName, source });
      }
    },
    configureServer(server) {
      const files = new Map(emojibaseFiles().map(({ fileName, source }) => [`/${fileName}`, source]));
      server.middlewares.use((request, response, next) => {
        const source = request.url === undefined ? undefined : files.get(request.url);
        if (source === undefined) {
          next();
          return;
        }
        response.setHeader("Content-Type", "application/json");
        response.end(source);
      });
    },
  };
}
