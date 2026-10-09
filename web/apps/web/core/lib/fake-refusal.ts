/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

// nerve's refusal as the web app's client gives it, for the tests of what a page shows of one.

import type { FieldError } from "@nerve/api-client";
import { ApiError } from "./api-error";

/** A refusal of status and code, naming fields, each by its path in the request's body and the code of its fault. */
export function refusal(status: number, code: string, fields: Pick<FieldError, "field" | "code">[] = []): ApiError {
  return new ApiError(status, {
    status,
    code,
    title: "",
    errors: fields.map(({ field, code: fault }) => ({ field, code: fault, message: "" })),
  });
}
