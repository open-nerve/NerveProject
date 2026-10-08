/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
import { Logo } from "@nerve/propel/emoji-icon-picker";
import { ProjectsOutline } from "@makeplane/propel/icons";
// nerve imports
import type { ICustomSearchSelectOption } from "@nerve/types";
import { BreadcrumbNavigationSearchDropdown, Breadcrumbs } from "@nerve/ui";
import { SwitcherLabel } from "@/components/common/switcher-label";
// hooks
import { useProject } from "@/hooks/store/use-project";
import { useNavigate } from "react-router";
import type { Project } from "@nerve/api-client";

type TProjectBreadcrumbProps = {
  workspaceSlug: string;
  projectId: string;
};

export const ProjectBreadcrumb = observer(function ProjectBreadcrumb(props: TProjectBreadcrumbProps) {
  const { workspaceSlug, projectId } = props;
  // router
  const navigate = useNavigate();
  // store hooks
  const { joinedProjectIds, getProjectById } = useProject();
  const currentProjectDetails = getProjectById(projectId);

  // store hooks

  if (!currentProjectDetails) return null;

  // derived values
  const switcherOptions = joinedProjectIds
    // oxlint-disable-next-line no-shadow
    .map((projectId) => {
      const project = getProjectById(projectId);
      return {
        value: projectId,
        query: project?.name,
        content: (
          <SwitcherLabel
            name={project?.name}
            logo_props={project?.logo_props}
            LabelIcon={ProjectsOutline}
            type="material"
          />
        ),
      };
    })
    .filter((option) => option !== undefined) as ICustomSearchSelectOption[];

  // helpers
  // oxlint-disable-next-line unicorn/consistent-function-scoping
  const renderIcon = (projectDetails: Project) => (
    <span className="grid size-4 flex-shrink-0 place-items-center">
      <Logo logo={projectDetails.logo_props} size={14} />
    </span>
  );

  return (
    <>
      <Breadcrumbs.Item
        component={
          <BreadcrumbNavigationSearchDropdown
            selectedItem={currentProjectDetails.id}
            navigationItems={switcherOptions}
            onChange={(value: string) => {
              navigate(`/${workspaceSlug}/projects/${value}/issues`);
            }}
            title={currentProjectDetails?.name}
            icon={renderIcon(currentProjectDetails)}
            handleOnClick={() => navigate(`/${workspaceSlug}/projects/${currentProjectDetails.id}/issues`)}
            shouldTruncate
          />
        }
        showSeparator={false}
      />
    </>
  );
});
