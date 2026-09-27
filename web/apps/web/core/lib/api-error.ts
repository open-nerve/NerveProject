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

/** The data of an answer of the generated client, or the answer as an ApiError when it is not a success. */
export function unwrap<T>(result: { data?: T; error?: unknown; response: Response }): T {
  if (result.response.ok) return result.data as T;
  throw new ApiError(result.response.status, isProblem(result.error) ? result.error : undefined);
}

function isProblem(body: unknown): body is Problem {
  return typeof body === "object" && body !== null && typeof (body as Partial<Problem>).code === "string";
}
