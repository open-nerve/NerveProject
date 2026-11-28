/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
// nerve imports
import { Input, InputGroup } from "@makeplane/propel/components/input";
import { SearchOutline } from "@makeplane/propel/icons";
import { DEFAULT_GLOBAL_VIEWS_LIST } from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";
// components
import { ListLayout } from "@/components/core/list";
import { PageHead } from "@/components/core/page-title";
import { PlanningReadFailure } from "@/components/planning/read-failure";
import { ViewListLoader } from "@/components/ui/loader/view-list-loader";
import { ViewListItem } from "@/components/views/view-list-item";
import { DefaultViewRow } from "@/components/workspace/views/default-view-row";
import { useWorkspaceViewsPageState } from "@/components/workspace/views/page-state";
// hooks
import { useViews } from "@/hooks/store/use-views";
import { useWorkspace } from "@/hooks/store/use-workspace";
import { useUserPermissions } from "@/hooks/store/user";
// lib
import { useSessionSWR } from "@/lib/use-session-swr";
import { workspaceViewsOn } from "@/lib/views-entry";
import type { Route } from "./+types/page";

/**
 * The workspace's views list (M7/P6b spec 2.7, 2.10, R12, R13), of the route's workspace: its search, which keeps the
 * rows whose names hold it, whatever its case and its spaces around (useWorkspaceViewsPageState), and says when none
 * does; the four default views, for anyone of the workspace, by their names' translations (DefaultViewRow); then, for
 * its admins and members (workspaceViewsOn), the workspace's own views as nerve lists them, read once as the page
 * opens (["WORKSPACE_VIEWS", …], which the views store keeps and the view page's switcher shares): waited for, why
 * nerve could not give them with a retry, else their rows, by the workspace as their container (ViewListItem). None of
 * its own is no empty state: the default rows are there. Its guests read none.
 */
function WorkspaceViewsPage({ params }: Route.ComponentProps) {
  const { workspaceSlug } = params;
  // nerve hooks
  const { t } = useTranslation();
  // store hooks
  const { getWorkspaceBySlug } = useWorkspace();
  const { getWorkspaceRoleByWorkspaceSlug } = useUserPermissions();
  const { getWorkspaceViews, fetchWorkspaceViews } = useViews();
  const { search, setSearch } = useWorkspaceViewsPageState();
  // derived values: of the route's workspace, never the router store's
  const workspace = getWorkspaceBySlug(workspaceSlug);
  const entry = workspaceViewsOn(getWorkspaceRoleByWorkspaceSlug(workspaceSlug));
  const views = useSessionSWR(
    entry && workspace ? ["WORKSPACE_VIEWS", workspace.id, workspace.slug] : null,
    (id, slug) => fetchWorkspaceViews({ id, slug })
  );
  const listed = getWorkspaceViews(workspaceSlug);

  if (!workspace) return null;

  /** Whether a row's name holds the search, whatever its case, its spaces around aside. */
  const holds = (name: string) => name.toLowerCase().includes(search.trim().toLowerCase());
  const defaults = DEFAULT_GLOBAL_VIEWS_LIST.filter((view) => holds(t(view.i18n_label)));
  const failed = entry && views.error !== undefined;
  const saved = entry && !failed ? listed?.filter((view) => holds(view.name)) : [];

  return (
    <>
      <PageHead title={`${workspace.name} - ${t("views")}`} />
      <div className="flex h-full w-full flex-col overflow-hidden">
        <div className="flex h-11 w-full items-center overflow-hidden border-b border-subtle px-5 py-3">
          <InputGroup size="2xl">
            <SearchOutline className="text-secondary" width={14} height={14} />
            <Input
              size="2xl"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder={t("common.search.label")}
              aria-label={t("common.search.label")}
            />
          </InputGroup>
        </div>
        <ListLayout>
          {defaults.map((view) => (
            <DefaultViewRow key={view.key} workspaceSlug={workspaceSlug} view={view} />
          ))}
          {saved?.map((view) => (
            <ViewListItem
              key={view.id}
              view={view}
              workspaceSlug={workspaceSlug}
              container={{ kind: "workspace", workspaceId: workspace.id, workspaceSlug }}
            />
          ))}
          {failed && <PlanningReadFailure error={views.error} onRetry={() => void views.mutate()} />}
          {saved === undefined && <ViewListLoader />}
          {defaults.length === 0 && saved?.length === 0 && !failed && (
            <p className="px-5 py-4 text-13 text-tertiary">{t("common.search.no_matches_found")}</p>
          )}
        </ListLayout>
      </div>
    </>
  );
}

export default observer(WorkspaceViewsPage);
