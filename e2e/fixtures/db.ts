import { PostgreSqlContainer } from "@testcontainers/postgresql";
import { Client, escapeIdentifier, type QueryResultRow } from "pg";

/** The PostgreSQL image, the same as the development database (deploy/compose.dev.yaml). */
const image = "postgres:18.6";

/** The database global setup migrates once; every worker gets a copy of it. */
export const templateDatabase = "nerve_template";

/** Global setup hands the server's URL to the workers in this variable. */
const serverUrlVariable = "NERVE_E2E_POSTGRES_URL";

/** A database of this run: nerve serves from it, stories assert on it. */
export interface Database {
  readonly name: string;
  readonly url: string;
  /** Runs one statement on this database and returns the rows. */
  query<Row extends QueryResultRow>(sql: string, params?: unknown[]): Promise<Row[]>;
}

/**
 * Starts the PostgreSQL server of this run and publishes its URL to the
 * workers. The testcontainers reaper (Ryuk) removes the container if the run
 * dies before stop is called.
 */
export async function startPostgres(): Promise<{ stop: () => Promise<void> }> {
  const container = await new PostgreSqlContainer(image).withDatabase("postgres").start();
  process.env[serverUrlVariable] = databaseUrl(container.getConnectionUri(), "postgres");
  return {
    stop: async () => {
      await container.stop();
    },
  };
}

/** Creates database name, as a copy of template when given, on the server startPostgres started. */
export async function createDatabase(name: string, template?: string): Promise<Database> {
  const serverUrl = process.env[serverUrlVariable];
  if (!serverUrl) {
    throw new Error(`${serverUrlVariable} is not set: run the stories with playwright test (see global-setup.ts)`);
  }
  const copy = template ? ` TEMPLATE ${escapeIdentifier(template)}` : "";
  await query(serverUrl, `CREATE DATABASE ${escapeIdentifier(name)}${copy}`);
  const url = databaseUrl(serverUrl, name);
  return { name, url, query: (sql, params) => query(url, sql, params) };
}

async function query<Row extends QueryResultRow>(url: string, sql: string, params?: unknown[]): Promise<Row[]> {
  const client = new Client({ connectionString: url });
  await client.connect();
  try {
    return (await client.query<Row>(sql, params)).rows;
  } finally {
    await client.end();
  }
}

function databaseUrl(serverUrl: string, name: string): string {
  const url = new URL(serverUrl);
  url.pathname = `/${name}`;
  url.searchParams.set("sslmode", "disable");
  return url.toString();
}
