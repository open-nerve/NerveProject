import { createDatabase, startPostgres, templateDatabase } from "./fixtures/db";
import { runNerve } from "./fixtures/server";

/**
 * Starts PostgreSQL once per run and migrates the template database with
 * bin/nerve migrate up; every worker copies the template (fixtures/test.ts).
 * The returned function is the global teardown.
 */
export default async function globalSetup(): Promise<() => Promise<void>> {
  const postgres = await startPostgres();
  const template = await createDatabase(templateDatabase);
  await runNerve(["migrate", "up"], template.url);
  return postgres.stop;
}
