/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
// nerve imports
import { useTranslation } from "@nerve/i18n";
// components
import { PageHead } from "@/components/core/page-title";
import { SettingsContentWrapper } from "@/components/settings/content-wrapper";
import { WorkspaceDetails } from "@/components/workspace/settings/workspace-details";
// hooks
import { useRouteWorkspace } from "@/hooks/use-route-workspace";
// local imports
import { GeneralWorkspaceSettingsHeader } from "./header";

function GeneralWorkspaceSettingsPage() {
  // store hooks
  const workspace = useRouteWorkspace();
  const { t } = useTranslation();
  // derived values
  const pageTitle = workspace ? t("workspace_settings.page_label", { workspace: workspace.name }) : undefined;

  return (
    <SettingsContentWrapper header={<GeneralWorkspaceSettingsHeader />}>
      <PageHead title={pageTitle} />
      <WorkspaceDetails />
    </SettingsContentWrapper>
  );
}

export default observer(GeneralWorkspaceSettingsPage);
