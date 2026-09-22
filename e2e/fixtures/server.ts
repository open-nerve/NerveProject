import { execFile, spawn, type ChildProcess } from "node:child_process";
import { once } from "node:events";
import { closeSync, mkdirSync, openSync } from "node:fs";
import { createServer, type AddressInfo } from "node:net";
import path from "node:path";
import { setTimeout as sleep } from "node:timers/promises";
import { promisify } from "node:util";

/** The binary under test: make build compiles it with the web frontend embedded. */
const binary = path.resolve(import.meta.dirname, "../../bin/nerve");

/** nerve connects with this application_name, so its sessions show in pg_stat_activity. */
export const applicationName = "nerve";

const readyTimeoutMs = 30_000;
const stopTimeoutMs = 30_000;
const commandTimeoutMs = 60_000;
const pollIntervalMs = 100;

/**
 * The worker-fixture timeout for `nerve` in test.ts: Playwright's default
 * (30 s, shared by setup and teardown) can be shorter than readyTimeoutMs
 * alone, so the fixture's own timeouts — and the "(log: …)" errors they
 * produce — could never fire first.
 */
export const nerveFixtureTimeoutMs = readyTimeoutMs + stopTimeoutMs + 10_000;

/** A nerve serve process of this run. */
export interface Nerve {
  readonly baseURL: string;
  /** Sends SIGTERM and waits for nerve to exit with code 0. */
  stop(): Promise<void>;
}

/**
 * Runs a nerve command, such as migrate up, with the test configuration on
 * the database at databaseUrl. It rejects when the command exits non-zero,
 * and kills it after commandTimeoutMs: global setup runs it before any
 * Playwright timeout applies.
 */
export async function runNerve(args: string[], databaseUrl: string): Promise<{ stdout: string; stderr: string }> {
  return promisify(execFile)(binary, args, {
    env: nerveEnv(databaseUrl),
    timeout: commandTimeoutMs,
    killSignal: "SIGKILL",
  });
}

/**
 * Starts nerve serve with the test configuration on a free local port and
 * waits until /readyz answers 200. Its output goes to logFile.
 */
export async function startNerve(databaseUrl: string, logFile: string): Promise<Nerve> {
  const addr = `127.0.0.1:${await freePort()}`;
  mkdirSync(path.dirname(logFile), { recursive: true });
  const log = openSync(logFile, "w");
  const child = spawn(binary, ["serve"], {
    env: { ...nerveEnv(databaseUrl), NERVE_SERVER__ADDR: addr },
    stdio: ["ignore", log, log],
  });
  closeSync(log); // the child has its own copy
  // A spawn failure (ENOENT/EACCES) emits "error" instead of "exit"; capture it
  // so waitUntilReady can surface it through the same log-path error below.
  let spawnError: Error | undefined;
  child.once("error", (err) => {
    spawnError = err;
  });
  const baseURL = `http://${addr}`;
  try {
    await waitUntilReady(child, `${baseURL}/readyz`, Date.now() + readyTimeoutMs, () => spawnError);
  } catch (err) {
    await kill(child);
    throw new Error(`nerve did not become ready (log: ${logFile})`, { cause: err });
  }
  return { baseURL, stop: () => stop(child, logFile) };
}

/** The test configuration on the given database; the caller's own NERVE_* variables are left out. */
function nerveEnv(databaseUrl: string): NodeJS.ProcessEnv {
  const url = new URL(databaseUrl);
  url.searchParams.set("application_name", applicationName);
  const inherited = Object.entries(process.env).filter(([name]) => !name.startsWith("NERVE_"));
  return { ...Object.fromEntries(inherited), NERVE_ENV: "test", NERVE_DATABASE__URL: url.toString() };
}

async function freePort(): Promise<number> {
  const server = createServer().listen(0, "127.0.0.1");
  await once(server, "listening");
  const { port } = server.address() as AddressInfo;
  server.close();
  await once(server, "close");
  return port;
}

/**
 * Polls /readyz until it answers 200; fails when nerve exits, fails to spawn,
 * or the deadline passes. Each request may only use the time left, so a
 * server that accepts the connection but never answers cannot hold the wait
 * past the deadline.
 */
async function waitUntilReady(
  child: ChildProcess,
  readyzUrl: string,
  deadline: number,
  spawnError: () => Error | undefined
): Promise<void> {
  const failure = spawnError();
  if (failure) {
    throw failure;
  }
  if (child.exitCode !== null) {
    throw new Error(`nerve exited with code ${child.exitCode}`);
  }
  const timeLeft = deadline - Date.now();
  if (timeLeft <= 0) {
    throw new Error(`${readyzUrl} did not answer 200 within ${readyTimeoutMs} ms`);
  }
  const ready = await fetch(readyzUrl, { signal: AbortSignal.timeout(timeLeft) }).then(
    (res) => res.ok,
    () => false // not listening yet, or no answer before the deadline
  );
  if (ready) {
    // Another worker's nerve can briefly answer on this port before this
    // child's own listener binds it; recheck after one more poll interval
    // that this child is still the process actually running.
    await sleep(pollIntervalMs);
    const lateFailure = spawnError();
    if (lateFailure) {
      throw lateFailure;
    }
    if (child.exitCode !== null || child.signalCode !== null) {
      throw new Error(
        `nerve exited with code ${child.exitCode}, signal ${child.signalCode} right after answering ready`
      );
    }
    return;
  }
  await sleep(pollIntervalMs);
  return waitUntilReady(child, readyzUrl, deadline, spawnError);
}

/** Kills a nerve that never became ready and waits until it is gone. */
async function kill(child: ChildProcess): Promise<void> {
  // No pid: the spawn failed, so there is no process and no "exit" to wait for.
  if (child.pid === undefined || child.exitCode !== null || child.signalCode !== null) {
    return;
  }
  const exited = once(child, "exit");
  child.kill("SIGKILL");
  await exited;
}

async function stop(child: ChildProcess, logFile: string): Promise<void> {
  if (child.exitCode === null && child.signalCode === null) {
    const exited = once(child, "exit");
    child.kill("SIGTERM");
    const timer = setTimeout(() => child.kill("SIGKILL"), stopTimeoutMs);
    await exited;
    clearTimeout(timer);
  }
  if (child.exitCode !== 0) {
    throw new Error(`nerve did not exit cleanly: code ${child.exitCode}, signal ${child.signalCode} (log: ${logFile})`);
  }
}
