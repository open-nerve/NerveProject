/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

// nerve imports
import { setPromiseToast } from "@nerve/propel/toast";
// hooks
import { useProject } from "@/hooks/store/use-project";
import type { ProjectToggleField } from "@/store/project/project.store";

/**
 * What a flip of a feature's switch does in the project's settings, in the features list and on a feature's own page:
 * the project store turns the feature the other way from nerve's last answer, in the change's turn
 * (ProjectStore.toggleProject; v0 design 7.7), not from what the switch shows; a toast says how it went.
 */
export function useFeatureToggle(workspaceSlug: string, projectId: string) {
  const { toggleProject } = useProject();

  return (field: ProjectToggleField) => {
    if (!workspaceSlug || !projectId) return;

    setPromiseToast(toggleProject(projectId, field), {
      loading: "Updating project feature...",
      success: {
        title: "Success!",
        message: () => "Project feature updated successfully.",
      },
      error: {
        title: "Error!",
        message: () => "Something went wrong while updating project feature. Please try again.",
      },
    });
  };
}
