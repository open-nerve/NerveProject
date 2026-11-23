/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { Problem } from "@nerve/api-client";

/** An answer of nerve that is not a success; `problem` is its problem+json body, when it has one. */
export class ApiError extends Error {
  constructor(
    readonly status: number,
    readonly problem: Problem | undefined
  ) {
    super(problem?.detail ?? problem?.title ?? `HTTP ${status}`);
    this.name = "ApiError";
  }
}

/**
 * A refusal the web foresees and so does not send: nerve's problem code of the refusal it would answer (P7b spec 2.4,
 * R4), which problemCode gives, so that it is said, and followed, as nerve's refusal by that code.
 */
export class ForeseenRefusal extends Error {
  constructor(readonly code: string) {
    super(code);
    this.name = "ForeseenRefusal";
  }
}

/**
 * nerve's problem code of an error of a call to nerve ("workspace.slug_taken"), or of a refusal the web foresees
 * (ForeseenRefusal), or undefined: no answer, no problem.
 */
export function problemCode(error: unknown): string | undefined {
  if (error instanceof ForeseenRefusal) return error.code;
  return error instanceof ApiError ? error.problem?.code : undefined;
}

/** The data of an answer of the generated client, or the answer as an ApiError when it is not a success. */
export function unwrap<T>(result: { data?: T; error?: unknown; response: Response }): T {
  if (result.response.ok) return result.data as T;
  throw new ApiError(result.response.status, isProblem(result.error) ? result.error : undefined);
}

function isProblem(body: unknown): body is Problem {
  return typeof body === "object" && body !== null && typeof (body as Partial<Problem>).code === "string";
}
