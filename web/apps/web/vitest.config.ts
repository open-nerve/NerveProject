/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { defineConfig } from "vitest/config";

// The tests read the route table and the source, in Node. Without this file vitest would load vite.config.ts,
// the app's build configuration, and run its React Router plugin, which the tests do not use. Imports that
// start with "@/" resolve through tsconfig.json's paths, as in the build.
export default defineConfig({
  resolve: {
    tsconfigPaths: true,
    // prosemirror-codemark ships no exports map, only "main" (CommonJS) and "module" (ESM). Its CommonJS build
    // require()s a second ProseMirror instance beside the ESM one @tiptap/pm imports, and a real editor throws
    // "Adding different instances of a keyed plugin". Production resolves "module"; only this package is sent
    // there, because a global mainFields moves every exports-less dependency onto an ESM build that vitest must
    // then transform, and the whole suite slows down.
    alias: [{ find: /^prosemirror-codemark$/, replacement: "prosemirror-codemark/dist/esm/index.js" }],
  },
  test: {
    environment: "node",
    setupFiles: ["./vitest.setup.ts"],
    // Tiptap's ESM default-imports CommonJS tippy.js, and only vite's module runner unwraps that default.
    server: { deps: { inline: [/@tiptap\//] } },
  },
});
