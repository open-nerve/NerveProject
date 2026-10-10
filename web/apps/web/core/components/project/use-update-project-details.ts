/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { UseFormSetError } from "react-hook-form";
import type { Project, ProjectUpdate } from "@nerve/api-client";
import { useTranslation } from "@nerve/i18n";
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
// hooks
import { useProject } from "@/hooks/store/use-project";
// lib
import { followInSession } from "@/lib/in-session";
// local imports
import { useProjectRefusal, type ProjectFormField } from "./project-refusal";

/** A project's general settings: the fields of ProjectUpdate its general page edits (M3 design 7.6). */
export type ProjectDetails = Required<
  Pick<ProjectUpdate, "name" | "identifier" | "description" | "network" | "logo_props" | "timezone">
>;

/** A project's general settings as nerve last answered them. */
export const projectDetailsOf = (project: Project): ProjectDetails => ({
  name: project.name,
  identifier: project.identifier,
  description: project.description,
  network: project.network,
  logo_props: project.logo_props,
  timezone: project.timezone,
});

/**
 * Changes a project's general settings to the values its page holds (M3 design 2 P3, 7.6). An identifier changed is
 * asked of nerve first (checkProjectIdentifier): one another project of the workspace has is said under it, and
 * nothing is sent. Else the page's fields are sent; nerve's refusal shows under the fields it names (the form's
 * setError), else in a toast (useProjectRefusal), and a change made is said so. Settles once the page has followed
 * the change, which it does only in the session the change was sent in (M3 design 7.1).
 */
export function useUpdateProjectDetails(): (
  project: Project,
  workspaceSlug: string,
  details: ProjectDetails,
  setError: UseFormSetError<ProjectDetails>
) => Promise<void> {
  const { updateProject, checkProjectIdentifier } = useProject();
  const { t } = useTranslation();
  const showRefusal = useProjectRefusal();

  return (project, workspaceSlug, details, setError) => {
    const underField = (field: ProjectFormField, message: string) => setError(field, { type: "server", message });
    const data: ProjectUpdate = {
      name: details.name,
      identifier: details.identifier,
      description: details.description,
      network: details.network,
      logo_props: details.logo_props,
      timezone: details.timezone,
    };
    return followInSession(
      async (): Promise<"updated" | "identifier_taken"> => {
        if (details.identifier !== project.identifier) {
          const { available } = await checkProjectIdentifier(workspaceSlug, details.identifier);
          if (!available) return "identifier_taken";
        }
        await updateProject(project.id, data);
        return "updated";
      },
      {
        done: (outcome) => {
          if (outcome === "identifier_taken") {
            underField("identifier", t("errors.project_identifier_taken"));
            return;
          }
          setToast({
            type: TOAST_TYPE.SUCCESS,
            title: t("toast.success"),
            message: t("project_settings.general.toast.success"),
          });
        },
        failed: (error) => showRefusal(error, underField),
      }
    );
  };
}
