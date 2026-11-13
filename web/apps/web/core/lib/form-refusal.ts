/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { useTranslation } from "@nerve/i18n";
// hooks
import { useRefusalToast } from "@/hooks/use-refusal-toast";
// lib
import { problemCode } from "@/lib/api-error";
import { errorMessageKey, fieldErrorKeys } from "@/lib/error-messages";

/** What a form shows of nerve's refusal: under the fields it names (their messages' i18n keys), or a toast. */
export type FormRefusal<F extends string> = { kind: "fields"; fields: Partial<Record<F, string>> } | { kind: "toast" };

/**
 * The messages of the field errors nerve's refusal names (fieldErrorKeys), by the form's field each is under: a field's
 * own, and an element of a list, which nerve names by its place in it (member_ids[2]), the list's (member_ids), the
 * first of its elements' messages.
 */
function messagesByField(error: unknown): Partial<Record<string, string>> {
  const messages: Partial<Record<string, string>> = {};
  for (const [path, message] of Object.entries(fieldErrorKeys(error)))
    messages[path.replace(/\[\d+\]$/, "")] ??= message;
  return messages;
}

/**
 * nerve's refusal of what a form sent (a project's, a state's, a label's, a module's), as the form shows it, among its
 * fields: a code of a value another resource already has, under the field taken names for it, whatever fields the
 * problem names; field errors under the form's fields when they name only those (a field the form shows none under is
 * not one of them, as for needsErrorBanner), an element of a list under the list's field; else nerve's reason in a
 * toast.
 */
export function formRefusal<F extends string>(
  error: unknown,
  fields: readonly F[],
  taken: Readonly<Record<string, F>>
): FormRefusal<F> {
  const field = taken[problemCode(error) ?? ""];
  if (field) {
    // the message keyed by the field's name; typed as fieldErrorKeys' messages are, since TypeScript takes those
    // as Partial<Record<F, string>> but not an object literal keyed by a value of the generic F
    const named: Partial<Record<string, string>> = { [field]: errorMessageKey(error) };
    return { kind: "fields", fields: named };
  }
  const messages = messagesByField(error);
  const named = Object.keys(messages);
  const shown: readonly string[] = fields;
  if (named.length === 0 || named.some((name) => !shown.includes(name))) return { kind: "toast" };
  return { kind: "fields", fields: messages };
}

/**
 * Shows nerve's refusal of what a form sent as formRefusal decides: each message under its field, by the form's
 * underField (its setError), in the order of fields; else nerve's reason in a toast (useRefusalToast).
 */
export function useFormRefusal<F extends string>(
  fields: readonly F[],
  taken: Readonly<Record<string, F>>
): (error: unknown, underField: (field: F, message: string) => void) => void {
  const { t } = useTranslation();
  const toastRefusal = useRefusalToast();
  return (error, underField) => {
    const refusal = formRefusal(error, fields, taken);
    if (refusal.kind === "toast") {
      toastRefusal(error);
      return;
    }
    for (const field of fields) {
      const message = refusal.fields[field];
      if (message) underField(field, t(message));
    }
  };
}
