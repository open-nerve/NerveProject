/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
// nerve imports
import type { View } from "@nerve/api-client";
import { useTranslation } from "@nerve/i18n";
import { Spinner } from "@nerve/ui";
// components
import type { TPowerKContext } from "@/components/power-k/core/types";
import { PowerKMenuEmptyState } from "@/components/power-k/menus/empty-state";
import { PowerKViewsMenu } from "@/components/power-k/menus/views";
// hooks
import { useProject } from "@/hooks/store/use-project";
import { useViews } from "@/hooks/store/use-views";
import { useUserPermissions } from "@/hooks/store/user";
import { useRefusalMessage } from "@/hooks/use-refusal-toast";
// lib
import { useSessionSWR } from "@/lib/use-session-swr";
import { viewEntryOn } from "@/lib/views-entry";

type Props = {
  context: TPowerKContext;
  handleSelect: (view: View) => void;
};

/**
 * The views of the route's project, to open one of (P6a spec 2.8, R13): nerve's list of them, by the views list's read,
 * whose key it shares (["PROJECT_VIEWS", projectId]: open beside the list, the two make one read), for those the ways
 * into the project's views are offered to (viewEntryOn, by the address's workspace and project, never the router's).
 * It reads each time it opens; it waits for nerve's first answer, or says why nerve could not give it. While the gate
 * is closed it reads nothing and says why at once, before any list the store may still hold: the project's views turned
 * off, in the views list's words (ViewsDisabled); else no role in it, as nerve answers the read of its views.
 */
export const PowerKOpenProjectViewsMenu = observer(function PowerKOpenProjectViewsMenu(props: Props) {
  const { context, handleSelect } = props;
  const { workspaceSlug, projectId } = context.params;
  // nerve hooks
  const { t } = useTranslation();
  // store hooks
  const { getProjectById } = useProject();
  const { getProjectRoleByWorkspaceSlugAndProjectId } = useUserPermissions();
  const { getProjectViews, fetchProjectViews } = useViews();
  const refusalMessage = useRefusalMessage();
  // derived values
  const project = getProjectById(projectId);
  const role = workspaceSlug ? getProjectRoleByWorkspaceSlugAndProjectId(workspaceSlug, projectId) : undefined;
  const entry = viewEntryOn(project, role);
  const read = useSessionSWR(entry ? ["PROJECT_VIEWS", projectId] : null, (id) => fetchProjectViews(id));
  const views = getProjectViews(projectId);

  if (!entry)
    return (
      <PowerKMenuEmptyState
        emptyText={
          project && !project.issue_views_view
            ? t("disabled_project.empty_state.view.title")
            : t("errors.project_not_found")
        }
      />
    );
  if (views === undefined)
    return read.error === undefined ? <Spinner /> : <PowerKMenuEmptyState emptyText={refusalMessage(read.error)} />;

  return <PowerKViewsMenu views={views} onSelect={handleSelect} />;
});
