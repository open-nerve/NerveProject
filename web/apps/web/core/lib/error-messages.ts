/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { FieldError } from "@nerve/api-client";
import { ApiError } from "@/lib/api-error";
import { SessionUnavailableError } from "@/lib/auth/token-manager";

/**
 * The message of every problem code an operation lists in x-problem-codes of api/dist/openapi.yaml, which a
 * test keeps equal to the keys (M2 design 3.11, 7.3; M3 design 12 constraint 4: a phase that declares a code adds
 * its message here, in the errors namespace of both languages). Any other code, such as the unauthorized that an
 * operation needing a bearer token can also answer, gets errors.unknown from errorMessageKey.
 */
export const PROBLEM_MESSAGES: Readonly<Record<string, string>> = {
  bad_request: "errors.bad_request",
  payload_too_large: "errors.payload_too_large",
  rate_limited: "errors.rate_limited",
  internal_error: "errors.internal_error",
  validation_failed: "errors.validation_failed",
  server_busy: "errors.server_busy",
  forbidden: "errors.forbidden",
  "identity.signup_disabled": "errors.signup_disabled",
  "identity.email_taken": "errors.email_taken",
  "identity.invalid_credentials": "errors.invalid_credentials",
  "identity.account_deactivated": "errors.account_deactivated",
  "identity.refresh_token_invalid": "errors.refresh_token_invalid",
  "identity.current_password_incorrect": "errors.current_password_incorrect",
  "identity.api_token_not_found": "errors.api_token_not_found",
  "workspace.not_found": "errors.workspace_not_found",
  "workspace.creation_disabled": "errors.workspace_creation_disabled",
  "workspace.slug_taken": "errors.workspace_slug_taken",
  "workspace.member_not_found": "errors.workspace_member_not_found",
  "workspace.own_membership": "errors.workspace_own_membership",
  "workspace.sole_admin": "errors.workspace_sole_admin",
  "workspace.invitation_not_found": "errors.workspace_invitation_not_found",
  "workspace.invitation_responded": "errors.workspace_invitation_responded",
  "workspace.invitation_email_mismatch": "errors.workspace_invitation_email_mismatch",
  "project.identifier_taken": "errors.project_identifier_taken",
  "project.name_taken": "errors.project_name_taken",
  "project.not_found": "errors.project_not_found",
  "project.archived": "errors.project_archived",
  "project.member_not_found": "errors.project_member_not_found",
  "project.own_membership": "errors.project_own_membership",
  "project.role_too_high": "errors.project_role_too_high",
  "project.sole_admin": "errors.project_sole_admin",
  "project.state_name_taken": "errors.project_state_name_taken",
  "project.state_not_found": "errors.project_state_not_found",
  "project.state_last_in_group": "errors.project_state_last_in_group",
  "project.state_default": "errors.project_state_default",
  "project.label_name_taken": "errors.project_label_name_taken",
  "project.label_not_found": "errors.project_label_not_found",
};

/** The message of every FieldError.code, shown under the field it names. */
export const FIELD_ERROR_MESSAGES: Readonly<Record<FieldError["code"], string>> = {
  required: "errors.field.required",
  invalid_format: "errors.field.invalid_format",
  too_short: "errors.field.too_short",
  too_long: "errors.field.too_long",
  out_of_range: "errors.field.out_of_range",
  not_allowed: "errors.field.not_allowed",
  duplicate: "errors.field.duplicate",
  weak_password: "errors.field.weak_password",
  common_password: "errors.field.common_password",
  must_be_future: "errors.field.must_be_future",
  contains_url: "errors.field.contains_url",
};

/** The i18n key of the message for an error of a call to nerve. */
export function errorMessageKey(error: unknown): string {
  if (error instanceof ApiError) return PROBLEM_MESSAGES[error.problem?.code ?? ""] ?? "errors.unknown";
  if (error instanceof SessionUnavailableError) return "errors.unreachable";
  return "errors.unknown";
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
