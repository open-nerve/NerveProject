/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

// lib
import { formRefusal, useFormRefusal, type FormRefusal } from "@/lib/form-refusal";

/** The fields of a project's forms (its creation, its general settings) under which nerve's refusals show. */
export type ProjectFormField = "name" | "identifier";

/** What a project's form shows of nerve's refusal: under the fields it names (their messages' i18n keys), or a toast. */
export type ProjectRefusal = FormRefusal<ProjectFormField>;

/** The fields, in the order their messages show. */
const FIELDS: readonly ProjectFormField[] = ["name", "identifier"];

/** nerve's codes for a name or an identifier another project of the workspace has, each under its field. */
const TAKEN: Readonly<Record<string, ProjectFormField>> = {
  "project.name_taken": "name",
  "project.identifier_taken": "identifier",
};

/**
 * nerve's refusal of a project's creation or change (M3 design 2 P1, P3), as its form shows it (formRefusal): a name
 * or an identifier taken, under its field; field errors under the form's fields when they name only those; else
 * nerve's reason in a toast.
 */
export function projectRefusal(error: unknown): ProjectRefusal {
  return formRefusal(error, FIELDS, TAKEN);
}

/**
 * Shows nerve's refusal of a project's form as projectRefusal decides (useFormRefusal): each message under its field,
 * by the form's underField (its setError), else nerve's reason in a toast.
 */
export function useProjectRefusal(): (
  error: unknown,
  underField: (field: ProjectFormField, message: string) => void
) => void {
  return useFormRefusal(FIELDS, TAKEN);
}
