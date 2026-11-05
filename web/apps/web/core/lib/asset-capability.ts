/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */

/** Validate only: re-serializing a signed URL can change the signed bytes. */
export function capabilityFailure(url: string, expiresAt: string): "invalid-grant" | "expired" | undefined {
  if ([...url].some((char) => char.charCodeAt(0) < 32 || char.charCodeAt(0) === 127) || /[ \\#]/.test(url))
    return "invalid-grant";
  const relative = url[0] === "/" && url[1] !== "/";
  const authority = /^https?:\/\/([^/?#]+)/i.exec(url)?.[1];
  if (!relative && (!authority || authority.includes("@"))) return "invalid-grant";
  try {
    const parsed = new URL(url, "https://local.invalid");
    if (!["http:", "https:"].includes(parsed.protocol) || parsed.username || parsed.password) return "invalid-grant";
  } catch {
    return "invalid-grant";
  }
  const expiry = Date.parse(expiresAt);
  if (!Number.isFinite(expiry)) return "invalid-grant";
  if (expiry <= Date.now()) return "expired";
}
