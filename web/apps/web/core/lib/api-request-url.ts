/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

/**
 * Ensures an API request URL has a trailing slash on its path component, preserving the query string and hash.
 * The domains that still call their old addresses through APIService (M6–M8) were written for slash-terminated
 * routes; the interceptor adds the slash so callers need not remember it.
 *
 * Examples:
 *   "/api/foo"             -> "/api/foo/"
 *   "/api/foo/"            -> "/api/foo/"
 *   "/api/foo?x=1"         -> "/api/foo/?x=1"
 *   "/api/foo#frag"        -> "/api/foo/#frag"
 *   "https://h/api/foo"    -> "https://h/api/foo/"
 *   "" | "/"               -> unchanged
 */
export function ensureAPITrailingSlash(url: string): string {
  if (!url) return url;
  const boundary = url.search(/[?#]/);
  const path = boundary === -1 ? url : url.slice(0, boundary);
  const suffix = boundary === -1 ? "" : url.slice(boundary);
  if (!path || path.endsWith("/")) return url;
  return `${path}/${suffix}`;
}

/**
 * Same as `ensureAPITrailingSlash`, but leaves absolute URLs unchanged: API requests go to relative paths on this
 * origin, so an absolute URL is a signed object-storage address, and adding a slash would invalidate its signature.
 */
export function normalizeAPIRequestURL(url: string): string {
  if (/^[a-z][a-z\d+\-.]*:/i.test(url)) return url; // an absolute URL starts with its scheme
  return ensureAPITrailingSlash(url);
}
