/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { describe, expect, it } from "vitest";
import { createClient } from "@nerve/api-client";
import { ApiError, unwrap } from "./api-error";
import { authMiddleware } from "./auth/auth-middleware";
import { SessionChangedError } from "./auth/token-manager";

// unwrap on real answers of the generated client, from a fetch that answers what each test gives it.
function clientAnswering(response: Response) {
  return createClient({ baseUrl: "http://nerve.test", fetch: async () => response });
}

const problem = { status: 409, code: "identity.email_taken", title: "Conflict", detail: "The email is taken." };

describe("unwrap", () => {
  it("returns the data of a success", async () => {
    const info = { product: "nerve", signup_enabled: true };
    const answer = new Response(JSON.stringify(info), { status: 200, headers: { "Content-Type": "application/json" } });
    expect(unwrap(await clientAnswering(answer).GET("/api/v0/instance"))).toEqual(info);
  });

  it("returns nothing for a 204", async () => {
    const answer = new Response(null, { status: 204 });
    const result = await clientAnswering(answer).POST("/api/v0/auth/logout", { body: { refresh_token: "rt" } });
    expect(unwrap(result)).toBeUndefined();
  });

  it("throws a problem+json answer as an ApiError that carries the problem", async () => {
    const answer = new Response(JSON.stringify(problem), {
      status: 409,
      headers: { "Content-Type": "application/problem+json" },
    });
    const result = await clientAnswering(answer).POST("/api/v0/auth/register", {
      body: { email: "a@example.com", password: "x" },
    });
    expect(() => unwrap(result)).toThrow(ApiError);
    expect(() => unwrap(result)).toThrow(expect.objectContaining({ status: 409, problem, message: problem.detail }));
  });

  it("throws an answer without a problem, e.g. a proxy's HTML page, as an ApiError without one", async () => {
    const answer = new Response("<html>Bad Gateway</html>", { status: 502, headers: { "Content-Type": "text/html" } });
    const result = await clientAnswering(answer).GET("/api/v0/instance");
    expect(() => unwrap(result)).toThrow(
      expect.objectContaining({ status: 502, problem: undefined, message: "HTTP 502" })
    );
  });

  it("does not take a JSON answer without a problem code for a problem", async () => {
    const answer = new Response(JSON.stringify({ message: "upstream timed out" }), {
      status: 504,
      headers: { "Content-Type": "application/json" },
    });
    const result = await clientAnswering(answer).GET("/api/v0/instance");
    expect(() => unwrap(result)).toThrow(
      expect.objectContaining({ status: 504, problem: undefined, message: "HTTP 504" })
    );
  });

  it("lets a request stopped by a session change through as its SessionChangedError, not as an ApiError", async () => {
    // The renewal after the request's 401 finds that another tab changed the session, and the auth middleware
    // stops the request. Its caller gets that SessionChangedError itself, not the 401 as an ApiError, and so
    // can tell a change of session from nerve refusing the request.
    const answer = new Response(JSON.stringify({ status: 401, code: "unauthorized", title: "Unauthorized" }), {
      status: 401,
      headers: { "Content-Type": "application/problem+json" },
    });
    const changed = new SessionChangedError();
    const renewed: string[] = [];
    const api = clientAnswering(answer);
    api.use(
      authMiddleware({
        state: { status: "signed-in", loginId: "login-x" },
        accessToken: async () => "at-x",
        renew: async (sent) => {
          renewed.push(sent);
          throw changed;
        },
        endSession: async () => {},
      })
    );
    const error = await (async () => unwrap(await api.GET("/api/v0/me")))().catch((thrown: unknown) => thrown);
    expect(renewed).toEqual(["at-x"]);
    expect(error).toBe(changed);
    expect(error).not.toBeInstanceOf(ApiError);
  });
});
