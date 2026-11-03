import { createHash } from "node:crypto";

import { expect } from "@playwright/test";

import type { Database } from "./db";
import { filePixel } from "./storage";

/** Read the returned capability without a business bearer and compare actual bytes and SHA. */
export async function expectImageBytes(base: string, address: string | null, expiresAt: string | null) {
  expect(address).toBeTruthy();
  expect(expiresAt).toBeTruthy();
  if (!address || !expiresAt) throw new Error("image capability is incomplete");
  const target = new URL(address, base);
  const expiry = Date.parse(expiresAt);
  expect(expiry - Date.now()).toBeGreaterThan(290_000);
  expect(expiry - Date.now()).toBeLessThanOrEqual(65 * 60_000);
  const payload = target.searchParams.get("payload");
  if (payload) {
    const claim: unknown = JSON.parse(Buffer.from(payload, "base64url").toString());
    expect(claim).toMatchObject({ inline: true, expires_at: expiry / 1000 });
  }
  const response = await fetch(target, { credentials: "omit", signal: AbortSignal.timeout(15_000) });
  expect(response.status).toBe(200);
  expect(response.headers.get("content-type")).toBe("image/png");
  expect(response.headers.get("content-disposition")).toContain("inline;");
  const bytes = Buffer.from(await response.arrayBuffer());
  expect(bytes).toEqual(filePixel);
  expect(createHash("sha256").update(bytes).digest("hex")).toBe(imageSHA);
}

export const imageSHA = createHash("sha256").update(filePixel).digest("hex");

/** Complete row snapshots catch audit or metadata changes hidden by a response-only assertion. */
export function imageRows(db: Database, ids: string[]) {
  return db.query<{ row: unknown }>(
    "SELECT to_jsonb(a) AS row FROM file_assets a WHERE id=ANY($1::uuid[]) ORDER BY id",
    [ids]
  );
}
