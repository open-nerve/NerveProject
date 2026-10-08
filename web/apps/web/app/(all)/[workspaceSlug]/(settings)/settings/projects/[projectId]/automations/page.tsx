/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
// nerve imports
import { EUserPermissions, EUserPermissionsLevel } from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
import type { ProjectUpdate } from "@nerve/api-client";
import { NotAuthorizedView } from "@/components/auth-screens/not-authorized-view";
import { AutoArchiveAutomation } from "@/components/automation";
import { PageHead } from "@/components/core/page-title";
import { SettingsContentWrapper } from "@/components/settings/content-wrapper";
import { SettingsHeading } from "@/components/settings/heading";
// hooks
import { useProject } from "@/hooks/store/use-project";
import { useUserPermissions } from "@/hooks/store/user";
// local imports
import type { Route } from "./+types/page";
import { AutomationsProjectSettingsHeader } from "./header";

function AutomationSettingsPage({ params }: Route.ComponentProps) {
  // router
  const { projectId } = params;
  // store hooks
  const { allowPermissions } = useUserPermissions();
  const { currentProjectDetails: projectDetails, updateProject, toggleAutoArchive } = useProject();

  const { t } = useTranslation();

  // derived values
  const canPerformProjectAdminActions = allowPermissions([EUserPermissions.ADMIN], EUserPermissionsLevel.PROJECT);

  /** Sends a change of the auto-archiving; a refusal shows a toast. */
  const send = async (change: () => Promise<unknown>) => {
    if (!projectDetails) return;

    try {
      await change();
    } catch {
      setToast({
        type: TOAST_TYPE.ERROR,
        title: "Error!",
        message: "Something went wrong. Please try again.",
      });
    }
  };
  const handleChange = (formData: Pick<ProjectUpdate, "archive_in">) => send(() => updateProject(projectId, formData));
  const handleToggle = () => send(() => toggleAutoArchive(projectId));

  // derived values
  const pageTitle = projectDetails?.name ? `${projectDetails?.name} - Automations` : undefined;

  if (!canPerformProjectAdminActions) {
    return <NotAuthorizedView section="settings" isProjectView className="h-auto" />;
  }

  return (
    <SettingsContentWrapper header={<AutomationsProjectSettingsHeader />} hugging>
      <PageHead title={pageTitle} />
      <section className={`w-full ${canPerformProjectAdminActions ? "" : "opacity-60"}`}>
        <SettingsHeading
          title={t("project_settings.automations.heading")}
          description={t("project_settings.automations.description")}
        />
        <div className="mt-6">
          <AutoArchiveAutomation handleChange={handleChange} handleToggle={handleToggle} />
        </div>
      </section>
    </SettingsContentWrapper>
  );
}

export default observer(AutomationSettingsPage);
