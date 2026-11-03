import { createHash } from "node:crypto";
import { readFile } from "node:fs/promises";
import path from "node:path";

import { expect } from "@playwright/test";

import type { Database } from "../db";

type StoredRow = { id: string; [column: string]: unknown };

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
