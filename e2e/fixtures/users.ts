import { expect } from "@playwright/test";

import type { Database } from "./db";
import { runNerve } from "./server";

// The server administrator's commands, nerve users (M2 design 3.17), on a
// worker's database: the worker's nerve sees what they change on its next
// request.

/** Every log line, so that a password in any of them shows. */
const allLogs = { NERVE_LOG__LEVEL: "debug" };

/** The start of a log line in the test configuration's text format. */
const logLine = /^time=\S+ level=[A-Z]+ msg=/m;

/**
 * Runs nerve users with args, and input as its standard input, and returns
 * its output: the one line the command prints. The command logs what it
 * did, at INFO, so its logs are there to search: neither the output nor
 * the logs, at every level, hold the input, a password.
 */
export async function nerveUsers(db: Database, args: string[], input?: string): Promise<string> {
  const { stdout, stderr } = await runNerve(["users", ...args], db.url, input, allLogs);
  const label = `nerve users ${args.join(" ")}`;
  expect(stderr, `${label} logs`).toMatch(logLine);
  expectNoPassword(label, stdout + stderr, input);
  return stdout;
}

/**
 * Runs nerve users with args, and input as its standard input, and expects
 * it to fail: exit code 1, no output, and "nerve: <message>" as the last
 * line of stderr. stderr does not hold the input, a password, either.
 */
export async function nerveUsersFails(db: Database, args: string[], message: string, input?: string): Promise<void> {
  const failure = await runNerve(["users", ...args], db.url, input, allLogs).then(
    () => undefined,
    (err: { code?: unknown; stdout?: string; stderr?: string }) => err
  );
  const label = `nerve users ${args.join(" ")}`;
  expect(failure, `${label} fails`).toBeDefined();
  expect(failure?.code, label).toBe(1);
  expect(failure?.stdout, label).toBe("");
  expect(failure?.stderr?.endsWith(`nerve: ${message}\n`), `${label}: ${failure?.stderr}`).toBe(true);
  expectNoPassword(label, failure?.stderr ?? "", input);
}

/**
 * Fails when text holds the password of input, the line the command reads
 * without its line ending: as is, in hex of either case, or in either
 * base64 alphabet. The unpadded base64 spellings also find the padded ones.
 */
function expectNoPassword(label: string, text: string, input?: string): void {
  const password = input?.replace(/\r?\n$/, "");
  if (!password) {
    return;
  }
  const bytes = Buffer.from(password);
  const hex = bytes.toString("hex");
  for (const spelling of [
    password,
    hex,
    hex.toUpperCase(),
    bytes.toString("base64").replace(/=+$/, ""),
    bytes.toString("base64url"),
  ]) {
    expect(text.includes(spelling), `${label}: the output or the logs hold the password as ${spelling}`).toBe(false);
  }
}
