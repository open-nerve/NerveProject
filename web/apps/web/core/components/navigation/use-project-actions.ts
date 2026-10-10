/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useCallback, useState } from "react";
import { useTranslation } from "@nerve/i18n";
import type { TNavigationItem } from "@/components/navigation/tab-navigation-root";
import { useCopyLink } from "@/hooks/use-copy-link";

type UseProjectActionsProps = {
  workspaceSlug: string;
  projectId: string;
  activeItem?: TNavigationItem;
};

export const useProjectActions = ({ workspaceSlug, projectId, activeItem }: UseProjectActionsProps) => {
  const [leaveProjectModalOpen, setLeaveProjectModalOpen] = useState(false);
  const { t } = useTranslation();
  const copyLink = useCopyLink();

  const handleLeaveProject = useCallback(() => {
    setLeaveProjectModalOpen(true);
  }, []);

  // the link of the tab open, else of the project's work items
  const handleCopyText = () =>
    copyLink(
      activeItem?.href ?? `/${workspaceSlug}/projects/${projectId}/issues`,
      t("project_link_copied_to_clipboard")
    );

  const handleLeaveProjectModal = useCallback((open: boolean) => {
    setLeaveProjectModalOpen(open);
  }, []);

  return {
    leaveProjectModalOpen,
    handleLeaveProject,
    handleCopyText,
    handleLeaveProjectModal,
  };
};
