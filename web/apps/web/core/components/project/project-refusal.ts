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

/** The fields of a project's forms (its creation, its general settings) under which nerve's refusals show. */
export type ProjectFormField = "name" | "identifier";

/** What a project's form shows of nerve's refusal: under the fields it names (their messages' i18n keys), or a toast. */
export type ProjectRefusal = { kind: "fields"; fields: Partial<Record<ProjectFormField, string>> } | { kind: "toast" };

/** The fields, in the order their messages show. */
const FIELDS: readonly ProjectFormField[] = ["name", "identifier"];

/** nerve's codes for a name or an identifier another project of the workspace has, each under its field. */
const TAKEN: Readonly<Record<string, ProjectFormField>> = {
  "project.name_taken": "name",
  "project.identifier_taken": "identifier",
};

/**
 * nerve's refusal of a project's creation or change (M3 design 2 P1, P3), as its form shows it: a name or an
 * identifier taken, under its field; field errors under the form's fields when they name only those
 * (needsErrorBanner: a lead nerve does not allow is not one of them); else nerve's reason in a toast.
 */
export function projectRefusal(error: unknown): ProjectRefusal {
  const code = error instanceof ApiError ? (error.problem?.code ?? "") : "";
  const taken = TAKEN[code];
  if (taken) return { kind: "fields", fields: { [taken]: PROBLEM_MESSAGES[code] } };
  if (needsErrorBanner(error, FIELDS)) return { kind: "toast" };
  const { name, identifier } = fieldErrorKeys(error);
  return { kind: "fields", fields: { name, identifier } };
}

/**
 * Shows nerve's refusal of a project's form as projectRefusal decides: each message under its field, by the form's
 * underField (its setError), else nerve's reason in a toast.
 */
export function useProjectRefusal(): (
  error: unknown,
  underField: (field: ProjectFormField, message: string) => void
) => void {
  const { t } = useTranslation();
  const toastRefusal = useRefusalToast();
  return (error, underField) => {
    const refusal = projectRefusal(error);
    if (refusal.kind === "toast") {
      toastRefusal(error);
      return;
    }
    for (const field of FIELDS) {
      const message = refusal.fields[field];
      if (message) underField(field, t(message));
    }
  };
}
