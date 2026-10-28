/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useRef } from "react";
import { observer } from "mobx-react";
// icons
import { CyclesOutline, RightSidePaneOutline } from "@makeplane/propel/icons";
// nerve imports
import { useTranslation } from "@nerve/i18n";
import { IconButton } from "@nerve/propel/icon-button";
import type { ICustomSearchSelectOption } from "@nerve/types";
import { Breadcrumbs, BreadcrumbNavigationSearchDropdown, Header } from "@nerve/ui";
import { cn } from "@nerve/utils";
// components
import { BreadcrumbLink } from "@/components/common/breadcrumb-link";
import { SwitcherLabel } from "@/components/common/switcher-label";
import { CycleQuickActions } from "@/components/cycles/quick-actions";
// hooks
import { useCycle } from "@/hooks/store/use-cycle";
import { useProject } from "@/hooks/store/use-project";
import { useNavigate } from "react-router";
import useLocalStorage from "@/hooks/use-local-storage";
// components
import { CommonProjectBreadcrumbs } from "@/components/breadcrumbs/common";

type TProps = {
  workspaceSlug: string;
  projectId: string;
  cycleId: string;
};

export const CycleIssuesHeader = observer(function CycleIssuesHeader(props: TProps) {
  // refs
  const parentRef = useRef<HTMLDivElement>(null);
  // router
  const navigate = useNavigate();
  const { workspaceSlug, projectId, cycleId } = props;
  // i18n
  const { t } = useTranslation();
  // store hooks
  const { currentProjectCycleIds, getCycleById } = useCycle();
  const { loader } = useProject();

  const { setValue, storedValue } = useLocalStorage("cycle_sidebar_collapsed", false);

  const isSidebarCollapsed = storedValue === true;
  const toggleSidebar = () => {
    setValue(!isSidebarCollapsed);
  };

  // derived values
  const cycleDetails = getCycleById(cycleId);

  const switcherOptions = currentProjectCycleIds
    ?.map((id) => {
      const _cycle = id === cycleId ? cycleDetails : getCycleById(id);
      if (!_cycle) return;
      return {
        value: _cycle.id,
        query: _cycle.name,
        content: <SwitcherLabel name={_cycle.name} LabelIcon={CyclesOutline} />,
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
                  label="Cycles"
                  href={`/${workspaceSlug}/projects/${projectId}/cycles`}
                  icon={<CyclesOutline className="h-4 w-4 text-tertiary" />}
                />
              }
            />
            <Breadcrumbs.Item
              component={
                <BreadcrumbNavigationSearchDropdown
                  selectedItem={cycleId}
                  navigationItems={switcherOptions}
                  onChange={(value: string) => {
                    navigate(`/${workspaceSlug}/projects/${projectId}/cycles/${value}`);
                  }}
                  title={cycleDetails?.name}
                  icon={
                    <Breadcrumbs.Icon>
                      <CyclesOutline className="size-4 flex-shrink-0 text-tertiary" />
                    </Breadcrumbs.Icon>
                  }
                  isLast
                />
              }
              isLast
            />
          </Breadcrumbs>
        </div>
      </Header.LeftItem>
      <Header.RightItem className="items-center">
        <div className="hidden items-center gap-2 md:flex">
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
          <CycleQuickActions
            parentRef={parentRef}
            cycleId={cycleId}
            projectId={projectId}
            workspaceSlug={workspaceSlug}
            customClassName="flex-shrink-0 flex items-center justify-center size-[26px] bg-layer-1/70 rounded-sm"
          />
        </div>
      </Header.RightItem>
    </Header>
  );
});
