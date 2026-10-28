/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useRef } from "react";
import { observer } from "mobx-react";

// nerve imports
import { LockOutline, ViewsOutline } from "@makeplane/propel/icons";
import { Tooltip } from "@makeplane/propel/components/tooltip";
import type { ICustomSearchSelectOption } from "@nerve/types";
import { EViewAccess } from "@nerve/types";
import { Breadcrumbs, Header, BreadcrumbNavigationSearchDropdown } from "@nerve/ui";
// components
import { BreadcrumbLink } from "@/components/common/breadcrumb-link";
import { SwitcherIcon, SwitcherLabel } from "@/components/common/switcher-label";
import { ViewQuickActions } from "@/components/views/quick-actions";
// hooks
import { useProject } from "@/hooks/store/use-project";
import { useProjectView } from "@/hooks/store/use-project-view";
import { useNavigate } from "react-router";
// components
import { CommonProjectBreadcrumbs } from "@/components/breadcrumbs/common";

type TProps = {
  workspaceSlug: string;
  projectId: string;
  viewId: string;
};

export const ProjectViewIssuesHeader = observer(function ProjectViewIssuesHeader(props: TProps) {
  // refs
  const parentRef = useRef(null);
  // router
  const navigate = useNavigate();
  const { workspaceSlug, projectId, viewId } = props;
  // store hooks
  const { loader } = useProject();
  const { projectViewIds, getViewById } = useProjectView();

  const viewDetails = getViewById(viewId);

  if (!viewDetails) return;

  const switcherOptions = projectViewIds
    ?.map((id) => {
      const _view = id === viewId ? viewDetails : getViewById(id);
      if (!_view) return;
      return {
        value: _view.id,
        query: _view.name,
        content: <SwitcherLabel logo_props={_view.logo_props} name={_view.name} LabelIcon={ViewsOutline} />,
      };
    })
    .filter((option) => option !== undefined) as ICustomSearchSelectOption[];

  return (
    <Header>
      <Header.LeftItem>
        <Breadcrumbs isLoading={loader === "init-loader"}>
          <CommonProjectBreadcrumbs workspaceSlug={workspaceSlug} projectId={projectId} />
          <Breadcrumbs.Item
            component={
              <BreadcrumbLink
                label="Views"
                href={`/${workspaceSlug}/projects/${projectId}/views`}
                icon={<ViewsOutline className="h-4 w-4 text-tertiary" />}
              />
            }
          />
          <Breadcrumbs.Item
            component={
              <BreadcrumbNavigationSearchDropdown
                selectedItem={viewId}
                navigationItems={switcherOptions}
                onChange={(value: string) => {
                  navigate(`/${workspaceSlug}/projects/${projectId}/views/${value}`);
                }}
                title={viewDetails?.name}
                icon={
                  <Breadcrumbs.Icon>
                    <SwitcherIcon logo_props={viewDetails.logo_props} LabelIcon={ViewsOutline} size={16} />
                  </Breadcrumbs.Icon>
                }
                isLast
              />
            }
          />
        </Breadcrumbs>

        {viewDetails?.access === EViewAccess.PRIVATE ? (
          <div className="cursor-default text-tertiary">
            <Tooltip label="Private">
              <LockOutline className="h-4 w-4" />
            </Tooltip>
          </div>
        ) : (
          <></>
        )}
      </Header.LeftItem>
      <Header.RightItem className="items-center">
        <div className="hidden md:block">
          <ViewQuickActions
            parentRef={parentRef}
            customClassName="flex-shrink-0 flex items-center justify-center size-[26px] bg-layer-1/70 rounded-sm"
            projectId={projectId}
            view={viewDetails}
            workspaceSlug={workspaceSlug}
          />
        </div>
      </Header.RightItem>
    </Header>
  );
});
