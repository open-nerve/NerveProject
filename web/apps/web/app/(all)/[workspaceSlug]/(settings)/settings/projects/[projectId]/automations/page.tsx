/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
// nerve imports
import { EUserPermissions, EUserPermissionsLevel } from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";
import { NotAuthorizedView } from "@/components/auth-screens/not-authorized-view";
import { AutoArchiveAutomation } from "@/components/automation";
import { PageHead } from "@/components/core/page-title";
import { SettingsContentWrapper } from "@/components/settings/content-wrapper";
import { SettingsHeading } from "@/components/settings/heading";
// hooks
import { useProject } from "@/hooks/store/use-project";
import { useUserPermissions } from "@/hooks/store/user";
// local imports
import { AutomationsProjectSettingsHeader } from "./header";

function AutomationSettingsPage() {
  // store hooks
  const { allowPermissions } = useUserPermissions();
  const { currentProjectDetails: projectDetails } = useProject();

  const { t } = useTranslation();

  // derived values
  const canPerformProjectAdminActions = allowPermissions([EUserPermissions.ADMIN], EUserPermissionsLevel.PROJECT);
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
          <AutoArchiveAutomation />
        </div>
      </section>
    </SettingsContentWrapper>
  );
}

export default observer(AutomationSettingsPage);
