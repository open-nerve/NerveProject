/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useState } from "react";
import { observer } from "mobx-react";
// nerve imports
import { DEFAULT_GLOBAL_VIEWS_LIST } from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";
import { Button } from "@nerve/propel/button";
import { ViewsOutline } from "@makeplane/propel/icons";
import type { ICustomSearchSelectOption } from "@nerve/types";
import { Breadcrumbs, Header, BreadcrumbNavigationSearchDropdown } from "@nerve/ui";
// components
import { BreadcrumbLink } from "@/components/common/breadcrumb-link";
import { SwitcherLabel } from "@/components/common/switcher-label";
import { DefaultWorkspaceViewQuickActions } from "@/components/workspace/views/default-view-quick-action";
import { CreateUpdateWorkspaceViewModal } from "@/components/workspace/views/modal";
import { WorkspaceViewQuickActions } from "@/components/workspace/views/quick-action";
// hooks
import { useGlobalView } from "@/hooks/store/use-global-view";
import { useNavigate } from "react-router";

type TProps = {
  workspaceSlug: string;
  globalViewId?: string;
};

export const GlobalIssuesHeader = observer(function GlobalIssuesHeader(props: TProps) {
  // states
  const [createViewModal, setCreateViewModal] = useState(false);
  // router
  const navigate = useNavigate();
  const { workspaceSlug, globalViewId } = props;
  // store hooks
  const { getViewDetailsById, currentWorkspaceViews } = useGlobalView();
  const { t } = useTranslation();

  const viewDetails = globalViewId ? getViewDetailsById(globalViewId) : undefined;

  const isDefaultView = DEFAULT_GLOBAL_VIEWS_LIST.find((view) => view.key === globalViewId);

  const defaultViewDetails = DEFAULT_GLOBAL_VIEWS_LIST.find((view) => view.key === globalViewId);

  const defaultOptions = DEFAULT_GLOBAL_VIEWS_LIST.map((view) => ({
    value: view.key,
    query: view.key,
    content: <SwitcherLabel name={t(view.i18n_label)} LabelIcon={ViewsOutline} />,
  }));

  const workspaceOptions = (currentWorkspaceViews || []).map((view) => {
    const _view = getViewDetailsById(view);
    if (!_view) return;
    return {
      value: _view.id,
      query: _view.name,
      content: <SwitcherLabel name={_view.name} LabelIcon={ViewsOutline} />,
    };
  });

  const switcherOptions = [...defaultOptions, ...workspaceOptions].filter(
    (option) => option !== undefined
  ) as ICustomSearchSelectOption[];
  return (
    <>
      <CreateUpdateWorkspaceViewModal isOpen={createViewModal} onClose={() => setCreateViewModal(false)} />
      <Header>
        <Header.LeftItem>
          <Breadcrumbs>
            <Breadcrumbs.Item
              component={
                <BreadcrumbLink label={t("views")} icon={<ViewsOutline className="h-4 w-4 text-tertiary" />} />
              }
            />
            <Breadcrumbs.Item
              component={
                <BreadcrumbNavigationSearchDropdown
                  selectedItem={globalViewId || ""}
                  navigationItems={switcherOptions}
                  onChange={(value: string) => {
                    navigate(`/${workspaceSlug}/workspace-views/${value}`);
                  }}
                  title={viewDetails?.name ?? t(defaultViewDetails?.i18n_label ?? "")}
                  icon={
                    <Breadcrumbs.Icon>
                      <ViewsOutline className="size-4 flex-shrink-0 text-tertiary" />
                    </Breadcrumbs.Icon>
                  }
                  isLast
                />
              }
              isLast
            />
          </Breadcrumbs>
        </Header.LeftItem>

        <Header.RightItem className="items-center">
          <Button variant="primary" size="lg" onClick={() => setCreateViewModal(true)}>
            {t("workspace_views.add_view")}
          </Button>
          <div className="hidden md:block">
            {viewDetails && <WorkspaceViewQuickActions workspaceSlug={workspaceSlug} view={viewDetails} />}
            {isDefaultView && defaultViewDetails && (
              <DefaultWorkspaceViewQuickActions workspaceSlug={workspaceSlug} view={defaultViewDetails} />
            )}
          </div>
        </Header.RightItem>
      </Header>
    </>
  );
});
