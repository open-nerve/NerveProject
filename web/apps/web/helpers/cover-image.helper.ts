/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import DefaultCoverImage from "@/app/assets/cover-images/image_1.webp?url";

/**
 * The cover shown where a project or a user has none of its own: nerve gives a project's and a user's
 * cover_image_url as null until uploads arrive (M5; M3 design 3.2).
 */
export const DEFAULT_COVER_IMAGE_URL = DefaultCoverImage;

/** The cover's own URL, or the fallback when it has none. */
export function getCoverImageDisplayURL(imageUrl: string | null | undefined, fallbackUrl: string): string {
  return imageUrl || fallbackUrl;
}
