/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { useTranslation } from "@nerve/i18n";
// hooks
import { useRefusalToast } from "@/hooks/use-refusal-toast";
// lib
import { ApiError } from "@/lib/api-error";
import { PROBLEM_MESSAGES, fieldErrorKeys, needsErrorBanner } from "@/lib/error-messages";

/** What a form shows of nerve's refusal: under the fields it names (their messages' i18n keys), or a toast. */
export type FormRefusal<F extends string> = { kind: "fields"; fields: Partial<Record<F, string>> } | { kind: "toast" };

/**
 * nerve's refusal of what a form sent (a project's, a state's, a label's), as the form shows it, among its fields: a
 * code of a value another resource already has, under the field taken names for it, whatever fields the problem names;
 * field errors under the form's fields when they name only those (needsErrorBanner: a field the form shows none under
 * is not one of them); else nerve's reason in a toast.
 */
export function formRefusal<F extends string>(
  error: unknown,
  fields: readonly F[],
  taken: Readonly<Record<string, F>>
): FormRefusal<F> {
  const code = error instanceof ApiError ? (error.problem?.code ?? "") : "";
  const field = taken[code];
  if (field) {
    // by the field's name, as fieldErrorKeys gives a problem's fields
    const named: Partial<Record<string, string>> = { [field]: PROBLEM_MESSAGES[code] };
    return { kind: "fields", fields: named };
  }
  if (needsErrorBanner(error, fields)) return { kind: "toast" };
  return { kind: "fields", fields: fieldErrorKeys(error) };
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
