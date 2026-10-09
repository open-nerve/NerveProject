/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { FieldError } from "@nerve/api-client";
// lib
import { ApiError } from "@/lib/api-error";
import { FIELD_ERROR_MESSAGES, errorMessageKey } from "@/lib/error-messages";

/**
 * What the invitation form shows of nerve's refusal of its invitations (M3 design 2, W4): the message under each row
 * nerve names, when it names rows of the form alone; else nerve's reason, in a toast. Each message is an i18n key.
 */
export type InvitationRefusal =
  | { kind: "rows"; rows: { index: number; message: string }[] }
  | { kind: "toast"; message: string };

/**
 * Why an address of a row cannot be invited, by nerve's code for its email: it is an active member's; or it is
 * invited already, pending or declined, or twice in the form.
 */
const ROW_MESSAGES: Partial<Record<FieldError["code"], string>> = {
  not_allowed: "workspace_settings.settings.members.modal.errors.already_member",
  duplicate: "workspace_settings.settings.members.modal.errors.already_invited",
};

/** The refusal of a form of rows rows, as the form shows it. */
export function invitationRefusal(error: unknown, rows: number): InvitationRefusal {
  const named = error instanceof ApiError ? (error.problem?.errors ?? []) : [];
  const onRows = named.flatMap(({ field, code }) => {
    const index = Number(/^invitations\[(\d+)\]\.email$/.exec(field)?.[1] ?? -1);
    return index >= 0 && index < rows ? [{ index, message: ROW_MESSAGES[code] ?? FIELD_ERROR_MESSAGES[code] }] : [];
  });
  if (onRows.length > 0 && onRows.length === named.length) return { kind: "rows", rows: onRows };
  return { kind: "toast", message: errorMessageKey(error) };
}
