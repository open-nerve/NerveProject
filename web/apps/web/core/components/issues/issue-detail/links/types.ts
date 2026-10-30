/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { IssueLinkCreate, IssueLinkPatch } from "@nerve/api-client";

export type TLinkOperations = {
  create: (data: IssueLinkCreate) => Promise<void>;
  update: (linkId: string, data: IssueLinkPatch) => Promise<void>;
  remove: (linkId: string) => Promise<void>;
};
