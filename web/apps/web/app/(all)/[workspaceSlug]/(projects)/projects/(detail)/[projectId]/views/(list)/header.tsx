/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
// ui
import { Button } from "@nerve/propel/button";
import { ViewsOutline } from "@makeplane/propel/icons";
import { useTranslation } from "@nerve/i18n";
import { Breadcrumbs, Header } from "@nerve/ui";
// components
import { CommonProjectBreadcrumbs } from "@/components/breadcrumbs/common";
import { BreadcrumbLink } from "@/components/common/breadcrumb-link";
import { ViewListHeader } from "@/components/views/view-list-header";
// hooks
import { useCommandPalette } from "@/hooks/store/use-command-palette";
import { useProject } from "@/hooks/store/use-project";
import { useUserPermissions } from "@/hooks/store/user";
// lib
import { viewEntryOn, viewsCreatableBy } from "@/lib/views-entry";

type TProps = {
  workspaceSlug: string;
  projectId: string;
};

/**
 * The header of a project's views list (P6a spec 2.2, 2.3), of the route's project: its crumbs; the view of the list
 * (its search, order and filters) for everyone the ways into the project's views are offered to (viewEntryOn); and a
 * new view, by the project's dialogs' host (ProjectLevelModals), for those who may create one (viewsCreatableBy: its
 * admins and members, while its views are turned on).
 */
export const ProjectViewsHeader = observer(function ProjectViewsHeader(props: TProps) {
  const { workspaceSlug, projectId } = props;
  // nerve hooks
  const { t } = useTranslation();
  // store hooks
  const { toggleCreateViewModal } = useCommandPalette();
  const { loader, getProjectById } = useProject();
  const { getProjectRoleByWorkspaceSlugAndProjectId } = useUserPermissions();
  // derived values
  const project = getProjectById(projectId);
  const role = getProjectRoleByWorkspaceSlugAndProjectId(workspaceSlug, projectId);

  return (
    <Header>
      <Header.LeftItem>
        <Breadcrumbs isLoading={loader === "init-loader"}>
          <CommonProjectBreadcrumbs workspaceSlug={workspaceSlug} projectId={projectId} />
          <Breadcrumbs.Item
            component={
              <BreadcrumbLink
                label={t("views")}
                href={`/${workspaceSlug}/projects/${projectId}/views`}
                icon={<ViewsOutline className="h-4 w-4 text-tertiary" />}
                isLast
              />
            }
            isLast
          />
        </Breadcrumbs>
      </Header.LeftItem>
      <Header.RightItem>
        {viewEntryOn(project, role) && <ViewListHeader projectId={projectId} />}
        {viewsCreatableBy(project, role) && (
          <Button variant="primary" size="lg" onClick={() => toggleCreateViewModal(true)}>
            {t("project_views.add_view")}
          </Button>
        )}
      </Header.RightItem>
    </Header>
  );
});
