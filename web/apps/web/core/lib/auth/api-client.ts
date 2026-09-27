/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { createClient } from "@nerve/api-client";

// The web app's clients of nerve's API (M2 design 7.1). They send to the page's own origin, where nerve
// serves both the app and the API.

/** The client of the operations that need no token: the instance, sign-in and sign-up. */
export const publicClient = createClient();
