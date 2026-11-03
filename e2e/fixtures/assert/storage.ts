import { createHash } from "node:crypto";
import { readFile } from "node:fs/promises";
import path from "node:path";

import { expect } from "@playwright/test";

import type { Database } from "../db";

type StoredRow = { id: string; [column: string]: unknown };

/** Read-only snapshots retain full precision and all recovery/audit columns. */
export async function assetRows(db: Database, ids: string[]): Promise<StoredRow[]> {
  const rows = await db.query<{ row: StoredRow }>(
    "SELECT to_jsonb(a) AS row FROM file_assets a WHERE id=ANY($1::uuid[]) ORDER BY id",
    [ids]
  );
  expect(rows).toHaveLength(ids.length);
  return rows.map(({ row }) => row);
}

export function imageHTML(...ids: string[]): string {
  return `<p>Body</p>${ids.map((id) => `<image-component src="${id}"></image-component>`).join("")}`;
}

/** A live bound asset keeps its final and audit; only its elapsed staging lease is reclaimed. */
export async function expectRetainedBoundAssets(db: Database, before: StoredRow[], root: string) {
  expect(before.length).toBeGreaterThan(0);
  await Promise.all(
    before.map(async (previous) => {
      expect(previous).toMatchObject({
        bound_at: expect.any(String),
        is_uploaded: true,
        is_deleted: false,
        deleted_at: null,
      });
      const [row] = await assetRows(db, [previous.id]);
      const metadata = previous.storage_metadata as Record<string, unknown>;
      expect(row).toEqual({
        ...previous,
        storage_revision: Number(previous.storage_revision) + (metadata.staging ? 2 : 0),
        storage_metadata: { ...metadata, staging: null },
      });
      const final = metadata.final as { key: string; sha256: string };
      const bytes = await readFile(path.join(root, "objects", final.key, "data"));
      expect(createHash("sha256").update(bytes).digest("hex")).toBe(final.sha256);
    })
  );
}

/** An old unbound preparation expires even while its host is live; its final keeps the new 60d tombstone. */
export async function expectExpiredPreparations(db: Database, before: StoredRow[], root: string, startedAt: Date) {
  expect(before.length).toBeGreaterThan(0);
  const cutoffs = await db.query<{ cutoff: string }>(
    `SELECT to_jsonb((metadata->'output'->'cursor'->>'cutoff')::timestamptz) AS cutoff FROM river_job
      WHERE kind='asset.cleanup' AND state='completed' AND attempted_at>$1::timestamptz`,
    [startedAt.toISOString()]
  );
  expect(cutoffs.length).toBeGreaterThan(0);
  await Promise.all(
    before.map(async (previous) => {
      expect(previous).toMatchObject({ bound_at: null, is_uploaded: true, is_deleted: false, deleted_at: null });
      const [{ row } = { row: undefined }] = await db.query<{ row: StoredRow }>(
        "SELECT to_jsonb(a) AS row FROM file_assets a WHERE id=$1",
        [previous.id]
      );
      expect(row).toBeDefined();
      if (!row) throw new Error(`expired preparation disappeared: ${previous.id}`);
      expect(cutoffs.map(({ cutoff }) => cutoff)).toContain(row.deleted_at);
      const metadata = previous.storage_metadata as Record<string, unknown>;
      expect(metadata).toMatchObject({ candidates: [], origin: "upload", schema_version: 1 });
      expect(row).toEqual({
        ...previous,
        is_deleted: true,
        deleted_at: row.deleted_at,
        updated_at: row.deleted_at,
        deleted_by_id: null,
        updated_by_id: null,
        storage_revision: Number(previous.storage_revision) + 2,
        storage_metadata: { ...metadata, staging: null },
      });
      const final = metadata.final as { key: string; sha256: string };
      const bytes = await readFile(path.join(root, "objects", final.key, "data"));
      expect(createHash("sha256").update(bytes).digest("hex")).toBe(final.sha256);
    })
  );
}
