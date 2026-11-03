import { expect } from "@playwright/test";

import type { Database } from "../db";

interface ImageReference {
  oid: string;
  schema: string;
  child: string;
  name: string;
  column: string;
  parentSchema: string;
  parent: string;
  target: string;
  columns: number;
  targetColumns: number;
  action: string;
  deferrable: boolean;
  deferred: boolean;
  indexed: boolean;
}

/** Image owners reference an asset; they are not children owned by it. Prove the complete contract before excluding it. */
export async function imageReferenceOIDs(db: Database): Promise<Set<string>> {
  const references = await db.query<ImageReference>(
    `SELECT c.oid::text AS oid, ns.nspname::text AS schema, child.relname::text AS child,
            c.conname::text AS name, a.attname::text AS column, pns.nspname::text AS "parentSchema",
            parent.relname::text AS parent, b.attname::text AS target,
            cardinality(c.conkey) AS columns, cardinality(c.confkey) AS "targetColumns",
            c.confdeltype::text AS action, c.condeferrable AS deferrable, c.condeferred AS deferred,
            EXISTS (SELECT 1 FROM pg_index i WHERE i.indrelid = c.conrelid AND i.indisvalid
              AND i.indpred IS NULL AND i.indexprs IS NULL AND i.indkey[0] = c.conkey[1]) AS indexed
       FROM pg_constraint c
       JOIN pg_class child ON child.oid = c.conrelid
       JOIN pg_namespace ns ON ns.oid = child.relnamespace
       JOIN pg_class parent ON parent.oid = c.confrelid
       JOIN pg_namespace pns ON pns.oid = parent.relnamespace
       JOIN pg_attribute a ON a.attrelid = c.conrelid AND a.attnum = c.conkey[1]
       JOIN pg_attribute b ON b.attrelid = c.confrelid AND b.attnum = c.confkey[1]
      WHERE c.contype = 'f' AND c.confrelid = 'public.file_assets'::regclass
      ORDER BY ns.nspname COLLATE "C", child.relname COLLATE "C", c.conname COLLATE "C"`
  );
  const expected = [
    { child: "projects", name: "projects_cover_image_asset_fk", column: "cover_image_asset_id" },
    { child: "users", name: "users_avatar_asset_fk", column: "avatar_asset_id" },
    { child: "users", name: "users_cover_image_asset_fk", column: "cover_image_asset_id" },
    { child: "workspaces", name: "workspaces_logo_asset_fk", column: "logo_asset_id" },
  ].map(({ child, name, column }) => ({
    child,
    name,
    column,
    schema: "public",
    parentSchema: "public",
    parent: "file_assets",
    target: "id",
    columns: 1,
    targetColumns: 1,
    action: "a",
    deferrable: false,
    deferred: false,
    indexed: true,
  }));
  expect(
    references.map(({ oid: _oid, ...reference }) => reference),
    "all incoming image references must be the four exact, immediate NO ACTION references with unconditional indexes"
  ).toEqual(expected);
  return new Set(references.map(({ oid }) => oid));
}
