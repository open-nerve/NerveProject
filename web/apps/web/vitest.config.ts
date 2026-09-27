/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { defineConfig } from "vitest/config";

// The tests read the route table and the source, in Node. Without this file vitest would load vite.config.ts,
// the app's build configuration, and run its React Router plugin, which the tests do not use. Imports that
// start with "@/" resolve through tsconfig.json's paths, as in the build.
export default defineConfig({ resolve: { tsconfigPaths: true }, test: { environment: "node" } });
