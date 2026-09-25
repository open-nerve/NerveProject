/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { assetDuplicationHandlers } from "@/helpers/asset-duplication";

// Marks the assets in HTML that a Nerve editor copied (text/nerve-editor-html) for duplication. Any page can write
// that clipboard type, so the HTML is parsed in an inert document: nothing in it loads or runs. The editor parses
// the result again, against its schema, when it pastes it.
export const processAssetDuplication = (htmlContent: string): { processedHtml: string } => {
  const { body } = new DOMParser().parseFromString(htmlContent, "text/html");
  for (const [selector, handler] of Object.entries(assetDuplicationHandlers)) {
    body.querySelectorAll(selector).forEach(handler);
  }
  return { processedHtml: body.innerHTML };
};
