/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { InstanceInfo } from "@nerve/api-client";
import { unwrap } from "@/lib/api-error";
import { publicClient } from "@/lib/auth/api-client";

export class InstanceService {
  async getInstanceInfo(): Promise<InstanceInfo> {
    return unwrap(await publicClient.GET("/api/v0/instance"));
  }
}
