/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { UseFormSetError } from "react-hook-form";
import type { OrganizationSize, SlugAvailability, Workspace, WorkspaceCreate } from "@nerve/api-client";
import { useTranslation } from "@nerve/i18n";
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
// hooks
import { useWorkspace } from "@/hooks/store/use-workspace";
import { useRefusalToast } from "@/hooks/use-refusal-toast";
// lib
import { ApiError } from "@/lib/api-error";
import { FIELD_ERROR_MESSAGES, fieldErrorKeys, needsErrorBanner } from "@/lib/error-messages";
import { followInSession } from "@/lib/in-session";

/** A creation form's values: no size until one is chosen (null, which keeps a select of sizes controlled). */
export type CreationForm = Pick<WorkspaceCreate, "name" | "slug"> & { organization_size: OrganizationSize | null };

/** The slug that text typed into a slug's field makes, as the field shows it and the form sends it. */
export const slugFrom = (text: string): string => text.toLowerCase().replace(/ /g, "-");

/** The fields of the creation's form that nerve may refuse, each with the i18n key of the message under it. */
type CreationFields = Partial<Record<"name" | "slug", string>>;

/** What the creation's form shows of nerve's refusal: under the fields it names, or its reason in a toast. */
export type CreationRefusal = { kind: "fields"; fields: CreationFields } | { kind: "toast" };

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

/** The fields of the creation's form, under which nerve's field errors show. */
const FIELDS: readonly (keyof CreationFields)[] = ["name", "slug"];

/**
 * The slug's own messages in place of those fieldErrorKeys gives any field for two of nerve's codes: a reserved slug
 * is not allowed; one of other characters, not of the format.
 */
const SLUG_FIELD_MESSAGES: Partial<Record<string, string>> = {
  [FIELD_ERROR_MESSAGES.not_allowed]: SLUG_MESSAGES.reserved,
  [FIELD_ERROR_MESSAGES.invalid_format]: SLUG_MESSAGES.invalid,
};

/**
 * nerve's refusal of a creation (M3 design 2 W1), as the form shows it: a slug taken after its check, under the slug;
 * field errors under the form's fields when they name only those (needsErrorBanner); else nerve's reason in a toast.
 */
export function creationRefusal(error: unknown): CreationRefusal {
  if (error instanceof ApiError && error.problem?.code === "workspace.slug_taken")
    return { kind: "fields", fields: { slug: SLUG_MESSAGES.taken } };
  if (needsErrorBanner(error, FIELDS)) return { kind: "toast" };
  const { name, slug } = fieldErrorKeys(error);
  return { kind: "fields", fields: { name, slug: slug && (SLUG_FIELD_MESSAGES[slug] ?? slug) } };
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
  const toastRefusal = useRefusalToast();

  // the slug's check decides: an unavailable slug is not sent
  const attempt = async (form: CreationForm): Promise<Attempt> => {
    const availability = await checkWorkspaceSlug(form.slug);
    if (!availability.available) return { kind: "unavailable", reason: availability.reason ?? "taken" };
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
        else toastRefusal(error);
      },
    });
    return workspace;
  };
}
