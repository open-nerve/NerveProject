/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import DefaultCoverImage from "@/app/assets/cover-images/image_1.webp?url";

/** Static fallback for an empty saved cover or a new project with no selected cover. */
export const DEFAULT_COVER_IMAGE_URL = DefaultCoverImage;

/** The cover's own URL, or the fallback when it has none. */
export function getCoverImageDisplayURL(imageUrl: string | null | undefined, fallbackUrl: string): string {
  return imageUrl || fallbackUrl;
}
