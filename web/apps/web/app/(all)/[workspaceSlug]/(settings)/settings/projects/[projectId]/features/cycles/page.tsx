/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
// nerve imports
import { useTranslation } from "@nerve/i18n";
// components
import { EUserPermissions, EUserPermissionsLevel } from "@nerve/constants";
import { NotAuthorizedView } from "@/components/auth-screens/not-authorized-view";
import { PageHead } from "@/components/core/page-title";
import { ProjectSettingsFeatureControlItem } from "@/components/settings/project/content/feature-control-item";
import { SettingsContentWrapper } from "@/components/settings/content-wrapper";
// hooks
import { useProject } from "@/hooks/store/use-project";
import { useUserPermissions } from "@/hooks/store/user";
// local imports
import type { Route } from "./+types/page";
import { FeaturesCyclesProjectSettingsHeader } from "./header";
import { SettingsHeading } from "@/components/settings/heading";

function FeaturesCyclesSettingsPage({ params }: Route.ComponentProps) {
  // router: the route's project, as the store gives it
  const { workspaceSlug, projectId } = params;
  // store hooks
  const { allowPermissions } = useUserPermissions();
  const { getProjectById } = useProject();
  const project = getProjectById(projectId);
  // translation
  const { t } = useTranslation();
  // derived values
  const pageTitle = project?.name
    ? `${project.name} settings - ${t("project_settings.features.cycles.short_title")}`
    : undefined;
  const canPerformProjectAdminActions = allowPermissions(
    [EUserPermissions.ADMIN],
    EUserPermissionsLevel.PROJECT,
    workspaceSlug,
    projectId
  );

  if (!canPerformProjectAdminActions) {
    return <NotAuthorizedView section="settings" isProjectView className="h-auto" />;
  }

  return (
    <SettingsContentWrapper header={<FeaturesCyclesProjectSettingsHeader />}>
      <PageHead title={pageTitle} />
      <section className="w-full">
        <SettingsHeading
          title={t("project_settings.features.cycles.title")}
          description={t("project_settings.features.cycles.description")}
        />
        <div className="mt-7">
          <ProjectSettingsFeatureControlItem
            title={t("project_settings.features.cycles.toggle_title")}
            description={t("project_settings.features.cycles.toggle_description")}
            featureProperty="cycle_view"
            projectId={projectId}
            value={!!project?.cycle_view}
            workspaceSlug={workspaceSlug}
          />
        </div>
      </section>
    </SettingsContentWrapper>
  );
}

export default observer(FeaturesCyclesSettingsPage);
