import { execFile } from "node:child_process";
import { writeFile } from "node:fs/promises";
import { promisify } from "node:util";

import { PostgreSqlContainer } from "@testcontainers/postgresql";
import { Client, Pool, escapeIdentifier, type QueryResultRow } from "pg";

/** The PostgreSQL image, the same as the development database (deploy/compose.dev.yaml). */
const image = "postgres:18.6";

/** The database global setup migrates once; every worker gets a copy of it. */
export const templateDatabase = "nerve_template";

/** Global setup hands the server's URL and its container to the workers in these variables. */
const serverUrlVariable = "NERVE_E2E_POSTGRES_URL";
const containerVariable = "NERVE_E2E_POSTGRES_CONTAINER";

/** A database of this run: nerve serves from it, stories assert on it. */
export interface Database {
  readonly name: string;
  readonly url: string;
  /** Runs one statement on this database, through its own pool, and returns the rows. */
  query<Row extends QueryResultRow>(sql: string, params?: unknown[]): Promise<Row[]>;
  /** Writes a plain SQL pg_dump of this database to file, for the artifacts of a failed test. */
  dump(file: string): Promise<void>;
  /** Closes the pool. */
  close(): Promise<void>;
}

/**
 * Starts the PostgreSQL server of this run and publishes its URL and
 * container to the workers. The testcontainers reaper (Ryuk) removes the
 * container if the run dies before stop is called.
 */
export async function startPostgres(): Promise<{ stop: () => Promise<void> }> {
  const container = await new PostgreSqlContainer(image).withDatabase("postgres").start();
  process.env[serverUrlVariable] = databaseUrl(container.getConnectionUri(), "postgres");
  process.env[containerVariable] = container.getId();
  return {
    stop: async () => {
      await container.stop();
    },
  };
}

// pg waits forever by default; a server that accepts connections but never
// answers must fail the query instead of hanging the run. pg_dump runs in
// the container and is killed after dumpTimeoutMs.
const connectTimeoutMs = 10_000;
const queryTimeoutMs = 30_000;
const dumpTimeoutMs = 60_000;

/**
 * Creates database name, as a copy of template when given, on the server
 * startPostgres started, and returns its URL.
 */
export async function createDatabase(name: string, template?: string): Promise<string> {
  const serverUrl = requireVariable(serverUrlVariable);
  const copy = template ? ` TEMPLATE ${escapeIdentifier(template)}` : "";
  const admin = new Client({
    connectionString: serverUrl,
    connectionTimeoutMillis: connectTimeoutMs,
    query_timeout: queryTimeoutMs,
  });
  await admin.connect();
  try {
    await admin.query(`CREATE DATABASE ${escapeIdentifier(name)}${copy}`);
  } finally {
    await admin.end();
  }
  return databaseUrl(serverUrl, name);
}

/** Opens a pool on database name of the server startPostgres started: a worker's own database. */
export function openDatabase(name: string): Database {
  const serverUrl = requireVariable(serverUrlVariable);
  const url = databaseUrl(serverUrl, name);
  const pool = new Pool({
    connectionString: url,
    max: 2,
    connectionTimeoutMillis: connectTimeoutMs,
    query_timeout: queryTimeoutMs,
  });
  return {
    name,
    url,
    query: async <Row extends QueryResultRow>(sql: string, params?: unknown[]) =>
      (await pool.query<Row>(sql, params)).rows,
    dump: (file) => dump(serverUrl, name, file),
    close: () => pool.end(),
  };
}

async function dump(serverUrl: string, name: string, file: string): Promise<void> {
  const user = decodeURIComponent(new URL(serverUrl).username);
  const { stdout } = await promisify(execFile)(
    "docker",
    ["exec", requireVariable(containerVariable), "pg_dump", "--username", user, "--no-owner", "--dbname", name],
    { timeout: dumpTimeoutMs, killSignal: "SIGKILL", maxBuffer: 256 * 1024 * 1024 }
  );
  await writeFile(file, stdout);
}

function requireVariable(name: string): string {
  const value = process.env[name];
  if (!value) {
    throw new Error(`${name} is not set: run the stories with playwright test (see global-setup.ts)`);
  }
  return value;
}

function databaseUrl(serverUrl: string, name: string): string {
  const url = new URL(serverUrl);
  url.pathname = `/${name}`;
  url.searchParams.set("sslmode", "disable");
  return url.toString();
}
