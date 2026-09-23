/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
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
  },
});
