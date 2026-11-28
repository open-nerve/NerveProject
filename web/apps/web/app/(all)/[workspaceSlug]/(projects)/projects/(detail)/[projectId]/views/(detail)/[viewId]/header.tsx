/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useState } from "react";
import { observer } from "mobx-react";
import { useNavigate } from "react-router";
// nerve imports
import { ViewsOutline } from "@makeplane/propel/icons";
import { EUserPermissions } from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";
import { Button } from "@nerve/propel/button";
import type { ICustomSearchSelectOption } from "@nerve/types";
import { EIssuesStoreType } from "@nerve/types";
import { Breadcrumbs, Header, BreadcrumbNavigationSearchDropdown } from "@nerve/ui";
// components
import { CommonProjectBreadcrumbs } from "@/components/breadcrumbs/common";
import { BreadcrumbLink } from "@/components/common/breadcrumb-link";
import { SwitcherIcon, SwitcherLabel } from "@/components/common/switcher-label";
import { HeaderFilters } from "@/components/issues/filters";
import { ViewQuickActions } from "@/components/views/quick-actions";
import { useShownView } from "@/components/views/use-view-page-read";
// hooks
import { useCommandPalette } from "@/hooks/store/use-command-palette";
import { useProject } from "@/hooks/store/use-project";
import { useViews } from "@/hooks/store/use-views";
import { useUserPermissions } from "@/hooks/store/user";
// lib
import { useSessionSWR } from "@/lib/use-session-swr";

type TProps = {
  workspaceSlug: string;
  projectId: string;
  viewId: string;
};

/**
 * The header of a project's view's page (P6a spec 2.2, 2.6, R20; re-96 m8), of the route's project and view: its
 * crumbs to the project's views in every state of the page; and while the page shows the view (useShownView), the
 * view's crumb, whose switcher goes to another of the project's views (read once it has opened: the views list's read,
 * gated by viewEntryOn; until it answers it offers the view alone), the layouts and display of the page's draft of the
 * view (stopped while a change of the view is out, R29), a new work item of the page for the project's admins and
 * members (nerve's rule for work items, as the project
 * page's), and the view's quick actions, under the host of its page (the layout's ViewActionsHost): deleting the view
 * goes to the project's views. The list's controls and the new work item show from a medium width, the quick actions at
 * any width; the small screen's row of the page's layouts and display is the layout's (ProjectViewMobileHeader).
 */
export const ProjectViewIssuesHeader = observer(function ProjectViewIssuesHeader(props: TProps) {
  const { workspaceSlug, projectId, viewId } = props;
  // router
  const navigate = useNavigate();
  // nerve hooks
  const { t } = useTranslation();
  // store hooks
  const { loader } = useProject();
  const { getProjectRoleByWorkspaceSlugAndProjectId } = useUserPermissions();
  const { getProjectViews, fetchProjectViews, isUpdating } = useViews();
  const { toggleCreateIssueModal } = useCommandPalette();
  // derived values
  const { entry, view } = useShownView(workspaceSlug, projectId, viewId);
  const role = getProjectRoleByWorkspaceSlugAndProjectId(workspaceSlug, projectId);
  const canCreateWorkItem = role === EUserPermissions.ADMIN || role === EUserPermissions.MEMBER;
  // the project's views, read once the switcher has opened
  const [switcherOpened, setSwitcherOpened] = useState(false);
  useSessionSWR(entry && switcherOpened ? ["PROJECT_VIEWS", projectId] : null, (id) => fetchProjectViews(id));
  const switcherOptions = (getProjectViews(projectId) ?? (view ? [view] : [])).map(
    (listed): ICustomSearchSelectOption => ({
      value: listed.id,
      query: listed.name,
      content: <SwitcherLabel logo_props={listed.logo_props} name={listed.name} LabelIcon={ViewsOutline} />,
    })
  );

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
                isLast={!view}
              />
            }
            isLast={!view}
          />
          {view && (
            <Breadcrumbs.Item
              component={
                <BreadcrumbNavigationSearchDropdown
                  selectedItem={viewId}
                  navigationItems={switcherOptions}
                  onOpen={() => setSwitcherOpened(true)}
                  onChange={(value: string) => {
                    void navigate(`/${workspaceSlug}/projects/${projectId}/views/${value}`);
                  }}
                  title={view.name}
                  icon={
                    <Breadcrumbs.Icon>
                      <SwitcherIcon logo_props={view.logo_props} LabelIcon={ViewsOutline} size={16} />
                    </Breadcrumbs.Icon>
                  }
                  isLast
                />
              }
              isLast
            />
          )}
        </Breadcrumbs>
      </Header.LeftItem>
      {view && (
        <Header.RightItem className="items-center">
          <div className="hidden items-center gap-2 md:flex">
            <HeaderFilters storeType={EIssuesStoreType.PROJECT_VIEW} entityId={viewId} disabled={isUpdating(viewId)} />
            {canCreateWorkItem && (
              <Button
                variant="primary"
                size="lg"
                onClick={() => toggleCreateIssueModal(true, EIssuesStoreType.PROJECT_VIEW)}
              >
                {t("issue.add.label")}
              </Button>
            )}
          </div>
          <ViewQuickActions
            customClassName="flex-shrink-0 flex items-center justify-center size-[26px] bg-layer-1/70 rounded-sm"
            container={{ kind: "project", projectId }}
            viewId={viewId}
            workspaceSlug={workspaceSlug}
          />
        </Header.RightItem>
      )}
    </Header>
  );
});
