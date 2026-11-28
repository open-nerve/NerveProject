/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useState } from "react";
import { observer } from "mobx-react";
import { useNavigate } from "react-router";
// nerve imports
import { PreferencesOutline, ViewsOutline } from "@makeplane/propel/icons";
import { DEFAULT_GLOBAL_VIEWS_LIST } from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";
import { Button } from "@nerve/propel/button";
import type { ICustomSearchSelectOption } from "@nerve/types";
import { EIssuesStoreType } from "@nerve/types";
import { Breadcrumbs, Header, BreadcrumbNavigationSearchDropdown } from "@nerve/ui";
// components
import { BreadcrumbLink } from "@/components/common/breadcrumb-link";
import { SwitcherIcon, SwitcherLabel } from "@/components/common/switcher-label";
import { DisplayFiltersSelection, FiltersDropdown } from "@/components/issues/issue-layouts/filters";
import { usePageDisplay } from "@/components/issues/use-page-display";
import { useViewsPage } from "@/components/views/actions-host";
import { ViewQuickActions } from "@/components/views/quick-actions";
import { useShownWorkspaceView } from "@/components/views/use-view-page-read";
import { WorkItemFiltersToggle } from "@/components/work-item-filters/filters-toggle";
import { DefaultViewMenu } from "@/components/workspace/views/default-view-menu";
// hooks
import { useIssues } from "@/hooks/store/use-issues";
import { useViews } from "@/hooks/store/use-views";
import { useWorkspace } from "@/hooks/store/use-workspace";
import { useUserPermissions } from "@/hooks/store/user";
// lib
import { useSessionSWR } from "@/lib/use-session-swr";
import { workspaceViewsCreatableBy, workspaceViewsOn } from "@/lib/views-entry";
// store
import { workspacePageId } from "@/store/issue/workspace/page-id";

type TProps = {
  workspaceSlug: string;
  /** The route's view: a default view's key, or one of the workspace's own views' id. */
  globalViewId: string;
};

/**
 * The header of a workspace's view's page (M7/P6b spec 2.7, 2.8, R6, R12), of the route's workspace and view:
 * its crumb of the workspace's views, a link to their list, in every state of the page; and while the page shows the
 * view (a default view once its preferences are read, a view of the workspace's own once the page's read of it seeded
 * its draft: useShownWorkspaceView), the view's crumb, whose switcher goes to another of the four default views or,
 * for the workspace's admins and members (workspaceViewsOn), of its own views (read once it has opened; until they
 * answer it offers the view shown), the toggle of the page's filters and its display (the spreadsheet's options, no
 * layouts), a new view, by the page's dialog of one, for those who may make one (workspaceViewsCreatableBy), and the
 * view's quick actions: a default view's open and copy its page's link, a view of the workspace's own's are its row's,
 * under the host of its page (the layout's ViewActionsHost). While the page's draft is being saved (Update view out,
 * ViewStore.isUpdating; spec 2.9, as P6a R29), the toggle offers no Add filter and the display stops.
 */
export const WorkspaceViewHeader = observer(function WorkspaceViewHeader(props: TProps) {
  const { workspaceSlug, globalViewId } = props;
  // router
  const navigate = useNavigate();
  // nerve hooks
  const { t } = useTranslation();
  // store hooks
  const { openCreate } = useViewsPage();
  const { getWorkspaceBySlug } = useWorkspace();
  const { getWorkspaceRoleByWorkspaceSlug } = useUserPermissions();
  const { getWorkspaceViews, fetchWorkspaceViews, isUpdating } = useViews();
  const { issuesFilter } = useIssues(EIssuesStoreType.GLOBAL);
  // derived values
  const workspace = getWorkspaceBySlug(workspaceSlug);
  const role = getWorkspaceRoleByWorkspaceSlug(workspaceSlug);
  const entry = workspaceViewsOn(role);
  const page = workspacePageId(workspaceSlug, globalViewId);
  const { display } = usePageDisplay(EIssuesStoreType.GLOBAL, page);
  const busy = isUpdating(page);
  const defaultView = DEFAULT_GLOBAL_VIEWS_LIST.find((view) => view.key === globalViewId);
  const saved = useShownWorkspaceView(workspaceSlug, globalViewId);
  const title = defaultView ? issuesFilter.getIssueFilters(page) && t(defaultView.i18n_label) : saved?.name;
  // the workspace's own views, read once the switcher has opened
  const [switcherOpened, setSwitcherOpened] = useState(false);
  useSessionSWR(
    entry && switcherOpened && workspace ? ["WORKSPACE_VIEWS", workspace.id, workspace.slug] : null,
    (id, slug) => fetchWorkspaceViews({ id, slug })
  );
  const switcherOptions: ICustomSearchSelectOption[] = [
    ...DEFAULT_GLOBAL_VIEWS_LIST.map((view) => ({
      value: view.key,
      query: t(view.i18n_label),
      content: <SwitcherLabel name={t(view.i18n_label)} LabelIcon={ViewsOutline} />,
    })),
    ...(entry ? (getWorkspaceViews(workspaceSlug) ?? (saved ? [saved] : [])) : []).map((view) => ({
      value: view.id,
      query: view.name,
      content: <SwitcherLabel logo_props={view.logo_props} name={view.name} LabelIcon={ViewsOutline} />,
    })),
  ];

  return (
    <Header>
      <Header.LeftItem>
        <Breadcrumbs>
          <Breadcrumbs.Item
            component={
              <BreadcrumbLink
                label={t("views")}
                href={`/${workspaceSlug}/workspace-views`}
                icon={<ViewsOutline className="h-4 w-4 text-tertiary" />}
                isLast={!title}
              />
            }
            isLast={!title}
          />
          {title && (
            <Breadcrumbs.Item
              component={
                <BreadcrumbNavigationSearchDropdown
                  selectedItem={globalViewId}
                  navigationItems={switcherOptions}
                  onOpen={() => setSwitcherOpened(true)}
                  onChange={(value: string) => {
                    void navigate(`/${workspaceSlug}/workspace-views/${value}`);
                  }}
                  title={title}
                  icon={
                    <Breadcrumbs.Icon>
                      <SwitcherIcon logo_props={saved?.logo_props} LabelIcon={ViewsOutline} size={16} />
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
      {title && (
        <Header.RightItem className="items-center">
          <WorkItemFiltersToggle entityType={EIssuesStoreType.GLOBAL} entityId={page} disabled={busy} />
          <FiltersDropdown
            miniIcon={<PreferencesOutline className="size-3.5" />}
            title={t("common.display")}
            placement="bottom-end"
            disabled={busy}
          >
            <DisplayFiltersSelection {...display} />
          </FiltersDropdown>
          {workspaceViewsCreatableBy(role) && (
            <Button variant="primary" size="lg" onClick={openCreate}>
              {t("workspace_views.add_view")}
            </Button>
          )}
          {defaultView ? (
            <DefaultViewMenu
              workspaceSlug={workspaceSlug}
              view={defaultView}
              customClassName="flex-shrink-0 flex items-center justify-center size-[26px] bg-layer-1/70 rounded-sm"
            />
          ) : (
            workspace && (
              <ViewQuickActions
                customClassName="flex-shrink-0 flex items-center justify-center size-[26px] bg-layer-1/70 rounded-sm"
                container={{ kind: "workspace", workspaceId: workspace.id, workspaceSlug }}
                viewId={globalViewId}
                workspaceSlug={workspaceSlug}
              />
            )
          )}
        </Header.RightItem>
      )}
    </Header>
  );
});
