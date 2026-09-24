/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useState, useRef } from "react";
import { useNavigate } from "react-router";
import {
  ArchiveOutline,
  LinkOutline,
  LogOutOutline,
  MoreHorizontalOutline,
  SettingsOutline,
} from "@makeplane/propel/icons";
// nerve imports
import { useTranslation } from "@nerve/i18n";
import { CustomMenu } from "@nerve/ui";

type Props = {
  workspaceSlug: string;
  project: {
    id: string;
  };
  isAuthorized: boolean;
  onCopyText: () => void;
  onLeaveProject: () => void;
};

export function ProjectActionsMenu({ workspaceSlug, project, isAuthorized, onCopyText, onLeaveProject }: Props) {
  // states
  const [isMenuActive, setIsMenuActive] = useState(false);
  // translation
  const { t } = useTranslation();
  // refs
  const actionSectionRef = useRef<HTMLDivElement | null>(null);
  // router
  const navigate = useNavigate();

  return (
    <CustomMenu
      customButton={
        <span
          ref={actionSectionRef}
          className="grid place-items-center rounded-sm p-0.5 text-placeholder hover:bg-layer-1"
          onClick={() => setIsMenuActive(!isMenuActive)}
        >
          <MoreHorizontalOutline className="size-4" />
        </span>
      }
      className="flex-shrink-0"
      customButtonClassName="grid place-items-center"
      placement="bottom-start"
      ariaLabel={t("aria_labels.projects_sidebar.toggle_quick_actions_menu")}
      useCaptureForOutsideClick
      closeOnSelect
      onMenuClose={() => setIsMenuActive(false)}
    >
      <CustomMenu.MenuItem onClick={onCopyText}>
        <span className="flex items-center justify-start gap-2">
          <LinkOutline className="h-3.5 w-3.5 stroke-[1.5]" />
          <span>{t("copy_link")}</span>
        </span>
      </CustomMenu.MenuItem>
      {isAuthorized && (
        <CustomMenu.MenuItem
          onClick={() => {
            navigate(`/${workspaceSlug}/projects/${project?.id}/archives/issues`);
          }}
        >
          <div className="flex cursor-pointer items-center justify-start gap-2">
            <ArchiveOutline className="h-3.5 w-3.5 stroke-[1.5]" />
            <span>{t("archives")}</span>
          </div>
        </CustomMenu.MenuItem>
      )}
      <CustomMenu.MenuItem
        onClick={() => {
          navigate(`/${workspaceSlug}/settings/projects/${project?.id}`);
        }}
      >
        <div className="flex cursor-pointer items-center justify-start gap-2">
          <SettingsOutline className="h-3.5 w-3.5 stroke-[1.5]" />
          <span>{t("settings")}</span>
        </div>
      </CustomMenu.MenuItem>
      {/* Leave project */}
      {!isAuthorized && (
        <CustomMenu.MenuItem onClick={onLeaveProject}>
          <div className="flex items-center justify-start gap-2">
            <LogOutOutline className="h-3.5 w-3.5 stroke-[1.5]" />
            <span>{t("leave_project")}</span>
          </div>
        </CustomMenu.MenuItem>
      )}
    </CustomMenu>
  );
}
