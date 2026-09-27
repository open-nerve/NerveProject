import { randomBytes } from "node:crypto";

import type { components } from "@nerve/api-client";
import { expect, type BrowserContext, type Page, type TestInfo } from "@playwright/test";

import type { Api } from "./api";

export type AuthTokens = components["schemas"]["AuthTokens"];
export type ApiTokenCreated = components["schemas"]["ApiTokenCreated"];

/** The key of the token manager's record in localStorage (M2 design 7.1). */
const authKey = "nerve.auth";

/** The token manager's record: the refresh token, and the login_id of the sign-in it came from. */
export interface AuthRecord {
  refresh_token: string;
  login_id: string;
}

/** The record a sign-in with tokens writes: a new login_id of 16 random bytes in hexadecimal. */
function newRecord(tokens: AuthTokens): AuthRecord {
  return { refresh_token: tokens.refresh_token, login_id: randomBytes(16).toString("hex") };
}

/**
 * Signs the pages of context in with tokens at the nerve of baseURL (M2 design 9.5): before the first
 * page there loads, its localStorage gets the record a sign-in writes, and the page refreshes it itself.
 * Only that first load writes it: later loads, reloads and other tabs find the record the pages keep,
 * refreshed or removed, as in a browser.
 */
export async function signInContext(context: BrowserContext, baseURL: string, tokens: AuthTokens): Promise<void> {
  await context.addInitScript(
    ({ origin, key, record }) => {
      const seeded = `${key}.e2e-seeded`;
      if (window.location.origin !== origin || localStorage.getItem(seeded) !== null) {
        return;
      }
      localStorage.setItem(seeded, "1");
      localStorage.setItem(key, record);
    },
    { origin: new URL(baseURL).origin, key: authKey, record: JSON.stringify(newRecord(tokens)) }
  );
}

/** The record in the localStorage of page, or null when it has none. */
export async function recordOf(page: Page): Promise<AuthRecord | null> {
  const text = await page.evaluate((key) => localStorage.getItem(key), authKey);
  return text === null ? null : (JSON.parse(text) as AuthRecord);
}

/** A password that meets the rules and is not common. */
export const password = "Tr0ub4dor&3";

/**
 * An address of this run of this test: the tests of a worker share its
 * database, and --repeat-each runs a test again in the same worker.
 */
export function emailFor(testInfo: TestInfo, label = "user"): string {
  return `${label}-${testInfo.testId}-${testInfo.repeatEachIndex}-${testInfo.retry}@example.com`;
}

/** Signs email up through the API and returns the new session's tokens. */
export async function register(api: Api, email: string, headers: Record<string, string> = {}): Promise<AuthTokens> {
  const { data, error, response } = await api.POST("/api/v0/auth/register", {
    body: { email, password },
    headers,
  });
  expect(response.status, `register ${email}: ${JSON.stringify(error)}`).toBe(201);
  if (!data) {
    throw new Error(`register ${email} answered 201 without tokens`);
  }
  return data;
}

/** Signs email in with the fixture's password and returns the new session's tokens. */
export async function login(api: Api, email: string, headers: Record<string, string> = {}): Promise<AuthTokens> {
  const { data, error, response } = await api.POST("/api/v0/auth/login", {
    body: { email, password },
    headers,
  });
  expect(response.status, `login ${email}: ${JSON.stringify(error)}`).toBe(200);
  if (!data) {
    throw new Error(`login ${email} answered 200 without tokens`);
  }
  return data;
}

/** The Authorization header of a bearer token: an access token or a personal access token. */
export function bearer(token: string): Record<string, string> {
  return { Authorization: `Bearer ${token}` };
}

/** Creates a personal access token with the bearer token given, and returns it with its token. */
export async function createPAT(
  api: Api,
  token: string,
  body: components["schemas"]["ApiTokenCreate"] = {}
): Promise<ApiTokenCreated> {
  const { data, error, response } = await api.POST("/api/v0/me/api-tokens", { body, headers: bearer(token) });
  expect(response.status, `create a personal access token: ${JSON.stringify(error)}`).toBe(201);
  if (!data) {
    throw new Error("createApiToken answered 201 without the token");
  }
  return data;
}

/** Exchanges refreshToken for the session's next tokens. */
export async function refresh(api: Api, refreshToken: string): Promise<AuthTokens> {
  const { data, error, response } = await api.POST("/api/v0/auth/refresh", { body: { refresh_token: refreshToken } });
  expect(response.status, `refresh: ${JSON.stringify(error)}`).toBe(200);
  if (!data) {
    throw new Error("refresh answered 200 without tokens");
  }
  return data;
}
