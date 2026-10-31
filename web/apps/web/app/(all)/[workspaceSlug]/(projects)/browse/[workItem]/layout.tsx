/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { Fragment } from "react";
import { observer } from "mobx-react";
import type { IssueDetail } from "@nerve/api-client";
import type { SWRResponse } from "swr";
import { Outlet } from "react-router";
import { ContentWrapper } from "@/components/core/content-wrapper";
import { useIssueDetail } from "@/hooks/store/use-issue-detail";
import { useSession } from "@/lib/auth/use-session";
import { useSessionSWR } from "@/lib/use-session-swr";
import { ProjectWorkItemDetailsHeader } from "./header";
import type { Route } from "./+types/layout";

export type IssueRouteRead = SWRResponse<IssueDetail | undefined>;

const ProjectIssueDetailsLayout = observer(function ProjectIssueDetailsLayout({
  params,
}: Pick<Route.ComponentProps, "params">) {
  const { workspaceSlug, workItem } = params;
  const { fetchIssueWithIdentifier, rootIssueStore } = useIssueDetail();
  const { loginId } = useSession();
  const workspaceID = rootIssueStore.rootStore.workspaceRoot.getWorkspaceBySlug(workspaceSlug)?.id;
  const identity = rootIssueStore.permissionReads.identity(workspaceID, [], true);
  const read = useSessionSWR(
    ["issue-by-identifier", workspaceSlug, workItem, identity],
    (slug, identifier) => fetchIssueWithIdentifier(slug, identifier),
    { dedupingInterval: 0 }
  );
  const { data, isLoading, error } = read;
  // One proven lookup supplies both halves; a cached identifier map is not a route read.
  const scope = JSON.stringify([workspaceSlug, workItem, data?.project_id, data?.id, identity, loginId]);
  return (
    <Fragment key={scope}>
      <ProjectWorkItemDetailsHeader workspaceSlug={workspaceSlug} issueId={isLoading || error ? undefined : data?.id} />
      <ContentWrapper className="overflow-hidden">
        <Outlet context={read} />
      </ContentWrapper>
    </Fragment>
  );
});

export default ProjectIssueDetailsLayout;
