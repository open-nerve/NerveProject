/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useCallback, useState } from "react";
import { useTranslation } from "@nerve/i18n";
import type { TNavigationItem } from "@/components/navigation/tab-navigation-root";
import { useCopyLink, useCopyProjectLink } from "@/hooks/use-copy-link";

type UseProjectActionsProps = {
  projectId: string;
  activeItem?: TNavigationItem;
};

export const useProjectActions = ({ projectId, activeItem }: UseProjectActionsProps) => {
  const [leaveProjectModalOpen, setLeaveProjectModalOpen] = useState(false);
  const { t } = useTranslation();
  const copyLink = useCopyLink();
  const copyProjectLink = useCopyProjectLink();

  const handleLeaveProject = useCallback(() => {
    setLeaveProjectModalOpen(true);
  }, []);

  // the link of the tab open, else of the project's work items
  const handleCopyText = () =>
    activeItem ? copyLink(activeItem.href, t("project_link_copied_to_clipboard")) : copyProjectLink(projectId);

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
