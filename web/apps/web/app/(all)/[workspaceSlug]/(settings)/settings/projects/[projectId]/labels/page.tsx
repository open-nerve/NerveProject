/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useEffect, useRef } from "react";
import { combine } from "@atlaskit/pragmatic-drag-and-drop/combine";
import { autoScrollForElements } from "@atlaskit/pragmatic-drag-and-drop-auto-scroll/element";
import { observer } from "mobx-react";
// components
import { EUserPermissions, EUserPermissionsLevel } from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";
import { NotAuthorizedView } from "@/components/auth-screens/not-authorized-view";
import { PageHead } from "@/components/core/page-title";
import { ProjectSettingsLabelList } from "@/components/labels";
import { SettingsContentWrapper } from "@/components/settings/content-wrapper";
// hooks
import { useProject } from "@/hooks/store/use-project";
import { useUserPermissions } from "@/hooks/store/user";
// local imports
import type { Route } from "./+types/page";
import { LabelsProjectSettingsHeader } from "./header";

function LabelsSettingsPage({ params }: Route.ComponentProps) {
  // router: the address's project, as the store gives it (the wrapper shows the page once it has)
  const { workspaceSlug, projectId } = params;
  // store hooks
  const { getProjectById } = useProject();
  const { allowPermissions } = useUserPermissions();
  const { t } = useTranslation();

  const project = getProjectById(projectId);
  const pageTitle = project ? `${project.name} - ${t("common.labels")}` : undefined;

  const scrollableContainerRef = useRef<HTMLDivElement | null>(null);

  // its members see its labels; its guests are told they may not, as on its states (M3 design 2 W11)
  const canPerformProjectMemberActions = allowPermissions(
    [EUserPermissions.ADMIN, EUserPermissions.MEMBER],
    EUserPermissionsLevel.PROJECT,
    workspaceSlug,
    projectId
  );

  // Enable Auto Scroll for Labels list
  useEffect(() => {
    const element = scrollableContainerRef.current;

    if (!element) return;

    return combine(
      autoScrollForElements({
        element,
      })
    );
  }, []);

  if (!canPerformProjectMemberActions) {
    return <NotAuthorizedView section="settings" isProjectView className="h-auto" />;
  }

  return (
    <SettingsContentWrapper header={<LabelsProjectSettingsHeader />}>
      <PageHead title={pageTitle} />
      <div ref={scrollableContainerRef} className="size-full">
        <ProjectSettingsLabelList workspaceSlug={workspaceSlug} projectId={projectId} />
      </div>
    </SettingsContentWrapper>
  );
}

export default observer(LabelsSettingsPage);
