/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { observer } from "mobx-react";
import { useNavigate } from "react-router";
import { useTheme } from "next-themes";
// nerve imports
import { EUserPermissions } from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";
// assets
import darkViewsAsset from "@/app/assets/empty-state/disabled-feature/views-dark.webp?url";
import lightViewsAsset from "@/app/assets/empty-state/disabled-feature/views-light.webp?url";
// components
import { DetailedEmptyState } from "@/components/empty-state/detailed-empty-state-root";
// hooks
import { useUserPermissions } from "@/hooks/store/user";

type Props = { workspaceSlug: string; projectId: string };

/**
 * A project's views turned off (P6a spec 2.3, 2.6), of the route's project: it says so in the views' own words and
 * picture, and only the project's admins may go to its features of views to turn them on. The views list and a view's
 * page show it in their place, and read nothing (as PlanningDisabled for cycles and modules).
 */
export const ViewsDisabled = observer(function ViewsDisabled({ workspaceSlug, projectId }: Props) {
  const navigate = useNavigate();
  const { resolvedTheme } = useTheme();
  const { t } = useTranslation();
  const { getProjectRoleByWorkspaceSlugAndProjectId } = useUserPermissions();
  const role = getProjectRoleByWorkspaceSlugAndProjectId(workspaceSlug, projectId);
  return (
    <div className="flex h-full w-full items-center justify-center">
      <DetailedEmptyState
        title={t("disabled_project.empty_state.view.title")}
        description={t("disabled_project.empty_state.view.description")}
        assetPath={resolvedTheme === "light" ? lightViewsAsset : darkViewsAsset}
        primaryButton={{
          text: t("disabled_project.empty_state.view.primary_button.text"),
          onClick: () => navigate(`/${workspaceSlug}/settings/projects/${projectId}/features/views`),
          disabled: role !== EUserPermissions.ADMIN,
        }}
      />
    </div>
  );
});
