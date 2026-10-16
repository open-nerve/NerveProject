/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { ArrowNarrowLeftOutline } from "@makeplane/propel/icons";
import { observer } from "mobx-react";
// nerve imports
import { ROLE_DETAILS } from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";
import { IconButton } from "@nerve/propel/icon-button";
// components
import { WorkspaceLogo } from "@/components/workspace/logo";
// hooks
import { useUserPermissions } from "@/hooks/store/user";
import { useRouteWorkspace } from "@/hooks/use-route-workspace";
import { useNavigate } from "react-router";

export const WorkspaceSettingsSidebarHeader = observer(function WorkspaceSettingsSidebarHeader() {
  // router
  const navigate = useNavigate();
  // store hooks
  const { getWorkspaceRoleByWorkspaceSlug } = useUserPermissions();
  const workspace = useRouteWorkspace();
  // derived values
  const currentWorkspaceRole = workspace ? getWorkspaceRoleByWorkspaceSlug(workspace.slug) : undefined;
  // translation
  const { t } = useTranslation();

  if (!currentWorkspaceRole) return null;

  return (
    <div className="shrink-0">
      <div className="flex items-center gap-1 py-3 pr-5 pl-4 text-body-md-medium">
        <IconButton
          variant="ghost"
          size="base"
          icon={ArrowNarrowLeftOutline}
          aria-label={t("common.go_back")}
          onClick={() => navigate(`/${workspace?.slug}`)}
        />
        <p>{t("workspace_settings.sidebar_title")}</p>
      </div>
      <div className="mt-1.5 flex items-center justify-between gap-2 px-5 py-0.5">
        <div className="flex items-center gap-2 truncate">
          <WorkspaceLogo
            logo={workspace?.logo_url}
            name={workspace?.name}
            classNames="shrink-0 size-8 border border-subtle"
          />
          <div className="truncate">
            <p className="truncate text-body-sm-medium">{workspace?.name}</p>
            <p className="truncate text-caption-md-regular">{t(ROLE_DETAILS[currentWorkspaceRole].i18n_title)}</p>
          </div>
        </div>
      </div>
    </div>
  );
});
