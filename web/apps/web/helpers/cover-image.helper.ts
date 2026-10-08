/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { getFileURL } from "@nerve/utils";

import DefaultCoverImage from "@/app/assets/cover-images/image_1.webp?url";

/**
 * The cover shown where a project has none of its own: nerve gives cover_image_url as null until uploads arrive
 * (M5; M3 design 3.2).
 */
export const DEFAULT_COVER_IMAGE_URL = DefaultCoverImage;

/**
 * Gets the correct display URL for a cover image
 * - The default cover: returned as-is (served from assets folder)
 * - Uploaded assets: processed through getFileURL (adds backend URL)
 */
export function getCoverImageDisplayURL(imageUrl: string | null | undefined, fallbackUrl: string): string {
  if (!imageUrl) {
    return fallbackUrl;
  }

  if (imageUrl === DEFAULT_COVER_IMAGE_URL) {
    return imageUrl;
  }

  return getFileURL(imageUrl) || imageUrl;
}
