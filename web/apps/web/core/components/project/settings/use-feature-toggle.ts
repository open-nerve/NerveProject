/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

// nerve imports
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
// hooks
import { useProject } from "@/hooks/store/use-project";
import { useRefusalToast } from "@/hooks/use-refusal-toast";
// lib
import { followInSession } from "@/lib/in-session";
import type { ProjectToggleField } from "@/store/project/project.store";

/**
 * What a flip of a feature's switch does in the project's settings, in the features list and on a feature's own page:
 * the project store turns the feature the other way from nerve's last answer, in the change's turn
 * (ProjectStore.toggleProject; v0 design 7.7), not from what the switch shows. A toast says it is done, or nerve's
 * reason for refusing it, only in the session the flip was sent in (M3 design 7.1).
 */
export function useFeatureToggle(workspaceSlug: string, projectId: string) {
  const { toggleProject } = useProject();
  const toastRefusal = useRefusalToast();

  return (field: ProjectToggleField) => {
    if (!workspaceSlug || !projectId) return;

    void followInSession(() => toggleProject(projectId, field), {
      done: () =>
        setToast({
          type: TOAST_TYPE.SUCCESS,
          title: "Success!",
          message: "Project feature updated successfully.",
        }),
      failed: toastRefusal,
    });
  };
}
