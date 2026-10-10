/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { UseFormSetError } from "react-hook-form";
import type { Project, ProjectCreate } from "@nerve/api-client";
import { useTranslation } from "@nerve/i18n";
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
// components
import { useProjectRefusal } from "@/components/project/project-refusal";
// hooks
import { useProject } from "@/hooks/store/use-project";
// lib
import { followInSession } from "@/lib/in-session";

/** A creation form's values: the fields of ProjectCreate the form asks for, no lead until one is picked. */
export type ProjectCreationForm = Pick<
  ProjectCreate,
  "name" | "identifier" | "description" | "network" | "logo_props"
> & {
  project_lead_id: string | null;
};

/**
 * Creates a project of the workspace from a creation form's values (M3 design 2 P1, 7.6): the fields of ProjectCreate
 * the form has, a lead only once picked; shows nerve's refusal under the fields it names (the form's setError), else
 * in a toast (useProjectRefusal); says so once created. Gives the project created, or undefined when it was not, or
 * when the tab moved to another account meanwhile: the page is that account's then, and goes no further (M3 design
 * 7.1).
 */
export function useCreateProject(): (
  workspaceSlug: string,
  form: ProjectCreationForm,
  setError: UseFormSetError<ProjectCreationForm>
) => Promise<Project | undefined> {
  const { createProject } = useProject();
  const { t } = useTranslation();
  const showRefusal = useProjectRefusal();

  return async (workspaceSlug, form, setError) => {
    const data: ProjectCreate = {
      name: form.name,
      identifier: form.identifier,
      description: form.description,
      network: form.network,
      logo_props: form.logo_props,
      project_lead_id: form.project_lead_id ?? undefined,
    };
    let created: Project | undefined;
    await followInSession(() => createProject(workspaceSlug, data), {
      done: (project) => {
        created = project;
        setToast({ type: TOAST_TYPE.SUCCESS, title: t("success"), message: t("project_created_successfully") });
      },
      failed: (error) => showRefusal(error, (field, message) => setError(field, { type: "server", message })),
    });
    return created;
  };
}
