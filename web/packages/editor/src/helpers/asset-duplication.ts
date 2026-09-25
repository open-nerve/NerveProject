/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { v4 as uuidv4 } from "uuid";
import { ECustomImageAttributeNames, ECustomImageStatus } from "@/extensions/custom-image/types";

// marks one element of the pasted HTML (parsed by processAssetDuplication) in place
type AssetDuplicationHandler = (element: Element) => void;

// An image of an uploaded asset (its src is the asset's id, not a web address) becomes a new image that
// duplicates the asset.
const imageComponentHandler: AssetDuplicationHandler = (element) => {
  const src = element.getAttribute("src");
  if (!src || src.startsWith("http")) return;
  element.setAttribute(ECustomImageAttributeNames.STATUS, ECustomImageStatus.DUPLICATING);
  element.setAttribute(ECustomImageAttributeNames.ID, uuidv4());
};

export const assetDuplicationHandlers: Record<string, AssetDuplicationHandler> = {
  "image-component": imageComponentHandler,
};
