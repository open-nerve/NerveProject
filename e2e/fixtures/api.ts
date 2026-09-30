import { createHash } from "node:crypto";

import { createClient, type components } from "@nerve/api-client";
import { expect, type TestInfo } from "@playwright/test";

import { bearer } from "./auth";

/** The typed Nerve API client, generated from api/dist/openapi.yaml. */
export type Api = ReturnType<typeof createClient>;

export type Workspace = components["schemas"]["Workspace"];
export type WorkspaceCreate = components["schemas"]["WorkspaceCreate"];
export type WorkspacePreferences = components["schemas"]["WorkspacePreferences"];
export type WorkspacePreferencesUpdate = components["schemas"]["WorkspacePreferencesUpdate"];

/** Returns a client for the nerve at baseURL. */
export function createApi(baseURL: string): Api {
  return createClient({ baseUrl: baseURL });
}

/**
 * A slug of this run of this test, as emailFor gives an address: the tests
 * of a worker share its database, and --repeat-each runs a test again in
 * the same worker. It is label and 16 hexadecimal digits, well within the
 * 48 characters of a slug.
 */
export function slugFor(testInfo: TestInfo, label = "w"): string {
  const run = `${testInfo.testId}-${testInfo.repeatEachIndex}-${testInfo.retry}`;
  return `${label}-${createHash("sha256").update(run).digest("hex").slice(0, 16)}`;
}

/** Creates a workspace with the bearer token given, its caller the admin (M3 design 3.11), and returns it. */
export async function createWorkspace(api: Api, token: string, body: WorkspaceCreate): Promise<Workspace> {
  const { data, error, response } = await api.POST("/api/v0/workspaces", { body, headers: bearer(token) });
  expect(response.status, `create the workspace ${body.slug}: ${JSON.stringify(error)}`).toBe(201);
  if (!data) {
    throw new Error(`createWorkspace ${body.slug} answered 201 without the workspace`);
  }
  return data;
}
