import { expect } from "@playwright/test";
import { Client } from "pg";

import type { Database } from "./db";

/** Keep PostgreSQL's microseconds: never round the watermark through Date. */
export async function archiveWatermark(db: Database): Promise<string> {
  const [row] = await db.query<{ watermark: string }>("SELECT clock_timestamp()::text AS watermark");
  if (!row) throw new Error("database returned no clock watermark");
  return row.watermark;
}

/** A completed attempt must have STARTED after every recent-control write. */
export function completedPeriodicJobs(db: Database, kind: string, watermark: string, startedAt: Date, id?: string) {
  return db.query<{ id: string; attempted_at: string; finalized_at: string }>(
    `SELECT id::text, attempted_at::text, finalized_at::text FROM river_job
     WHERE kind = $4 AND state = 'completed'
       AND attempted_at > $1::timestamptz
       AND finalized_at > $2::timestamptz
       AND ($3::bigint IS NULL OR id = $3::bigint)
     ORDER BY id`,
    [watermark, startedAt.toISOString(), id ?? null, kind]
  );
}

/** A pinned, read-only row lock: synchronization only, never seed business rows. */
export async function blockArchiveProject(db: Database, projectId: string) {
  const client = new Client({ connectionString: db.url, connectionTimeoutMillis: 10_000, query_timeout: 10_000 });
  await client.connect();
  try {
    await client.query("BEGIN");
    await client.query("SELECT id FROM projects WHERE id = $1 FOR SHARE", [projectId]);
    const { rows: backendRows } = await client.query<{ pid: number }>("SELECT pg_backend_pid() AS pid");
    const pid = backendRows[0]?.pid;
    if (!pid) throw new Error("blocker has no backend pid");
    return {
      async waitForAttempt() {
        let id: string | undefined;
        await expect
          .poll(
            async () => {
              const rows = await db.query<{ id: string }>(
                `SELECT j.id::text FROM river_job j WHERE j.kind = 'issue.auto_archive' AND j.state = 'running'
             AND EXISTS (SELECT 1 FROM pg_stat_activity a
                         WHERE a.datname = current_database() AND $1::int = ANY(pg_blocking_pids(a.pid)))`,
                [pid]
              );
              id = rows[0]?.id;
              return rows.length;
            },
            { timeout: 10_000, intervals: [20, 50, 100] }
          )
          .toBe(1);
        if (!id) throw new Error("no blocked auto-archive attempt");
        return id;
      },
      async release() {
        try {
          await client.query("ROLLBACK");
        } finally {
          await client.end();
        }
      },
    };
  } catch (error) {
    await client.end();
    throw error;
  }
}
