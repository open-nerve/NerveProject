import { createDatabase, startPostgres, templateDatabase } from "./fixtures/db";
import { shareS3, startS3 } from "./fixtures/s3";
import { runNerve } from "./fixtures/server";

/**
 * Starts PostgreSQL and the S3 service once per run, before any browser, and
 * migrates the template database with bin/nerve migrate up; every worker
 * copies the template (fixtures/test.ts) and every S3 world takes its own
 * bucket from the shared service (fixtures/s3-world.ts). The returned function
 * is the global teardown.
 */
export default async function globalSetup(): Promise<() => Promise<void>> {
  const [postgres, s3] = await Promise.allSettled([startPostgres(), startS3()]);
  const releases = [
    ...(s3.status === "fulfilled" ? [() => s3.value.close()] : []),
    ...(postgres.status === "fulfilled" ? [() => postgres.value.stop()] : []),
  ];
  const stop = async () => {
    const results = await Promise.allSettled(releases.map((release) => release()));
    const errors = results.flatMap((result) => (result.status === "rejected" ? [result.reason] : []));
    if (errors.length) throw new AggregateError(errors, "global teardown failed");
  };
  try {
    if (postgres.status === "rejected") throw postgres.reason;
    if (s3.status === "rejected") throw s3.reason;
    shareS3(s3.value);
    await runNerve(["migrate", "up"], await createDatabase(templateDatabase));
  } catch (error) {
    // A teardown that also fails must not hide why setup failed.
    const cleanup = await stop().then(
      () => undefined,
      (reason: unknown) => ({ reason })
    );
    if (cleanup) {
      // oxlint-disable-next-line preserve-caught-error -- the caught error is the first of the aggregated errors.
      throw new AggregateError([error, cleanup.reason], "global setup and its teardown failed");
    }
    throw error;
  }
  return stop;
}
