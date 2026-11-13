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
import darkEmptyState from "@/app/assets/empty-state/disabled-feature/cycles-dark.webp?url";
import lightEmptyState from "@/app/assets/empty-state/disabled-feature/cycles-light.webp?url";
// components
import { DetailedEmptyState } from "@/components/empty-state/detailed-empty-state-root";
// hooks
import { useUserPermissions } from "@/hooks/store/user";

type Props = { workspaceSlug: string; projectId: string };

/**
 * A project's cycles turned off (P5a spec 2.5, P5b spec 2.1): it says so, and only the project's admins may go to its
 * features to turn them on. The list of its cycles and a cycle's work items show it in their place.
 */
export const CyclesDisabled = observer(function CyclesDisabled({ workspaceSlug, projectId }: Props) {
  const navigate = useNavigate();
  const { resolvedTheme } = useTheme();
  const { t } = useTranslation();
  const { getProjectRoleByWorkspaceSlugAndProjectId } = useUserPermissions();
  const role = getProjectRoleByWorkspaceSlugAndProjectId(workspaceSlug, projectId);
  return (
    <div className="flex h-full w-full items-center justify-center">
      <DetailedEmptyState
        title={t("disabled_project.empty_state.cycle.title")}
        description={t("disabled_project.empty_state.cycle.description")}
        assetPath={resolvedTheme === "light" ? lightEmptyState : darkEmptyState}
        primaryButton={{
          text: t("disabled_project.empty_state.cycle.primary_button.text"),
          onClick: () => navigate(`/${workspaceSlug}/settings/projects/${projectId}/features/cycles`),
          disabled: role !== EUserPermissions.ADMIN,
        }}
      />
    </div>
  );
});
