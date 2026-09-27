/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { FieldError } from "@nerve/api-client";
import { ApiError } from "@/lib/api-error";
import { SessionUnavailableError } from "@/lib/auth/token-manager";

export enum EPageTypes {
  PUBLIC = "PUBLIC",
  NON_AUTHENTICATED = "NON_AUTHENTICATED",
  ONBOARDING = "ONBOARDING",
  AUTHENTICATED = "AUTHENTICATED",
}

export enum EAuthModes {
  SIGN_IN = "SIGN_IN",
  SIGN_UP = "SIGN_UP",
}

/**
 * The message of every problem code an operation lists in x-problem-codes of api/dist/openapi.yaml, which a
 * test keeps equal to the keys (M2 design 3.11, 7.3). Any other code, such as the unauthorized that an
 * operation needing a bearer token can also answer, gets auth.errors.unknown from errorMessageKey.
 */
export const PROBLEM_MESSAGES: Readonly<Record<string, string>> = {
  bad_request: "auth.errors.bad_request",
  payload_too_large: "auth.errors.payload_too_large",
  rate_limited: "auth.errors.rate_limited",
  internal_error: "auth.errors.internal_error",
  validation_failed: "auth.errors.validation_failed",
  server_busy: "auth.errors.server_busy",
  "identity.signup_disabled": "auth.errors.signup_disabled",
  "identity.email_taken": "auth.errors.email_taken",
  "identity.invalid_credentials": "auth.errors.invalid_credentials",
  "identity.account_deactivated": "auth.errors.account_deactivated",
  "identity.refresh_token_invalid": "auth.errors.refresh_token_invalid",
  "identity.current_password_incorrect": "auth.errors.current_password_incorrect",
  "identity.api_token_not_found": "auth.errors.api_token_not_found",
};

/** The message of every FieldError.code, shown under the field it names. */
export const FIELD_ERROR_MESSAGES: Readonly<Record<FieldError["code"], string>> = {
  required: "auth.errors.field.required",
  invalid_format: "auth.errors.field.invalid_format",
  too_short: "auth.errors.field.too_short",
  too_long: "auth.errors.field.too_long",
  out_of_range: "auth.errors.field.out_of_range",
  not_allowed: "auth.errors.field.not_allowed",
  weak_password: "auth.errors.field.weak_password",
  common_password: "auth.errors.field.common_password",
  must_be_future: "auth.errors.field.must_be_future",
  contains_url: "auth.errors.field.contains_url",
};

/** The i18n key of the message for an error of a call to nerve. */
export function errorMessageKey(error: unknown): string {
  if (error instanceof ApiError) return PROBLEM_MESSAGES[error.problem?.code ?? ""] ?? "auth.errors.unknown";
  if (error instanceof SessionUnavailableError) return "auth.errors.unreachable";
  return "auth.errors.unknown";
}

/** The i18n keys of the messages for the fields a problem names, by field. */
export function fieldErrorKeys(error: unknown): Partial<Record<string, string>> {
  if (!(error instanceof ApiError)) return {};
  return Object.fromEntries((error.problem?.errors ?? []).map((e) => [e.field, FIELD_ERROR_MESSAGES[e.code]]));
}

/**
 * Whether a form with the given fields shows an error of a call to nerve above it: unless every field the
 * error names is one of them, whose messages show under the fields. An error that names a field the form
 * does not have would otherwise show nothing.
 */
export function needsErrorBanner(error: unknown, fields: readonly string[]): boolean {
  const named = Object.keys(fieldErrorKeys(error));
  return named.length === 0 || named.some((field) => !fields.includes(field));
}
