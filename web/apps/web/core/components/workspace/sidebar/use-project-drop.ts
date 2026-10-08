/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { useParams } from "react-router";
// nerve imports
import { useTranslation } from "@nerve/i18n";
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
// hooks
import { useProject } from "@/hooks/store/use-project";

/**
 * What a drop of a project in the caller's sidebar does, in its list and in the extended one: the project dropped
 * moves before the one it was dropped on, or after his last at the end of the list (ProjectStore.updateProjectSortOrder
 * reckons the place in the change's turn); a drop on itself or on nothing moves nothing; a move that fails shows a
 * toast.
 */
export function useProjectDrop() {
  const { workspaceSlug } = useParams();
  const { t } = useTranslation();
  const { getProjectById, updateProjectSortOrder } = useProject();

  return (sourceId: string | undefined, destinationId: string | undefined, shouldDropAtEnd: boolean) => {
    if (!sourceId || !destinationId || !workspaceSlug) return;
    if (sourceId === destinationId) return;

    const source = getProjectById(sourceId);
    if (source)
      updateProjectSortOrder(source, destinationId, shouldDropAtEnd).catch(() => {
        setToast({
          type: TOAST_TYPE.ERROR,
          title: t("error"),
          message: t("something_went_wrong"),
        });
      });
  };
}
