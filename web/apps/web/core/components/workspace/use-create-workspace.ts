/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { UseFormSetError } from "react-hook-form";
import type { FieldError, OrganizationSize, SlugAvailability, Workspace, WorkspaceCreate } from "@nerve/api-client";
import { useTranslation } from "@nerve/i18n";
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
// hooks
import { useWorkspace } from "@/hooks/store/use-workspace";
// lib
import { ApiError } from "@/lib/api-error";
import { FIELD_ERROR_MESSAGES, errorMessageKey } from "@/lib/error-messages";
import { followInSession } from "@/lib/in-session";

/** A creation form's values: no size until one is chosen (null, which keeps a select of sizes controlled). */
export type CreationForm = Pick<WorkspaceCreate, "name" | "slug"> & { organization_size: OrganizationSize | null };

/** The slug that text typed into a slug's field makes, as the field shows it and the form sends it. */
export const slugFrom = (text: string): string => text.toLowerCase().replace(/ /g, "-");

/** The fields of the creation's form that nerve may refuse, each with the i18n key of the message under it. */
type CreationFields = Partial<Record<"name" | "slug", string>>;

/** What the creation's form shows of nerve's refusal: under the fields it names, or its reason in a toast. */
export type CreationRefusal = { kind: "fields"; fields: CreationFields } | { kind: "toast"; message: string };

/** Why nerve says a slug cannot name a new workspace (SlugAvailability.reason). */
type SlugUnavailable = NonNullable<SlugAvailability["reason"]>;

/** How a creation ends: the workspace created, or none, its slug unavailable. */
type Attempt = { kind: "created"; workspace: Workspace } | { kind: "unavailable"; reason: SlugUnavailable };

/** The message under the slug, by why nerve says it cannot name a new workspace. */
const SLUG_MESSAGES: Record<SlugUnavailable, string> = {
  taken: "workspace_creation.errors.validation.url_already_taken",
  reserved: "workspace_creation.errors.validation.url_reserved",
  invalid: "workspace_creation.errors.validation.url_alphanumeric",
};

/** A field error of nerve's on the slug: a reserved slug is not allowed; one of other characters, not of the format. */
const SLUG_CODES: Partial<Record<FieldError["code"], string>> = {
  not_allowed: SLUG_MESSAGES.reserved,
  invalid_format: SLUG_MESSAGES.invalid,
};

/** nerve's refusal of a creation (M3 design 2 W1), as the form shows it. */
export function creationRefusal(error: unknown): CreationRefusal {
  if (!(error instanceof ApiError)) return { kind: "toast", message: errorMessageKey(error) };
  if (error.problem?.code === "workspace.slug_taken") return { kind: "fields", fields: { slug: SLUG_MESSAGES.taken } };
  const named = error.problem?.errors ?? [];
  const fields: CreationFields = {};
  for (const { field, code } of named) {
    if (field === "slug") fields.slug = SLUG_CODES[code] ?? FIELD_ERROR_MESSAGES[code];
    if (field === "name") fields.name = FIELD_ERROR_MESSAGES[code];
  }
  const shown = Object.keys(fields).length;
  return shown > 0 && shown === named.length
    ? { kind: "fields", fields }
    : { kind: "toast", message: errorMessageKey(error) };
}

/**
 * Creates a workspace from a creation form's values (M3 design 2 W1, 3.10): asks nerve first whether the slug can name
 * it, and creates it only then, with the fields of WorkspaceCreate the form has; shows nerve's refusal under the fields
 * it names (the form's setError), else in a toast; says so once created. Gives the workspace created, or undefined
 * when it was not, or when the tab moved to another account meanwhile: the page is that account's then, and goes no
 * further (M3 design 7.1).
 */
export function useCreateWorkspace(): (
  form: CreationForm,
  setError: UseFormSetError<CreationForm>
) => Promise<Workspace | undefined> {
  const { checkWorkspaceSlug, createWorkspace } = useWorkspace();
  const { t } = useTranslation();

  // the slug's check decides: an unavailable slug is not sent
  const attempt = async (form: CreationForm): Promise<Attempt> => {
    const slug = await checkWorkspaceSlug(form.slug);
    if (!slug.available) return { kind: "unavailable", reason: slug.reason ?? "taken" };
    const data: WorkspaceCreate = {
      name: form.name,
      slug: form.slug,
      organization_size: form.organization_size ?? undefined,
    };
    return { kind: "created", workspace: await createWorkspace(data) };
  };

  return async (form, setError) => {
    // nerve's reasons, each under the field it is about
    const refused = (fields: CreationFields) => {
      if (fields.name) setError("name", { type: "server", message: t(fields.name) });
      if (fields.slug) setError("slug", { type: "server", message: t(fields.slug) });
    };
    let workspace: Workspace | undefined;
    await followInSession(() => attempt(form), {
      done: (outcome) => {
        if (outcome.kind === "unavailable") {
          refused({ slug: SLUG_MESSAGES[outcome.reason] });
          return;
        }
        workspace = outcome.workspace;
        setToast({
          type: TOAST_TYPE.SUCCESS,
          title: t("workspace_creation.toast.success.title"),
          message: t("workspace_creation.toast.success.message"),
        });
      },
      failed: (error) => {
        const refusal = creationRefusal(error);
        if (refusal.kind === "fields") refused(refusal.fields);
        else setToast({ type: TOAST_TYPE.ERROR, title: t("toast.error"), message: t(refusal.message) });
      },
    });
    return workspace;
  };
}
