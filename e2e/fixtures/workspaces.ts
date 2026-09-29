import { expect } from "@playwright/test";

import type { Database } from "./db";
import { runNerve } from "./server";

// The server administrator's commands on workspaces, nerve workspaces (M3
// design 3.11), on a worker's database: the worker's nerve sees what they
// change on its next request.

/** Runs nerve workspaces with args, and the variables of env, and returns its output: the one line it prints. */
export async function nerveWorkspaces(db: Database, args: string[], env: Record<string, string> = {}): Promise<string> {
  return (await runNerve(["workspaces", ...args], db.url, "", env)).stdout;
}

/**
 * Runs nerve workspaces with args and expects it to fail: exit code 1, no
 * output, and "nerve: <message>" as the last line of stderr.
 */
export async function nerveWorkspacesFails(db: Database, args: string[], message: string): Promise<void> {
  const failure = await runNerve(["workspaces", ...args], db.url).then(
    () => undefined,
    (err: { code?: unknown; stdout?: string; stderr?: string }) => err
  );
  const label = `nerve workspaces ${args.join(" ")}`;
  expect(failure, `${label} fails`).toBeDefined();
  expect(failure?.code, label).toBe(1);
  expect(failure?.stdout, label).toBe("");
  expect(failure?.stderr?.endsWith(`nerve: ${message}\n`), `${label}: ${failure?.stderr}`).toBe(true);
}
