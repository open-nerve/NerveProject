/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
import { EUserPermissions, EUserPermissionsLevel } from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";
// components
import { NotAuthorizedView } from "@/components/auth-screens/not-authorized-view";
import { PageHead } from "@/components/core/page-title";
import { ProjectStateRoot } from "@/components/project-states";
import { SettingsContentWrapper } from "@/components/settings/content-wrapper";
import { SettingsHeading } from "@/components/settings/heading";
// hook
import { useProject } from "@/hooks/store/use-project";
import { useUserPermissions } from "@/hooks/store/user";
// local imports
import type { Route } from "./+types/page";
import { StatesProjectSettingsHeader } from "./header";

function StatesSettingsPage({ params }: Route.ComponentProps) {
  // router: the address's project, as the store gives it (the wrapper shows the page once it has)
  const { workspaceSlug, projectId } = params;
  // store
  const { getProjectById } = useProject();
  const { allowPermissions } = useUserPermissions();

  const { t } = useTranslation();

  // derived values
  const project = getProjectById(projectId);
  const pageTitle = project ? `${project.name} - ${t("common.states")}` : undefined;
  // its members see its states; its guests are told they may not (M3 design 2 W11)
  const canPerformProjectMemberActions = allowPermissions(
    [EUserPermissions.ADMIN, EUserPermissions.MEMBER],
    EUserPermissionsLevel.PROJECT,
    workspaceSlug,
    projectId
  );

  if (!canPerformProjectMemberActions) {
    return <NotAuthorizedView section="settings" isProjectView className="h-auto" />;
  }

  return (
    <SettingsContentWrapper header={<StatesProjectSettingsHeader />}>
      <PageHead title={pageTitle} />
      <div className="w-full">
        <SettingsHeading
          title={t("project_settings.states.heading")}
          description={t("project_settings.states.description")}
        />
        <div className="mt-6">
          <ProjectStateRoot workspaceSlug={workspaceSlug} projectId={projectId} />
        </div>
      </div>
    </SettingsContentWrapper>
  );
}

export default observer(StatesSettingsPage);
