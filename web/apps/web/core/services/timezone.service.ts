/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { Timezone } from "@nerve/api-client";
import { unwrap } from "@/lib/api-error";
import { publicClient } from "@/lib/auth/api-client";

export class TimezoneService {
  /** The time zones to choose from, with their offsets now (M2 design 5.3): public, like the instance. */
  async list(): Promise<Timezone[]> {
    return unwrap(await publicClient.GET("/api/v0/timezones")).data;
  }
}
