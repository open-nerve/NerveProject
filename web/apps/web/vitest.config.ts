/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { defineConfig } from "vitest/config";

// The tests read the route table and the source, in Node. Without this file vitest would load vite.config.ts,
// the app's build configuration, and run its React Router plugin, which the tests do not use.
export default defineConfig({ test: { environment: "node" } });
