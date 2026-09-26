import type { components } from "@nerve/api-client";
import { expect, type TestInfo } from "@playwright/test";

import type { Api } from "./api";

export type AuthTokens = components["schemas"]["AuthTokens"];
export type ApiTokenCreated = components["schemas"]["ApiTokenCreated"];

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
