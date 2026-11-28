/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
// nerve imports
import { ViewsOutline } from "@makeplane/propel/icons";
import { useTranslation } from "@nerve/i18n";
import { Button } from "@nerve/propel/button";
import { Breadcrumbs, Header } from "@nerve/ui";
// components
import { BreadcrumbLink } from "@/components/common/breadcrumb-link";
import { useViewsPage } from "@/components/views/actions-host";
// hooks
import { useUserPermissions } from "@/hooks/store/user";
// lib
import { workspaceViewsCreatableBy } from "@/lib/views-entry";

/**
 * The header of the workspace's views list (M7/P6b spec 2.7, 2.10), of the route's workspace: its crumb, the views (the
 * page itself, no link), and a new view of the workspace's own, by the page's dialog of one (ViewActionsHost), for those
 * who may make one (workspaceViewsCreatableBy: its admins and members, by their role there).
 */
export const WorkspaceViewsHeader = observer(function WorkspaceViewsHeader(props: { workspaceSlug: string }) {
  const { workspaceSlug } = props;
  // nerve hooks
  const { t } = useTranslation();
  // store hooks
  const { openCreate } = useViewsPage();
  const { getWorkspaceRoleByWorkspaceSlug } = useUserPermissions();

  return (
    <Header>
      <Header.LeftItem>
        <Breadcrumbs>
          <Breadcrumbs.Item
            component={
              <BreadcrumbLink label={t("views")} icon={<ViewsOutline className="h-4 w-4 text-tertiary" />} isLast />
            }
            isLast
          />
        </Breadcrumbs>
      </Header.LeftItem>
      {workspaceViewsCreatableBy(getWorkspaceRoleByWorkspaceSlug(workspaceSlug)) && (
        <Header.RightItem>
          <Button variant="primary" size="lg" onClick={openCreate}>
            {t("workspace_views.add_view")}
          </Button>
        </Header.RightItem>
      )}
    </Header>
  );
});
