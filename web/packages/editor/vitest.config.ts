/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { fileURLToPath } from "node:url";
import { defineConfig } from "vitest/config";

export default defineConfig({
  resolve: {
    alias: {
      "@": fileURLToPath(new URL("./src", import.meta.url)),
    },
    // prosemirror-codemark ships no exports map, only "main" (CommonJS) and "module" (ESM), and vitest
    // resolves "main" by default. That build's require("prosemirror-state") loads dist/index.cjs beside
    // the ESM dist/index.js that @tiptap/pm/state imports: two module instances, and the first Editor
    // throws "RangeError: Adding different instances of a keyed plugin". The production build already
    // prefers "module"; this makes the tests resolve the same way.
    mainFields: ["module", "main"],
  },
  test: {
    environment: "jsdom",
    setupFiles: ["./vitest.setup.ts"],
    server: {
      deps: {
        // is-emoji-supported ships ESM under dist/esm/ without "type": "module", so vitest runs it through
        // Vite, which loads its source map and warns that the sources it names were not published. Node
        // loads the file itself. (prosemirror-codemark has the same maps, but its ESM build imports
        // "./plugin" without an extension, which only Vite resolves, so it stays inlined; its patch in
        // patches/ drops the comments that point to the maps.)
        external: [/\/is-emoji-supported\//],
      },
    },
  },
});
