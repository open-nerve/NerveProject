/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
import { useNavigate } from "react-router";
import { useTheme } from "next-themes";
// nerve imports
import { EUserPermissions, SITE_NAME } from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";
// assets
import darkIntakeAsset from "@/app/assets/empty-state/disabled-feature/intake-dark.webp?url";
import lightIntakeAsset from "@/app/assets/empty-state/disabled-feature/intake-light.webp?url";
// components
import { PageHead } from "@/components/core/page-title";
import { DetailedEmptyState } from "@/components/empty-state/detailed-empty-state-root";
import { InboxIssueRoot } from "@/components/inbox";
// hooks
import { useProject } from "@/hooks/store/use-project";
import { useUserPermissions } from "@/hooks/store/user";
import type { Route } from "./+types/page";

/**
 * A project's intake page (P5a spec 2.5, 2.7, R19): by the route's project and the caller's role in it, never the
 * router store's. Its intake turned off: the disabled page, its way to turning it on the project admin's, and no
 * intake, so no read of its list; else the intake, titled by the project's name. The tab and the selection are the
 * address's (the intake reads them), the filters the layout's.
 */
function ProjectInboxPage({ params }: Route.ComponentProps) {
  /// router
  const navigate = useNavigate();
  const { workspaceSlug, projectId } = params;
  // theme hook
  const { resolvedTheme } = useTheme();
  // nerve hooks
  const { t } = useTranslation();
  // hooks
  const { getProjectById } = useProject();
  const { getProjectRoleByWorkspaceSlugAndProjectId } = useUserPermissions();
  // derived values
  const project = getProjectById(projectId);
  const canPerformEmptyStateActions =
    getProjectRoleByWorkspaceSlugAndProjectId(workspaceSlug, projectId) === EUserPermissions.ADMIN;
  const resolvedPath = resolvedTheme === "light" ? lightIntakeAsset : darkIntakeAsset;

  // No access to inbox
  if (project?.intake_view === false)
    return (
      <div className="flex h-full w-full items-center justify-center">
        <DetailedEmptyState
          title={t("disabled_project.empty_state.inbox.title")}
          description={t("disabled_project.empty_state.inbox.description")}
          assetPath={resolvedPath}
          primaryButton={{
            text: t("disabled_project.empty_state.inbox.primary_button.text"),
            onClick: () => {
              navigate(`/${workspaceSlug}/settings/projects/${projectId}/features/intake`);
            },
            disabled: !canPerformEmptyStateActions,
          }}
        />
      </div>
    );

  // derived values
  const pageTitle = t("inbox_issue.page_label", { workspace: project?.name || SITE_NAME });

  return (
    <div className="flex h-full flex-col">
      <PageHead title={pageTitle} />
      <div className="h-full w-full overflow-hidden">
        <InboxIssueRoot workspaceSlug={workspaceSlug} projectId={projectId} />
      </div>
    </div>
  );
}

export default observer(ProjectInboxPage);
