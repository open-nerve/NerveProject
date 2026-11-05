/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { Fragment, useEffect, useState } from "react";
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

/** The route's read as its page sees it. */
export type IssueRouteRead = Pick<SWRResponse<IssueDetail | undefined>, "data" | "error" | "mutate"> & {
  /** The visit's own lookup has not answered yet, or an explicit retry of it is on its way. */
  pending: boolean;
};

const ProjectIssueDetailsLayout = observer(function ProjectIssueDetailsLayout({
  params,
}: Pick<Route.ComponentProps, "params">) {
  const { workspaceSlug, workItem } = params;
  const { rootIssueStore } = useIssueDetail();
  const { loginId } = useSession();
  const workspaceID = rootIssueStore.rootStore.workspaceRoot.getWorkspaceBySlug(workspaceSlug)?.id;
  const identity = rootIssueStore.permissionReads.identity(workspaceID, [], true);
  // Every entry into a route is a visit of its own, Back and Forward included.
  return (
    <IssueRouteVisit
      key={JSON.stringify([workspaceSlug, workItem, identity, loginId])}
      workspaceSlug={workspaceSlug}
      workItem={workItem}
      identity={identity}
    />
  );
});

/**
 * One visit of the route, whose lookup is its own: sent as the visit begins, and again only by an explicit retry. An
 * answer SWR keeps from an earlier visit of the route is not this visit's read, so nothing interactive mounts until
 * the visit's lookup has answered. The proven route owns the detail host and the input that host preserves, so a
 * reconnect alone does not read it again, as for the shared host reads (M5 design §226).
 */
const IssueRouteVisit = observer(function IssueRouteVisit({
  workspaceSlug,
  workItem,
  identity,
}: {
  workspaceSlug: string;
  workItem: string;
  identity: string;
}) {
  const { fetchIssueWithIdentifier } = useIssueDetail();
  const { data, error, isValidating, mutate } = useSessionSWR(
    ["issue-by-identifier", workspaceSlug, workItem, identity],
    (slug, identifier) => fetchIssueWithIdentifier(slug, identifier),
    { dedupingInterval: 0, revalidateOnMount: false, revalidateOnReconnect: false }
  );
  const [answered, setAnswered] = useState(false);
  useEffect(() => {
    let visiting = true;
    // mutate reads the key anew, never joining an earlier visit's request, and settles once SWR holds its answer.
    void mutate().finally(() => {
      if (visiting) setAnswered(true);
    });
    return () => {
      visiting = false;
    };
  }, [mutate]);
  const pending = !answered || isValidating;
  // One proven lookup supplies both halves; a cached identifier map is not a route read.
  const scope = JSON.stringify([data?.project_id, data?.id]);
  return (
    <Fragment key={scope}>
      <ProjectWorkItemDetailsHeader
        workspaceSlug={workspaceSlug}
        issueId={pending || error ? undefined : data?.id}
        projectId={pending || error ? undefined : (data?.project_id ?? undefined)}
      />
      <ContentWrapper className="overflow-hidden">
        <Outlet context={{ data, error, mutate, pending } satisfies IssueRouteRead} />
      </ContentWrapper>
    </Fragment>
  );
});

export default ProjectIssueDetailsLayout;
