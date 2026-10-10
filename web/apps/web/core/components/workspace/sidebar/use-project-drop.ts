/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { useParams } from "react-router";
// hooks
import { useProject } from "@/hooks/store/use-project";
import { useRefusalToast } from "@/hooks/use-refusal-toast";
// lib
import { followInSession } from "@/lib/in-session";

/**
 * What a drop of a project in the caller's sidebar does, in its list and in the extended one: the project dropped
 * moves before the one it was dropped on, or after his last at the end of the list (ProjectStore.updateProjectSortOrder
 * reckons the place in the change's turn); a drop on itself or on nothing moves nothing. A move that fails says why,
 * followed in the session that sent it (M3 design 7.1).
 */
export function useProjectDrop() {
  const { workspaceSlug } = useParams();
  const toastRefusal = useRefusalToast();
  const { getProjectById, updateProjectSortOrder } = useProject();

  return (sourceId: string | undefined, destinationId: string | undefined, shouldDropAtEnd: boolean) => {
    if (!sourceId || !destinationId || !workspaceSlug) return;
    if (sourceId === destinationId) return;

    const source = getProjectById(sourceId);
    if (source)
      void followInSession(() => updateProjectSortOrder(source, destinationId, shouldDropAtEnd), {
        failed: toastRefusal,
      });
  };
}
