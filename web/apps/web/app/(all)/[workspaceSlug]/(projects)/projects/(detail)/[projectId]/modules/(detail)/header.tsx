/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useRef } from "react";
import { observer } from "mobx-react";
// icons
import { ModuleOutline, RightSidePaneOutline } from "@makeplane/propel/icons";
// nerve imports
import { useTranslation } from "@nerve/i18n";
import type { ICustomSearchSelectOption } from "@nerve/types";
import { Breadcrumbs, Header, BreadcrumbNavigationSearchDropdown } from "@nerve/ui";
import { cn } from "@nerve/utils";
// components
import { BreadcrumbLink } from "@/components/common/breadcrumb-link";
import { SwitcherLabel } from "@/components/common/switcher-label";
import { ModuleQuickActions } from "@/components/modules";
// hooks
import { useModule } from "@/hooks/store/use-module";
import { useProject } from "@/hooks/store/use-project";
import { useNavigate } from "react-router";
import useLocalStorage from "@/hooks/use-local-storage";
// components
import { CommonProjectBreadcrumbs } from "@/components/breadcrumbs/common";
import { IconButton } from "@nerve/propel/icon-button";

type TProps = {
  workspaceSlug: string;
  projectId: string;
  moduleId: string;
};

export const ModuleIssuesHeader = observer(function ModuleIssuesHeader(props: TProps) {
  // refs
  const parentRef = useRef<HTMLDivElement>(null);
  // router
  const navigate = useNavigate();
  const { workspaceSlug, projectId, moduleId } = props;
  // hooks
  const { t } = useTranslation();
  // store hooks
  const { projectModuleIds, getModuleById } = useModule();
  const { loader } = useProject();
  // local storage
  const { setValue, storedValue } = useLocalStorage("module_sidebar_collapsed", "false");
  // derived values
  const isSidebarCollapsed = storedValue ? storedValue === "true" : false;
  const moduleDetails = getModuleById(moduleId);

  const toggleSidebar = () => {
    setValue(`${!isSidebarCollapsed}`);
  };

  const switcherOptions = projectModuleIds
    ?.map((id) => {
      const _module = id === moduleId ? moduleDetails : getModuleById(id);
      if (!_module) return;
      return {
        value: _module.id,
        query: _module.name,
        content: <SwitcherLabel name={_module.name} LabelIcon={ModuleOutline} />,
      };
    })
    .filter((option) => option !== undefined) as ICustomSearchSelectOption[];

  return (
    <Header>
      <Header.LeftItem>
        <div className="flex items-center gap-2">
          <Breadcrumbs onBack={() => navigate(-1)} isLoading={loader === "init-loader"}>
            <CommonProjectBreadcrumbs workspaceSlug={workspaceSlug} projectId={projectId} />
            <Breadcrumbs.Item
              component={
                <BreadcrumbLink
                  label="Modules"
                  href={`/${workspaceSlug}/projects/${projectId}/modules`}
                  icon={<ModuleOutline className="h-4 w-4 text-tertiary" />}
                  isLast
                />
              }
              isLast
            />
            <Breadcrumbs.Item
              component={
                <BreadcrumbNavigationSearchDropdown
                  selectedItem={moduleId}
                  navigationItems={switcherOptions}
                  onChange={(value: string) => {
                    navigate(`/${workspaceSlug}/projects/${projectId}/modules/${value}`);
                  }}
                  title={moduleDetails?.name}
                  icon={<ModuleOutline className="size-3.5 flex-shrink-0 text-tertiary" />}
                  isLast
                />
              }
            />
          </Breadcrumbs>
        </div>
      </Header.LeftItem>
      <Header.RightItem className="items-center">
        <IconButton
          variant="tertiary"
          size="lg"
          icon={RightSidePaneOutline}
          aria-label={t("aria_labels.toggle_details")}
          onClick={toggleSidebar}
          className={cn({
            "bg-accent-subtle text-accent-primary": !isSidebarCollapsed,
          })}
        />
        <ModuleQuickActions
          parentRef={parentRef}
          moduleId={moduleId}
          projectId={projectId}
          workspaceSlug={workspaceSlug}
          customClassName="flex-shrink-0 flex items-center justify-center bg-layer-1/70 rounded-sm size-[26px]"
        />
      </Header.RightItem>
    </Header>
  );
});
