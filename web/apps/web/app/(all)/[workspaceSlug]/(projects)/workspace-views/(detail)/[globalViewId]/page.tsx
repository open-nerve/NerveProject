/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
import { useNavigate } from "react-router";
// nerve imports
import { DEFAULT_GLOBAL_VIEWS_LIST, type DefaultWorkspaceViewKey } from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";
import { EIssuesStoreType } from "@nerve/types";
import { Loader } from "@nerve/ui";
// assets
import emptyView from "@/app/assets/empty-state/view.svg?url";
// components
import { EmptyState } from "@/components/common/empty-state";
import { PageHead } from "@/components/core/page-title";
import { WorkspaceViewLayoutRoot } from "@/components/issues/issue-layouts/roots/workspace-view-layout-root";
import { PlanningReadFailure } from "@/components/planning/read-failure";
import { useViewsPage } from "@/components/views/actions-host";
import { useShownWorkspaceView, useViewPageRead } from "@/components/views/use-view-page-read";
import { useDefaultViewRead } from "@/components/workspace/views/use-default-view-read";
// hooks
import { useWorkspace } from "@/hooks/store/use-workspace";
import { useUserPermissions } from "@/hooks/store/user";
// lib
import { useSession } from "@/lib/auth/use-session";
import { workspaceViewsOn } from "@/lib/views-entry";
import type { Route } from "./+types/page";

/**
 * A workspace's view's page (M7/P6b spec 2.8, R8, R11), of the route's workspace and view, a default view's key or one
 * of the workspace's own views' id (its layout is keyed by it): each session's visit its own. A default view reads no
 * view: its preferences. A view of the workspace's own is for its admins and members (workspaceViewsOn): to its guest
 * it does not exist, and nothing is read.
 */
function WorkspaceViewPage({ params }: Route.ComponentProps) {
  const { workspaceSlug, globalViewId } = params;
  // nerve hooks
  const { loginId } = useSession();
  // store hooks
  const { getWorkspaceRoleByWorkspaceSlug } = useUserPermissions();
  const defaultView = DEFAULT_GLOBAL_VIEWS_LIST.find((view) => view.key === globalViewId);

  if (defaultView) return <DefaultViewVisit key={loginId} workspaceSlug={workspaceSlug} view={defaultView} />;
  if (!workspaceViewsOn(getWorkspaceRoleByWorkspaceSlug(workspaceSlug)))
    return <ViewNotFound workspaceSlug={workspaceSlug} />;
  return <SavedViewVisit key={loginId} workspaceSlug={workspaceSlug} viewId={globalViewId} />;
}

/** The page's title: the workspace's name, and the view's. */
const WorkspaceViewTitle = observer(function WorkspaceViewTitle(props: { workspaceSlug: string; name: string }) {
  const { getWorkspaceBySlug } = useWorkspace();
  const workspace = getWorkspaceBySlug(props.workspaceSlug);
  return <PageHead title={workspace ? `${workspace.name} - ${props.name}` : undefined} />;
});

/**
 * One visit of a default view's page: its preferences read as it mounts, its own (useDefaultViewRead), then the
 * workspace's work items under them and the view's own term.
 */
const DefaultViewVisit = observer(function DefaultViewVisit(props: {
  workspaceSlug: string;
  view: { key: DefaultWorkspaceViewKey; i18n_label: string };
}) {
  const { workspaceSlug, view } = props;
  // nerve hooks
  const { t } = useTranslation();
  const read = useDefaultViewRead(workspaceSlug, view.key);

  if (!read) return null;

  return (
    <>
      <WorkspaceViewTitle workspaceSlug={workspaceSlug} name={t(view.i18n_label)} />
      <div className="h-full w-full">
        <WorkspaceViewLayoutRoot workspaceSlug={workspaceSlug} globalViewId={view.key} />
      </div>
    </>
  );
});

/**
 * One visit of a workspace's own view's page, in one session, whose read of the view is its own (useViewPageRead): it
 * waits for nerve's answer; says that the view does not exist to the caller (nerve has none, refuses it, or gives a
 * project's view of that id: useShownWorkspaceView), with the way to the workspace's views; why nerve could not read it,
 * with a retry; shows nothing of the view its own deletion took (its host leaving: it goes on to the workspace's
 * views); and else the workspace's work items under the page's draft of the view.
 */
const SavedViewVisit = observer(function SavedViewVisit(props: { workspaceSlug: string; viewId: string }) {
  const { workspaceSlug, viewId } = props;
  // store hooks
  const { pageLeaving } = useViewsPage();
  const read = useViewPageRead(EIssuesStoreType.GLOBAL, workspaceSlug, viewId);
  const view = useShownWorkspaceView(workspaceSlug, viewId);

  if (read.state === "reading")
    return (
      <Loader className="flex h-full flex-col p-5">
        <Loader.Item height="30px" width="100%" />
        <Loader.Item height="30px" width="100%" />
      </Loader>
    );
  if (read.state === "failed") return <PlanningReadFailure error={read.error} onRetry={read.retry} />;
  // the page's own deletion took the view: it is leaving for the workspace's views (P6a re-review n1)
  if (read.state === "read" && !view && pageLeaving.current) return null;
  if (read.state === "not-found" || !view) return <ViewNotFound workspaceSlug={workspaceSlug} />;

  return (
    <>
      <WorkspaceViewTitle workspaceSlug={workspaceSlug} name={view.name} />
      <div className="h-full w-full">
        <WorkspaceViewLayoutRoot workspaceSlug={workspaceSlug} globalViewId={viewId} />
      </div>
    </>
  );
});

/** The view does not exist to the caller (R11): the way to the workspace's views (R30). */
function ViewNotFound({ workspaceSlug }: { workspaceSlug: string }) {
  // router
  const navigate = useNavigate();
  // nerve hooks
  const { t } = useTranslation();
  return (
    <EmptyState
      image={emptyView}
      title={t("workspace_views.not_found.title")}
      description={t("workspace_views.not_found.description")}
      primaryButton={{
        text: t("workspace_views.not_found.action"),
        onClick: () => navigate(`/${workspaceSlug}/workspace-views`),
      }}
    />
  );
}

export default observer(WorkspaceViewPage);
