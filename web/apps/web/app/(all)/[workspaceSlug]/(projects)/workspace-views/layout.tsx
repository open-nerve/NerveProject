/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { Outlet, useNavigate } from "react-router";
import { observer } from "mobx-react";
import { DEFAULT_GLOBAL_VIEWS_LIST } from "@nerve/constants";
import { EmptyStateDetailed } from "@nerve/propel/empty-state";
import useSWR from "swr";
import { useGlobalView } from "@/hooks/store/use-global-view";
import { AppHeader } from "@/components/core/app-header";
import { ContentWrapper } from "@/components/core/content-wrapper";
import { GlobalIssuesHeader } from "./header";
import type { Route } from "./+types/layout";

function GlobalIssuesLayout({ params }: Route.ComponentProps) {
  const { fetchAllGlobalViews, getViewDetailsById } = useGlobalView();
  const navigate = useNavigate();
  const { workspaceSlug } = params;
  const { isLoading } = useSWR(
    workspaceSlug ? `WORKSPACE_GLOBAL_VIEWS_${workspaceSlug}` : null,
    workspaceSlug ? () => fetchAllGlobalViews(workspaceSlug) : null,
    { revalidateIfStale: false, revalidateOnFocus: false }
  );
  return (
    <>
      <AppHeader
        header={<GlobalIssuesHeader workspaceSlug={params.workspaceSlug} globalViewId={params.globalViewId} />}
      />
      <ContentWrapper>
        {!isLoading &&
        params.globalViewId &&
        !DEFAULT_GLOBAL_VIEWS_LIST.some((view) => view.key === params.globalViewId) &&
        !getViewDetailsById(params.globalViewId) ? (
          <EmptyStateDetailed
            title="View does not exist"
            description="The view you are looking for does not exist or you don't have permission to view it."
            assetKey="view"
            actions={[
              {
                label: "Go to All work items",
                onClick: () => navigate(`/${workspaceSlug}/workspace-views/all-issues`),
                variant: "primary",
              },
            ]}
          />
        ) : (
          <Outlet />
        )}
      </ContentWrapper>
    </>
  );
}

export default observer(GlobalIssuesLayout);
