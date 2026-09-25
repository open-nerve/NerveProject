import { createDatabase, startPostgres, templateDatabase } from "./fixtures/db";
import { runNerve } from "./fixtures/server";

/**
 * Starts PostgreSQL once per run and migrates the template database with
 * bin/nerve migrate up; every worker copies the template (fixtures/test.ts).
 * The returned function is the global teardown.
 */
export default async function globalSetup(): Promise<() => Promise<void>> {
  const postgres = await startPostgres();
  await runNerve(["migrate", "up"], await createDatabase(templateDatabase));
  return postgres.stop;
}
