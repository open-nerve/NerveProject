/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { isImageID } from "@/extensions/custom-image/input";
/** The transaction plugin, not clipboard markers, creates target operations. */
export const assetDuplicationHandlers: Record<string, (element: Element) => void> = {
  "image-component": (element) => {
    if (!isImageID(element.getAttribute("src"))) element.remove();
  },
  img: (element) => element.remove(),
};
