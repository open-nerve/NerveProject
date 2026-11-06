import { createHash } from "node:crypto";
import { mkdtemp, rm, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import path from "node:path";

import { test as base, type TestInfo } from "@playwright/test";

import { createApi, type Api } from "./api";
import { createDatabase, openDatabase, templateDatabase, type Database } from "./db";
import { startNerve, type Nerve } from "./server";
import { sharedS3, type S3Bucket, type S3Storage } from "./s3";

export interface S3World {
  readonly api: Api;
  readonly db: Database;
  readonly nerve: Nerve;
  readonly storage: S3Storage;
  readonly bucket: S3Bucket;
  readonly secretsDirectory: string;
  close(): Promise<void>;
}

/**
 * Every world owns its database, private bucket/CORS, secrets and process. The S3 service is the run's
 * (global-setup.ts): closing the world removes its buckets, never the service.
 */
export async function createS3World(info: TestInfo, env: Record<string, string> = {}): Promise<S3World> {
  const hash = createHash("sha256")
    .update(`${info.testId}:${info.workerIndex}:${info.repeatEachIndex}:${info.retry}`)
    .digest("hex")
    .slice(0, 16);
  let storage: S3Storage | undefined;
  let db: Database | undefined;
  let nerve: Nerve | undefined;
  let directory: string | undefined;
  let closing: Promise<void> | undefined;
  const close = () =>
    (closing ??= (async () => {
      const errors: unknown[] = [];
      // HTTP/jobs stop before its buckets are removed, then assertion connections and secrets.
      for (const release of [
        () => nerve?.stop(),
        () => storage?.close(),
        () => db?.close(),
        () => directory && rm(directory, { recursive: true, force: true }),
      ]) {
        try {
          // eslint-disable-next-line no-await-in-loop -- Close in HTTP, storage, pool, secret ownership order.
          await release();
        } catch (error) {
          errors.push(error);
        }
      }
      if (errors.length) throw new AggregateError(errors, "S3 world cleanup failed");
    })());
  try {
    storage = sharedS3();
    const bucket = await storage.createBucket();
    const name = `e2e_s3_${hash}`;
    await createDatabase(name, templateDatabase);
    db = openDatabase(name);
    directory = await mkdtemp(path.join(tmpdir(), "nerve-s3-e2e-"));
    // Keep only ownership identifiers, never credentials, so failed-startup
    // checks can inspect this world's resources while other worlds run.
    await writeFile(
      info.outputPath("s3-resources.json"),
      JSON.stringify({ containerName: storage.name, bucket: bucket.connection.bucket, secretsDirectory: directory }) +
        "\n"
    );
    const accessFile = path.join(directory, "access"),
      secretFile = path.join(directory, "secret");
    await writeFile(accessFile, bucket.connection.access_key_id, { mode: 0o600 });
    await writeFile(secretFile, bucket.connection.secret_access_key, { mode: 0o600 });
    nerve = await startNerve(db.url, info.outputPath("s3-nerve.log"), {
      NERVE_FILES__DRIVER: "s3",
      NERVE_FILES__S3__ENDPOINT: bucket.connection.endpoint,
      NERVE_FILES__S3__REGION: bucket.connection.region,
      NERVE_FILES__S3__BUCKET: bucket.connection.bucket,
      NERVE_FILES__S3__PREFIX: bucket.connection.prefix,
      NERVE_FILES__S3__PATH_STYLE: "true",
      NERVE_FILES__S3__ACCESS_KEY_ID_FILE: accessFile,
      NERVE_FILES__S3__SECRET_ACCESS_KEY_FILE: secretFile,
      ...env,
    });
    await bucket.cors(nerve.baseURL);
    return { api: createApi(nerve.baseURL), db, nerve, storage, bucket, secretsDirectory: directory, close };
  } catch (error) {
    const errors = [error];
    try {
      await close();
    } catch (cleanup) {
      errors.push(cleanup);
    }
    if (errors.length > 1) {
      // eslint-disable-next-line preserve-caught-error -- AggregateError keeps both errors and the original cause in its third argument.
      throw new AggregateError(errors, "S3 world startup and cleanup failed", { cause: error });
    }
    throw error;
  }
}

export const s3Test = base.extend<{ s3World: S3World }>({
  s3World: [
    // oxlint-disable-next-line no-empty-pattern -- Playwright reads fixture dependencies from the pattern
    async ({}, use, info) => {
      const world = await createS3World(info);
      try {
        await use(world);
      } finally {
        try {
          if (info.status !== info.expectedStatus) {
            const dump = info.outputPath("s3-database.sql");
            await world.db.dump(dump);
            await info.attach("s3-database", { path: dump, contentType: "application/sql" });
            await info.attach("s3-nerve", { path: info.outputPath("s3-nerve.log"), contentType: "text/plain" });
          }
        } finally {
          await world.close();
        }
      }
    },
    { timeout: 200_000 },
  ],
});
