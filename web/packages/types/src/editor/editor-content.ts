/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

/**
 * Editor JSON content type - locally defined to avoid external dependencies
 */

export type JSONContent = {
  attrs?: Record<string, unknown>;
  marks?: {
    type: string;
    attrs?: Record<string, unknown>;
    [key: string]: unknown;
  }[];
  [key: string]: unknown;
};
