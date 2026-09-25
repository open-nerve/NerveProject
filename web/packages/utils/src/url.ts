/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

/**
 * Validates that a next_path parameter is safe for redirection.
 * Only allows relative paths starting with "/" to prevent open redirect vulnerabilities.
 *
 * @param url - The next_path URL to validate
 * @returns True if the URL is a safe relative path, false otherwise
 *
 * @example
 * isValidNextPath("/dashboard") // true
 * isValidNextPath("/workspace/123") // true
 * isValidNextPath("https://malicious.com") // false
 * isValidNextPath("//malicious.com") // false (protocol-relative)
 * isValidNextPath("javascript:alert(1)") // false
 * isValidNextPath("") // false
 * isValidNextPath("dashboard") // false (must start with /)
 * isValidNextPath("\\malicious") // false (backslash)
 * isValidNextPath("  /dashboard  ") // true (trimmed)
 */
export function isValidNextPath(url: string): boolean {
  if (!url || typeof url !== "string") return false;

  // Trim leading/trailing whitespace
  const trimmedUrl = url.trim();

  if (!trimmedUrl) return false;

  // Only allow relative paths starting with /
  if (!trimmedUrl.startsWith("/")) return false;

  // Block protocol-relative URLs (//example.com) - open redirect vulnerability
  if (trimmedUrl.startsWith("//")) return false;

  // Block backslashes which can be used for path traversal or Windows-style paths
  if (trimmedUrl.includes("\\")) return false;

  try {
    // Use URL constructor with a dummy base to normalize and validate the path
    const normalizedUrl = new URL(trimmedUrl, "http://localhost");

    // Ensure the path is still relative (no host change from our dummy base)
    if (normalizedUrl.hostname !== "localhost" || normalizedUrl.protocol !== "http:") {
      return false;
    }

    // Use the normalized pathname for additional security checks
    const pathname = normalizedUrl.pathname;

    // Additional security checks for malicious patterns in the normalized path
    const maliciousPatterns = [
      /javascript:/i,
      /data:/i,
      /vbscript:/i,
      /<script/i,
      /on\w+=/i, // Event handlers like onclick=, onload=
    ];

    return !maliciousPatterns.some((pattern) => pattern.test(pathname));
  } catch (_error) {
    // If URL constructor fails, it's an invalid path
    return false;
  }
}
